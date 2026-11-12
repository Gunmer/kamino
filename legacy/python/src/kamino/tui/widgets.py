"""Widgets y helpers de render compartidos por las pantallas."""

from __future__ import annotations

from rich.text import Text

from kamino import texts
from kamino.conflicts import Conflict, ConflictKind, Hunk, HunkRow, RowDecision, RowKind


def format_size(size: int) -> str:
    """Tamaño compacto y legible."""
    return texts.size_label(size)


def conflict_kind_label(conflict: Conflict) -> str:
    return {
        ConflictKind.TEXT: "texto",
        ConflictKind.BINARY: "binario",
        ConflictKind.SYMLINK: "enlace",
        ConflictKind.LARGE: "texto grande (>5 MB)",
    }[conflict.kind]


def conflict_detail(conflict: Conflict) -> str:
    if conflict.kind is ConflictKind.SYMLINK:
        return f"{conflict.local_link} → {conflict.clone_link}"
    if conflict.kind is ConflictKind.TEXT:
        lines = sum(
            1 for hunk in conflict.hunks for row in hunk.rows if row.kind is not RowKind.EQUAL
        )
        return f"{len(conflict.hunks)} bloques · {lines} {'línea' if lines == 1 else 'líneas'}"
    return f"{format_size(conflict.local_size)} / {format_size(conflict.clone_size)}"


def review_label(clone_rows: int, total_rows: int) -> str:
    return f"{texts.HYBRID} ({clone_rows} de {total_rows} líneas)"


def _display(line: str | None) -> str:
    return (line or "").rstrip("\r\n")


def hunk_header(hunk: Hunk, total: int | None = None) -> Text:
    """Cabecera no decidible del bloque actual."""
    text = Text()
    suffix = f" de {total}" if total is not None else ""
    text.append(f"Bloque {hunk.index}{suffix}", style="bold")
    changed = sum(1 for row in hunk.rows if row.kind is not RowKind.EQUAL)
    text.append(
        f" · {changed} {'línea' if changed == 1 else 'líneas'} cambiadas",
        style="cyan",
    )
    return text


def context_text(row: HunkRow) -> Text:
    """Contexto no decidible, mostrado una sola vez."""
    return Text(f"  {_display(row.local)}", style="dim")


def _marker(chosen: bool, unicode: bool = True) -> str:
    if unicode:
        return texts.MARKER_CHOSEN if chosen else texts.MARKER_NOT_CHOSEN
    return texts.MARKER_CHOSEN_ASCII if chosen else texts.MARKER_NOT_CHOSEN_ASCII


def row_text(
    row: HunkRow,
    decision: RowDecision,
    *,
    line_numbers: bool = False,
    unicode: bool = True,
) -> Text:
    """Render one decidable row with both sides and the final-file marker."""
    text = Text()
    local_selected = decision is RowDecision.LOCAL
    if row.kind is RowKind.REPLACE:
        text.append(
            f"  {_marker(local_selected, unicode)} original  {_display(row.local)}",
            style="yellow",
        )
        text.append("\n")
        text.append(
            f"  {_marker(not local_selected, unicode)} clon      {_display(row.clone)}",
            style="green" if not local_selected else "dim",
        )
    elif row.kind is RowKind.LOCAL_ONLY:
        label = texts.DEPLOY_ROW_ORIGINAL_ONLY
        text.append(
            f"  {_marker(local_selected, unicode)} {label}  {_display(row.local)}"
            + ("" if local_selected else f"  [{texts.DEPLOY_ROW_CLONE}]"),
            style="yellow" if local_selected else "dim",
        )
    else:
        label = texts.DEPLOY_ROW_CLONE_ONLY
        text.append(
            f"  {_marker(not local_selected, unicode)} {label}  {_display(row.clone)}"
            + ("" if not local_selected else f"  [{texts.DEPLOY_ROW_ORIGINAL}]"),
            style="green" if not local_selected else "dim",
        )
    return text


def render_block(
    hunk: Hunk,
    decisions: list[RowDecision],
    *,
    cursor: int | None = None,
    unicode: bool = True,
) -> Text:
    """Render an entire block; context once and every changed row with markers."""
    text = Text()
    decision_index = 0
    for row in hunk.rows:
        if row.kind is RowKind.EQUAL:
            text.append_text(context_text(row))
            text.append("\n")
            continue
        decision = decisions[decision_index]
        rendered = row_text(row, decision, unicode=unicode)
        if cursor is not None and decision_index == cursor:
            rendered.stylize("reverse")
        text.append_text(rendered)
        text.append("\n")
        decision_index += 1
    return text


def conflict_warning() -> Text:
    return Text(texts.DIFF_WARNING, style="yellow")
