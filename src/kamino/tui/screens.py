"""Screens for the interactive deploy review."""

from __future__ import annotations

from collections.abc import Sequence

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal, Vertical
from textual.screen import Screen
from textual.widgets import (
    Button,
    DataTable,
    Footer,
    Header,
    ListItem,
    ListView,
    SelectionList,
    Static,
)
from textual.widgets.selection_list import Selection

from kamino.archive import AppEntry
from kamino.conflicts import (
    Conflict,
    ConflictDecisions,
    ConflictKind,
    FileAction,
    FileDecision,
    HunkRow,
    RowDecision,
    RowKind,
    apply_all,
    compose,
    omit_all,
    row_count,
    validate_merge,
)
from kamino.deploy import DeployClassification, DeployPlan, DeployResult, DeploySession
from kamino.errors import ConcurrentChangeError, ConflictError, KaminoError
from kamino.tui.widgets import (
    conflict_detail,
    conflict_kind_label,
    conflict_warning,
    context_text,
    hunk_header,
    row_text,
)

_ACTION_LABELS = {
    FileAction.APPLY: "aplicar clon",
    FileAction.OMIT: "omitir (local)",
}


class AppSelectionScreen(Screen[list[str] | None]):
    """Choose which deployable apps to include."""

    BINDINGS = [Binding("escape", "cancel", "Cancelar")]

    def __init__(self, apps: Sequence[AppEntry]) -> None:
        super().__init__()
        self.apps = list(apps)

    def compose(self) -> ComposeResult:
        yield Header(show_clock=False)
        yield Static(
            "Selecciona las apps que quieres desplegar. Ninguna está marcada por defecto.",
            id="title",
        )
        if self.apps:
            selections = [
                Selection(f"{app.name} ({app.id})", app.id, initial_state=False)
                for app in self.apps
            ]
            yield SelectionList[str](*selections, id="apps")
        else:
            yield Static("Este clon no tiene apps desplegables en esta máquina.", id="empty")
        with Horizontal(id="buttons"):
            yield Button("Seleccionar todas", id="all")
            yield Button("Seleccionar ninguna", id="none")
            yield Button("Continuar", id="continue", variant="primary")
            yield Button("Cancelar", id="cancel", variant="error")
        yield Footer()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        if event.button.id == "cancel":
            self.dismiss(None)
            return
        if event.button.id == "all":
            if self.apps:
                self.query_one("#apps", SelectionList).select_all()
            return
        if event.button.id == "none":
            if self.apps:
                self.query_one("#apps", SelectionList).deselect_all()
            return
        if event.button.id == "continue":
            selected = list(self.query_one("#apps", SelectionList).selected) if self.apps else []
            self.dismiss(selected)

    def action_cancel(self) -> None:
        self.dismiss(None)


