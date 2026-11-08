"""Shared widgets and render helpers for the restore TUI."""

from __future__ import annotations

from rich.text import Text

from kamino.conflicts import (
    Conflict,
    ConflictKind,
    Hunk,
    HunkRow,
    RowDecision,
    RowKind,
    row_count,
)


def format_size(size: int) -> str:
    """Return a compact human-readable size."""
    value = float(size)
    for unit in ("B", "KB", "MB", "GB"):
        if value < 1024 or unit == "GB":
            if unit == "B":
                return f"{int(value)} {unit}"
            return f"{value:.1f} {unit}"
        value /= 1024
    return f"{size} B"


def conflict_kind_label(conflict: Conflict) -> str:
    return {
        ConflictKind.TEXT: "texto",
        ConflictKind.BINARY: "binario",
        ConflictKind.SYMLINK: "enlace",
        ConflictKind.LARGE: "grande (>5 MB)",
    }[conflict.kind]


def conflict_detail(conflict: Conflict) -> str:
    if conflict.kind is ConflictKind.SYMLINK:
        return f"{conflict.local_link} → {conflict.backup_link}"
    if conflict.kind is ConflictKind.TEXT:
        lines = row_count(conflict)
        return f"{len(conflict.hunks)} bloques · {lines} {'línea' if lines == 1 else 'líneas'}"
    return f"{format_size(conflict.local_size)} / {format_size(conflict.backup_size)}"


def _display(line: str | None) -> str:
    return (line or "").rstrip("\r\n")


def hunk_header(hunk: Hunk) -> Text:
    """Render a non-decidable block header."""
    local_range = _range(hunk.local_start, hunk.local_end)
    backup_range = _range(hunk.backup_start, hunk.backup_end)
    changed = sum(1 for row in hunk.rows if row.kind is not RowKind.EQUAL)
    text = Text()
    text.append(f"Bloque {hunk.index}", style="bold")
    text.append(
        f" · local {local_range} · respaldo {backup_range} · {changed} "
        f"{'línea' if changed == 1 else 'líneas'}",
        style="cyan",
    )
    return text


def _range(start: int, end: int) -> str:
    if end <= start:
        return "-"
    return f"{start}-{end - 1}"


def context_text(row: HunkRow) -> Text:
    """Render a non-decidable context line."""
    return Text(f"  {_display(row.local)}", style="dim")


def row_text(row: HunkRow, decision: RowDecision) -> Text:
    """Render a decidable row with the chosen side highlighted."""
    local_selected = decision is RowDecision.LOCAL
    text = Text()
    if row.kind is RowKind.REPLACE:
        _append_pair(text, row.local, row.backup, local_selected)
    elif row.kind is RowKind.LOCAL_ONLY:
        _append_single(text, "-", row.local, local_selected, "local", "red")
    else:
        _append_single(text, "+", row.backup, not local_selected, "respaldo", "green")
    return text


def _append_pair(text: Text, local: str | None, backup: str | None, local_selected: bool) -> None:
    if local_selected:
        text.append(f"* - {_display(local)}  [local]", style="bold red")
        text.append("\n")
        text.append(f"  + {_display(backup)}", style="dim green")
    else:
        text.append(f"  - {_display(local)}", style="dim red")
        text.append("\n")
        text.append(f"* + {_display(backup)}  [respaldo]", style="bold green")


def _append_single(
    text: Text,
    prefix: str,
    line: str | None,
    chosen: bool,
    label: str,
    color: str,
) -> None:
    if chosen:
        text.append(f"* {prefix} {_display(line)}  [{label}]", style=f"bold {color}")
    else:
        text.append(f"  {prefix} {_display(line)}  [omitir]", style=f"dim {color}")


def conflict_warning() -> Text:
    text = Text()
    text.append(
        "Los diffs pueden contener datos sensibles (tokens, claves o rutas). "
        "No se guardan ni se registran fuera de esta pantalla.",
        style="yellow",
    )
    return text
