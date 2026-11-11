"""Flujo interactivo de despliegue (D0-D5)."""

from __future__ import annotations

import threading
from collections.abc import Sequence
from dataclasses import dataclass, field
from pathlib import Path
from typing import TYPE_CHECKING

from textual import work
from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import ScrollableContainer
from textual.screen import Screen
from textual.widgets import DataTable, Input, OptionList, ProgressBar, Static
from textual.widgets.option_list import Option

from kamino import texts
from kamino.archive import read_manifest
from kamino.catalog.models import AppDefinition
from kamino.config import AppConfig, ConfigLoadResult, load_config
from kamino.conflicts import (
    Conflict,
    ConflictDecisions,
    ConflictKind,
    FileAction,
    FileDecision,
    RowDecision,
    RowKind,
    apply_all,
    compose,
    omit_all,
    row_count,
    validate_merge,
)
from kamino.deploy import (
    DeployClassification,
    DeployPlan,
    DeployResult,
    DeploySession,
    DeployStatus,
    prepare_deploy,
    result_code,
)
from kamino.errors import (
    KaminoError,
    OperationCancelled,
    PartialApplyError,
)
from kamino.progress import ProgressEvent
from kamino.snapshot import snapshot_files
from kamino.tui.common import (
    Choice,
    ChoiceDialog,
    ErrorAction,
    ErrorScreen,
    ShortcutBar,
    WarningScreen,
)
from kamino.tui.widgets import (
    conflict_detail,
    conflict_kind_label,
    conflict_warning,
    format_size,
    hunk_header,
    render_block,
    review_label,
)

if TYPE_CHECKING:
    from kamino.tui.app import KaminoApp


@dataclass
class DeployState:
    """Opciones, sesión y decisiones compartidas por D0-D5."""

    home: Path | None = None
    catalog: dict[str, AppDefinition] | None = None
    catalog_dir: Path | None = None
    application_dirs: Sequence[Path] | None = None
    path: str | None = None
    snapshots_dir: Path | None = None
    clone_file: Path | None = None
    selected: list[str] | None = None
    requested: list[str] | None = None
    dry_run: bool = False
    verbose: bool = False
    include_file: bool = True
    include_apps: bool = True
    file_error: str | None = None
    apps_error: str | None = None
    session: DeploySession | None = None
    plan: DeployPlan | None = None
    decisions: ConflictDecisions = field(default_factory=ConflictDecisions)
    classification: DeployClassification | None = None
    last_error: str | None = None

    @property
    def steps(self) -> tuple[str, ...]:
        steps: list[str] = []
        if self.include_file:
            steps.append("clon")
        if self.include_apps:
            steps.append("apps")
        steps.extend(("conflictos", "confirmación"))
        return tuple(steps)


def _finish_flow(state: DeployState, app, mission: str, code: int) -> None:
    if state.session is not None:
        state.session.close()
        state.session = None
    app.finish_flow(mission, code)


def step_label(state: DeployState, step: str) -> str:
    steps = state.steps
    return texts.CLONE_STEP_LABEL.format(
        current=steps.index(step) + 1,
        total=len(steps),
        title=texts.STEP_TITLES.get(step, step),
    )


def _prepare_session(state: DeployState, clone_file: Path) -> DeploySession:
    return prepare_deploy(
        clone_file,
        home=state.home,
        catalog=state.catalog,
        catalog_dir=state.catalog_dir,
        application_dirs=state.application_dirs,
        path=state.path,
        snapshots_dir=state.snapshots_dir,
    )


def _apply_requested(state: DeployState) -> None:
    """Filtra la selección y el aviso de `--apps` tras preparar la sesión."""
    if state.requested is None or state.session is None:
        return
    deployable = {app.id for app in state.session.deployable_apps}
    state.selected = [app_id for app_id in state.requested if app_id in deployable]
    missing = [app_id for app_id in state.requested if app_id not in deployable]
    state.apps_error = (
        "aviso: app no desplegable o desconocida: " + ", ".join(missing) if missing else None
    )
    state.include_apps = state.selected is None or state.apps_error is not None


def _warnings_text(session: DeploySession) -> str:
    values = [*session.warnings, *session.notices]
    return "\n".join(values)


