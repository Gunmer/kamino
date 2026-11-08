"""Interactive deploy application and its public entry point."""

from __future__ import annotations

import os

from textual.app import App
from textual.binding import Binding

from kamino.conflicts import ConflictDecisions
from kamino.deploy import DeployPlan, DeployResult, DeploySession
from kamino.errors import KaminoError
from kamino.tui.screens import (
    AppSelectionScreen,
    ConflictSummaryScreen,
    ErrorScreen,
    HelpScreen,
)


def _default_theme() -> str:
    """Devuelve el tema ANSI del terminal, salvo que TEXTUAL_THEME lo sobrescriba."""
    return os.environ.get("TEXTUAL_THEME", "ansi-dark")


class KaminoDeployApp(App[DeployResult | None]):
    """Review conflicts, decide line by line and apply a deploy."""

    CSS_PATH = "styles.tcss"
    TITLE = "kamino · despliegue"
    theme = _default_theme()
    BINDINGS = [Binding("question_mark", "help", "Ayuda")]

    def __init__(self, session: DeploySession, selected: list[str] | None = None) -> None:
        super().__init__()
        self.session = session
        self.selected = selected
        self.plan: DeployPlan | None = None
        self.error: KaminoError | None = None
        self.decisions = ConflictDecisions()

    def on_mount(self) -> None:
        if self.selected is None:
            self.push_screen(AppSelectionScreen(self.session.deployable_apps), self._apps_chosen)
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


def run_deploy_tui(
    session: DeploySession, selected: list[str] | None = None
) -> DeployResult | None:
    """Run the interactive deploy and return its result (``None`` on cancel)."""
    app = KaminoDeployApp(session, selected)
    result = app.run()
    if app.error is not None:
        raise app.error
    return result
