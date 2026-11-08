"""Restore use case: plan the restore, resolve conflicts and apply it safely.

The session keeps the safety invariants in one place: verify the archive before
extracting, extract only into a temporary directory, create a snapshot before
writing, re-check destinations immediately before applying, and never write
when the review is cancelled.

Classification of every manifest file against its destination:

- ``NEW``: the destination does not exist; it is created.
- ``IDENTICAL``: content matches (sha256 for regular files, target for symlinks,
  and CRLF-normalised text for text files); it is omitted.
- ``DIFFERENT``: content or type differs; a content conflict is added to
  ``RestorePlan.conflicts`` for an explicit decision, while a type change keeps
  the existing overwrite behaviour.
- ``UNSUPPORTED``: the destination is a directory, socket, pipe...; it is
  omitted.
"""

from __future__ import annotations

import hashlib
import shutil
import stat
import tempfile
from collections.abc import Sequence
from dataclasses import dataclass, field
from enum import StrEnum
from pathlib import Path
from typing import Literal

from kamino.archive import (
    APPS_DIR,
    AppEntry,
    FileEntry,
    Manifest,
    extract_archive,
    read_file_from_archive,
    read_manifest,
    verify_archive,
)
from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.conflicts import (
    MAX_TEXT_BYTES,
    Conflict,
    ConflictDecisions,
    FileAction,
    FileDecision,
    HunkDecision,
    LocalFile,
    apply_all,
    build_conflict,
    compose,
    validate_merge,
)
from kamino.detect import is_installed
from kamino.errors import (
    ConcurrentChangeError,
    ConflictError,
    DecisionRequiredError,
    InvalidMergeError,
    PathError,
)
from kamino.ghostty import GHOSTTY_APP_ID, MAX_CONFIG_BYTES, absolute_references
from kamino.paths import SafeHome, to_absolute, to_relative
from kamino.snapshot import SNAPSHOT_APP_ID, create_snapshot


class RestoreStatus(StrEnum):
    """Classification of a manifest file against its destination (§5.2.4)."""

    NEW = "new"
    IDENTICAL = "identical"
    DIFFERENT = "different"
    UNSUPPORTED = "unsupported"


class ResultCategory(StrEnum):
    """Final outcome of an active manifest file."""

    RESTORED = "restored"
    MERGED = "merged"
    OMITTED = "omitted"


@dataclass(frozen=True)
class RestoreEntry:
    """A manifest file classified for restoration."""

    app_id: str
    path: str
    destination: Path
    size: int
    sha256: str | None
    mode: str
    link: str | None
    status: RestoreStatus


@dataclass(frozen=True)
class RestorePlan:
    """The restore plan: active manifest files, conflicts and omissions."""

    entries: list[RestoreEntry] = field(default_factory=list)
    notices: list[str] = field(default_factory=list)
    conflicts: list[Conflict] = field(default_factory=list)
    omitted: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class RestoreResult:
    """Result of a restore (§5.2.9).

    ``restored`` contains files written with the backup content, ``merged``
    contains real block mixes and ``omitted`` contains everything left
    untouched (not installed, identical, unsupported, omitted by decision and
    all-local reviews).
    """

    plan: RestorePlan
    manifest: Manifest
    output: Path | None
    dry_run: bool
    snapshot: Path | None
    restored: list[str]
    merged: list[str]
    omitted: list[str]


@dataclass(frozen=True)
class RestoreClassification:
    """Prospective result categories, used by the final confirmation screen."""

    restored: list[str]
    merged: list[str]
    omitted: list[str]


@dataclass(frozen=True)
class _DestinationFingerprint:
    kind: Literal["missing", "file", "symlink", "special"]
    sha256: str | None = None
    link: str | None = None


@dataclass(frozen=True)
class _ResolvedFile:
    path: str
    destination: Path
    content: bytes | None
    link: str | None
    mode: str
    category: ResultCategory
    write: bool
    local_text: str | None = None
    backup_text: str | None = None
    merged_text: str | None = None