class ConflictSummaryScreen(Screen[None]):
    """List every conflict and collect the global or per-file decisions."""

    BINDINGS = [
        Binding("escape", "cancel", "Cancelar"),
        Binding("question_mark", "help", "Ayuda"),
    ]

    def __init__(
        self,
        session: DeploySession,
        plan: DeployPlan,
        decisions: ConflictDecisions,
    ) -> None:
        super().__init__()
        self.session = session
        self.plan = plan
        self.decisions = decisions
        self._row_paths: list[str] = []

    def compose(self) -> ComposeResult:
        yield Header(show_clock=False)
        yield Static("", id="counts")
        yield DataTable(id="conflicts")
        yield Static("", id="status")
        with Vertical(id="actions"):
            with Horizontal(classes="action-row"):
                yield Button("Revisar", id="review")
                yield Button("Aplicar", id="apply", variant="success")
                yield Button("Omitir", id="omit", variant="warning")
                yield Button("Aplicar todo", id="apply-all", variant="success")
                yield Button("Omitir todo", id="omit-all", variant="warning")
            with Horizontal(classes="action-row"):
                yield Button("Confirmar", id="confirm", variant="primary", disabled=True)
                yield Button("Cancelar", id="cancel", variant="error")
                yield Button("Ayuda", id="help")
        yield Footer()

    def on_mount(self) -> None:
        table = self.query_one(DataTable)
        table.cursor_type = "row"
        table.zebra_stripes = True
        table.add_columns("App", "Archivo", "Tipo", "Detalle", "Decisión")
        self.refresh_table()

    def refresh_table(self) -> None:
        table = self.query_one(DataTable)
        selected = table.cursor_row if table.row_count else 0
        table.clear()
        self._row_paths = []
        for conflict in self.plan.conflicts:
            self._row_paths.append(conflict.path)
            table.add_row(
                conflict.app_id,
                conflict.path,
                conflict_kind_label(conflict),
                conflict_detail(conflict),
                self._decision_text(conflict),
            )
        if self._row_paths:
            table.move_cursor(row=min(selected, len(self._row_paths) - 1))
        self._refresh_counts()
        self._refresh_status()

    def _decision_text(self, conflict: Conflict) -> str:
        decision = self.decisions.get(conflict.path)
        if decision is None:
            return "sin decidir"
        if decision.action is FileAction.REVIEW:
            clone = sum(1 for item in decision.rows if item is RowDecision.CLONE)
            return f"revisión ({clone}/{len(decision.rows)} clon)"
        return _ACTION_LABELS[decision.action]

    def _refresh_counts(self) -> None:
        new = sum(1 for entry in self.plan.entries if entry.status.value == "new")
        identical = sum(1 for entry in self.plan.entries if entry.status.value == "identical")
        unsupported = sum(1 for entry in self.plan.entries if entry.status.value == "unsupported")
        self.query_one("#counts", Static).update(
            f"Conflictos: {len(self.plan.conflicts)} · Nuevos: {new} · "
            f"Idénticos: {identical} · No soportados: {unsupported}"
        )

    def _refresh_status(self) -> None:
        missing = self.decisions.missing(self.plan.conflicts)
        confirm = self.query_one("#confirm", Button)
        confirm.disabled = bool(missing)
        status = self.query_one("#status", Static)
        if missing:
            names = ", ".join(conflict.path for conflict in missing[:3])
            extra = "" if len(missing) <= 3 else f" y {len(missing) - 3} más"
            status.update(f"Falta decidir: {names}{extra}")
        else:
            status.update("Todos los conflictos tienen decisión. Puedes confirmar.")

    def _selected_conflict(self) -> Conflict | None:
        if not self._row_paths:
            return None
        row = self.query_one(DataTable).cursor_row
        if row < 0 or row >= len(self._row_paths):
            return None
        path = self._row_paths[row]
        return next((conflict for conflict in self.plan.conflicts if conflict.path == path), None)

    def on_button_pressed(self, event: Button.Pressed) -> None:
        button = event.button.id
        if button == "cancel":
            self.action_cancel()
            return
        if button == "help":
            self.app.push_screen(HelpScreen())
            return
        if button == "apply-all":
            self.decisions.decisions.update(apply_all(self.plan.conflicts).decisions)
            self.refresh_table()
            return
        if button == "omit-all":
            self.decisions.decisions.update(omit_all(self.plan.conflicts).decisions)
            self.refresh_table()
            return
        if button == "confirm":
            self._confirm()
            return
        conflict = self._selected_conflict()
        if conflict is None:
            self._refresh_status()
            return
        if button == "apply":
            self.decisions.set(conflict.path, FileDecision(FileAction.APPLY))
        elif button == "omit":
            self.decisions.set(conflict.path, FileDecision(FileAction.OMIT))
        elif button == "review":
            self._review(conflict)
            return
        self.refresh_table()

    def _review(self, conflict: Conflict) -> None:
        if conflict.kind is not ConflictKind.TEXT:
            self.query_one("#status", Static).update(
                "Este conflicto se resuelve como archivo entero: aplica u omite."
            )
            return
        existing = self.decisions.get(conflict.path)
        self.app.push_screen(
            ConflictDetailScreen(conflict, existing),
            lambda decision: self._reviewed(conflict, decision),
        )

    def _reviewed(self, conflict: Conflict, decision: FileDecision | None) -> None:
        if decision is not None:
            self.decisions.set(conflict.path, decision)
        self.refresh_table()

    def _confirm(self) -> None:
        try:
            classification = self.session.preview(self.decisions)
        except ConflictError as exc:
            self.query_one("#status", Static).update(str(exc))
            return
        self.app.push_screen(
            ConfirmScreen(self.session, classification, self.decisions),
            self._confirmed,
        )

    def _confirmed(self, action: str | None) -> None:
        if action == "cancel":
            self.app.exit(None)
            return
        if action != "apply":
            return
        try:
            result = self.session.apply(self.decisions)
        except ConcurrentChangeError as exc:
            self.app.error = exc  # type: ignore[attr-defined]
            self.app.push_screen(ErrorScreen(str(exc), fatal=True))
            return
        except ConflictError as exc:
            self.query_one("#status", Static).update(str(exc))
            return
        except KaminoError as exc:
            self.app.error = exc  # type: ignore[attr-defined]
            self.app.push_screen(ErrorScreen(str(exc), fatal=True))
            return
        self.app.push_screen(SummaryScreen(result))

    def action_cancel(self) -> None:
        self.app.exit(None)

    def action_help(self) -> None:
        self.app.push_screen(HelpScreen())


