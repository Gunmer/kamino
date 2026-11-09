"""Flujo interactivo de actualización (U1)."""

from __future__ import annotations

import threading
from pathlib import Path
from typing import TYPE_CHECKING

from textual import work
from textual.app import ComposeResult
from textual.binding import Binding
from textual.screen import Screen
from textual.widgets import ProgressBar, Static

from kamino import __version__, texts
from kamino.errors import (
    KaminoError,
    OperationCancelled,
    UpdateError,
    UpdateErrorKind,
)
from kamino.progress import ProgressEvent
from kamino.tui.common import ErrorAction, ErrorScreen, ShortcutBar
from kamino.update import (
    ReleaseInfo,
    UpdateCheck,
    UpdateOutcome,
    UpdateResult,
    download_update,
    query_update,
    replace_binary,
)

if TYPE_CHECKING:
    from kamino.tui.app import KaminoApp


class UpdateScreen(Screen[None]):
    """U1: consulta, descarga/verificación, reemplazo y resultado."""

    BINDINGS = [
        Binding("enter", "continue", "Continuar", priority=True),
        Binding("escape", "escape", "Salir"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(
        self,
        *,
        current_version: str | None = None,
        target: Path | None = None,
        release: ReleaseInfo | None = None,
    ) -> None:
        super().__init__()
        self.current_version = current_version or __version__
        self.target = target
        self.release = release
        self._cancel = threading.Event()
        self._finished = False
        self._replacing = False
        self._result: UpdateResult | None = None
        self._cancelled = False

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    def compose(self) -> ComposeResult:
        yield Static(texts.UPDATE_TITLE, id="title")
        yield Static(texts.UPDATE_CHECKING, id="state")
        yield ProgressBar(total=1, id="progress")
        yield Static("", id="result")
        yield Static("", id="status")
        yield ShortcutBar(texts.SHORTCUTS_U1)

    def on_mount(self) -> None:
        self.query_one("#progress", ProgressBar).display = False
        self._query_worker()

    @work(thread=True)
    def _query_worker(self) -> None:
        try:
            check = query_update(self.current_version, target=self.target, release=self.release)
        except UpdateError as exc:
            self.app.call_from_thread(self._failed, exc)
            return
        self.app.call_from_thread(self._checked, check)

    def _checked(self, check: UpdateCheck) -> None:
        if self._cancel.is_set():
            self._cancelled_state()
            return
        if not check.should_update:
            self._finished = True
            self._result = UpdateResult(
                check.current_version,
                check.release.version,
                check.target,
                updated=False,
                outcome=check.outcome,
            )
            self._show_result(self._result)
            return
        self.query_one("#state", Static).update(texts.UPDATE_DOWNLOADING)
        self.query_one("#progress", ProgressBar).display = True
        self.query_one("#progress", ProgressBar).update(
            total=max(check.release.asset_size, 1), progress=0
        )
        self._download_worker(check)

    @work(thread=True)
    def _download_worker(self, check: UpdateCheck) -> None:
        temp: Path | None = None
        try:
            temp = download_update(
                check.release,
                check.target,
                progress=lambda event: self.app.call_from_thread(self._progress, event),
                cancelled=self._cancel.is_set,
            )
            if self._cancel.is_set():
                raise OperationCancelled("operación cancelada")
            self.app.call_from_thread(self._replacing_state)
            replace_binary(temp, check.target)
            temp = None
        except OperationCancelled:
            self.app.call_from_thread(self._cancelled_state)
            return
        except KaminoError as exc:
            self.app.call_from_thread(self._failed, exc)
            return
        finally:
            if temp is not None:
                temp.unlink(missing_ok=True)
        result = UpdateResult(
            check.current_version,
            check.release.version,
            check.target,
            updated=True,
            outcome=UpdateOutcome.UPDATED,
        )
        self.app.call_from_thread(self._show_result, result)

    def _progress(self, event: ProgressEvent) -> None:
        self.query_one("#progress", ProgressBar).update(
            total=max(event.total, 1), progress=event.current
        )
        self.query_one("#state", Static).update(texts.UPDATE_DOWNLOADING)

    def _replacing_state(self) -> None:
        self._replacing = True
        self.query_one("#state", Static).update(texts.UPDATE_REPLACING)

    def _cancelled_state(self) -> None:
        self._finished = True
        self._cancelled = True
        self.query_one("#progress", ProgressBar).display = False
        self.query_one("#state", Static).update(texts.UPDATE_CANCELLED)

    def _failed(self, exc: KaminoError) -> None:
        self._finished = True
        self.query_one("#progress", ProgressBar).display = False
        advice = texts.UPDATE_ERROR_ADVICE.get(
            getattr(exc, "kind", None),
            texts.UPDATE_ERROR_ADVICE[UpdateErrorKind.OTHER],
        )
        self.app.push_screen(
            ErrorScreen(
                str(exc),
                safety="El binario actual sigue intacto; el temporal se eliminó.",
                advice=advice,
                detail=repr(exc),
                show_menu=self.kamino.session.mode == "menu",
                actions=(
                    ErrorAction(
                        "retry",
                        texts.ERROR_RETRY,
                        self._retry,
                        key="r",
                    ),
                ),
            )
        )

    def _retry(self) -> None:
        self.app.pop_screen()
        self._finished = False
        self._cancelled = False
        self._replacing = False
        self.query_one("#result", Static).update("")
        self.query_one("#state", Static).update(texts.UPDATE_CHECKING)
        self._query_worker()

    def _show_result(self, result: UpdateResult) -> None:
        self._finished = True
        self._result = result
        self.query_one("#progress", ProgressBar).display = False
        if result.outcome is UpdateOutcome.UPDATED:
            state = texts.UPDATE_UPDATED.format(
                current=result.current_version,
                latest=result.latest_version,
                path=result.target,
            )
            body = "\n".join((state, texts.UPDATE_RESTART))
        elif result.outcome is UpdateOutcome.NEWER:
            body = texts.UPDATE_NEWER.format(
                current=result.current_version, latest=result.latest_version
            )
        else:
            body = texts.UPDATE_UP_TO_DATE.format(current=result.current_version)
        label = {
            UpdateOutcome.UPDATED: texts.UPDATE_RESULT_UPDATED,
            UpdateOutcome.NEWER: texts.UPDATE_RESULT_NEWER,
            UpdateOutcome.UP_TO_DATE: texts.UPDATE_RESULT_UP_TO_DATE,
        }[result.outcome]
        self.query_one("#state", Static).update(label)
        self.query_one("#result", Static).update(body)

    def action_continue(self) -> None:
        if not self._finished and not self._cancelled:
            return
        if self._cancelled:
            self.kamino.finish_flow(texts.UPDATE_CANCELLED, 1)
            return
        result = self._result
        if result is None:
            self.kamino.finish_flow("Actualización cancelada", 1)
            return
        if result.outcome is UpdateOutcome.UPDATED:
            summary = f"kamino actualizado a {result.latest_version}"
        elif result.outcome is UpdateOutcome.NEWER:
            summary = "kamino ya era más reciente"
        else:
            summary = "kamino ya estaba al día"
        self.kamino.finish_flow(summary, 0)

    def action_escape(self) -> None:
        if self._finished:
            self.action_continue()
            return
        if self._replacing:
            self.query_one("#status", Static).update(texts.UPDATE_CANNOT_CANCEL)
            return
        self._cancel.set()

    def action_interrupt(self) -> None:
        if self._replacing:
            self.query_one("#status", Static).update(texts.UPDATE_CANNOT_CANCEL)
            return
        if self._finished:
            self.action_continue()
            return
        self._cancel.set()
        self.query_one("#state", Static).update(texts.UPDATE_CANCELLED)

    def action_help(self) -> None:
        self.app.push_screen(
            self.kamino.flow_help(
                texts.SCREEN_U1,
                texts.HELP_U1,
                flow_steps=("actualización",),
                current_step="actualización",
            )
        )


def start_update_from_menu(app: KaminoApp) -> None:
    app.push_screen(UpdateScreen())


def run_update_flow(
    *,
    current_version: str | None = None,
    target: Path | None = None,
    release: ReleaseInfo | None = None,
) -> int:
    from kamino.tui.app import run_app

    return run_app(
        mode="direct",
        initial=UpdateScreen(current_version=current_version, target=target, release=release),
    )