class RestoreSession:
    """Prepared restore with ``plan``, ``preview`` and ``apply`` steps."""

    def __init__(
        self,
        backup_file: Path,
        *,
        home: Path,
        manifest: Manifest,
        restorable_apps: Sequence[AppEntry],
        not_installed_apps: Sequence[AppEntry],
        snapshots_dir: Path | None,
    ) -> None:
        self.backup_file = backup_file
        self.home = home
        self.manifest = manifest
        self.restorable_apps = list(restorable_apps)
        self.not_installed_apps = list(not_installed_apps)
        self.snapshots_dir = snapshots_dir
        self.notices = _ghostty_portability_notices(manifest, backup_file, home)
        self._staging: Path | None = None
        self._plan: RestorePlan | None = None
        self._fingerprints: dict[str, _DestinationFingerprint] = {}
        self._closed = False

    @property
    def staging(self) -> Path | None:
        return self._staging

    @property
    def snapshot_directory(self) -> Path:
        """Directory where the previous-state snapshot will be written."""
        if self.snapshots_dir is not None:
            return self.snapshots_dir
        return self.home / "Library" / "Application Support" / "kamino" / "snapshots"

    def plan(self, selected: list[str] | None = None, *, dry_run: bool = False) -> RestorePlan:
        """Classify the chosen apps and build every content conflict.

        With ``dry_run`` the archive is neither verified nor extracted; the
        conflict contents are read directly from it so the caller can print the
        diffs.
        """
        self._require_open()
        if not dry_run:
            verify_archive(self.backup_file, self.manifest)
            self.home.mkdir(parents=True, exist_ok=True)
            self._reset_staging()

        entries: list[RestoreEntry] = []
        conflicts: list[Conflict] = []
        omitted: list[str] = list(
            file.path for app in self.not_installed_apps for file in app.files
        )
        seen_conflicts: set[str] = set()
        fingerprints: dict[str, _DestinationFingerprint] = {}
        safe = SafeHome(self.home) if self.home.exists() else None
        try:
            for app, file in self._active_files(selected):
                destination = to_absolute(file.path, home=self.home)
                try:
                    local = _read_local(safe, file.path) if safe is not None else None
                except FileNotFoundError:
                    local = None
                if local is None:
                    status = RestoreStatus.NEW
                else:
                    backup_for_classification = self._backup_content(app.id, file, dry_run=dry_run)
                    status = _classify_local(local, file, backup_for_classification)
                entries.append(
                    RestoreEntry(
                        app_id=app.id,
                        path=file.path,
                        destination=destination,
                        size=file.size,
                        sha256=file.sha256,
                        mode=file.mode,
                        link=file.link,
                        status=status,
                    )
                )
                if status in (RestoreStatus.IDENTICAL, RestoreStatus.UNSUPPORTED):
                    omitted.append(file.path)
                if status is RestoreStatus.DIFFERENT and file.path not in seen_conflicts:
                    conflict = self._build_conflict(app.id, file, destination, local, dry_run)
                    if conflict is not None:
                        conflicts.append(conflict)
                        seen_conflicts.add(file.path)
                if status in (RestoreStatus.NEW, RestoreStatus.DIFFERENT) and safe is not None:
                    if file.path not in fingerprints:
                        fingerprints[file.path] = _fingerprint(safe, file.path)
                elif status is RestoreStatus.NEW and file.path not in fingerprints:
                    fingerprints[file.path] = _DestinationFingerprint("missing")
        finally:
            if safe is not None:
                safe.close()

        changed = [entry for entry in entries if entry.status is not RestoreStatus.UNSUPPORTED]
        if changed and self.home.exists():
            _validate_destinations(
                [entry for entry in changed if entry.status is not RestoreStatus.IDENTICAL],
                self.home,
            )
        plan = RestorePlan(
            entries=entries,
            notices=self.notices,
            conflicts=conflicts,
            omitted=list(dict.fromkeys(omitted)),
        )
        self._plan = plan
        self._fingerprints = fingerprints
        return plan

    def preview(self, decisions: ConflictDecisions) -> RestoreClassification:
        """Return the categories that applying ``decisions`` would produce."""
        plan, resolved = self._resolve(decisions)
        return _classification(plan, resolved)

    def dry_run_result(self, plan: RestorePlan) -> RestoreResult:
        """Build the result of a simulation (no writes, no snapshot)."""
        return RestoreResult(
            plan=plan,
            manifest=self.manifest,
            output=None,
            dry_run=True,
            snapshot=None,
            restored=[],
            merged=[],
            omitted=plan.omitted,
        )

    def apply(self, decisions: ConflictDecisions) -> RestoreResult:
        """Apply the decisions with a snapshot and atomic per-file writes."""
        self._require_open()
        plan, resolved = self._resolve(decisions)
        _validate_merges(resolved)

        writes = [item for item in resolved if item.write]
        paths = [item.path for item in writes]
        if not paths:
            return RestoreResult(
                plan=plan,
                manifest=self.manifest,
                output=self.backup_file,
                dry_run=False,
                snapshot=None,
                restored=[],
                merged=[],
                omitted=_merged_paths(plan, resolved),
            )

        with SafeHome(self.home) as safe:
            for path in paths:
                current = _fingerprint(safe, path)
                expected = self._fingerprints.get(path)
                if expected is None or current != expected:
                    raise ConcurrentChangeError(f"el destino cambió durante la revisión: {path}")

        snapshot = create_snapshot(
            [item.destination for item in writes], home=self.home, snapshots_dir=self.snapshots_dir
        )
        _write_resolved(writes, home=self.home)
        classification = _classification(plan, resolved)
        return RestoreResult(
            plan=plan,
            manifest=self.manifest,
            output=self.backup_file,
            dry_run=False,
            snapshot=snapshot,
            restored=classification.restored,
            merged=classification.merged,
            omitted=classification.omitted,
        )

    def close(self) -> None:
        """Remove the staging directory, if any."""
        if self._staging is not None:
            shutil.rmtree(self._staging, ignore_errors=True)
            self._staging = None
        self._closed = True

    def __enter__(self) -> RestoreSession:
        return self

    def __exit__(self, *_exc: object) -> None:
        self.close()

    def _require_open(self) -> None:
        if self._closed:
            raise RuntimeError("restore session is closed")

    def _reset_staging(self) -> None:
        if self._staging is not None:
            shutil.rmtree(self._staging, ignore_errors=True)
        self._staging = Path(tempfile.mkdtemp(prefix="kamino-restore-"))
        extract_archive(self.backup_file, self._staging)

    def _active_files(self, selected: list[str] | None) -> Sequence[tuple[AppEntry, FileEntry]]:
        active: list[tuple[AppEntry, FileEntry]] = []
        for app in self.restorable_apps:
            if selected is not None and app.id not in selected:
                continue
            active.extend((app, file) for file in app.files)
        return active

    def _backup_content(self, app_id: str, file: FileEntry, *, dry_run: bool) -> bytes | None:
        if file.link is not None or file.size > MAX_TEXT_BYTES:
            return None
        if self._staging is not None:
            source = self._staging / APPS_DIR / app_id / file.path
            if not source.is_file():
                return None
            return source.read_bytes()
        return read_file_from_archive(self.backup_file, app_id, file.path, limit=MAX_TEXT_BYTES + 1)

    def _build_conflict(
        self,
        app_id: str,
        file: FileEntry,
        destination: Path,
        local: LocalFile | None,
        dry_run: bool,
    ) -> Conflict | None:
        if local is None or file.link is not None:
            if local is None:
                return None
            return build_conflict(
                app_id=app_id,
                path=file.path,
                destination=destination,
                entry=file,
                local=local,
                backup_content=None,
            )
        backup_content = self._backup_content(app_id, file, dry_run=dry_run)
        return build_conflict(
            app_id=app_id,
            path=file.path,
            destination=destination,
            entry=file,
            local=local,
            backup_content=backup_content,
        )

    def _resolve(self, decisions: ConflictDecisions) -> tuple[RestorePlan, list[_ResolvedFile]]:
        plan = self._plan
        if plan is None:
            raise RuntimeError("call plan() before preview() or apply()")
        missing = decisions.missing(plan.conflicts)
        if missing:
            names = ", ".join(conflict.path for conflict in missing)
            raise DecisionRequiredError(f"faltan decisiones de conflicto: {names}")
        conflicts = {conflict.path: conflict for conflict in plan.conflicts}
        resolved: list[_ResolvedFile] = []
        done: set[str] = set()
        for entry in plan.entries:
            if entry.path in done:
                continue
            done.add(entry.path)
            if entry.status in (RestoreStatus.IDENTICAL, RestoreStatus.UNSUPPORTED):
                resolved.append(
                    _ResolvedFile(
                        entry.path,
                        entry.destination,
                        None,
                        None,
                        entry.mode,
                        ResultCategory.OMITTED,
                        False,
                    )
                )
                continue
            conflict = conflicts.get(entry.path)
            if conflict is None:
                content, link = self._backup_output(entry)
                resolved.append(
                    _ResolvedFile(
                        entry.path,
                        entry.destination,
                        content,
                        link,
                        entry.mode,
                        ResultCategory.RESTORED,
                        True,
                    )
                )
                continue
            decision = decisions.get(entry.path)
            if decision is None:
                raise DecisionRequiredError(f"falta la decisión de {entry.path}")
            resolved.append(self._resolve_conflict(entry, conflict, decision))
        return plan, resolved

    def _resolve_conflict(
        self, entry: RestoreEntry, conflict: Conflict, decision: FileDecision
    ) -> _ResolvedFile:
        if decision.action is FileAction.OMIT:
            return _ResolvedFile(
                entry.path,
                entry.destination,
                None,
                None,
                entry.mode,
                ResultCategory.OMITTED,
                False,
            )
        if decision.action is FileAction.APPLY:
            content, link = self._backup_output(entry)
            return _ResolvedFile(
                entry.path,
                entry.destination,
                content,
                link,
                entry.mode,
                ResultCategory.RESTORED,
                True,
            )
        if conflict.local_text is None or conflict.backup_text is None:
            raise ConflictError(f"{entry.path} no admite revisión por bloques")
        if len(decision.hunks) != len(conflict.hunks):
            raise ConflictError(f"la decisión de {entry.path} no cubre todos los bloques")
        merged = compose(conflict.local_text, conflict.backup_text, decision.hunks)
        if merged == conflict.local_text:
            return _ResolvedFile(
                entry.path,
                entry.destination,
                None,
                None,
                entry.mode,
                ResultCategory.OMITTED,
                False,
                local_text=conflict.local_text,
                backup_text=conflict.backup_text,
                merged_text=merged,
            )
        all_backup = all(item is HunkDecision.BACKUP for item in decision.hunks)
        category = ResultCategory.RESTORED if all_backup else ResultCategory.MERGED
        return _ResolvedFile(
            entry.path,
            entry.destination,
            merged.encode("utf-8"),
            None,
            entry.mode,
            category,
            True,
            local_text=conflict.local_text,
            backup_text=conflict.backup_text,
            merged_text=merged,
        )

    def _backup_output(self, entry: RestoreEntry) -> tuple[bytes | None, str | None]:
        if entry.link is not None:
            return None, entry.link
        if self._staging is None:
            raise RuntimeError("the archive is not extracted")
        source = self._staging / APPS_DIR / entry.app_id / entry.path
        return source.read_bytes(), None