class ConflictDetailScreen(Screen[FileDecision | None]):
    """Review a text conflict line by line."""

    BINDINGS = [
        Binding("escape", "save", "Volver"),
        Binding("space", "toggle", "Alternar"),
        Binding("enter", "toggle", "Alternar"),
        Binding("a", "all_clone", "Todo clon"),
        Binding("l", "all_local", "Todo local"),
        Binding("A", "block_clone", "Bloque clon"),
        Binding("L", "block_local", "Bloque local"),
        Binding("n", "next_block", "Siguiente bloque"),
        Binding("p", "prev_block", "Bloque anterior"),
        Binding("question_mark", "help", "Ayuda"),
    ]

    def __init__(self, conflict: Conflict, decision: FileDecision | None) -> None:
        super().__init__()
        self.conflict = conflict
        self.decisions = self._initial_decisions(decision)
        self._rows: list[HunkRow] = []
        self._item_to_row: list[int | None] = []
        self._item_block: list[int] = []
        self._row_widgets: list[Static] = []
        self._row_block: list[int] = []
        self._block_first_item: list[int] = []

    def _initial_decisions(self, decision: FileDecision | None) -> list[RowDecision]:
        if (
            decision is not None
            and decision.action is FileAction.REVIEW
            and len(decision.rows) == row_count(self.conflict)
        ):
            return list(decision.rows)
        return [RowDecision.CLONE for _row in range(row_count(self.conflict))]

    def compose(self) -> ComposeResult:
        yield Header(show_clock=False)
        validation = self.conflict.validation_format or "sin validación automática"
        yield Static(
            f"{self.conflict.path} · {conflict_kind_label(self.conflict)} · "
            f"validación: {validation}",
            id="title",
        )
        yield Static(conflict_warning(), id="warning")
        yield ListView(id="hunks")
        yield Static("", id="hunk-error")
        with Vertical(id="actions"):
            with Horizontal(classes="action-row"):
                yield Button("Alternar", id="toggle", variant="primary")
                yield Button("Todo clon (A)", id="all-clone")
                yield Button("Todo local (L)", id="all-local")
            with Horizontal(classes="action-row"):
                yield Button("Bloque clon (Shift+A)", id="block-clone")
                yield Button("Bloque local (Shift+L)", id="block-local")
            with Horizontal(classes="action-row"):
                yield Button("Volver", id="back", variant="warning")
                yield Button("Ayuda", id="help")
        yield Footer()

    async def on_mount(self) -> None:
        await self._build_list()

    async def _build_list(self) -> None:
        list_view = self.query_one("#hunks", ListView)
        items: list[ListItem] = []
        for block, hunk in enumerate(self.conflict.hunks):
            items.append(ListItem(Static(hunk_header(hunk)), disabled=True))
            self._item_to_row.append(None)
            self._item_block.append(block)
            first_item: int | None = None
            for row in hunk.rows:
                if row.kind is RowKind.EQUAL:
                    items.append(ListItem(Static(context_text(row)), disabled=True))
                    self._item_to_row.append(None)
                    self._item_block.append(block)
                    continue
                if first_item is None:
                    first_item = len(items)
                row_index = len(self._rows)
                widget = Static(row_text(row, self.decisions[row_index]))
                items.append(ListItem(widget))
                self._item_to_row.append(row_index)
                self._item_block.append(block)
                self._row_widgets.append(widget)
                self._row_block.append(block)
                self._rows.append(row)
            if first_item is not None:
                self._block_first_item.append(first_item)
        await list_view.extend(items)
        if self._block_first_item:
            list_view.index = self._block_first_item[0]
        list_view.focus()

    def _current_row(self) -> int | None:
        list_view = self.query_one("#hunks", ListView)
        index = list_view.index
        if index is None or index < 0 or index >= len(self._item_to_row):
            return None
        return self._item_to_row[index]

    def _current_block(self) -> int | None:
        list_view = self.query_one("#hunks", ListView)
        index = list_view.index
        if index is None or index < 0 or index >= len(self._item_block):
            return None
        return self._item_block[index]

    def _update_row(self, row_index: int) -> None:
        row = self._rows[row_index]
        self._row_widgets[row_index].update(row_text(row, self.decisions[row_index]))

    def on_list_view_selected(self, event: ListView.Selected) -> None:
        event.stop()
        index = event.index
        if index is None or index < 0 or index >= len(self._item_to_row):
            return
        row_index = self._item_to_row[index]
        if row_index is None:
            self._move_to_nearest_decidable(index)
            return
        self._toggle_row(row_index)

    def _move_to_nearest_decidable(self, index: int) -> None:
        for offset in range(1, len(self._item_to_row)):
            for candidate in (index - offset, index + offset):
                if (
                    0 <= candidate < len(self._item_to_row)
                    and self._item_to_row[candidate] is not None
                ):
                    self.query_one("#hunks", ListView).index = candidate
                    return

    def _toggle_row(self, row_index: int) -> None:
        self.decisions[row_index] = (
            RowDecision.LOCAL
            if self.decisions[row_index] is RowDecision.CLONE
            else RowDecision.CLONE
        )
        self._update_row(row_index)

    def _set_all(self, decision: RowDecision) -> None:
        self.decisions = [decision for _item in self.decisions]
        for row_index in range(len(self.decisions)):
            self._update_row(row_index)

    def _set_block(self, decision: RowDecision) -> None:
        block = self._current_block()
        if block is None:
            return
        for row_index, row_block in enumerate(self._row_block):
            if row_block == block:
                self.decisions[row_index] = decision
                self._update_row(row_index)

    def _jump_block(self, step: int) -> None:
        block = self._current_block()
        if block is None:
            return
        target = block + step
        if 0 <= target < len(self._block_first_item):
            self.query_one("#hunks", ListView).index = self._block_first_item[target]

    def on_button_pressed(self, event: Button.Pressed) -> None:
        button = event.button.id
        if button == "toggle":
            self.action_toggle()
        elif button == "all-clone":
            self._set_all(RowDecision.CLONE)
        elif button == "all-local":
            self._set_all(RowDecision.LOCAL)
        elif button == "block-clone":
            self._set_block(RowDecision.CLONE)
        elif button == "block-local":
            self._set_block(RowDecision.LOCAL)
        elif button == "back":
            self.action_save()
        elif button == "help":
            self.app.push_screen(HelpScreen())

    def action_toggle(self) -> None:
        row_index = self._current_row()
        if row_index is None:
            index = self.query_one("#hunks", ListView).index
            if index is not None:
                self._move_to_nearest_decidable(index)
            return
        self._toggle_row(row_index)

    def action_all_clone(self) -> None:
        self._set_all(RowDecision.CLONE)

    def action_all_local(self) -> None:
        self._set_all(RowDecision.LOCAL)

    def action_block_clone(self) -> None:
        self._set_block(RowDecision.CLONE)

    def action_block_local(self) -> None:
        self._set_block(RowDecision.LOCAL)

    def action_next_block(self) -> None:
        self._jump_block(1)

    def action_prev_block(self) -> None:
        self._jump_block(-1)

    def action_save(self) -> None:
        if self.conflict.local_text is None or self.conflict.clone_text is None:
            self.dismiss(None)
            return
        merged = compose(self.conflict.local_text, self.conflict.clone_text, self.decisions)
        validation = validate_merge(
            self.conflict.path,
            self.conflict.local_text,
            self.conflict.clone_text,
            merged,
        )
        if validation.status == "invalid":
            self.query_one("#hunk-error", Static).update(
                f"Mezcla inválida: {validation.message}. Corrige las líneas o vuelve a intentarlo."
            )
            return
        decision = FileDecision(FileAction.REVIEW, tuple(self.decisions))
        if validation.status == "warning":
            self.app.push_screen(
                WarningScreen(validation.message),
                lambda apply: self.dismiss(decision) if apply else None,
            )
            return
        self.dismiss(decision)

    def action_help(self) -> None:
        self.app.push_screen(HelpScreen())


