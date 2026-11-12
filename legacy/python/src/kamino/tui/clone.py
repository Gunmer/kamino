"""Flujo interactivo de clonación (C1-C4)."""

from __future__ import annotations

import threading
from collections.abc import Sequence
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import TYPE_CHECKING

from textual import work
from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import VerticalScroll
from textual.screen import Screen
from textual.widgets import DataTable, Input, ProgressBar, Static

from kamino import texts
from kamino.archive import destination_state
from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.clone import (
    ClonePlan,
    CloneResult,
    default_name,
    free_suffix,
    plan_clone,
    validate_destination,
    write_clone,
)
from kamino.config import AppConfig, ConfigLoadResult, load_config
from kamino.defaults import catalog_dir as default_catalog_dir
from kamino.detect import is_installed
from kamino.errors import KaminoError, OperationCancelled
from kamino.plan import CountResult, count_files
from kamino.progress import ProgressEvent
from kamino.tui.common import (
    Choice,
    ChoiceDialog,
    ErrorAction,
    ErrorScreen,
    ShortcutBar,
)
from kamino.tui.widgets import format_size

if TYPE_CHECKING:
    from kamino.tui.app import KaminoApp


@dataclass
class CloneState:
    """Opciones y resultados compartidos por C1-C4."""

    home: Path | None = None
    catalog: dict[str, AppDefinition] | None = None
    catalog_dir: Path | None = None
    application_dirs: Sequence[Path] | None = None
    path: str | None = None
    selected: list[str] | None = None
    output: Path | None = None
    default_output_dir: Path | None = None
    force: bool = False
    dry_run: bool = False
    verbose: bool = False
    now: datetime | None = None
    include_apps: bool = True
    include_destination: bool = True
    apps_error: str | None = None
    output_error: str | None = None
    missing_requested: list[str] = field(default_factory=list)
    planned: ClonePlan | None = None
    result: CloneResult | None = None
    counts: dict[str, int] = field(default_factory=dict)
    not_detected: list[AppDefinition] = field(default_factory=list)

    @property
    def steps(self) -> tuple[str, ...]:
        steps: list[str] = []
        if self.include_apps:
            steps.append("apps")
        if self.include_destination:
            steps.append("destino")
        steps.append("revisión")
        return tuple(steps)


def _root(state: CloneState) -> Path:
    return state.home if state.home is not None else Path.home()


def _catalog(state: CloneState) -> dict[str, AppDefinition]:
    if state.catalog is not None:
        return state.catalog
    directory = (
        state.catalog_dir if state.catalog_dir is not None else default_catalog_dir(_root(state))
    )
    return load_catalog(user_dir=directory)


def _installed(state: CloneState) -> tuple[list[AppDefinition], list[AppDefinition]]:
    catalog = _catalog(state)
    installed: list[AppDefinition] = []
    missing: list[AppDefinition] = []
    for app_id in sorted(catalog):
        app = catalog[app_id]
        if is_installed(
            app,
            home=_root(state),
            application_dirs=state.application_dirs,
            path=state.path,
        ):
            installed.append(app)
        else:
            missing.append(app)
    return installed, missing


def step_label(state: CloneState, step: str) -> str:
    steps = state.steps
    return texts.CLONE_STEP_LABEL.format(
        current=steps.index(step) + 1,
        total=len(steps),
        title=texts.STEP_TITLES.get(step, step),
    )


class FilterInput(Input):
    """Campo de filtro que mantiene los atajos a/n sobre las filas visibles."""

    def on_key(self, event) -> None:
        if event.key == "a":
            self.screen.action_all()  # type: ignore[attr-defined]
            event.stop()
        elif event.key == "n":
            self.screen.action_none()  # type: ignore[attr-defined]
            event.stop()


