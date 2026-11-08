"""Screens for the interactive restore review."""

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
    HunkDecision,
    apply_all,
    compose,
    omit_all,
    validate_merge,
)
from kamino.errors import ConcurrentChangeError, ConflictError, KaminoError
from kamino.restore import RestoreClassification, RestorePlan, RestoreResult, RestoreSession
from kamino.tui.widgets import (
    conflict_detail,
    conflict_kind_label,
    conflict_warning,
    hunk_text,
)

_ACTION_LABELS = {
    FileAction.APPLY: "aplicar respaldo",
    FileAction.OMIT: "omitir (local)",
}


class AppSelectionScreen(Screen[list[str] | None]):
    """Choose which restorable apps to include."""

    BINDINGS = [Binding("escape", "cancel", "Cancelar")]

    def __init__(self, apps: Sequence[AppEntry]) -> None:
        super().__init__()
        self.apps = list(apps)

    def compose(self) -> ComposeResult:
        yield Header(show_clock=False)
        yield Static(
            "Selecciona las apps que quieres restaurar. Ninguna está marcada por defecto.",
            id="title",
        )
        if self.apps:
            selections = [
                Selection(f"{app.name} ({app.id})", app.id, initial_state=False)
                for app in self.apps
            ]
            yield SelectionList[str](*selections, id="apps")
        else:
            yield Static("Este respaldo no tiene apps restaurables en esta máquina.", id="empty")
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
        session: RestoreSession,
        plan: RestorePlan,
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
            backup = sum(1 for item in decision.hunks if item is HunkDecision.BACKUP)
            return f"revisión ({backup}/{len(decision.hunks)} respaldo)"
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
    """Review a text conflict block by block."""

    BINDINGS = [
        Binding("escape", "save", "Volver"),
        Binding("space", "toggle", "Alternar"),
        Binding("enter", "toggle", "Alternar"),
        Binding("a", "all_backup", "Todo respaldo"),
        Binding("l", "all_local", "Todo local"),
        Binding("question_mark", "help", "Ayuda"),
    ]

    def __init__(self, conflict: Conflict, decision: FileDecision | None) -> None:
        super().__init__()
        self.conflict = conflict
        self.decisions = self._initial_decisions(decision)

    def _initial_decisions(self, decision: FileDecision | None) -> list[HunkDecision]:
        if (
            decision is not None
            and decision.action is FileAction.REVIEW
            and len(decision.hunks) == len(self.conflict.hunks)
        ):
            return list(decision.hunks)
        return [HunkDecision.BACKUP for _hunk in self.conflict.hunks]

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
                yield Button("Todo respaldo (A)", id="all-backup")
                yield Button("Todo local (L)", id="all-local")
            with Horizontal(classes="action-row"):
                yield Button("Volver", id="back", variant="warning")
                yield Button("Ayuda", id="help")
        yield Footer()

    async def on_mount(self) -> None:
        await self._refresh_hunks()

    async def _refresh_hunks(self) -> None:
        list_view = self.query_one("#hunks", ListView)
        index = list_view.index if list_view.index is not None else 0
        await list_view.clear()
        items = [
            ListItem(Static(hunk_text(hunk, self.decisions[hunk.index - 1])))
            for hunk in self.conflict.hunks
        ]
        await list_view.extend(items)
        if items:
            list_view.index = min(index, len(items) - 1)

    async def on_button_pressed(self, event: Button.Pressed) -> None:
        button = event.button.id
        if button == "toggle":
            await self._toggle()
        elif button == "all-backup":
            await self._set_all(HunkDecision.BACKUP)
        elif button == "all-local":
            await self._set_all(HunkDecision.LOCAL)
        elif button == "back":
            self.action_save()
        elif button == "help":
            self.app.push_screen(HelpScreen())

    async def _toggle(self) -> None:
        list_view = self.query_one("#hunks", ListView)
        index = list_view.index
        if index is None or index < 0 or index >= len(self.decisions):
            return
        self.decisions[index] = (
            HunkDecision.LOCAL
            if self.decisions[index] is HunkDecision.BACKUP
            else HunkDecision.BACKUP
        )
        await self._refresh_hunks()

    async def _set_all(self, decision: HunkDecision) -> None:
        self.decisions = [decision for _item in self.decisions]
        await self._refresh_hunks()

    async def action_toggle(self) -> None:
        await self._toggle()

    async def action_all_backup(self) -> None:
        await self._set_all(HunkDecision.BACKUP)

    async def action_all_local(self) -> None:
        await self._set_all(HunkDecision.LOCAL)

    def action_save(self) -> None:
        if self.conflict.local_text is None or self.conflict.backup_text is None:
            self.dismiss(None)
            return
        merged = compose(self.conflict.local_text, self.conflict.backup_text, self.decisions)
        validation = validate_merge(
            self.conflict.path,
            self.conflict.local_text,
            self.conflict.backup_text,
            merged,
        )
        if validation.status == "invalid":
            self.query_one("#hunk-error", Static).update(
                f"Mezcla inválida: {validation.message}. Corrige los bloques o vuelve a intentarlo."
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
        session: RestoreSession,
        classification: RestoreClassification,
        decisions: ConflictDecisions,
    ) -> None:
        super().__init__()
        self.session = session
        self.classification = classification
        self.decisions = decisions

    def compose(self) -> ComposeResult:
        yield Header(show_clock=False)
        yield Static("Confirmación de la restauración", id="title")
        yield Static(
            f"Restaurados: {len(self.classification.restored)} · "
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
    """Final result after applying the restore."""

    BINDINGS = [Binding("escape", "exit", "Salir")]

    def __init__(self, result: RestoreResult) -> None:
        super().__init__()
        self.result = result

    def compose(self) -> ComposeResult:
        yield Header(show_clock=False)
        yield Static("Restauración completada", id="title")
        yield Static(
            f"Restaurados: {len(self.result.restored)} · "
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
        yield Static("Ayuda de la restauración interactiva", id="title")
        yield Static(
            "Apps: ninguna está marcada por defecto. Usa Seleccionar todas, "
            "Seleccionar ninguna, Continuar o Cancelar.\n\n"
            "Conflictos: Revisar abre el detalle por bloques; Aplicar restaura el "
            "archivo del respaldo; Omitir conserva el local. Aplicar todo y Omitir "
            "todo afectan a todos los conflictos y luego puedes cambiarlos uno a uno.\n\n"
            "Detalle: Espacio o Enter alterna el bloque resaltado. A toma todos los "
            "bloques del respaldo y L todos los locales. Escape vuelve y valida la mezcla.\n\n"
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
        yield Static("No se pudo completar la restauración", id="title")
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
