"""Conflict detection, diffing, composition and validation for restores.

The restore flow keeps this module free of interface code: it receives the local
state and the backup bytes and returns the conflict model that both the CLI and
the TUI consume. Composition happens here too, so the interface never injects
arbitrary bytes into a destination.

Rules:

- Regular files are compared as UTF-8 text when possible. CRLF is normalised to
  LF only for comparison, so a file that differs only in line endings is
  identical.
- A regular file with NUL bytes or invalid UTF-8 is binary. Text larger than
  :data:`MAX_TEXT_BYTES` or with more than :data:`MAX_DIFF_LINES` lines is
  treated as a whole-file conflict without a line diff.
- Symlinks with different targets are whole-file conflicts.
- Hunks are generated with :class:`difflib.SequenceMatcher` using
  :data:`CONTEXT_LINES` lines of context. Every hunk starts on the backup side
  and can be switched to the local side.
- Composition keeps equal lines from the local file (including their endings),
  adapts backup lines to the dominant local line ending and preserves the final
  newline of the chosen side.
"""

from __future__ import annotations

import configparser
import json
import plistlib
import tomllib
import xml.etree.ElementTree as ElementTree
from collections.abc import Sequence
from dataclasses import dataclass, field
from difflib import SequenceMatcher
from enum import StrEnum
from hashlib import sha256
from pathlib import Path
from typing import Literal

from kamino.archive import FileEntry

#: Text above this size is a whole-file conflict to protect difflib and the TUI.
MAX_TEXT_BYTES = 5 * 1024 * 1024

#: Text with more lines than this is a whole-file conflict.
MAX_DIFF_LINES = 20_000

#: Lines of context around every generated hunk.
CONTEXT_LINES = 3


class ConflictKind(StrEnum):
    """Why a file needs an explicit decision."""

    TEXT = "text"
    BINARY = "binary"
    LARGE = "large"
    SYMLINK = "symlink"


class HunkDecision(StrEnum):
    """Which side of a hunk is written."""

    BACKUP = "backup"
    LOCAL = "local"


class FileAction(StrEnum):
    """How a conflicted file is resolved."""

    APPLY = "apply"
    OMIT = "omit"
    REVIEW = "review"


@dataclass(frozen=True)
class LocalFile:
    """Snapshot of the destination read during planning."""

    kind: Literal["file", "symlink", "special"]
    data: bytes = b""
    link: str | None = None
    mode: str = "0644"


@dataclass(frozen=True)
class Hunk:
    """A contiguous change block with enough context to review it."""

    index: int
    #: 1-based, end-exclusive local line range covered by the block.
    local_start: int
    local_end: int
    #: 1-based, end-exclusive backup line range covered by the block.
    backup_start: int
    backup_end: int
    #: Original lines, including their line endings.
    local_lines: tuple[str, ...]
    backup_lines: tuple[str, ...]
    #: Normalised context lines before and after the changed opcodes.
    context_before: tuple[str, ...] = ()
    context_after: tuple[str, ...] = ()
    decision: HunkDecision = HunkDecision.BACKUP


@dataclass(frozen=True)
class FileDecision:
    """Decision for one conflicted file.

    ``REVIEW`` carries one decision per hunk, in the same order as
    :attr:`Conflict.hunks`. ``APPLY`` and ``OMIT`` ignore the list.
    """

    action: FileAction
    hunks: tuple[HunkDecision, ...] = ()


@dataclass
class ConflictDecisions:
    """Map of relative path -> explicit decision."""

    decisions: dict[str, FileDecision] = field(default_factory=dict)

    def set(self, path: str, decision: FileDecision) -> None:
        self.decisions[path] = decision

    def get(self, path: str) -> FileDecision | None:
        return self.decisions.get(path)

    def missing(self, conflicts: Sequence[Conflict]) -> list[Conflict]:
        """Return conflicts that still lack a decision."""
        return [conflict for conflict in conflicts if conflict.path not in self.decisions]


@dataclass(frozen=True)
class MergeValidation:
    """Result of validating a composed file."""

    status: str
    message: str = ""


class _MergeStatus(StrEnum):
    VALID = "valid"
    INVALID = "invalid"
    WARNING = "warning"
    UNVALIDATED = "unvalidated"