class CloneAppsScreen(Screen[None]):
    """C1: apps instaladas, recuento en segundo plano y selección."""

    BINDINGS = [
        Binding("space", "toggle", "Marcar"),
        Binding("a", "all", "Todas", priority=True),
        Binding("n", "none", "Ninguna", priority=True),
        Binding("slash", "filter", "Filtro"),
        Binding("enter", "continue", "Continuar", priority=True),
        Binding("escape", "cancel", "Atrás"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(self, state: CloneState, *, error: str | None = None) -> None:
        super().__init__()
        self.state = state
        self.error = error
        self.installed: list[AppDefinition] = []
        self.selected: set[str] = set(state.selected or [])
        self._filter = ""

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    def compose(self) -> ComposeResult:
        yield Static(step_label(self.state, "apps"), id="step")
        yield Static(texts.CLONE_APPS_TITLE, id="title")
        yield Static("", id="counts")
        yield FilterInput(placeholder=texts.CLONE_FILTER, id="filter")
        yield DataTable(id="apps")
        yield Static("", id="status")
        yield Static("", id="not-detected")
        yield ShortcutBar(texts.SHORTCUTS_C1)

    def on_mount(self) -> None:
        self.query_one("#filter", Input).display = False
        table = self.query_one(DataTable)
        table.cursor_type = "row"
        table.add_columns("", texts.COLUMN_APP, texts.COLUMN_ARCHIVOS)
        table.focus()
        self.installed, self.state.not_detected = _installed(self.state)
        self._rebuild_table()
        self.query_one("#not-detected", Static).update(
            texts.CLONE_NOT_DETECTED.format(
                apps=", ".join(app.id for app in self.state.not_detected)
            )
            if self.state.not_detected
            else ""
        )
        if self.error:
            self.query_one("#status", Static).update(self.error)
        self._count_worker(self.installed)

    @work(thread=True)
    def _count_worker(self, apps: Sequence[AppDefinition]) -> None:
        result = count_files(apps, home=_root(self.state))
        self.app.call_from_thread(self._counts_ready, result)

    def _counts_ready(self, result: CountResult) -> None:
        self.state.counts = dict(result.counts)
        self.selected = {app_id for app_id in self.selected if self.state.counts.get(app_id, 0) > 0}
        self._rebuild_table()
        self._refresh_controls()

    def _visible(self) -> list[AppDefinition]:
        needle = self._filter.casefold()
        return [
            app
            for app in self.installed
            if not needle or needle in app.id.casefold() or needle in app.name.casefold()
        ]

    def _rebuild_table(self) -> None:
        table = self.query_one(DataTable)
        had_focus = table.has_focus
        cursor_row = table.cursor_row if table.row_count else 0
        table.clear()
        for app in self._visible():
            count = self.state.counts.get(app.id)
            checked = "☑" if app.id in self.selected else "☐"
            if count is None:
                files = "…"
            elif count == 0:
                files = texts.CLONE_ZERO_FILES
                checked = "—"
            else:
                files = texts.CLONE_FILES.format(count=count)
            table.add_row(checked, f"{app.name} ({app.id})", files, key=app.id)
        if table.row_count:
            table.move_cursor(row=min(cursor_row, table.row_count - 1))
        if had_focus:
            table.focus()
        self._refresh_controls()

    def _refresh_controls(self) -> None:
        count = len(self.selected)
        self.query_one("#counts", Static).update(texts.CLONE_APPS_SUMMARY.format(count=count))

    def on_data_table_row_selected(self, event: DataTable.RowSelected) -> None:
        self._toggle_row(str(event.row_key.value))

    def on_input_changed(self, event: Input.Changed) -> None:
        self._filter = event.value
        self._rebuild_table()

    def _toggle_row(self, app_id: str) -> None:
        count = self.state.counts.get(app_id)
        if count == 0:
            return
        if app_id in self.selected:
            self.selected.discard(app_id)
        else:
            self.selected.add(app_id)
        self._rebuild_table()

    def action_toggle(self) -> None:
        table = self.query_one(DataTable)
        if table.row_count == 0:
            return
        row_key = table.coordinate_to_cell_key((table.cursor_row, 0)).row_key
        self._toggle_row(str(row_key.value))

    def action_all(self) -> None:
        for app in self._visible():
            if self.state.counts.get(app.id, 0) > 0:
                self.selected.add(app.id)
        self._rebuild_table()

    def action_none(self) -> None:
        for app in self._visible():
            self.selected.discard(app.id)
        self._rebuild_table()

    def action_filter(self) -> None:
        filter_input = self.query_one("#filter", Input)
        filter_input.display = True
        filter_input.focus()

    def action_continue(self) -> None:
        if not self.selected:
            self.query_one("#status", Static).update(texts.CLONE_NO_SELECTION)
            return
        self.state.selected = [app.id for app in self.installed if app.id in self.selected]
        if self.state.include_destination:
            self.app.push_screen(CloneDestinationScreen(self.state))
        else:
            self.app.push_screen(CloneReviewScreen(self.state))

    def action_cancel(self) -> None:
        filter_input = self.query_one("#filter", Input)
        if filter_input.display:
            filter_input.value = ""
            self._filter = ""
            filter_input.display = False
            self._rebuild_table()
            self.query_one(DataTable).focus()
            return
        self.kamino.finish_flow(texts.TITLE_CANCEL_CLONE, 1)

    def action_interrupt(self) -> None:
        self.action_cancel()

    def action_help(self) -> None:
        self.app.push_screen(
            self.kamino.flow_help(
                texts.SCREEN_C1,
                texts.HELP_C1,
                flow_steps=self.state.steps,
                current_step="apps",
            )
        )


class CloneDestinationScreen(Screen[None]):
    """C2: destino con validación en vivo y diálogo de reemplazo."""

    BINDINGS = [
        Binding("enter", "continue", "Continuar", priority=True),
        Binding("f2", "cwd", "Directorio actual"),
        Binding("f3", "downloads", "Descargas"),
        Binding("f4", "desktop", "Escritorio"),
        Binding("escape", "back", "Atrás"),
        Binding("f1", "help", "Ayuda"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(self, state: CloneState) -> None:
        super().__init__()
        self.state = state

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    def compose(self) -> ComposeResult:
        yield Static(step_label(self.state, "destino"), id="step")
        yield Static(texts.CLONE_DESTINATION_TITLE, id="title")
        yield Static("", id="defaults")
        yield Input(placeholder=texts.LABEL_PATH_PLACEHOLDER, id="path")
        yield Static("", id="status")
        yield Static("", id="force-note")
        yield ShortcutBar(texts.SHORTCUTS_C2)

    def on_mount(self) -> None:
        state = self.state
        directory = state.default_output_dir if state.default_output_dir is not None else Path.cwd()
        default = state.output or (directory / default_name(state.now))
        path_input = self.query_one("#path", Input)
        path_input.value = str(default.expanduser())
        path_input.focus()
        self.query_one("#defaults", Static).update(
            texts.LABEL_DEFAULT_NAME.format(name=default_name(state.now))
        )
        if state.output_error:
            self.query_one("#status", Static).update(state.output_error)
        if state.force:
            self.query_one("#force-note", Static).update(
                texts.STATUS_WARN + " " + texts.CLONE_FORCE_NOTICE
            )
        self._validate()

    def _absolute(self) -> Path:
        raw = self.query_one("#path", Input).value.strip()
        return Path(raw).expanduser().resolve()

    def _validate(self) -> str | None:
        path = self._absolute()
        error = validate_destination(
            path,
            allow_missing_parent=self._is_default_proposal(path),
        )
        self.query_one("#status", Static).update(error or "")
        return error

    def _is_default_proposal(self, path: Path) -> bool:
        if self.state.output is not None or self.state.default_output_dir is None:
            return False
        expected = (self.state.default_output_dir / default_name(self.state.now)).resolve()
        return path == expected

    def on_input_changed(self, event: Input.Changed) -> None:
        self._validate()

    def _set_directory(self, directory: Path) -> None:
        self.query_one("#path", Input).value = str(directory / default_name(self.state.now))

    def action_cwd(self) -> None:
        self._set_directory(Path.cwd())

    def action_downloads(self) -> None:
        self._set_directory(Path.home() / "Downloads")

    def action_desktop(self) -> None:
        self._set_directory(Path.home() / "Desktop")

    def action_continue(self) -> None:
        error = self._validate()
        if error is not None:
            return
        destination = self._absolute()
        self.state.output = destination
        if destination.exists() and not self.state.force:
            stat = destination.stat()
            created = datetime.fromtimestamp(stat.st_mtime).strftime("%Y-%m-%d %H:%M")
            body = texts.CLONE_REPLACE_BODY.format(
                path=destination,
                created=created,
                size=texts.size_label(stat.st_size),
            )
            self.app.push_screen(
                ReemplazarDialog(
                    texts.CLONE_REPLACE_TITLE,
                    body,
                    destination,
                ),
                self._replacement_chosen,
            )
            return
        self.app.push_screen(CloneReviewScreen(self.state))

    def _replacement_chosen(self, choice: str | None) -> None:
        if choice is None or choice == "back":
            return
        assert self.state.output is not None
        if choice == "other":
            other = free_suffix(self.state.output)
            self.query_one("#path", Input).value = str(other)
            self._validate()
            return
        if choice == "replace":
            self.state.force = True
            self.app.push_screen(CloneReviewScreen(self.state))

    def action_back(self) -> None:
        if len(self.app.screen_stack) <= 2:
            self.kamino.finish_flow(texts.TITLE_CANCEL_CLONE, 1)
            return
        self.app.pop_screen()

    def action_interrupt(self) -> None:
        self.action_back()

    def action_help(self) -> None:
        self.app.push_screen(
            self.kamino.flow_help(
                texts.SCREEN_C2,
                texts.HELP_C2,
                flow_steps=self.state.steps,
                current_step="destino",
            )
        )


class ReemplazarDialog(ChoiceDialog):
    """Diálogo de clon existente con foco inicial en Volver."""

    def __init__(self, title: str, body: str, destination: Path) -> None:
        self.destination = destination
        super().__init__(
            title,
            body,
            (
                Choice("other", texts.CLONE_REPLACE_OTHER),
                Choice("replace", texts.CLONE_REPLACE_FORCE, "error"),
            ),
            default="cancel",
        )


class CloneReviewScreen(Screen[None]):
    """C3: planificando, revisión y escritura con progreso y cancelación."""

    BINDINGS = [
        Binding("d", "toggle_skipped", "Omitidos"),
        Binding("f", "toggle_files", "Archivos"),
        Binding("enter", "primary", "Continuar", priority=True),
        Binding("escape", "back", "Volver"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(self, state: CloneState, *, error: str | None = None) -> None:
        super().__init__()
        self.state = state
        self.error = error
        self._cancel_event = threading.Event()
        self._plan_cancel = threading.Event()
        self._writing = False
        self._planning = False
        self._skipped_visible = False
        self._files_visible = False
        self._cancel_requested = False
        self._destination_state: tuple[int, int, int] | None = None

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    def compose(self) -> ComposeResult:
        yield Static(step_label(self.state, "revisión"), id="step")
        yield Static(texts.CLONE_REVIEW_TITLE, id="title")
        yield Static("", id="planning")
        with VerticalScroll(id="review-body"):
            yield Static("", id="summary")
            yield Static(texts.CLONE_SECRETS, id="secrets")
            yield Static("", id="skipped")
            yield Static("", id="files")
        yield ProgressBar(total=1, id="progress")
        yield Static("", id="status")
        yield Static("", id="exit-hint")
        yield ShortcutBar(texts.SHORTCUTS_C3)

    def on_mount(self) -> None:
        self.query_one("#skipped", Static).display = False
        self.query_one("#files", Static).display = False
        self.query_one("#progress", ProgressBar).display = False
        if self.state.dry_run:
            self.query_one("#exit-hint", Static).update(texts.CLONE_DRY_RUN_NOTE)
            self.query_one("#shortcut-bar", Static).update(texts.SHORTCUTS_C3_DRY_RUN)
        if self.error:
            self.query_one("#status", Static).update(self.error)
            return
        self._planning = True
        self.query_one("#planning", Static).update(texts.CLONE_PLANNING)
        self._plan_worker()

    @work(thread=True)
    def _plan_worker(self) -> None:
        try:
            planned = plan_clone(
                home=self.state.home,
                selected=self.state.selected,
                output=self.state.output,
                catalog=self.state.catalog,
                catalog_dir=self.state.catalog_dir,
                application_dirs=self.state.application_dirs,
                path=self.state.path,
                now=self.state.now,
                progress=lambda event: self.app.call_from_thread(self._planning_progress, event),
                cancelled=self._plan_cancel.is_set,
            )
        except OperationCancelled:
            return
        except KaminoError as exc:
            self.app.call_from_thread(self._plan_failed, exc)
            return
        if self._plan_cancel.is_set():
            return
        self.app.call_from_thread(self._planned, planned)

    def _planning_progress(self, event: ProgressEvent) -> None:
        if self._planning:
            self.query_one("#planning", Static).update(f"{texts.CLONE_PLANNING} {event.path or ''}")

    def _planned(self, planned: ClonePlan) -> None:
        self._planning = False
        self.state.planned = planned
        self.query_one("#planning", Static).update("")
        self.query_one("#summary", Static).update(self._summary(planned))
        skipped = "\n".join(f"  {item.path} ({item.reason})" for item in planned.plan.skipped)
        self.query_one("#skipped", Static).update(
            f"{texts.CLONE_SKIPPED_TITLE}\n{skipped}" if skipped else ""
        )
        grouped: dict[str, list[str]] = {}
        for operation in planned.plan.operations:
            grouped.setdefault(operation.app_id, []).append(operation.path)
        lines = [
            f"{app_id} ({len(paths)} archivos)" + "".join(f"\n  {path}" for path in paths)
            for app_id, paths in grouped.items()
        ]
        self.query_one("#files", Static).update(
            f"{texts.CLONE_INCLUDED_TITLE}\n" + "\n".join(lines)
        )
        self.query_one("#progress", ProgressBar).update(
            total=max(planned.file_count, 1), progress=0
        )

    def _plan_failed(self, exc: Exception) -> None:
        self._planning = False
        self.query_one("#planning", Static).update("")
        self.query_one("#status", Static).update(str(exc))

    def _summary(self, planned: ClonePlan) -> str:
        return "\n".join(
            (
                texts.CLONE_APPS_SUMMARY.format(count=len(planned.manifest.apps)),
                texts.CLONE_FILE_COUNT.format(count=planned.file_count),
                texts.CLONE_SIZE.format(size=format_size(planned.size)),
                texts.CLONE_DESTINATION.format(path=planned.output),
            )
        )

    def action_primary(self) -> None:
        if self.state.dry_run:
            self.action_back()
            return
        if self._writing or self._planning or self.state.planned is None:
            return
        self.action_start()

    def action_start(self) -> None:
        if self._writing or self.state.planned is None:
            return
        if not self.state.planned.plan.operations:
            self.query_one("#status", Static).update(texts.CLONE_NOTHING_TO_DO)
            return
        self._writing = True
        self._cancel_requested = False
        self._cancel_event.clear()
        self._destination_state = destination_state(self.state.planned.output)
        self.query_one("#progress", ProgressBar).display = True
        self.query_one("#status", Static).update(texts.CLONE_WRITING)
        self._write_worker()

    @work(thread=True)
    def _write_worker(self) -> None:
        assert self.state.planned is not None
        try:
            result = write_clone(
                self.state.planned,
                home=self.state.home,
                force=self.state.force,
                expected_state=self._destination_state,
                progress=lambda event: self.app.call_from_thread(self._progress, event),
                cancelled=self._cancel_event.is_set,
            )
        except OperationCancelled:
            self.app.call_from_thread(self._cancelled)
            return
        except KaminoError as exc:
            self.app.call_from_thread(self._failed, exc)
            return
        self.app.call_from_thread(self._done, result)

    def _progress(self, event: ProgressEvent) -> None:
        self.query_one("#progress", ProgressBar).update(
            total=max(event.total, 1), progress=event.current
        )
        self.query_one("#status", Static).update(f"{texts.CLONE_WRITING} {event.path or ''}")

    def _cancelled(self) -> None:
        self._writing = False
        self._cancel_requested = False
        self.query_one("#status", Static).update(texts.CLONE_CANCELLED)
        self.query_one("#progress", ProgressBar).display = False

    def _failed(self, exc: KaminoError) -> None:
        self._writing = False
        self.query_one("#status", Static).update(str(exc))
        self.query_one("#progress", ProgressBar).display = False
        self.app.push_screen(
            ErrorScreen(
                str(exc),
                detail=repr(exc),
                show_menu=self.kamino.session.mode == "menu",
                actions=(
                    ErrorAction(
                        "retry",
                        texts.BUTTON_RETRY,
                        self._retry,
                        key="r",
                    ),
                ),
            )
        )

    def _retry(self) -> None:
        self.app.pop_screen()
        self.query_one("#status", Static).update("")

    def _done(self, result: CloneResult) -> None:
        self._writing = False
        self.state.result = result
        if self._cancel_requested:
            self.query_one("#status", Static).update(texts.STATUS_RENAME_STARTED)
        self.app.push_screen(CloneSummaryScreen(self.state))

    def action_cancel(self) -> None:
        if not self._writing:
            return
        self._cancel_requested = True
        self._cancel_event.set()
        self.query_one("#status", Static).update(texts.CLONE_CANCELLING)

    def action_back(self) -> None:
        if self._writing:
            return
        if self._planning:
            self._planning = False
            self._plan_cancel.set()
        if len(self.app.screen_stack) <= 2:
            if self.state.dry_run:
                code = 3 if (self.state.planned and self.state.planned.plan.skipped) else 0
                self.kamino.finish_flow(texts.TITLE_SIMULATION, code)
            else:
                self.kamino.finish_flow(texts.TITLE_CANCEL_CLONE, 1)
            return
        if self._planning:
            self._planning = False
            self._plan_cancel.set()
        self.app.pop_screen()

    def action_interrupt(self) -> None:
        if self._writing:
            self.action_cancel()
            return
        self.action_back()

    def action_toggle_skipped(self) -> None:
        self._skipped_visible = not self._skipped_visible
        self.query_one("#skipped", Static).display = self._skipped_visible

    def action_toggle_files(self) -> None:
        self._files_visible = not self._files_visible
        self.query_one("#files", Static).display = self._files_visible

    def action_help(self) -> None:
        self.app.push_screen(
            self.kamino.flow_help(
                texts.SCREEN_C3,
                texts.HELP_C3,
                flow_steps=self.state.steps,
                current_step="revisión",
            )
        )


class CloneSummaryScreen(Screen[None]):
    """C4: resumen, omitidos, línea de salida y copia de ruta."""

    BINDINGS = [
        Binding("d", "toggle_skipped", "Omitidos"),
        Binding("c", "copy_path", "Copiar ruta"),
        Binding("enter", "continue", "Continuar", priority=True),
        Binding("escape", "continue", "Continuar"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(self, state: CloneState) -> None:
        super().__init__()
        self.state = state
        self._skipped_visible = False

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    @property
    def code(self) -> int:
        result = self.state.result
        skipped = result.plan.skipped if result is not None else []
        return 3 if (skipped or self.state.missing_requested) else 0

    def compose(self) -> ComposeResult:
        yield Static(f"{texts.ICON_SUCCESS} {texts.CLONE_SUMMARY_TITLE}", id="title")
        yield Static("", id="summary")
        yield Static("", id="skipped")
        yield Static("", id="exit-line")
        yield Static("", id="status")
        yield ShortcutBar(texts.SHORTCUTS_C4)

    def on_mount(self) -> None:
        result = self.state.result
        assert result is not None
        self.query_one("#skipped", Static).display = False
        self.query_one("#summary", Static).update(
            "\n".join(
                (
                    texts.CLONE_APPS_SUMMARY.format(count=len(result.manifest.apps)),
                    texts.CLONE_FILE_COUNT.format(count=len(result.plan.operations)),
                    texts.CLONE_SIZE.format(
                        size=format_size(sum(op.size for op in result.plan.operations))
                    ),
                    texts.CLONE_DESTINATION.format(path=result.output),
                )
            )
        )
        if self.code == 3:
            self.query_one("#exit-line", Static).update(texts.CLONE_EXIT_LINE)
        else:
            self.query_one("#exit-line", Static).display = False

    def _skipped_text(self) -> str:
        result = self.state.result
        if result is None:
            return ""
        return "\n".join(f"  {item.path} ({item.reason})" for item in result.plan.skipped)

    def action_toggle_skipped(self) -> None:
        self._skipped_visible = not self._skipped_visible
        skipped = self.query_one("#skipped", Static)
        skipped.display = self._skipped_visible
        skipped.update(self._skipped_text())

    def action_copy_path(self) -> None:
        result = self.state.result
        if result is None or result.output is None:
            return
        supported = self.kamino.copy_text(str(result.output))
        self.query_one("#status", Static).update(
            texts.CLONE_COPY_PATH if supported else texts.CLONE_COPY_UNSUPPORTED
        )

    def action_continue(self) -> None:
        result = self.state.result
        path = str(result.output) if result is not None else ""
        self.kamino.finish_flow(texts.TITLE_CLONE_CREATED.format(path=path), self.code)

    def action_interrupt(self) -> None:
        self.action_continue()

    def action_help(self) -> None:
        self.app.push_screen(
            self.kamino.flow_help(
                texts.SCREEN_C4,
                texts.HELP_C4,
                flow_steps=self.state.steps,
                current_step="revisión",
            )
        )


def _clone_state(
    *,
    home: Path | None,
    catalog: dict[str, AppDefinition] | None,
    catalog_dir: Path | None,
    selected: list[str] | None,
    output: Path | None,
    default_output_dir: Path | None,
    force: bool,
    dry_run: bool,
    verbose: bool,
    application_dirs: Sequence[Path] | None,
    path: str | None,
    now: datetime | None,
    include_apps: bool,
    include_destination: bool,
    apps_error: str | None,
    output_error: str | None,
    missing_requested: list[str],
) -> CloneState:
    return CloneState(
        home=home,
        catalog=catalog,
        catalog_dir=catalog_dir,
        application_dirs=application_dirs,
        path=path,
        selected=selected,
        output=output,
        default_output_dir=default_output_dir,
        force=force,
        dry_run=dry_run,
        verbose=verbose,
        now=now,
        include_apps=include_apps,
        include_destination=include_destination,
        apps_error=apps_error,
        output_error=output_error,
        missing_requested=missing_requested,
    )


def start_clone_from_menu(app: KaminoApp) -> None:
    state = _clone_state(
        home=None,
        catalog=None,
        catalog_dir=app.config.catalog_dir,
        selected=None,
        output=None,
        default_output_dir=app.config.clone_output_dir,
        force=False,
        dry_run=False,
        verbose=False,
        application_dirs=None,
        path=None,
        now=None,
        include_apps=True,
        include_destination=True,
        apps_error=None,
        output_error=None,
        missing_requested=[],
    )
    app.push_screen(CloneAppsScreen(state))


def clone_initial_screen(state: CloneState) -> Screen:
    """Elige la primera pantalla visible según los argumentos válidos."""
    if not state.include_apps and not state.include_destination:
        return CloneReviewScreen(state)
    if state.apps_error is not None:
        return CloneAppsScreen(state, error=state.apps_error)
    if state.output_error is not None:
        return CloneDestinationScreen(state)
    if state.include_apps:
        return CloneAppsScreen(state)
    return CloneDestinationScreen(state)


def run_clone_flow(
    *,
    home: Path | None = None,
    catalog: dict[str, AppDefinition] | None = None,
    selected: list[str] | None = None,
    requested: list[str] | None = None,
    output: Path | None = None,
    force: bool = False,
    dry_run: bool = False,
    verbose: bool = False,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
    now: datetime | None = None,
    apps_error: str | None = None,
    output_error: str | None = None,
    missing_requested: list[str] | None = None,
    config_result: ConfigLoadResult | None = None,
) -> int:
    """Ejecuta el flujo en modo directo y devuelve su código."""
    from kamino.tui.app import run_app

    resolved = config_result if config_result is not None else load_config(home=home)
    include_destination = output is None or output_error is not None

    def selection(config: AppConfig) -> tuple[list[str] | None, str | None, list[str]]:
        if requested is None:
            return selected, apps_error, list(missing_requested or [])
        definitions = catalog if catalog is not None else load_catalog(user_dir=config.catalog_dir)
        installed = {
            app.id
            for app in definitions.values()
            if is_installed(
                app,
                home=home,
                application_dirs=application_dirs,
                path=path,
            )
        }
        chosen = [app_id for app_id in requested if app_id in installed]
        missing = [app_id for app_id in requested if app_id not in installed]
        error = "aviso: app no instalada o desconocida: " + ", ".join(missing) if missing else None
        return chosen, error, missing

    def build_state(config: AppConfig) -> CloneState:
        chosen, error, missing = selection(config)
        include_apps = (chosen is None and output_error is None) or error is not None
        return _clone_state(
            home=home,
            catalog=catalog,
            catalog_dir=config.catalog_dir,
            selected=chosen,
            output=output,
            default_output_dir=config.clone_output_dir,
            force=force,
            dry_run=dry_run,
            verbose=verbose,
            application_dirs=application_dirs,
            path=path,
            now=now,
            include_apps=include_apps,
            include_destination=include_destination,
            apps_error=error,
            output_error=output_error,
            missing_requested=missing,
        )

    def after_config(app: KaminoApp) -> Screen:
        return clone_initial_screen(build_state(app.config))

    state = build_state(resolved.config)
    return run_app(
        mode="direct",
        initial=clone_initial_screen(state),
        config_result=resolved,
        config_home=home,
        after_config=after_config,
    )
