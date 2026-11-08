"""Interactive restore application and its public entry point."""

from __future__ import annotations

from textual.app import App
from textual.binding import Binding

from kamino.conflicts import ConflictDecisions
from kamino.errors import KaminoError
from kamino.restore import RestorePlan, RestoreResult, RestoreSession
from kamino.tui.screens import (
    AppSelectionScreen,
    ConflictSummaryScreen,
    ErrorScreen,
    HelpScreen,
)


class KaminoRestoreApp(App[RestoreResult | None]):
    """Review conflicts, decide block by block and apply a restore."""

    CSS_PATH = "styles.tcss"
    TITLE = "kamino · restauración"
    BINDINGS = [Binding("question_mark", "help", "Ayuda")]

    def __init__(self, session: RestoreSession, selected: list[str] | None = None) -> None:
        super().__init__()
        self.session = session
        self.selected = selected
        self.plan: RestorePlan | None = None
        self.error: KaminoError | None = None
        self.decisions = ConflictDecisions()

    def on_mount(self) -> None:
        if self.selected is None:
            self.push_screen(AppSelectionScreen(self.session.restorable_apps), self._apps_chosen)
        else:
            self._start(self.selected)

    def _apps_chosen(self, selected: list[str] | None) -> None:
        if selected is None:
            self.exit(None)
            return
        self._start(selected)

    def _start(self, selected: list[str]) -> None:
        try:
            self.plan = self.session.plan(selected)
        except KaminoError as exc:
            self.error = exc
            self.push_screen(ErrorScreen(str(exc), fatal=True))
            return
        self.selected = selected
        self.push_screen(ConflictSummaryScreen(self.session, self.plan, self.decisions))

    def action_help(self) -> None:
        self.push_screen(HelpScreen())


def run_restore_tui(
    session: RestoreSession, selected: list[str] | None = None
) -> RestoreResult | None:
    """Run the interactive restore and return its result (``None`` on cancel)."""
    app = KaminoRestoreApp(session, selected)
    result = app.run()
    if app.error is not None:
        raise app.error
    return result