@dataclass(frozen=True)
class Conflict:
    """A destination and a backup file that differ in content or link target."""

    app_id: str
    path: str
    destination: Path
    kind: ConflictKind
    local_size: int
    backup_size: int
    local_mode: str
    backup_mode: str
    local_sha256: str | None
    backup_sha256: str | None
    local_link: str | None
    backup_link: str | None
    local_text: str | None
    backup_text: str | None
    hunks: tuple[Hunk, ...] = ()
    validation_format: str | None = None


def build_conflict(
    *,
    app_id: str,
    path: str,
    destination: Path,
    entry: FileEntry,
    local: LocalFile,
    backup_content: bytes | None,
) -> Conflict | None:
    """Build a conflict, or return ``None`` when the file needs no decision.

    ``None`` covers new files, identical content (after CRLF normalisation),
    mode-only changes and type changes between a symlink and a regular file;
    those keep the existing overwrite semantics.
    """
    local_sha = sha256(local.data).hexdigest() if local.kind == "file" else None
    backup_sha = entry.sha256

    if entry.link is not None:
        if local.kind != "symlink" or local.link is None:
            return None
        if local.link == entry.link:
            return None
        return Conflict(
            app_id=app_id,
            path=path,
            destination=destination,
            kind=ConflictKind.SYMLINK,
            local_size=0,
            backup_size=0,
            local_mode=local.mode,
            backup_mode=entry.mode,
            local_sha256=None,
            backup_sha256=None,
            local_link=local.link,
            backup_link=entry.link,
            local_text=None,
            backup_text=None,
        )

    if local.kind != "file":
        return None
    if backup_sha is not None and local_sha == backup_sha:
        return None

    local_text = _decode_text(local.data)
    if backup_content is None:
        # The size in the manifest is enough to reject oversized diffs; the
        # content is only needed for a text diff.
        if entry.size > MAX_TEXT_BYTES or len(local.data) > MAX_TEXT_BYTES:
            return _whole_file_conflict(
                app_id, path, destination, entry, local, local_sha, ConflictKind.LARGE
            )
        return _whole_file_conflict(
            app_id, path, destination, entry, local, local_sha, ConflictKind.BINARY
        )

    backup_text = _decode_text(backup_content)
    if local_text is None or backup_text is None:
        if entry.size > MAX_TEXT_BYTES or len(local.data) > MAX_TEXT_BYTES:
            kind = ConflictKind.LARGE
        else:
            kind = ConflictKind.BINARY
        return _whole_file_conflict(app_id, path, destination, entry, local, local_sha, kind)

    if _normalise_newlines(local_text) == _normalise_newlines(backup_text):
        return None

    if (
        entry.size > MAX_TEXT_BYTES
        or len(local.data) > MAX_TEXT_BYTES
        or _line_count(backup_text) > MAX_DIFF_LINES
        or _line_count(local_text) > MAX_DIFF_LINES
    ):
        return _whole_file_conflict(
            app_id, path, destination, entry, local, local_sha, ConflictKind.LARGE
        )

    hunks = build_hunks(local_text, backup_text)
    return Conflict(
        app_id=app_id,
        path=path,
        destination=destination,
        kind=ConflictKind.TEXT,
        local_size=len(local.data),
        backup_size=entry.size,
        local_mode=local.mode,
        backup_mode=entry.mode,
        local_sha256=local_sha,
        backup_sha256=backup_sha,
        local_link=None,
        backup_link=None,
        local_text=local_text,
        backup_text=backup_text,
        hunks=hunks,
        validation_format=validation_format(path),
    )


def _whole_file_conflict(
    app_id: str,
    path: str,
    destination: Path,
    entry: FileEntry,
    local: LocalFile,
    local_sha: str | None,
    kind: ConflictKind,
) -> Conflict:
    return Conflict(
        app_id=app_id,
        path=path,
        destination=destination,
        kind=kind,
        local_size=len(local.data),
        backup_size=entry.size,
        local_mode=local.mode,
        backup_mode=entry.mode,
        local_sha256=local_sha,
        backup_sha256=entry.sha256,
        local_link=None,
        backup_link=None,
        local_text=None,
        backup_text=None,
        validation_format=validation_format(path),
    )