class ConfirmScreen(Screen[str | None]):
    """Final confirmation with prospective counts and the snapshot directory."""

    BINDINGS = [Binding("escape", "back", "Volver")]

    def __init__(
        self,
        session: DeploySession,
        classification: DeployClassification,
        decisions: ConflictDecisions,
    ) -> None:
        super().__init__()
        self.session = session
        self.classification = classification
        self.decisions = decisions

    def compose(self) -> ComposeResult:
        yield Header(show_clock=False)
        yield Static("Confirmación del despliegue", id="title")
        yield Static(
            f"Desplegados: {len(self.classification.deployed)} · "
            f"Mezclados: {len(self.classification.merged)} · "
            f"Omitidos: {len(self.classification.omitted)}",
            id="counts",
        )
        yield Static("Snapshot previo en:", id="snapshot-label")
        yield Static(str(self.session.snapshot_directory), id="snapshot-dir")
        yield Static(
            "Se escribirá solo lo indicado. Volver o cancelar no escribe ni crea snapshot.",
            id="warning",
        )
        with Horizontal(id="actions"):
            yield Button("Aplicar", id="apply", variant="primary")
            yield Button("Volver", id="back", variant="warning")
            yield Button("Cancelar", id="cancel", variant="error")
        yield Footer()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        if event.button.id == "apply":
            self.dismiss("apply")
        elif event.button.id == "back":
            self.action_back()
        elif event.button.id == "cancel":
            self.dismiss("cancel")

    def action_back(self) -> None:
        self.dismiss("back")