def prepare_restore(
    backup_file: Path,
    *,
    home: Path | None = None,
    catalog: dict[str, AppDefinition] | None = None,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
    snapshots_dir: Path | None = None,
) -> RestoreSession:
    """Read the manifest and split manifest apps into restorable and missing."""
    root = home if home is not None else Path.home()
    definitions = (
        catalog
        if catalog is not None
        else load_catalog(user_dir=root / ".config" / "kamino" / "apps")
    )
    manifest = read_manifest(backup_file)
    restorable: list[AppEntry] = []
    missing: list[AppEntry] = []
    for app in manifest.apps:
        if _is_app_installed(app, definitions, application_dirs, path, root):
            restorable.append(app)
        else:
            missing.append(app)
    return RestoreSession(
        backup_file,
        home=root,
        manifest=manifest,
        restorable_apps=restorable,
        not_installed_apps=missing,
        snapshots_dir=snapshots_dir,
    )


def run_restore(
    backup_file: Path,
    *,
    home: Path | None = None,
    selected: list[str] | None = None,
    dry_run: bool = False,
    catalog: dict[str, AppDefinition] | None = None,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
    snapshots_dir: Path | None = None,
) -> RestoreResult:
    """Run the complete restore and return the typed result.

    This is the non-interactive wrapper: every conflict is resolved by applying
    the backup side. ``dry_run`` only builds the plan and conflicts.
    """
    session = prepare_restore(
        backup_file,
        home=home,
        catalog=catalog,
        application_dirs=application_dirs,
        path=path,
        snapshots_dir=snapshots_dir,
    )
    try:
        plan = session.plan(selected, dry_run=dry_run)
        if dry_run:
            return session.dry_run_result(plan)
        return session.apply(apply_all(plan.conflicts))
    finally:
        session.close()