def build_hunks(local_text: str, backup_text: str) -> tuple[Hunk, ...]:
    """Return the change blocks between two text files."""
    local_lines = local_text.splitlines(keepends=True)
    backup_lines = backup_text.splitlines(keepends=True)
    local_norm = _normalise_newlines(local_text).splitlines(keepends=True)
    backup_norm = _normalise_newlines(backup_text).splitlines(keepends=True)
    matcher = SequenceMatcher(None, local_norm, backup_norm, autojunk=False)

    hunks: list[Hunk] = []
    for group in matcher.get_grouped_opcodes(CONTEXT_LINES):
        if not any(opcode[0] != "equal" for opcode in group):
            continue
        local_start = group[0][1]
        local_end = group[-1][2]
        backup_start = group[0][3]
        backup_end = group[-1][4]
        before: tuple[str, ...] = ()
        after: tuple[str, ...] = ()
        if group[0][0] == "equal":
            before = tuple(line.rstrip("\n") for line in local_norm[group[0][1] : group[0][2]])
        if group[-1][0] == "equal":
            after = tuple(line.rstrip("\n") for line in local_norm[group[-1][1] : group[-1][2]])
        hunks.append(
            Hunk(
                index=len(hunks) + 1,
                local_start=local_start + 1,
                local_end=local_end + 1,
                backup_start=backup_start + 1,
                backup_end=backup_end + 1,
                local_lines=tuple(local_lines[local_start:local_end]),
                backup_lines=tuple(backup_lines[backup_start:backup_end]),
                context_before=before,
                context_after=after,
            )
        )
    return tuple(hunks)


def compose(local_text: str, backup_text: str, decisions: Sequence[HunkDecision]) -> str:
    """Compose the result from the selected side of every hunk.

    Equal lines always come from the local file. Backup lines adopt the
    dominant local line ending, but a missing final newline is preserved.
    """
    local_lines = local_text.splitlines(keepends=True)
    backup_lines = backup_text.splitlines(keepends=True)
    local_norm = _normalise_newlines(local_text).splitlines(keepends=True)
    backup_norm = _normalise_newlines(backup_text).splitlines(keepends=True)
    matcher = SequenceMatcher(None, local_norm, backup_norm, autojunk=False)
    full = matcher.get_opcodes()
    grouped = list(matcher.get_grouped_opcodes(CONTEXT_LINES))
    if len(grouped) != len(decisions):
        raise ValueError("the decision list does not match the conflict hunks")

    hunk_for_opcode = _map_grouped_opcodes(full, grouped)
    eol = _dominant_eol(local_text)
    output: list[str] = []
    for index, (tag, i1, i2, j1, j2) in enumerate(full):
        if tag == "equal":
            output.extend(local_lines[i1:i2])
            continue
        hunk_index = hunk_for_opcode[index]
        if decisions[hunk_index] == HunkDecision.LOCAL:
            output.extend(local_lines[i1:i2])
        else:
            output.extend(_adapt_line(line, eol) for line in backup_lines[j1:j2])
    return "".join(output)


def _map_grouped_opcodes(
    full: Sequence[tuple[str, int, int, int, int]],
    grouped: Sequence[Sequence[tuple[str, int, int, int, int]]],
) -> dict[int, int]:
    """Map each non-equal full opcode to its hunk index."""
    mapped: dict[int, int] = {}
    cursor = 0
    for hunk_index, group in enumerate(grouped):
        for opcode in group:
            if opcode[0] == "equal":
                continue
            found = None
            for index in range(cursor, len(full)):
                if full[index] == opcode:
                    found = index
                    break
            if found is None:
                raise ValueError("inconsistent diff grouping")
            cursor = found + 1
            mapped[found] = hunk_index
    return mapped


def _dominant_eol(text: str) -> str:
    crlf = text.count("\r\n")
    lf = text.count("\n") - crlf
    return "\r\n" if crlf > lf else "\n"


def _adapt_line(line: str, eol: str) -> str:
    if line.endswith("\r\n"):
        return line[:-2] + eol
    if line.endswith("\n"):
        return line[:-1] + eol
    return line