class SummaryScreen(Screen[None]):
    """Final result after applying the deploy."""

    BINDINGS = [Binding("escape", "exit", "Salir")]

    def __init__(self, result: DeployResult) -> None:
        super().__init__()
        self.result = result

    def compose(self) -> ComposeResult:
        yield Header(show_clock=False)
        yield Static("Despliegue completado", id="title")
        yield Static(
            f"Desplegados: {len(self.result.deployed)} · "
            f"Mezclados: {len(self.result.merged)} · "
            f"Omitidos: {len(self.result.omitted)}",
            id="counts",
        )
        snapshot = str(self.result.snapshot) if self.result.snapshot else "no se creó (sin cambios)"
        yield Static(f"Snapshot: {snapshot}", id="snapshot")
        with Horizontal(id="actions"):
            yield Button("Salir", id="exit", variant="primary")
        yield Footer()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        if event.button.id == "exit":
            self.action_exit()

    def action_exit(self) -> None:
        self.app.exit(self.result)


class HelpScreen(Screen[None]):
    """Spanish help for keys, flow and secrets."""

    BINDINGS = [Binding("escape", "close", "Cerrar")]

    def compose(self) -> ComposeResult:
        yield Header(show_clock=False)
        yield Static("Ayuda del despliegue interactivo", id="title")
        yield Static(
            "Apps: ninguna está marcada por defecto. Usa Seleccionar todas, "
            "Seleccionar ninguna, Continuar o Cancelar.\n\n"
            "Conflictos: Revisar abre el detalle por líneas; Aplicar despliega el "
            "archivo del clon; Omitir conserva el local. Aplicar todo y Omitir "
            "todo afectan a todos los conflictos y luego puedes cambiarlos uno a uno.\n\n"
            "Detalle: Espacio o Enter alterna la línea resaltada. A toma todas las "
            "líneas del clon y L todas las locales; Shift+A y Shift+L solo el "
            "bloque actual. n y p saltan entre bloques. Escape vuelve y valida la mezcla.\n\n"
            "Confirmación: Aplicar escribe; Volver regresa a la tabla; Cancelar no "
            "escribe ni crea snapshot. Escape sale.\n\n"
            "Los diffs pueden contener secretos. No se guardan ni se envían a ningún sitio.",
            id="help-body",
        )
        with Horizontal(id="actions"):
            yield Button("Cerrar", id="close", variant="primary")
        yield Footer()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        if event.button.id == "close":
            self.action_close()

    def action_close(self) -> None:
        self.dismiss(None)


