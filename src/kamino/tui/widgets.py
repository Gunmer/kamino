"""Shared widgets and render helpers for the restore TUI."""

from __future__ import annotations

from rich.text import Text

from kamino.conflicts import Conflict, ConflictKind, Hunk, HunkDecision


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


def decision_label(decision: HunkDecision | None) -> str:
    if decision is HunkDecision.LOCAL:
        return "local"
    return "respaldo"


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
        return f"{len(conflict.hunks)} bloques"
    return f"{format_size(conflict.local_size)} / {format_size(conflict.backup_size)}"


def hunk_text(hunk: Hunk, decision: HunkDecision) -> Text:
    """Render one hunk with both sides and the current decision."""
    local_last = max(hunk.local_start, hunk.local_end - 1)
    backup_last = max(hunk.backup_start, hunk.backup_end - 1)
    text = Text()
    text.append(f"Bloque {hunk.index}", style="bold")
    text.append(
        f" · local {hunk.local_start}-{local_last} · respaldo {hunk.backup_start}-{backup_last}"
        f" · decisión: {decision_label(decision)}\n",
        style="cyan" if decision is HunkDecision.BACKUP else "yellow",
    )
    for line in hunk.local_lines:
        style = "red" if decision is HunkDecision.LOCAL else "dim red"
        text.append(f"- {line.rstrip()}\n", style=style)
    for line in hunk.backup_lines:
        style = "green" if decision is HunkDecision.BACKUP else "dim green"
        text.append(f"+ {line.rstrip()}\n", style=style)
    return text


def conflict_warning() -> Text:
    text = Text()
    text.append(
        "Los diffs pueden contener datos sensibles (tokens, claves o rutas). "
        "No se guardan ni se registran fuera de esta pantalla.",
        style="yellow",
    )
    return text
