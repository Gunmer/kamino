"""Deploy use case: plan the deploy, resolve conflicts and apply it safely.

The session keeps the safety invariants in one place: verify the archive before
extracting, extract only into a temporary directory, create a snapshot before
writing, re-check destinations immediately before applying, and never write
when the review is cancelled.

Classification of every manifest file against its destination:

- ``NEW``: the destination does not exist; it is created.
- ``IDENTICAL``: content matches (sha256 for regular files, target for symlinks,
  and CRLF-normalised text for text files); it is omitted.
- ``DIFFERENT``: content or type differs; a content conflict is added to
  ``DeployPlan.conflicts`` for an explicit decision, while a type change keeps
  the existing overwrite behaviour.
- ``UNSUPPORTED``: the destination is a directory, socket, pipe...; it is
  omitted.
"""

from __future__ import annotations

import hashlib
import platform
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
    LocalFile,
    RowDecision,
    apply_all,
    build_conflict,
    compose,
    row_count,
    validate_merge,
)
from kamino.defaults import catalog_dir as default_catalog_dir
from kamino.defaults import snapshots_dir as default_snapshots_dir
from kamino.detect import is_installed
from kamino.errors import (
    ConcurrentChangeError,
    ConflictError,
    DecisionRequiredError,
    InvalidMergeError,
    OperationCancelled,
    PartialApplyError,
    PathError,
)
from kamino.ghostty import GHOSTTY_APP_ID, MAX_CONFIG_BYTES, absolute_references
from kamino.paths import SafeHome, to_absolute, to_relative
from kamino.progress import (
    CancelCallback,
    Phase,
    ProgressCallback,
    emit,
    ensure_not_cancelled,
)
from kamino.snapshot import SNAPSHOT_APP_ID, create_snapshot, snapshot_files


class DeployStatus(StrEnum):
    """Classification of a manifest file against its destination (§5.2.4)."""

    NEW = "new"
    IDENTICAL = "identical"
    DIFFERENT = "different"
    UNSUPPORTED = "unsupported"


class ResultCategory(StrEnum):
    """Final outcome of an active manifest file."""

    DEPLOYED = "deployed"
    MERGED = "merged"
    OMITTED = "omitted"
    UNCHANGED = "unchanged"


@dataclass(frozen=True)
class DeployEntry:
    """A manifest file classified for deployment."""

    app_id: str
    path: str
    destination: Path
    size: int
    sha256: str | None
    mode: str
    link: str | None
    status: DeployStatus


@dataclass(frozen=True)
class DeployPlan:
    """The deploy plan: active manifest files, conflicts and categories."""

    entries: list[DeployEntry] = field(default_factory=list)
    notices: list[str] = field(default_factory=list)
    conflicts: list[Conflict] = field(default_factory=list)
    #: Archivos no soportados o conservados en el original: producen código 3.
    omitted: list[str] = field(default_factory=list)
    #: Clones perfectos: no se escriben y no producen código 3.
    unchanged: list[str] = field(default_factory=list)
    #: Apps del manifiesto que no están instaladas en este Mac.
    not_installed: list[str] = field(default_factory=list)
    #: Ids pedidos con `--apps` que no se pueden cumplir: producen código 3.
    missing_requested: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class DeployResult:
    """Result of a deploy (§5.2.9).

    ``deployed`` contains files written with the clone content, ``merged``
    contains real block mixes, ``omitted`` contains conflicts kept on the
    original and unsupported files, ``unchanged`` contains perfect clones and
    ``not_installed`` contains manifest apps with no base on this Mac.
    """

    plan: DeployPlan
    manifest: Manifest
    output: Path | None
    dry_run: bool
    snapshot: Path | None
    deployed: list[str]
    merged: list[str]
    omitted: list[str]
    unchanged: list[str] = field(default_factory=list)
    not_installed: list[str] = field(default_factory=list)
    missing_requested: list[str] = field(default_factory=list)
    snapshot_files: int = 0