class DeployFileScreen(Screen[None]):
    """D0: elige un .tar.gz del directorio actual y valida su manifiesto."""

    BINDINGS = [
        Binding("enter", "continue", "Continuar", priority=True),
        Binding("escape", "back", "Atrás"),
        Binding("f1", "help", "Ayuda"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(self, state: DeployState, *, error: str | None = None) -> None:
        super().__init__()
        self.state = state
        self.error = error

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    def compose(self) -> ComposeResult:
        if self.state.include_file:
            yield Static(step_label(self.state, "clon"), id="step")
        yield Static(texts.DEPLOY_FILE_TITLE, id="title")
        yield DataTable(id="files")
        yield Input(placeholder=texts.LABEL_CLONE_FILE_PLACEHOLDER, id="path")
        yield Static("", id="warnings")
        yield Static("", id="status")
        yield ShortcutBar(texts.SHORTCUTS_D0)

    def on_mount(self) -> None:
        table = self.query_one(DataTable)
        table.cursor_type = "row"
        table.add_columns(texts.COLUMN_ARCHIVO, texts.COLUMN_TAMANO)
        for candidate in sorted(Path.cwd().glob("*.tar.gz")):
            table.add_row(candidate.name, format_size(candidate.stat().st_size), key=str(candidate))
        path_input = self.query_one("#path", Input)
        if self.state.clone_file is not None:
            path_input.value = str(self.state.clone_file)
        path_input.focus()
        if self.error or self.state.file_error:
            self.query_one("#status", Static).update(self.error or self.state.file_error)

    def on_data_table_row_highlighted(self, event: DataTable.RowHighlighted) -> None:
        if event.row_key is not None:
            self.query_one("#path", Input).value = str(event.row_key.value)

    def on_data_table_row_selected(self, event: DataTable.RowSelected) -> None:
        self.query_one("#path", Input).value = str(event.row_key.value)

    def on_input_changed(self, event: Input.Changed) -> None:
        self._validate()

    def _path(self) -> Path:
        return Path(self.query_one("#path", Input).value.strip()).expanduser()

    def _validate(self) -> Path | None:
        text = self.query_one("#path", Input).value.strip()
        status = self.query_one("#status", Static)
        if not text:
            status.update(texts.DEPLOY_NO_FILE)
            return None
        path = Path(text).expanduser().resolve()
        if not path.is_file():
            status.update(texts.DEPLOY_FILE_INVALID.format(message="no existe o no es un archivo"))
            return None
        status.update("")
        return path

    def action_continue(self) -> None:
        path = self._validate()
        if path is None:
            return
        try:
            manifest = read_manifest(path)
        except KaminoError as exc:
            self.query_one("#status", Static).update(texts.DEPLOY_FILE_INVALID.format(message=exc))
            return
        try:
            session = _prepare_session(self.state, path)
        except KaminoError as exc:
            self.query_one("#status", Static).update(texts.DEPLOY_FILE_INVALID.format(message=exc))
            return
        del manifest
        self.state.clone_file = path
        self.state.session = session
        self.kamino.cleanup = session.close
        _apply_requested(self.state)
        warnings = _warnings_text(session)
        self.query_one("#warnings", Static).update(warnings)
        self._advance()

    def _advance(self) -> None:
        if self.state.include_apps:
            self.app.push_screen(DeployAppsScreen(self.state))
        else:
            self.app.push_screen(DeployPrepareScreen(self.state))

    def action_back(self) -> None:
        _finish_flow(self.state, self.kamino, texts.TITLE_DEPLOY_CANCELLED, 1)

    def action_interrupt(self) -> None:
        self.action_back()

    def action_help(self) -> None:
        self.app.push_screen(
            self.kamino.flow_help(
                texts.SCREEN_D0,
                texts.HELP_D0,
                flow_steps=self.state.steps,
                current_step="clon",
                secrets=False,
            )
        )


class DeployAppsScreen(Screen[None]):
    """D1: apps instaladas con archivos y no instaladas informativas."""

    BINDINGS = [
        Binding("space", "toggle", "Marcar"),
        Binding("a", "all", "Todas"),
        Binding("n", "none", "Ninguna"),
        Binding("enter", "continue", "Continuar", priority=True),
        Binding("escape", "back", "Atrás"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(self, state: DeployState) -> None:
        super().__init__()
        self.state = state
        self.selected: set[str] = set(state.selected or [])
        self._preparation_cancel = threading.Event()
        self._preparing = False

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    def compose(self) -> ComposeResult:
        yield Static(step_label(self.state, "apps"), id="step")
        yield Static(texts.DEPLOY_APPS_TITLE, id="title")
        yield Static("", id="counts")
        yield DataTable(id="apps")
        yield Static("", id="not-installed")
        yield Static("", id="warnings")
        yield ProgressBar(total=1, id="progress")
        yield Static("", id="status")
        yield ShortcutBar(texts.SHORTCUTS_D1)

    def on_mount(self) -> None:
        session = self.state.session
        assert session is not None
        table = self.query_one(DataTable)
        table.cursor_type = "row"
        table.add_columns("", texts.COLUMN_APP, texts.COLUMN_ARCHIVOS)
        self._populate_table()
        self.query_one("#not-installed", Static).update(
            "\n".join(
                (
                    texts.DEPLOY_NOT_INSTALLED_TITLE,
                    texts.DEPLOY_NOT_INSTALLED_NOTE,
                    *[f"  {app.name} ({app.id})" for app in session.not_installed_apps],
                )
            )
            if session.not_installed_apps
            else ""
        )
        warnings = _warnings_text(session)
        self.query_one("#warnings", Static).update(warnings)
        self.query_one("#progress", ProgressBar).display = False
        table.focus()
        if self.state.apps_error:
            self.query_one("#status", Static).update(self.state.apps_error)
        self._refresh_controls()

    def on_data_table_row_selected(self, event: DataTable.RowSelected) -> None:
        self._toggle_row(str(event.row_key.value))

    def _populate_table(self) -> None:
        table = self.query_one(DataTable)
        session = self.state.session
        if session is None:
            return
        had_focus = table.has_focus
        cursor_row = table.cursor_row if table.row_count else 0
        table.clear()
        for app in session.deployable_apps:
            checked = "☑" if app.id in self.selected else "☐"
            table.add_row(
                checked,
                f"{app.name} ({app.id})",
                texts.CLONE_FILES.format(count=len(app.files)),
                key=app.id,
            )
        if table.row_count:
            table.move_cursor(row=min(cursor_row, table.row_count - 1))
        if had_focus:
            table.focus()
        self._refresh_controls()

    def _refresh_table(self) -> None:
        self._populate_table()

    def _refresh_controls(self) -> None:
        self.query_one("#counts", Static).update(
            texts.CLONE_APPS_SUMMARY.format(count=len(self.selected))
        )

    def _toggle_row(self, app_id: str) -> None:
        if self._preparing:
            return
        if app_id in self.selected:
            self.selected.discard(app_id)
        else:
            self.selected.add(app_id)
        self._refresh_table()

    def action_toggle(self) -> None:
        table = self.query_one(DataTable)
        if table.row_count == 0 or self._preparing:
            return
        row_key = table.coordinate_to_cell_key((table.cursor_row, 0)).row_key
        self._toggle_row(str(row_key.value))

    def action_all(self) -> None:
        if self.state.session is None:
            return
        self.selected = {app.id for app in self.state.session.deployable_apps}
        self._refresh_table()

    def action_none(self) -> None:
        self.selected.clear()
        self._refresh_table()

    def action_continue(self) -> None:
        if not self.selected or self._preparing:
            return
        assert self.state.session is not None
        self.state.selected = [
            app.id for app in self.state.session.deployable_apps if app.id in self.selected
        ]
        self._start_preparation()

    def _start_preparation(self) -> None:
        self._preparing = True
        self._preparation_cancel.clear()
        self.query_one("#progress", ProgressBar).display = True
        self.query_one("#status", Static).update(texts.DEPLOY_PREPARING)
        self._refresh_controls()
        self._prepare_worker()

    @work(thread=True)
    def _prepare_worker(self) -> None:
        assert self.state.session is not None
        try:
            plan = self.state.session.plan(
                self.state.selected,
                dry_run=self.state.dry_run,
                requested=self.state.requested,
                progress=lambda event: self.app.call_from_thread(self._progress, event),
                cancelled=self._preparation_cancel.is_set,
            )
        except OperationCancelled:
            self.app.call_from_thread(self._cancelled)
            return
        except KaminoError as exc:
            self.app.call_from_thread(self._failed, exc)
            return
        self.app.call_from_thread(self._planned, plan)

    def _progress(self, event: ProgressEvent) -> None:
        phase_label = {
            "verifying": texts.DEPLOY_VERIFYING,
            "extracting": texts.DEPLOY_EXTRACTING,
            "classifying": texts.DEPLOY_CLASSIFYING,
        }.get(event.phase.value, texts.DEPLOY_PREPARING)
        total = max(event.total, 1)
        self.query_one("#progress", ProgressBar).update(total=total, progress=event.current)
        self.query_one("#status", Static).update(phase_label)

    def _cancelled(self) -> None:
        self._preparing = False
        self.query_one("#progress", ProgressBar).display = False
        self.query_one("#status", Static).update(texts.STATUS_PREPARING_CANCELLED)
        self._refresh_controls()

    def _failed(self, exc: KaminoError) -> None:
        self._preparing = False
        self.query_one("#progress", ProgressBar).display = False
        self.query_one("#status", Static).update(str(exc))
        self.state.last_error = str(exc)
        self._refresh_controls()
        self.app.push_screen(
            ErrorScreen(
                str(exc),
                safety=texts.ERROR_NOTHING_WRITTEN,
                advice=texts.ADVICE_BACK_TO_D0,
                detail=repr(exc),
                show_menu=self.kamino.session.mode == "menu",
                actions=(
                    ErrorAction(
                        "d0",
                        "Volver a D0",
                        self._back_to_d0,
                        key="d",
                    ),
                ),
            )
        )

    def _planned(self, plan: DeployPlan) -> None:
        self._preparing = False
        self.query_one("#progress", ProgressBar).display = False
        self.state.plan = plan
        self.state.decisions = ConflictDecisions()
        self._advance_after_plan(plan)

    def _advance_after_plan(self, plan: DeployPlan) -> None:
        if plan.conflicts:
            self.app.push_screen(DeployConflictsScreen(self.state))
            return
        try:
            classification = self.state.session.preview(self.state.decisions)  # type: ignore[union-attr]
        except KaminoError as exc:
            self.query_one("#status", Static).update(str(exc))
            return
        self.state.classification = classification
        self.app.push_screen(DeployConfirmScreen(self.state))

    def _back_to_d0(self) -> None:
        self.app.pop_screen()
        while len(self.app.screen_stack) > 2 and not isinstance(self.app.screen, DeployFileScreen):
            self.app.pop_screen()
        if isinstance(self.app.screen, DeployFileScreen):
            screen = self.app.screen
            screen.query_one("#status", Static).update(self.state.last_error or "")
            return
        self.state.include_file = True
        self.app.push_screen(DeployFileScreen(self.state, error=self.state.last_error))

    def action_cancel(self) -> None:
        if self._preparing:
            self._preparation_cancel.set()
            self.query_one("#status", Static).update(texts.STATUS_CANCELLING)

    def action_back(self) -> None:
        if self._preparing:
            return
        if len(self.app.screen_stack) <= 2:
            _finish_flow(self.state, self.kamino, texts.TITLE_DEPLOY_CANCELLED, 1)
            return
        self.app.pop_screen()

    def action_interrupt(self) -> None:
        if self._preparing:
            self.action_cancel()
            return
        self.action_back()

    def action_help(self) -> None:
        self.app.push_screen(
            self.kamino.flow_help(
                texts.SCREEN_D1,
                texts.HELP_D1,
                flow_steps=self.state.steps,
                current_step="apps",
            )
        )


class DeployPrepareScreen(Screen[None]):
    """Preparación sin selección visible: verifica, extrae y clasifica."""

    BINDINGS = [
        Binding("escape", "cancel", "Cancelar"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(self, state: DeployState) -> None:
        super().__init__()
        self.state = state
        self._cancel = threading.Event()
        self._finished = False

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    def compose(self) -> ComposeResult:
        yield Static(texts.TITLE_PREPARING, id="title")
        yield ProgressBar(total=1, id="progress")
        yield Static("", id="status")
        yield ShortcutBar(texts.SHORTCUTS_PREPARE)

    def on_mount(self) -> None:
        self._worker()

    @work(thread=True)
    def _worker(self) -> None:
        assert self.state.session is not None
        try:
            plan = self.state.session.plan(
                self.state.selected,
                dry_run=self.state.dry_run,
                requested=self.state.requested,
                progress=lambda event: self.app.call_from_thread(self._progress, event),
                cancelled=self._cancel.is_set,
            )
        except OperationCancelled:
            self.app.call_from_thread(self._cancelled)
            return
        except KaminoError as exc:
            self.app.call_from_thread(self._failed, exc)
            return
        self.app.call_from_thread(self._planned, plan)

    def _progress(self, event: ProgressEvent) -> None:
        self.query_one("#progress", ProgressBar).update(
            total=max(event.total, 1), progress=event.current
        )
        self.query_one("#status", Static).update(event.path or texts.DEPLOY_PREPARING)

    def _cancelled(self) -> None:
        self._finished = True
        self.query_one("#status", Static).update(texts.STATUS_PREPARING_CANCELLED)
        _finish_flow(self.state, self.kamino, texts.TITLE_DEPLOY_CANCELLED, 1)

    def _failed(self, exc: KaminoError) -> None:
        self._finished = True
        self.app.push_screen(
            ErrorScreen(
                str(exc),
                safety=texts.ERROR_NOTHING_WRITTEN,
                advice=texts.ADVICE_BACK_TO_D0,
                detail=repr(exc),
                show_menu=self.kamino.session.mode == "menu",
            )
        )

    def _planned(self, plan: DeployPlan) -> None:
        self._finished = True
        self.state.plan = plan
        self.state.decisions = ConflictDecisions()
        if plan.conflicts:
            self.app.push_screen(DeployConflictsScreen(self.state))
            return
        self.state.classification = self.state.session.preview(self.state.decisions)  # type: ignore[union-attr]
        self.app.push_screen(DeployConfirmScreen(self.state))

    def action_cancel(self) -> None:
        self._cancel.set()
        self.query_one("#status", Static).update(texts.STATUS_CANCELLING)

    def action_interrupt(self) -> None:
        self.action_cancel()

    def action_help(self) -> None:
        self.app.push_screen(
            self.kamino.flow_help(
                texts.SCREEN_PREPARE,
                texts.HELP_PREPARE,
                flow_steps=self.state.steps,
                current_step="conflictos",
            )
        )


class DeployConflictsScreen(Screen[None]):
    """D2: tabla de conflictos, conteos y decisiones globales."""

    BINDINGS = [
        Binding("r", "review", "Revisar"),
        Binding("c", "apply", "Aplicar clon"),
        Binding("o", "omit", "Conservar original"),
        Binding("C", "apply_all", "Todo clon"),
        Binding("O", "omit_all", "Todo original"),
        Binding("enter", "confirm", "Confirmar", priority=True),
        Binding("escape", "back", "Retirada"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(self, state: DeployState) -> None:
        super().__init__()
        self.state = state
        self._row_paths: list[str] = []

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    def compose(self) -> ComposeResult:
        yield Static(step_label(self.state, "conflictos"), id="step")
        yield Static(texts.DEPLOY_CONFLICTS_TITLE, id="title")
        yield Static("", id="counts")
        yield DataTable(id="conflicts")
        yield Static("", id="status")
        yield ShortcutBar(texts.SHORTCUTS_D2)

    def on_mount(self) -> None:
        table = self.query_one(DataTable)
        table.cursor_type = "row"
        table.zebra_stripes = True
        table.add_columns(
            texts.COLUMN_APP,
            texts.COLUMN_ARCHIVO,
            texts.COLUMN_TIPO,
            texts.COLUMN_DETALLE,
            texts.COLUMN_DECISION,
        )
        self.refresh_table()
        table.focus()

    def refresh_table(self) -> None:
        assert self.state.plan is not None
        table = self.query_one(DataTable)
        selected = table.cursor_row if table.row_count else 0
        table.clear()
        self._row_paths = []
        for conflict in self.state.plan.conflicts:
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
        decision = self.state.decisions.get(conflict.path)
        if decision is None:
            return "sin decidir"
        if decision.action is FileAction.REVIEW:
            clone = sum(1 for item in decision.rows if item is RowDecision.CLONE)
            return review_label(clone, len(decision.rows))
        return {
            FileAction.APPLY: "aplicar clon",
            FileAction.OMIT: "conservar original",
        }[decision.action]

    def _refresh_counts(self) -> None:
        plan = self.state.plan
        assert plan is not None
        new = sum(1 for entry in plan.entries if entry.status is DeployStatus.NEW)
        identical = sum(1 for entry in plan.entries if entry.status is DeployStatus.IDENTICAL)
        unsupported = sum(1 for entry in plan.entries if entry.status is DeployStatus.UNSUPPORTED)
        notices = len(plan.notices)
        self.query_one("#counts", Static).update(
            " · ".join(
                (
                    texts.DEPLOY_CHANGED.format(count=len(plan.conflicts)),
                    texts.DEPLOY_NEW.format(count=new),
                    texts.DEPLOY_PERFECT.format(count=identical),
                    texts.DEPLOY_UNSUPPORTED.format(count=unsupported),
                    texts.DEPLOY_NOTICES.format(count=notices),
                )
            )
        )

    def _refresh_status(self) -> None:
        assert self.state.plan is not None
        missing = self.state.decisions.missing(self.state.plan.conflicts)
        status = self.query_one("#status", Static)
        if missing:
            names = ", ".join(conflict.path for conflict in missing[:3])
            extra = "" if len(missing) <= 3 else f" y {len(missing) - 3} más"
            status.update(f"Falta decidir: {names}{extra}")
        else:
            status.update(texts.STATUS_ALL_DECIDED)

    def _selected_conflict(self) -> Conflict | None:
        assert self.state.plan is not None
        if not self._row_paths:
            return None
        row = self.query_one(DataTable).cursor_row
        if row < 0 or row >= len(self._row_paths):
            return None
        path = self._row_paths[row]
        return next(
            (conflict for conflict in self.state.plan.conflicts if conflict.path == path),
            None,
        )

    def action_review(self) -> None:
        conflict = self._selected_conflict()
        if conflict is None:
            return
        if conflict.kind is not ConflictKind.TEXT:
            self.query_one("#status", Static).update(texts.STATUS_CONFLICT_WHOLE_FILE)
            return
        existing = self.state.decisions.get(conflict.path)
        self.app.push_screen(
            DeployDetailScreen(conflict, existing),
            lambda decision: self._reviewed(conflict, decision),
        )

    def _reviewed(self, conflict: Conflict, decision: FileDecision | None) -> None:
        if decision is not None:
            self.state.decisions.set(conflict.path, decision)
        self.refresh_table()

    def action_apply(self) -> None:
        conflict = self._selected_conflict()
        if conflict is not None:
            self.state.decisions.set(conflict.path, FileDecision(FileAction.APPLY))
            self.refresh_table()

    def action_omit(self) -> None:
        conflict = self._selected_conflict()
        if conflict is not None:
            self.state.decisions.set(conflict.path, FileDecision(FileAction.OMIT))
            self.refresh_table()

    def action_apply_all(self) -> None:
        assert self.state.plan is not None
        self.state.decisions.decisions.update(apply_all(self.state.plan.conflicts).decisions)
        self.refresh_table()

    def action_omit_all(self) -> None:
        assert self.state.plan is not None
        self.state.decisions.decisions.update(omit_all(self.state.plan.conflicts).decisions)
        self.refresh_table()

    def action_confirm(self) -> None:
        assert self.state.plan is not None
        if self.state.decisions.missing(self.state.plan.conflicts):
            return
        try:
            classification = self.state.session.preview(self.state.decisions)  # type: ignore[union-attr]
        except KaminoError as exc:
            self.query_one("#status", Static).update(str(exc))
            return
        self.state.classification = classification
        self.app.push_screen(DeployConfirmScreen(self.state))

    def action_back(self) -> None:
        self.app.push_screen(
            ChoiceDialog(
                texts.TITLE_BACK_TO_APPS,
                texts.TITLE_BACK_TO_APPS_BODY,
                (Choice("discard", texts.BUTTON_DISCARD, "error"),),
                default="cancel",
            ),
            self._discard_chosen,
        )

    def _discard_chosen(self, choice: str | None) -> None:
        if choice != "discard":
            return
        self.state.decisions = ConflictDecisions()
        self.state.plan = None
        if self.state.include_apps:
            while len(self.app.screen_stack) > 2 and not isinstance(
                self.app.screen, DeployAppsScreen
            ):
                self.app.pop_screen()
            return
        if self.state.include_file:
            while len(self.app.screen_stack) > 2 and not isinstance(
                self.app.screen, DeployFileScreen
            ):
                self.app.pop_screen()
            return
        _finish_flow(self.state, self.kamino, texts.TITLE_DEPLOY_CANCELLED, 1)

    def action_interrupt(self) -> None:
        _finish_flow(self.state, self.kamino, texts.TITLE_DEPLOY_CANCELLED, 1)

    def action_help(self) -> None:
        self.app.push_screen(
            self.kamino.flow_help(
                texts.SCREEN_D2,
                texts.HELP_D2,
                flow_steps=self.state.steps,
                current_step="conflictos",
                secrets=True,
            )
        )


class DeployDetailScreen(Screen[FileDecision | None]):
    """D3: un bloque a la vez, con marcadores y validación al volver."""

    BINDINGS = [
        Binding("up", "cursor_up", "Subir"),
        Binding("down", "cursor_down", "Bajar"),
        Binding("space", "toggle", "Alternar"),
        Binding("enter", "toggle", "Alternar"),
        Binding("c", "block_clone", "Bloque clon"),
        Binding("o", "block_original", "Bloque original"),
        Binding("C", "all_clone", "Todo clon"),
        Binding("O", "all_original", "Todo original"),
        Binding("tab", "next_block", "Siguiente bloque"),
        Binding("shift+tab", "prev_block", "Bloque anterior"),
        Binding("escape", "save", "Volver"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(self, conflict: Conflict, decision: FileDecision | None) -> None:
        super().__init__()
        self.conflict = conflict
        self.decisions = self._initial_decisions(decision)
        self.block = 0
        self.cursor = 0
        self._starts: list[int] = []
        self._counts: list[int] = []
        total = 0
        for hunk in conflict.hunks:
            self._starts.append(total)
            decidable = sum(1 for row in hunk.rows if row.kind is not RowKind.EQUAL)
            self._counts.append(decidable)
            total += decidable

    def _initial_decisions(self, decision: FileDecision | None) -> list[RowDecision]:
        if (
            decision is not None
            and decision.action is FileAction.REVIEW
            and len(decision.rows) == row_count(self.conflict)
        ):
            return list(decision.rows)
        return [RowDecision.CLONE for _ in range(row_count(self.conflict))]

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    def compose(self) -> ComposeResult:
        yield Static(texts.DEPLOY_DETAIL_TITLE, id="title")
        yield Static("", id="block-header")
        yield Static(conflict_warning(), id="warning")
        with ScrollableContainer(id="block-scroll"):
            yield Static("", id="block-body")
        yield Static("", id="block-footer")
        yield Static("", id="detail-error")
        yield Static(texts.DEPLOY_DECISION_HINT, id="hint")
        yield ShortcutBar(texts.SHORTCUTS_D3)

    def on_mount(self) -> None:
        self._render_block()

    def _current_hunk(self):
        return self.conflict.hunks[self.block]

    def _render_block(self) -> None:
        hunk = self._current_hunk()
        start = self._starts[self.block]
        count = self._counts[self.block]
        self.cursor = min(self.cursor, max(count - 1, 0))
        decisions = self.decisions[start : start + count]
        self.query_one("#block-header", Static).update(hunk_header(hunk, len(self.conflict.hunks)))
        self.query_one("#block-body", Static).update(
            render_block(
                hunk,
                decisions,
                cursor=self.cursor,
                unicode=self.kamino.unicode,
            )
        )
        clone_lines = len((self.conflict.clone_text or "").replace("\r\n", "\n").splitlines())
        original_lines = len((self.conflict.local_text or "").replace("\r\n", "\n").splitlines())
        self.query_one("#block-footer", Static).update(
            texts.DEPLOY_BLOCK_COUNTS.format(clone=clone_lines, original=original_lines)
        )
        self.query_one("#detail-error", Static).update("")

    def action_cursor_up(self) -> None:
        count = self._counts[self.block]
        if count:
            self.cursor = (self.cursor - 1) % count
            self._render_block()

    def action_cursor_down(self) -> None:
        count = self._counts[self.block]
        if count:
            self.cursor = (self.cursor + 1) % count
            self._render_block()

    def action_toggle(self) -> None:
        global_index = self._starts[self.block] + self.cursor
        self.decisions[global_index] = (
            RowDecision.LOCAL
            if self.decisions[global_index] is RowDecision.CLONE
            else RowDecision.CLONE
        )
        self._render_block()

    def _set_block(self, decision: RowDecision) -> None:
        start = self._starts[self.block]
        for index in range(start, start + self._counts[self.block]):
            self.decisions[index] = decision
        self._render_block()

    def _set_all(self, decision: RowDecision) -> None:
        self.decisions = [decision for _ in self.decisions]
        self._render_block()

    def action_block_clone(self) -> None:
        self._set_block(RowDecision.CLONE)

    def action_block_original(self) -> None:
        self._set_block(RowDecision.LOCAL)

    def action_all_clone(self) -> None:
        self._set_all(RowDecision.CLONE)

    def action_all_original(self) -> None:
        self._set_all(RowDecision.LOCAL)

    def action_next_block(self) -> None:
        if self.block + 1 < len(self.conflict.hunks):
            self.block += 1
            self.cursor = 0
            self._render_block()

    def action_prev_block(self) -> None:
        if self.block > 0:
            self.block -= 1
            self.cursor = 0
            self._render_block()

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
            self.query_one("#detail-error", Static).update(
                texts.DEPLOY_VALIDATION_INVALID.format(message=validation.message)
                + " Corrige las líneas o vuelve a intentarlo."
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

    def action_interrupt(self) -> None:
        self.dismiss(None)

    def action_help(self) -> None:
        self.app.push_screen(
            self.kamino.flow_help(
                texts.SCREEN_D3,
                texts.HELP_D3,
                flow_steps=("conflictos",),
                current_step="conflictos",
                secrets=True,
            )
        )


class DeployConfirmScreen(Screen[None]):
    """D4: resumen previo, protegidos y confirmación con foco en Retirada."""

    BINDINGS = [
        Binding("escape", "back", "Retirada"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(self, state: DeployState) -> None:
        super().__init__()
        self.state = state
        self._applying = False

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    def _classification(self) -> DeployClassification:
        assert self.state.classification is not None
        return self.state.classification

    def _new_count(self) -> int:
        assert self.state.plan is not None
        new_paths = {
            entry.path for entry in self.state.plan.entries if entry.status is DeployStatus.NEW
        }
        return sum(1 for path in self._classification().deployed if path in new_paths)

    def _write_destinations(self) -> list[Path]:
        assert self.state.plan is not None
        changed = set(self._classification().deployed) | set(self._classification().merged)
        return [entry.destination for entry in self.state.plan.entries if entry.path in changed]

    def compose(self) -> ComposeResult:
        yield Static(step_label(self.state, "confirmación"), id="step")
        yield Static(texts.DEPLOY_CONFIRM_TITLE, id="title")
        yield Static("", id="counts")
        yield Static("", id="protected")
        yield Static("", id="open-apps")
        yield ProgressBar(total=1, id="progress")
        yield Static("", id="status")
        yield OptionList(
            Option(texts.DEPLOY_ABORT, id="abort"),
            Option(texts.DEPLOY_BACK, id="back"),
            Option(texts.DEPLOY_APPLY_BUTTON, id="apply"),
            id="confirm-options",
            compact=True,
        )
        yield Static("", id="exit-hint")
        yield ShortcutBar(texts.SHORTCUTS_D4)

    def on_mount(self) -> None:
        classification = self._classification()
        assert self.state.plan is not None
        new = self._new_count()
        replaced = len(classification.deployed) - new
        self.query_one("#counts", Static).update(
            "\n".join(
                (
                    texts.DEPLOY_DEPLOYED.format(count=len(classification.deployed)),
                    texts.DEPLOY_DEPLOYED_DETAIL.format(new=new, replaced=replaced),
                    texts.DEPLOY_HYBRIDS.format(count=len(classification.merged)),
                    texts.DEPLOY_NOT_DEPLOYED.format(count=len(classification.omitted)),
                    texts.DEPLOY_UNCHANGED.format(count=len(classification.unchanged)),
                )
            )
        )
        protected = snapshot_files(self._write_destinations(), home=self.state.home)
        self.query_one("#protected", Static).update(
            "\n".join(
                (
                    texts.DEPLOY_PROTECTED.format(count=protected),
                    texts.DEPLOY_SNAPSHOT_DIR.format(
                        path=self.state.session.snapshot_directory  # type: ignore[union-attr]
                    ),
                )
            )
        )
        self.query_one("#open-apps", Static).update(texts.DEPLOY_OPEN_APPS)
        self.query_one("#progress", ProgressBar).display = False
        options = self.query_one("#confirm-options", OptionList)
        options.highlighted = 1
        options.focus()

    def on_option_list_option_selected(self, event: OptionList.OptionSelected) -> None:
        event.stop()
        actions = {
            "abort": self.action_abort,
            "back": self.action_back,
            "apply": self.action_apply,
        }
        action = actions.get(event.option.id or "")
        if action is not None:
            action()

    def action_back(self) -> None:
        if self._applying:
            return
        self.app.pop_screen()

    def action_abort(self) -> None:
        if self._applying:
            return
        _finish_flow(self.state, self.kamino, texts.TITLE_DEPLOY_ABORTED, 1)

    def action_interrupt(self) -> None:
        if self._applying:
            self.query_one("#status", Static).update(texts.DEPLOY_AFTER_SNAPSHOT)
            return
        self.action_abort()

    def action_apply(self) -> None:
        if self._applying:
            return
        self._applying = True
        self.query_one("#progress", ProgressBar).display = True
        self.query_one("#status", Static).update(texts.DEPLOY_APPLYING)
        self.query_one("#confirm-options", OptionList).disabled = True
        self._apply_worker()

    @work(thread=True)
    def _apply_worker(self) -> None:
        assert self.state.session is not None
        try:
            result = self.state.session.apply(
                self.state.decisions,
                progress=lambda event: self.app.call_from_thread(self._progress, event),
            )
        except PartialApplyError as exc:
            self.app.call_from_thread(self._partial, exc)
            return
        except KaminoError as exc:
            self.app.call_from_thread(self._failed, exc)
            return
        self.app.call_from_thread(self._done, result)

    def _progress(self, event: ProgressEvent) -> None:
        self.query_one("#progress", ProgressBar).update(
            total=max(event.total, 1), progress=event.current
        )
        self.query_one("#status", Static).update(f"{texts.DEPLOY_APPLYING} {event.path or ''}")

    def _partial(self, exc: PartialApplyError) -> None:
        self._applying = False
        self.app.push_screen(DeploySummaryScreen(self.state, partial=exc))

    def _failed(self, exc: KaminoError) -> None:
        self._applying = False
        self.query_one("#status", Static).update(str(exc))
        self.app.push_screen(
            ErrorScreen(
                str(exc),
                safety=texts.ERROR_NOTHING_WRITTEN,
                advice=texts.ADVICE_RETRY_CONFLICT,
                detail=repr(exc),
                show_menu=self.kamino.session.mode == "menu",
            )
        )

    def _done(self, result: DeployResult) -> None:
        self._applying = False
        self.app.push_screen(DeploySummaryScreen(self.state, result=result))

    def action_help(self) -> None:
        self.app.push_screen(
            self.kamino.flow_help(
                texts.SCREEN_D4,
                texts.HELP_D4,
                flow_steps=self.state.steps,
                current_step="confirmación",
                secrets=True,
            )
        )


class DeploySummaryScreen(Screen[None]):
    """D5: resultado por categorías, holocrón y despliegue incompleto."""

    BINDINGS = [
        Binding("c", "copy_path", "Copiar holocrón"),
        Binding("enter", "continue", "Continuar", priority=True),
        Binding("escape", "continue", "Continuar"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(
        self,
        state: DeployState,
        *,
        result: DeployResult | None = None,
        partial: PartialApplyError | None = None,
    ) -> None:
        super().__init__()
        self.state = state
        self.result = result
        self.partial = partial

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    @property
    def code(self) -> int:
        if self.partial is not None:
            return 1
        return result_code(self.result) if self.result is not None else 1

    def compose(self) -> ComposeResult:
        yield Static(
            f"{texts.ICON_ERROR} {texts.DEPLOY_INCOMPLETE_TITLE}"
            if self.partial is not None
            else f"{texts.ICON_SUCCESS} {texts.DEPLOY_SUMMARY_TITLE}",
            id="title",
        )
        yield Static("", id="counts")
        yield Static("", id="snapshot")
        yield Static("", id="undo")
        yield Static("", id="exit-line")
        yield Static("", id="status")
        yield ShortcutBar(texts.SHORTCUTS_D5)

    def on_mount(self) -> None:
        if self.partial is not None:
            self.query_one("#counts", Static).update(
                texts.DEPLOY_PARTIAL.format(
                    applied=len(self.partial.applied), failed=self.partial.failed
                )
            )
            self.query_one("#snapshot", Static).update(
                texts.DEPLOY_PARTIAL_SNAPSHOT.format(path=self.partial.snapshot)
                if self.partial.snapshot
                else ""
            )
            self.query_one("#undo", Static).display = False
            self.query_one("#exit-line", Static).update(f"Salida 1: {texts.STATUS_ERROR}")
            return
        result = self.result
        assert result is not None
        self.query_one("#counts", Static).update(
            "\n".join(
                (
                    texts.DEPLOY_DEPLOYED.format(count=len(result.deployed)),
                    texts.DEPLOY_HYBRIDS.format(count=len(result.merged)),
                    texts.DEPLOY_OMITTED.format(count=len(result.omitted)),
                    texts.DEPLOY_UNCHANGED.format(count=len(result.unchanged)),
                    texts.NOT_INSTALLED + f": {len(result.not_installed)}",
                    texts.DEPLOY_PROTECTED.format(count=result.snapshot_files),
                )
            )
        )
        self.query_one("#snapshot", Static).update(
            texts.DEPLOY_SNAPSHOT_PATH.format(path=result.snapshot)
            if result.snapshot is not None
            else texts.DEPLOY_NO_SNAPSHOT
        )
        if result.snapshot is not None:
            self.query_one("#undo", Static).update(texts.DEPLOY_UNDO.format(path=result.snapshot))
        else:
            self.query_one("#undo", Static).display = False
        if self.code != 0:
            self.query_one("#exit-line", Static).update(
                texts.DEPLOY_EXIT_LINE.format(code=self.code)
            )
        else:
            self.query_one("#exit-line", Static).display = False

    def _snapshot_path(self) -> Path | None:
        if self.partial is not None:
            return self.partial.snapshot
        if self.result is not None:
            return self.result.snapshot
        return None

    def action_copy_path(self) -> None:
        path = self._snapshot_path()
        if path is None:
            return
        supported = self.kamino.copy_text(str(path))
        self.query_one("#status", Static).update(
            texts.DEPLOY_COPY_PATH if supported else texts.CLONE_COPY_UNSUPPORTED
        )

    def action_continue(self) -> None:
        if self.partial is not None:
            _finish_flow(self.state, self.kamino, texts.DEPLOY_INCOMPLETE_TITLE, 1)
            return
        result = self.result
        summary = (
            f"Despliegue: {len(result.deployed)} desplegados, {len(result.merged)} híbridos"
            if result is not None
            else "Despliegue"
        )
        _finish_flow(self.state, self.kamino, summary, self.code)

    def action_interrupt(self) -> None:
        self.action_continue()

    def action_help(self) -> None:
        self.app.push_screen(
            self.kamino.flow_help(
                texts.SCREEN_D5,
                texts.HELP_D5,
                flow_steps=self.state.steps,
                current_step="confirmación",
                secrets=False,
            )
        )


def deploy_initial_screen(state: DeployState) -> Screen:
    if state.include_file:
        return DeployFileScreen(
            state,
            error=state.file_error,
        )
    if state.include_apps:
        return DeployAppsScreen(state)
    return DeployPrepareScreen(state)


def start_deploy_from_menu(app: KaminoApp) -> None:
    state = DeployState(
        catalog_dir=app.config.catalog_dir,
        snapshots_dir=app.config.snapshots_dir,
    )
    app.push_screen(DeployFileScreen(state))


def run_deploy_flow(
    *,
    clone_file: Path,
    home: Path | None = None,
    catalog: dict[str, AppDefinition] | None = None,
    selected: list[str] | None = None,
    requested: list[str] | None = None,
    dry_run: bool = False,
    verbose: bool = False,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
    snapshots_dir: Path | None = None,
    apps_error: str | None = None,
    file_error: str | None = None,
    session: DeploySession | None = None,
    config_result: ConfigLoadResult | None = None,
) -> int:
    """Ejecuta el flujo directo y devuelve su código."""
    from kamino.tui.app import run_app

    resolved = config_result if config_result is not None else load_config(home=home)

    def build_state(config: AppConfig, *, use_session: bool) -> DeployState:
        return DeployState(
            home=home,
            catalog=catalog,
            catalog_dir=config.catalog_dir,
            application_dirs=application_dirs,
            path=path,
            snapshots_dir=snapshots_dir if snapshots_dir is not None else config.snapshots_dir,
            clone_file=clone_file,
            selected=selected,
            requested=requested,
            dry_run=dry_run,
            verbose=verbose,
            include_file=file_error is not None,
            include_apps=selected is None or apps_error is not None,
            file_error=file_error,
            apps_error=apps_error,
            session=session if use_session else None,
        )

    state = build_state(resolved.config, use_session=resolved.error is None)
    if resolved.error is None and not state.include_file and state.session is None:
        try:
            state.session = _prepare_session(state, clone_file)
        except KaminoError as exc:
            state.include_file = True
            state.file_error = str(exc)
    _apply_requested(state)

    def after_config(app: KaminoApp) -> Screen:
        nonlocal state
        if state.session is not None:
            state.session.close()
            state.session = None
        state = build_state(app.config, use_session=False)
        if not state.include_file and state.session is None:
            try:
                state.session = _prepare_session(state, clone_file)
            except KaminoError as exc:
                state.include_file = True
                state.file_error = str(exc)
        _apply_requested(state)
        return deploy_initial_screen(state)

    try:
        return run_app(
            mode="direct",
            initial=deploy_initial_screen(state),
            config_result=resolved,
            config_home=home,
            after_config=after_config,
        )
    finally:
        if state.session is not None:
            state.session.close()