def _read_local(safe: SafeHome | None, rel: str) -> LocalFile:
    if safe is None:
        raise FileNotFoundError(rel)
    info = safe.lstat(rel)
    mode = format(stat.S_IMODE(info.st_mode), "04o")
    if stat.S_ISLNK(info.st_mode):
        return LocalFile(kind="symlink", link=safe.readlink(rel), mode=mode)
    if stat.S_ISREG(info.st_mode):
        return LocalFile(kind="file", data=safe.read_bytes(rel), mode=mode)
    return LocalFile(kind="special", mode=mode)


def _classify_local(
    local: LocalFile, entry: FileEntry, backup_content: bytes | None
) -> RestoreStatus:
    if local.kind == "special":
        return RestoreStatus.UNSUPPORTED
    if local.kind == "symlink":
        if entry.link is None:
            return RestoreStatus.DIFFERENT
        return RestoreStatus.IDENTICAL if local.link == entry.link else RestoreStatus.DIFFERENT
    if entry.link is not None:
        return RestoreStatus.DIFFERENT
    if entry.sha256 is not None and hashlib.sha256(local.data).hexdigest() == entry.sha256:
        return RestoreStatus.IDENTICAL
    if backup_content is not None and _text_equal(local.data, backup_content):
        return RestoreStatus.IDENTICAL
    return RestoreStatus.DIFFERENT


