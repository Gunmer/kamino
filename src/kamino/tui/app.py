"""Aplicación única de Kamino: menú, flujos y estado de sesión."""

from __future__ import annotations

import os
import sys
from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path
from typing import Literal

from textual.app import App
from textual.binding import Binding
from textual.screen import Screen
from textual.widgets import Static

from kamino import texts
from kamino.config import AppConfig, ConfigLoadResult, default_config, load_config
from kamino.tui.menu import MenuScreen


class SizeGuard(Static):
    """Aviso a pantalla completa que conserva la pantalla inferior y bloquea q."""

    can_focus = True
    BINDINGS = [
        Binding("q", "noop", "Sin salir"),
        Binding("escape", "noop", "Sin salir"),
    ]

    def __init__(self) -> None:
        super().__init__(
            "\n".join(
                (
                    texts.STATUS_WARN,
                    texts.ERROR_SMALL_TITLE,
                    "",
                    texts.ERROR_SMALL_BODY,
                )
            ),
            id="size-guard",
        )

    def on_mount(self) -> None:
        self.focus()

    def action_noop(self) -> None:
        return None


def default_theme() -> str:
    """Tema ANSI del terminal, salvo que `TEXTUAL_THEME` lo sobrescriba."""
    return os.environ.get("TEXTUAL_THEME", "ansi-dark")


def _default_phrase() -> str:
    import random

    return random.choice(texts.PHRASES)


def _terminal_utf8() -> bool:
    if os.environ.get("KAMINO_NO_LOGO"):
        return False
    encoding = (getattr(sys.stdout, "encoding", None) or "").lower()
    return "utf" in encoding


@dataclass
class SessionState:
    """Estado que vive mientras la app está abierta."""

    phrase: str
    last_mission: str | None = None
    last_code: int = 0
    mode: Literal["menu", "direct"] = "menu"