class WarningScreen(Screen[bool]):
    """Warn that an original was invalid and let the user apply anyway."""

    BINDINGS = [Binding("escape", "back", "Volver")]

    def __init__(self, message: str) -> None:
        super().__init__()
        self.message = message

    def compose(self) -> ComposeResult:
        yield Header(show_clock=False)
        yield Static("Aviso de validación", id="title")
        yield Static(
            f"{self.message}. Puedes aplicar la mezcla igualmente.",
            id="warning",
        )
        with Horizontal(id="actions"):
            yield Button("Aplicar de todas formas", id="apply", variant="warning")
            yield Button("Volver al detalle", id="back", variant="primary")
        yield Footer()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        if event.button.id == "apply":
            self.dismiss(True)
        elif event.button.id == "back":
            self.action_back()

    def action_back(self) -> None:
        self.dismiss(False)


class ErrorScreen(Screen[None]):
    """Fatal or recoverable error in the interactive flow."""

    BINDINGS = [Binding("escape", "dismiss_error", "Volver")]

    def __init__(self, message: str, *, fatal: bool = False) -> None:
        super().__init__()
        self.message = message
        self.fatal = fatal

    def compose(self) -> ComposeResult:
        yield Header(show_clock=False)
        yield Static("No se pudo completar el despliegue", id="title")
        yield Static(self.message, id="error")
        with Horizontal(id="actions"):
            if not self.fatal:
                yield Button("Volver", id="back", variant="warning")
            yield Button("Salir", id="exit", variant="error")
        yield Footer()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        if event.button.id == "exit":
            self.app.exit(None)
        elif event.button.id == "back":
            self.action_dismiss_error()

    def action_dismiss_error(self) -> None:
        if self.fatal:
            self.app.exit(None)
        else:
            self.dismiss(None)