def _text_equal(local: bytes, backup: bytes) -> bool:
    if b"\x00" in local or b"\x00" in backup:
        return False
    try:
        local_text = local.decode("utf-8")
        backup_text = backup.decode("utf-8")
    except UnicodeDecodeError:
        return False
    return local_text.replace("\r\n", "\n") == backup_text.replace("\r\n", "\n")


def _fingerprint(safe: SafeHome, rel: str) -> _DestinationFingerprint:
    try:
        info = safe.lstat(rel)
    except FileNotFoundError:
        return _DestinationFingerprint("missing")
    if stat.S_ISLNK(info.st_mode):
        return _DestinationFingerprint("symlink", link=safe.readlink(rel))
    if stat.S_ISREG(info.st_mode):
        return _DestinationFingerprint(
            "file", sha256=hashlib.sha256(safe.read_bytes(rel)).hexdigest()
        )
    return _DestinationFingerprint("special")


def _validate_destinations(entries: Sequence[RestoreEntry], home: Path) -> None:
    """Reject intermediate symlinks before creating the previous-state snapshot."""
    with SafeHome(home) as safe:
        for entry in entries:
            try:
                safe.lstat(entry.path)
            except FileNotFoundError:
                continue


def _write_resolved(entries: Sequence[_ResolvedFile], *, home: Path) -> None:
    with SafeHome(home) as safe:
        for entry in entries:
            if entry.link is not None:
                safe.replace_with_symlink(entry.path, entry.link)
            else:
                if entry.content is None:
                    raise RuntimeError(f"missing content for {entry.path}")
                safe.write_file(entry.path, entry.content, mode=int(entry.mode, 8))


def _validate_merges(entries: Sequence[_ResolvedFile]) -> None:
    for entry in entries:
        if not entry.write or entry.merged_text is None:
            continue
        if entry.local_text is None or entry.backup_text is None:
            continue
        validation = validate_merge(
            entry.path, entry.local_text, entry.backup_text, entry.merged_text
        )
        if validation.status == "invalid":
            raise InvalidMergeError(f"la mezcla de {entry.path} no es válida: {validation.message}")


def _classification(plan: RestorePlan, resolved: Sequence[_ResolvedFile]) -> RestoreClassification:
    restored = [entry.path for entry in resolved if entry.category is ResultCategory.RESTORED]
    merged = [entry.path for entry in resolved if entry.category is ResultCategory.MERGED]
    omitted = list(plan.omitted)
    omitted.extend(entry.path for entry in resolved if entry.category is ResultCategory.OMITTED)
    return RestoreClassification(
        restored=restored,
        merged=merged,
        omitted=list(dict.fromkeys(omitted)),
    )


def _merged_paths(plan: RestorePlan, resolved: Sequence[_ResolvedFile]) -> list[str]:
    return _classification(plan, resolved).omitted


def _is_app_installed(
    app: AppEntry,
    definitions: dict[str, AppDefinition],
    application_dirs: Sequence[Path] | None,
    path: str | None,
    home: Path,
) -> bool:
    """Whether a manifest app is installed on this machine.

    The synthetic ``snapshot`` app always counts as installed: a snapshot is
    the previous state of this same machine. Other apps are detected with
    ``is_installed``; without a definition they are treated as not installed.
    """
    if app.id == SNAPSHOT_APP_ID:
        return True
    definition = definitions.get(app.id)
    return definition is not None and is_installed(
        definition, home=home, application_dirs=application_dirs, path=path
    )


def _ghostty_portability_notices(manifest: Manifest, backup_file: Path, home: Path) -> list[str]:
    """Warn about absolute Ghostty paths that do not point to the target home."""
    notices: list[str] = []
    for app in manifest.apps:
        if app.id != GHOSTTY_APP_ID:
            continue
        for file in app.files:
            if file.link is not None:
                continue
            content = read_file_from_archive(backup_file, app.id, file.path, limit=MAX_CONFIG_BYTES)
            if content is None:
                continue
            try:
                text = content.decode("utf-8", errors="replace")
            except Exception:
                continue
            for reference in absolute_references(text):
                try:
                    to_relative(Path(reference), home=home)
                except PathError:
                    notices.append(
                        "aviso: Ghostty conserva una ruta absoluta que puede "
                        "requerir ajuste manual tras la restauración"
                    )
                    break
    return list(dict.fromkeys(notices))