def validate_merge(
    path: str,
    local_text: str,
    backup_text: str,
    merged_text: str,
) -> MergeValidation:
    """Validate a composed file with the standard library.

    Only block when both originals parse and the merge does not. If either
    original was already invalid, return a warning and allow applying.
    """
    parser = _parser_for(path)
    if parser is None:
        return MergeValidation(
            _MergeStatus.UNVALIDATED, "este formato no se valida automáticamente"
        )
    local_error = parser(local_text)
    backup_error = parser(backup_text)
    if local_error is not None or backup_error is not None:
        return MergeValidation(
            _MergeStatus.WARNING,
            "uno de los originales ya era inválido; se permite aplicar la mezcla",
        )
    merged_error = parser(merged_text)
    if merged_error is not None:
        return MergeValidation(_MergeStatus.INVALID, merged_error)
    return MergeValidation(_MergeStatus.VALID)


def validation_format(path: str) -> str | None:
    """Return the display name of the validator for ``path``, if any."""
    suffix = Path(path).suffix.lower()
    if suffix == ".toml":
        return "TOML"
    if suffix == ".json":
        return "JSON"
    if suffix in {".ini", ".cfg", ".gitconfig"}:
        return "INI"
    if suffix == ".plist":
        return "plist"
    return None


def apply_all(conflicts: Sequence[Conflict]) -> ConflictDecisions:
    """Decide every conflict by applying the backup side."""
    return ConflictDecisions(
        {conflict.path: FileDecision(FileAction.APPLY) for conflict in conflicts}
    )


def omit_all(conflicts: Sequence[Conflict]) -> ConflictDecisions:
    """Decide every conflict by keeping the local side."""
    return ConflictDecisions(
        {conflict.path: FileDecision(FileAction.OMIT) for conflict in conflicts}
    )


def review_all(conflict: Conflict, decision: HunkDecision = HunkDecision.BACKUP) -> FileDecision:
    """Build a ``REVIEW`` decision with every hunk on the same side."""
    return FileDecision(FileAction.REVIEW, tuple(decision for _hunk in conflict.hunks))


def _decode_text(data: bytes) -> str | None:
    if b"\x00" in data:
        return None
    try:
        return data.decode("utf-8")
    except UnicodeDecodeError:
        return None


def _normalise_newlines(text: str) -> str:
    return text.replace("\r\n", "\n")


def _line_count(text: str) -> int:
    return len(_normalise_newlines(text).splitlines())


def _parser_for(path: str):
    suffix = Path(path).suffix.lower()
    if suffix == ".toml":
        return _parse_toml
    if suffix == ".json":
        return _parse_json
    if suffix in {".ini", ".cfg", ".gitconfig"}:
        return _parse_ini
    if suffix == ".plist":
        return _parse_plist
    return None


def _parse_toml(text: str) -> str | None:
    try:
        tomllib.loads(text)
    except (tomllib.TOMLDecodeError, ValueError) as exc:
        return f"TOML inválido: {exc}"
    return None


def _parse_json(text: str) -> str | None:
    def reject_duplicates(pairs: list[tuple[str, object]]) -> dict[str, object]:
        result: dict[str, object] = {}
        for key, value in pairs:
            if key in result:
                raise ValueError(f"clave duplicada: {key!r}")
            result[key] = value
        return result

    try:
        json.loads(text, object_pairs_hook=reject_duplicates)
    except (json.JSONDecodeError, ValueError) as exc:
        return f"JSON inválido: {exc}"
    return None


def _parse_ini(text: str) -> str | None:
    parser = configparser.ConfigParser(strict=True, interpolation=None, default_section="\0")
    try:
        parser.read_string(text)
    except configparser.Error as exc:
        return f"INI inválido: {exc}"
    return None


def _parse_plist(text: str) -> str | None:
    data = text.encode("utf-8")
    try:
        plistlib.loads(data)
    except (plistlib.InvalidFileException, ValueError) as exc:
        return f"plist inválido: {exc}"
    duplicate = _plist_duplicate_key(data)
    if duplicate is not None:
        return f"plist inválido: clave duplicada: {duplicate!r}"
    return None


def _plist_duplicate_key(data: bytes) -> str | None:
    try:
        root = ElementTree.fromstring(data)
    except ElementTree.ParseError:
        return None
    for element in root.iter("dict"):
        seen: set[str] = set()
        for child in element:
            if child.tag != "key":
                continue
            key = child.text or ""
            if key in seen:
                return key
            seen.add(key)
    return None