class KaminoApp(App[int]):
    """Una sola app con modo menú y modo directo."""

    CSS_PATH = "styles.tcss"
    TITLE = texts.APP_NAME
    BINDINGS = [Binding("ctrl+c", "interrupt", "Cancelar", show=False, priority=True)]
    theme = default_theme()

    def __init__(
        self,
        *,
        mode: Literal["menu", "direct"] = "menu",
        initial: Screen | None = None,
        phrase: str | None = None,
        show_logo: bool | None = None,
        unicode: bool | None = None,
        config_result: ConfigLoadResult | None = None,
        config_home: Path | None = None,
        after_config: Callable[[KaminoApp], Screen] | None = None,
    ) -> None:
        super().__init__()
        self.theme = default_theme()
        self.session = SessionState(phrase=phrase if phrase is not None else _default_phrase())
        self.session.mode = mode
        self.initial = initial
        self.unicode = _terminal_utf8() if unicode is None else unicode
        self.show_logo = self.unicode if show_logo is None else show_logo
        self.cleanup = None
        self._guard_pending = False
        self.config_home = config_home
        self.config_result = (
            config_result
            if config_result is not None
            else ConfigLoadResult(config=default_config(config_home))
        )
        self.config = self.config_result.config
        self.config_error = self.config_result.error
        self.after_config = after_config

    def on_mount(self) -> None:
        if self.config_error is not None:
            from kamino.tui.config import ConfigScreen

            self.push_screen(
                ConfigScreen(
                    self.config_result,
                    home=self.config_home,
                    error_mode=True,
                    on_saved=self._after_config_saved,
                )
            )
        elif self.initial is not None:
            self.push_screen(self.initial)
        else:
            self.push_screen(MenuScreen())
        self.call_after_refresh(self._ensure_size_guard)
        self.set_interval(0.2, self._ensure_size_guard)

    def _after_config_saved(self, config: AppConfig) -> None:
        self.config = config
        self.config_error = None
        self.config_result = ConfigLoadResult(config=config, exists=True)
        next_screen = self.after_config(self) if self.after_config is not None else MenuScreen()
        self.switch_screen(next_screen)

    def push_screen(self, screen, callback=None, wait_for_dismiss=False, *, mode=None):  # type: ignore[override]
        if self.screen_stack:
            self.screen.display = False
        result = super().push_screen(
            screen, callback=callback, wait_for_dismiss=wait_for_dismiss, mode=mode
        )
        self.call_after_refresh(self._ensure_size_guard)
        self.call_after_refresh(self._repaint_screen)
        return result

    def pop_screen(self):  # type: ignore[override]
        result = super().pop_screen()
        if self.screen_stack:
            self.screen.display = True
        self.call_after_refresh(self._ensure_size_guard)
        self.call_after_refresh(self._repaint_screen)
        return result

    def switch_screen(self, screen: Screen | str):  # type: ignore[override]
        if self.screen_stack:
            self.screen.display = False
        result = super().switch_screen(screen)
        self.call_after_refresh(self._ensure_size_guard)
        self.call_after_refresh(self._repaint_screen)
        return result

    @property
    def _background_screens(self) -> list[Screen]:
        """No screen is composited below the active one.

        With the ansi-dark theme every screen background is `ansi_default`
        (alpha 0), which Textual treats as translucent and renders the
        previous screen's compositor underneath. Kamino screens always fill
        the terminal, so uncovered cells must show the terminal background,
        never a previous screen.
        """
        return []

    def _repaint_screen(self) -> None:
        """Repinta la pantalla completa para no dejar restos de la anterior."""
        if self.screen_stack:
            self.screen.refresh(repaint=True)

    def on_resize(self) -> None:
        self._ensure_size_guard()

    def _ensure_size_guard(self) -> None:
        if not self.screen_stack:
            return
        small = self.size.width < 80 or self.size.height < 24
        existing = list(self.screen.query("#size-guard"))
        if small:
            if not existing:
                self.screen.mount(SizeGuard())
            elif existing[0].is_mounted:
                existing[0].focus()
            return
        for guard in existing:
            guard.remove()
        if existing:
            self.screen.refresh(repaint=True)

    def _run_cleanup(self) -> None:
        cleanup = self.cleanup
        self.cleanup = None
        if cleanup is not None:
            cleanup()

    def exit(self, result=None, return_code: int = 0, message=None):  # type: ignore[override]
        self._run_cleanup()
        return super().exit(result=result, return_code=return_code, message=message)

    def on_unmount(self) -> None:
        self._run_cleanup()

    def finish_flow(self, mission: str, code: int) -> None:
        """Cierra un flujo: menú con última misión o salida directa con código."""
        self._run_cleanup()
        self.session.last_mission = mission
        self.session.last_code = code
        if self.session.mode == "direct":
            self.exit(code)
            return
        self.return_to_menu(code)

    def return_to_menu(self, code: int) -> None:
        self.session.last_code = code
        if self.session.mode == "direct":
            self.exit(code)
            return
        while len(self.screen_stack) > 2:
            super().pop_screen()
        menu = next(
            (screen for screen in self.screen_stack if isinstance(screen, MenuScreen)),
            None,
        )
        if menu is not None:
            menu.display = True
            menu.refresh_mission()
            menu.apply_height()
        self.call_after_refresh(self._repaint_screen)

    def action_interrupt(self) -> None:
        """Reparte Ctrl+C a la pantalla activa y respeta sus puntos de no retorno."""
        handler = getattr(self.screen, "action_interrupt", None)
        if callable(handler):
            handler()

    def copy_text(self, text: str) -> bool:
        """Intenta copiar con OSC 52 y avisa si el terminal no lo admite."""
        self.copy_to_clipboard(text)
        return clipboard_supported()

    def flow_help(
        self,
        screen: str,
        shortcuts: tuple[tuple[str, str], ...],
        *,
        flow_steps: tuple[str, ...] = (),
        current_step: str | None = None,
        secrets: bool = False,
    ):
        from kamino.tui.common import HelpContext, HelpScreen

        return HelpScreen(
            HelpContext(
                screen=screen,
                shortcuts=shortcuts,
                flow_steps=flow_steps,
                current_step=current_step,
                secrets=secrets,
            )
        )


def clipboard_supported() -> bool:
    """Heurística para terminales que no implementan la copia OSC 52."""
    program = os.environ.get("TERM_PROGRAM", "")
    if program == "Apple_Terminal":
        return False
    return not (os.environ.get("TMUX") or os.environ.get("STY"))


def run_app(
    *,
    mode: Literal["menu", "direct"],
    initial: Screen | None = None,
    phrase: str | None = None,
    show_logo: bool | None = None,
    unicode: bool | None = None,
    config_result: ConfigLoadResult | None = None,
    config_home: Path | None = None,
    after_config: Callable[[KaminoApp], Screen] | None = None,
) -> int:
    """Ejecuta la app y devuelve su código de salida."""
    resolved = config_result if config_result is not None else load_config(home=config_home)
    app = KaminoApp(
        mode=mode,
        initial=initial,
        phrase=phrase,
        show_logo=show_logo,
        unicode=unicode,
        config_result=resolved,
        config_home=config_home,
        after_config=after_config,
    )
    result = app.run()
    return int(result or 0)