@dataclass(frozen=True)
class DeployClassification:
    """Prospective result categories, used by the final confirmation screen."""

    deployed: list[str]
    merged: list[str]
    omitted: list[str]
    unchanged: list[str] = field(default_factory=list)
    not_installed: list[str] = field(default_factory=list)
    missing_requested: list[str] = field(default_factory=list)


def classification_code(classification: DeployClassification) -> int:
    """Código de salida: 3 solo por omitidos o `--apps` no cumplidos."""
    return 3 if (classification.omitted or classification.missing_requested) else 0


def result_code(result: DeployResult) -> int:
    """Código de salida de un despliegue terminado."""
    return 3 if (result.omitted or result.missing_requested) else 0


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
    clone_text: str | None = None
    merged_text: str | None = None


class DeploySession:
    """Prepared deploy with ``plan``, ``preview`` and ``apply`` steps."""

    def __init__(
        self,
        clone_file: Path,
        *,
        home: Path,
        manifest: Manifest,
        deployable_apps: Sequence[AppEntry],
        not_installed_apps: Sequence[AppEntry],
        snapshots_dir: Path | None,
    ) -> None:
        self.clone_file = clone_file
        self.home = home
        self.manifest = manifest
        self.deployable_apps = list(deployable_apps)
        self.not_installed_apps = list(not_installed_apps)
        self.snapshots_dir = snapshots_dir
        self.notices = _ghostty_portability_notices(manifest, clone_file, home)
        self.warnings = _platform_warnings(manifest)
        self._staging: Path | None = None
        self._plan: DeployPlan | None = None
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
        return default_snapshots_dir(self.home)

    def plan(
        self,
        selected: list[str] | None = None,
        *,
        dry_run: bool = False,
        requested: list[str] | None = None,
        progress: ProgressCallback | None = None,
        cancelled: CancelCallback | None = None,
    ) -> DeployPlan:
        """Classify the chosen apps and build every content conflict.

        With ``dry_run`` the archive is neither verified nor extracted; the
        conflict contents are read directly from it so the caller can print the
        diffs.
        """
        self._require_open()
        if not dry_run:
            verify_archive(self.clone_file, self.manifest, progress=progress, cancelled=cancelled)
            ensure_not_cancelled(cancelled)
            self.home.mkdir(parents=True, exist_ok=True)
            self._reset_staging(progress=progress, cancelled=cancelled)

        try:
            entries: list[DeployEntry] = []
            conflicts: list[Conflict] = []
            unchanged: list[str] = []
            omitted: list[str] = []
            not_installed = [file.path for app in self.not_installed_apps for file in app.files]
            deployable_ids = {app.id for app in self.deployable_apps}
            missing_requested = [
                app_id for app_id in (requested or []) if app_id not in deployable_ids
            ]
            active_files = list(self._active_files(selected))
            total = len(active_files)
            seen_conflicts: set[str] = set()
            fingerprints: dict[str, _DestinationFingerprint] = {}
            safe = SafeHome(self.home) if self.home.exists() else None
            try:
                for index, (app, file) in enumerate(active_files, start=1):
                    ensure_not_cancelled(cancelled)
                    emit(progress, Phase.CLASSIFYING, index, total, file.path)
                    destination = to_absolute(file.path, home=self.home)
                    try:
                        local = _read_local(safe, file.path) if safe is not None else None
                    except FileNotFoundError:
                        local = None
                    if local is None:
                        status = DeployStatus.NEW
                    else:
                        clone_for_classification = self._clone_content(
                            app.id, file, dry_run=dry_run
                        )
                        status = _classify_local(local, file, clone_for_classification)
                    entries.append(
                        DeployEntry(
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
                    if status is DeployStatus.IDENTICAL:
                        unchanged.append(file.path)
                    elif status is DeployStatus.UNSUPPORTED:
                        omitted.append(file.path)
                    if status is DeployStatus.DIFFERENT and file.path not in seen_conflicts:
                        conflict = self._build_conflict(app.id, file, destination, local, dry_run)
                        if conflict is not None:
                            conflicts.append(conflict)
                            seen_conflicts.add(file.path)
                    if status in (DeployStatus.NEW, DeployStatus.DIFFERENT) and safe is not None:
                        if file.path not in fingerprints:
                            fingerprints[file.path] = _fingerprint(safe, file.path)
                    elif status is DeployStatus.NEW and file.path not in fingerprints:
                        fingerprints[file.path] = _DestinationFingerprint("missing")
            finally:
                if safe is not None:
                    safe.close()

            ensure_not_cancelled(cancelled)
            changed = [entry for entry in entries if entry.status is not DeployStatus.UNSUPPORTED]
            if changed and self.home.exists():
                _validate_destinations(
                    [entry for entry in changed if entry.status is not DeployStatus.IDENTICAL],
                    self.home,
                )
            plan = DeployPlan(
                entries=entries,
                notices=self.notices,
                conflicts=conflicts,
                omitted=list(dict.fromkeys(omitted)),
                unchanged=list(dict.fromkeys(unchanged)),
                not_installed=list(dict.fromkeys(not_installed)),
                missing_requested=missing_requested,
            )
            self._plan = plan
            self._fingerprints = fingerprints
            return plan
        except OperationCancelled:
            self._clear_staging()
            raise

    def preview(self, decisions: ConflictDecisions) -> DeployClassification:
        """Return the categories that applying ``decisions`` would produce."""
        plan, resolved = self._resolve(decisions)
        return _classification(plan, resolved)

    def dry_run_result(self, plan: DeployPlan) -> DeployResult:
        """Build the result of a simulation (no writes, no snapshot)."""
        return DeployResult(
            plan=plan,
            manifest=self.manifest,
            output=None,
            dry_run=True,
            snapshot=None,
            deployed=[],
            merged=[],
            omitted=plan.omitted,
            unchanged=plan.unchanged,
            not_installed=plan.not_installed,
            missing_requested=plan.missing_requested,
        )

    def apply(
        self,
        decisions: ConflictDecisions,
        *,
        progress: ProgressCallback | None = None,
        cancelled: CancelCallback | None = None,
    ) -> DeployResult:
        """Apply the decisions with a snapshot and atomic per-file writes.

        Cancellation works until the holocrón is created and is ignored once
        writing destinations starts. A write failure raises
        :class:`PartialApplyError` with the applied files and the snapshot.
        """
        self._require_open()
        plan, resolved = self._resolve(decisions)
        _validate_merges(resolved)

        writes = [item for item in resolved if item.write]
        paths = [item.path for item in writes]
        if not paths:
            return DeployResult(
                plan=plan,
                manifest=self.manifest,
                output=self.clone_file,
                dry_run=False,
                snapshot=None,
                deployed=[],
                merged=[],
                omitted=_merged_paths(plan, resolved),
                unchanged=plan.unchanged,
                not_installed=plan.not_installed,
                missing_requested=plan.missing_requested,
            )

        with SafeHome(self.home) as safe:
            for path in paths:
                ensure_not_cancelled(cancelled)
                current = _fingerprint(safe, path)
                expected = self._fingerprints.get(path)
                if expected is None or current != expected:
                    raise ConcurrentChangeError(f"el destino cambió durante la revisión: {path}")

        protected = snapshot_files([item.destination for item in writes], home=self.home)
        ensure_not_cancelled(cancelled)
        snapshot = create_snapshot(
            [item.destination for item in writes], home=self.home, snapshots_dir=self.snapshots_dir
        )
        _write_resolved(
            writes,
            home=self.home,
            snapshot=snapshot,
            progress=progress,
        )
        classification = _classification(plan, resolved)
        return DeployResult(
            plan=plan,
            manifest=self.manifest,
            output=self.clone_file,
            dry_run=False,
            snapshot=snapshot,
            deployed=classification.deployed,
            merged=classification.merged,
            omitted=classification.omitted,
            unchanged=classification.unchanged,
            not_installed=classification.not_installed,
            missing_requested=classification.missing_requested,
            snapshot_files=protected,
        )

    def _clear_staging(self) -> None:
        if self._staging is not None:
            shutil.rmtree(self._staging, ignore_errors=True)
            self._staging = None

    def close(self) -> None:
        """Remove the staging directory, if any."""
        self._clear_staging()
        self._closed = True

    def __enter__(self) -> DeploySession:
        return self

    def __exit__(self, *_exc: object) -> None:
        self.close()

    def _require_open(self) -> None:
        if self._closed:
            raise RuntimeError("deploy session is closed")

    def _reset_staging(
        self,
        *,
        progress: ProgressCallback | None = None,
        cancelled: CancelCallback | None = None,
    ) -> None:
        if self._staging is not None:
            shutil.rmtree(self._staging, ignore_errors=True)
        self._staging = Path(tempfile.mkdtemp(prefix="kamino-deploy-"))
        try:
            extract_archive(self.clone_file, self._staging, progress=progress, cancelled=cancelled)
        except Exception:
            shutil.rmtree(self._staging, ignore_errors=True)
            self._staging = None
            raise

    def _active_files(self, selected: list[str] | None) -> Sequence[tuple[AppEntry, FileEntry]]:
        active: list[tuple[AppEntry, FileEntry]] = []
        for app in self.deployable_apps:
            if selected is not None and app.id not in selected:
                continue
            active.extend((app, file) for file in app.files)
        return active

    def _clone_content(self, app_id: str, file: FileEntry, *, dry_run: bool) -> bytes | None:
        if file.link is not None or file.size > MAX_TEXT_BYTES:
            return None
        if self._staging is not None:
            source = self._staging / APPS_DIR / app_id / file.path
            if not source.is_file():
                return None
            return source.read_bytes()
        return read_file_from_archive(self.clone_file, app_id, file.path, limit=MAX_TEXT_BYTES + 1)

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
                clone_content=None,
            )
        clone_content = self._clone_content(app_id, file, dry_run=dry_run)
        return build_conflict(
            app_id=app_id,
            path=file.path,
            destination=destination,
            entry=file,
            local=local,
            clone_content=clone_content,
        )

    def _resolve(self, decisions: ConflictDecisions) -> tuple[DeployPlan, list[_ResolvedFile]]:
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
            if entry.status in (DeployStatus.IDENTICAL, DeployStatus.UNSUPPORTED):
                category = (
                    ResultCategory.UNCHANGED
                    if entry.status is DeployStatus.IDENTICAL
                    else ResultCategory.OMITTED
                )
                resolved.append(
                    _ResolvedFile(
                        entry.path,
                        entry.destination,
                        None,
                        None,
                        entry.mode,
                        category,
                        False,
                    )
                )
                continue
            conflict = conflicts.get(entry.path)
            if conflict is None:
                content, link = self._clone_output(entry)
                resolved.append(
                    _ResolvedFile(
                        entry.path,
                        entry.destination,
                        content,
                        link,
                        entry.mode,
                        ResultCategory.DEPLOYED,
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
        self, entry: DeployEntry, conflict: Conflict, decision: FileDecision
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
            content, link = self._clone_output(entry)
            return _ResolvedFile(
                entry.path,
                entry.destination,
                content,
                link,
                entry.mode,
                ResultCategory.DEPLOYED,
                True,
            )
        if conflict.local_text is None or conflict.clone_text is None:
            raise ConflictError(f"{entry.path} no admite revisión por líneas")
        if len(decision.rows) != row_count(conflict):
            raise ConflictError(f"la decisión de {entry.path} no cubre todas las líneas")
        merged = compose(conflict.local_text, conflict.clone_text, decision.rows)
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
                clone_text=conflict.clone_text,
                merged_text=merged,
            )
        all_clone = all(item is RowDecision.CLONE for item in decision.rows)
        category = ResultCategory.DEPLOYED if all_clone else ResultCategory.MERGED
        return _ResolvedFile(
            entry.path,
            entry.destination,
            merged.encode("utf-8"),
            None,
            entry.mode,
            category,
            True,
            local_text=conflict.local_text,
            clone_text=conflict.clone_text,
            merged_text=merged,
        )

    def _clone_output(self, entry: DeployEntry) -> tuple[bytes | None, str | None]:
        if entry.link is not None:
            return None, entry.link
        if self._staging is None:
            raise RuntimeError("the archive is not extracted")
        source = self._staging / APPS_DIR / entry.app_id / entry.path
        return source.read_bytes(), None


def prepare_deploy(
    clone_file: Path,
    *,
    home: Path | None = None,
    catalog: dict[str, AppDefinition] | None = None,
    catalog_dir: Path | None = None,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
    snapshots_dir: Path | None = None,
) -> DeploySession:
    """Read the manifest and split manifest apps into deployable and missing."""
    root = home if home is not None else Path.home()
    definitions = (
        catalog
        if catalog is not None
        else load_catalog(
            user_dir=catalog_dir if catalog_dir is not None else default_catalog_dir(root)
        )
    )
    manifest = read_manifest(clone_file)
    deployable: list[AppEntry] = []
    missing: list[AppEntry] = []
    for app in manifest.apps:
        if _is_app_installed(app, definitions, application_dirs, path, root):
            deployable.append(app)
        else:
            missing.append(app)
    return DeploySession(
        clone_file,
        home=root,
        manifest=manifest,
        deployable_apps=deployable,
        not_installed_apps=missing,
        snapshots_dir=snapshots_dir,
    )


def run_deploy(
    clone_file: Path,
    *,
    home: Path | None = None,
    selected: list[str] | None = None,
    dry_run: bool = False,
    catalog: dict[str, AppDefinition] | None = None,
    catalog_dir: Path | None = None,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
    snapshots_dir: Path | None = None,
) -> DeployResult:
    """Run the complete deploy and return the typed result.

    This is the non-interactive wrapper: every conflict is resolved by applying
    the clone side. ``dry_run`` only builds the plan and conflicts.
    """
    session = prepare_deploy(
        clone_file,
        home=home,
        catalog=catalog,
        catalog_dir=catalog_dir,
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
    local: LocalFile, entry: FileEntry, clone_content: bytes | None
) -> DeployStatus:
    if local.kind == "special":
        return DeployStatus.UNSUPPORTED
    if local.kind == "symlink":
        if entry.link is None:
            return DeployStatus.DIFFERENT
        return DeployStatus.IDENTICAL if local.link == entry.link else DeployStatus.DIFFERENT
    if entry.link is not None:
        return DeployStatus.DIFFERENT
    if entry.sha256 is not None and hashlib.sha256(local.data).hexdigest() == entry.sha256:
        return DeployStatus.IDENTICAL
    if clone_content is not None and _text_equal(local.data, clone_content):
        return DeployStatus.IDENTICAL
    return DeployStatus.DIFFERENT


def _text_equal(local: bytes, clone: bytes) -> bool:
    if b"\x00" in local or b"\x00" in clone:
        return False
    try:
        local_text = local.decode("utf-8")
        clone_text = clone.decode("utf-8")
    except UnicodeDecodeError:
        return False
    return local_text.replace("\r\n", "\n") == clone_text.replace("\r\n", "\n")


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


def _validate_destinations(entries: Sequence[DeployEntry], home: Path) -> None:
    """Reject intermediate symlinks before creating the previous-state snapshot."""
    with SafeHome(home) as safe:
        for entry in entries:
            try:
                safe.lstat(entry.path)
            except FileNotFoundError:
                continue


def _write_resolved(
    entries: Sequence[_ResolvedFile],
    *,
    home: Path,
    snapshot: Path | None,
    progress: ProgressCallback | None = None,
) -> None:
    applied: list[str] = []
    total = len(entries)
    with SafeHome(home) as safe:
        for index, entry in enumerate(entries, start=1):
            try:
                if entry.link is not None:
                    safe.replace_with_symlink(entry.path, entry.link)
                else:
                    if entry.content is None:
                        raise RuntimeError(f"missing content for {entry.path}")
                    safe.write_file(entry.path, entry.content, mode=int(entry.mode, 8))
            except Exception as exc:
                raise PartialApplyError(
                    f"despliegue incompleto en {entry.path}: {exc}",
                    applied=applied,
                    failed=entry.path,
                    snapshot=snapshot,
                ) from exc
            applied.append(entry.path)
            emit(progress, Phase.WRITING, index, total, entry.path)


def _validate_merges(entries: Sequence[_ResolvedFile]) -> None:
    for entry in entries:
        if not entry.write or entry.merged_text is None:
            continue
        if entry.local_text is None or entry.clone_text is None:
            continue
        validation = validate_merge(
            entry.path, entry.local_text, entry.clone_text, entry.merged_text
        )
        if validation.status == "invalid":
            raise InvalidMergeError(
                f"el híbrido de {entry.path} no es válido: {validation.message}"
            )


def _classification(plan: DeployPlan, resolved: Sequence[_ResolvedFile]) -> DeployClassification:
    deployed = [entry.path for entry in resolved if entry.category is ResultCategory.DEPLOYED]
    merged = [entry.path for entry in resolved if entry.category is ResultCategory.MERGED]
    omitted = list(plan.omitted)
    omitted.extend(entry.path for entry in resolved if entry.category is ResultCategory.OMITTED)
    unchanged = list(plan.unchanged)
    unchanged.extend(entry.path for entry in resolved if entry.category is ResultCategory.UNCHANGED)
    return DeployClassification(
        deployed=deployed,
        merged=merged,
        omitted=list(dict.fromkeys(omitted)),
        unchanged=list(dict.fromkeys(unchanged)),
        not_installed=list(plan.not_installed),
        missing_requested=list(plan.missing_requested),
    )


def _merged_paths(plan: DeployPlan, resolved: Sequence[_ResolvedFile]) -> list[str]:
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


def _platform_warnings(manifest: Manifest) -> list[str]:
    """Avisos no bloqueantes si el clon viene de otra arquitectura o macOS."""
    warnings: list[str] = []
    current_arch = platform.machine()
    if manifest.arch and current_arch and manifest.arch != current_arch:
        warnings.append(
            f"el clon se creó en {manifest.arch} y este Mac es {current_arch}; "
            "el despliegue puede requerir ajustes"
        )
    current_macos = platform.mac_ver()[0]
    if manifest.macos_version and current_macos and manifest.macos_version != current_macos:
        warnings.append(
            f"el clon se creó con macOS {manifest.macos_version} y este Mac usa "
            f"{current_macos}; el despliegue puede requerir ajustes"
        )
    return warnings


def _ghostty_portability_notices(manifest: Manifest, clone_file: Path, home: Path) -> list[str]:
    """Warn about absolute Ghostty paths that do not point to the target home."""
    notices: list[str] = []
    for app in manifest.apps:
        if app.id != GHOSTTY_APP_ID:
            continue
        for file in app.files:
            if file.link is not None:
                continue
            content = read_file_from_archive(clone_file, app.id, file.path, limit=MAX_CONFIG_BYTES)
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
                        "requerir ajuste manual tras el despliegue"
                    )
                    break
    return list(dict.fromkeys(notices))
