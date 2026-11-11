"""Menú principal de Kamino."""

from __future__ import annotations

from typing import TYPE_CHECKING

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import OptionList, Static
from textual.widgets.option_list import Option

from kamino import texts
from kamino.tui.common import HelpContext, HelpScreen, ShortcutBar

if TYPE_CHECKING:
    from kamino.tui.app import KaminoApp


class MenuScreen(Screen[None]):
    """Wordmark, cápsulas, frase fija y las cinco opciones."""

    BINDINGS = [
        Binding("c", "clone", "Clonar"),
        Binding("d", "deploy", "Desplegar"),
        Binding("u", "update", "Actualizar"),
        Binding("q", "quit", "Abandonar"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def compose(self) -> ComposeResult:
        yield Static("", id="wordmark")
        yield Static("", id="subtitle")
        yield Static("", id="capsules")
        yield Static("", id="phrase")
        with Vertical(id="menu"):
            yield OptionList(
                Option(f"{texts.MENU_CLONE}  (c)", id="c"),
                Option(f"{texts.MENU_DEPLOY}  (d)", id="d"),
                Option(f"{texts.MENU_UPDATE}  (u)", id="u"),
                Option(texts.MENU_CONFIG, id="config"),
                Option(f"{texts.MENU_EXIT}  (q)", id="q"),
                id="menu-options",
                compact=True,
            )
        yield Static("", id="last-mission")
        yield Static("", id="menu-footer")
        yield ShortcutBar(texts.SHORTCUTS_MENU)

    def on_mount(self) -> None:
        self.refresh_mission()
        self.apply_height()
        options = self.query_one("#menu-options", OptionList)
        options.highlighted = 0
        options.focus()

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    def apply_height(self) -> None:
        height = self.size.height
        wordmark = self.query_one("#wordmark", Static)
        subtitle = self.query_one("#subtitle", Static)
        capsules = self.query_one("#capsules", Static)
        phrase = self.query_one("#phrase", Static)
        last_mission = self.query_one("#last-mission", Static)
        wordmark.display = self.kamino.show_logo
        if self.kamino.show_logo:
            wordmark.update("\n".join(texts.WORDMARK))
        subtitle.display = height >= 30
        subtitle.update(texts.SUBTITLE)
        capsules.display = height >= 26
        capsules.update(self._capsules())
        phrase.display = height >= 30
        phrase.update(self.kamino.session.phrase)
        last_mission.display = height >= 30 and bool(self.kamino.session.last_mission)
        self.query_one("#menu-footer", Static).update(texts.STATUS_OK if height >= 30 else "")

    def on_resize(self) -> None:
        self.apply_height()

    def _capsules(self) -> str:
        total = texts.CAPSULE_TOTAL
        empty = texts.CAPSULE_EMPTY if self.kamino.unicode else texts.CAPSULE_ASCII_EMPTY
        original = texts.CAPSULE_ORIGINAL if self.kamino.unicode else texts.CAPSULE_ASCII_ORIGINAL
        units = [empty for _ in range(total - 1)] + [original]
        return "  ".join(units)

    def refresh_mission(self) -> None:
        last = self.query_one("#last-mission", Static)
        mission = self.kamino.session.last_mission
        last.update(texts.LAST_MISSION.format(summary=mission) if mission else "")
        last.display = self.size.height >= 30 and bool(mission)

    def on_option_list_option_selected(self, event: OptionList.OptionSelected) -> None:
        event.stop()
        actions = {
            "c": self.action_clone,
            "d": self.action_deploy,
            "u": self.action_update,
            "config": self.action_config,
            "q": self.action_quit,
        }
        action = actions.get(event.option.id or "")
        if action is not None:
            action()

    def action_clone(self) -> None:
        from kamino.tui.clone import start_clone_from_menu

        start_clone_from_menu(self.kamino)

    def action_deploy(self) -> None:
        from kamino.tui.deploy import start_deploy_from_menu

        start_deploy_from_menu(self.kamino)

    def action_update(self) -> None:
        from kamino.tui.update import start_update_from_menu

        start_update_from_menu(self.kamino)

    def action_config(self) -> None:
        from kamino.tui.config import ConfigScreen

        self.app.push_screen(
            ConfigScreen(
                self.kamino.config_result,
                home=self.kamino.config_home,
            )
        )

    def action_quit(self) -> None:
        self.kamino.exit(self.kamino.session.last_code)

    def action_interrupt(self) -> None:
        self.action_quit()

    def action_help(self) -> None:
        self.app.push_screen(
            HelpScreen(
                HelpContext(
                    screen=texts.SCREEN_MENU,
                    shortcuts=texts.HELP_MENU,
                    flow_steps=(),
                )
            )
        )
