"""Pantallas transversales: ayuda, error común, terminal pequeña y diálogos."""

from __future__ import annotations

from collections.abc import Callable, Sequence
from dataclasses import dataclass

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import VerticalScroll
from textual.screen import Screen
from textual.widgets import OptionList, Static
from textual.widgets.option_list import Option

from kamino import texts


class ShortcutBar(Static):
    """Leyenda fija con las teclas disponibles en la pantalla."""

    def __init__(self, legend: str) -> None:
        super().__init__(legend, id="shortcut-bar", markup=False)


@dataclass(frozen=True)
class HelpContext:
    """Atajos y flujo que la ayuda contextual debe mostrar."""

    screen: str
    shortcuts: tuple[tuple[str, str], ...] = ()
    flow_steps: tuple[str, ...] = ()
    current_step: str | None = None
    secrets: bool = False


@dataclass(frozen=True)
class Choice:
    """Una acción de un diálogo."""

    id: str
    label: str
    variant: str = "default"


@dataclass(frozen=True)
class ErrorAction:
    """Una acción contextual de la pantalla de error."""

    id: str
    label: str
    callback: Callable[[], None]
    key: str
    variant: str = "default"


class HelpScreen(Screen[None]):
    """Ayuda contextual de la pantalla actual."""

    BINDINGS = [
        Binding("escape", "close", "Cerrar"),
        Binding("enter", "close", "Cerrar"),
        Binding("question_mark", "close", "Cerrar"),
    ]

    def __init__(self, context: HelpContext) -> None:
        super().__init__()
        self.context = context

    def compose(self) -> ComposeResult:
        yield Static(texts.HELP_TITLE, id="title")
        yield Static(
            texts.help_texts(
                self.context.screen,
                self.context.shortcuts,
                flow_steps=self.context.flow_steps,
                current_step=self.context.current_step,
                secrets=self.context.secrets,
                unicode=getattr(self.app, "unicode", True),
            ),
            id="help-body",
        )
        yield ShortcutBar(texts.SHORTCUTS_HELP)

    def action_close(self) -> None:
        self.dismiss(None)


class ErrorScreen(Screen[None]):
    """Error común de cuatro partes con detalle plegado y copiable."""

    BINDINGS = [
        Binding("t", "toggle_detail", "Detalle"),
        Binding("c", "copy_detail", "Copiar"),
        Binding("m", "menu", "Menú"),
        Binding("x", "exit", "Salir"),
        Binding("escape", "escape", "Volver"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(
        self,
        message: str,
        *,
        safety: str = texts.ERROR_NOTHING_WRITTEN,
        advice: str = texts.STATUS_REVIEW_DEFAULT,
        detail: str | None = None,
        actions: Sequence[ErrorAction] = (),
        show_menu: bool = True,
    ) -> None:
        super().__init__()
        self.message = message
        self.safety = safety
        self.advice = advice
        self.detail = detail
        self.actions = list(actions)
        self.show_menu = show_menu
        self.detail_visible = False

    def compose(self) -> ComposeResult:
        yield Static(texts.STATUS_ERROR, id="title")
        with VerticalScroll(id="error-body"):
            yield Static(f"{texts.ERROR_WHAT}\n{self.message}", id="error-what")
            yield Static(f"{texts.ERROR_SAFETY}\n{self.safety}", id="error-safety")
            yield Static(f"{texts.ERROR_ACTION}\n{self.advice}", id="error-action")
            yield Static("", id="error-detail")
        yield Static("", id="status")
        yield ShortcutBar(self._legend())

    def _legend(self) -> str:
        entries = [("t", "detalle"), ("c", "copiar")]
        entries.extend((action.key, action.label) for action in self.actions)
        if self.show_menu:
            entries.append(("m", texts.ERROR_BACK_TO_MENU))
        entries.append(("x", texts.ERROR_EXIT))
        return texts.shortcut_line(*entries)

    def on_mount(self) -> None:
        self.query_one("#error-detail", Static).display = False

    def on_key(self, event) -> None:
        for action in self.actions:
            if event.key == action.key:
                event.stop()
                action.callback()
                return

    def action_menu(self) -> None:
        if not self.show_menu:
            return
        self.app.return_to_menu(1)  # type: ignore[attr-defined]

    def action_exit(self) -> None:
        self.app.exit(1)

    def action_escape(self) -> None:
        if self.show_menu:
            self.action_menu()
            return
        self.action_exit()

    def action_toggle_detail(self) -> None:
        self.detail_visible = not self.detail_visible
        detail = self.query_one("#error-detail", Static)
        detail.display = self.detail_visible
        if self.detail_visible:
            detail.update(self.detail or texts.STATUS_NO_DETAIL)

    def action_copy_detail(self) -> None:
        text = self.detail or self.message
        supported = self.app.copy_text(text)  # type: ignore[attr-defined]
        status = self.query_one("#status", Static)
        status.update(texts.ERROR_COPIED if supported else texts.ERROR_COPY_UNSUPPORTED)

    def action_help(self) -> None:
        self.app.push_screen(
            HelpScreen(
                HelpContext(
                    screen=texts.SCREEN_ERROR,
                    shortcuts=(("t", "mostrar u ocultar detalle"), ("c", "copiar detalle")),
                )
            )
        )


class SmallTerminalScreen(Screen[None]):
    """Aviso por debajo de 80x24; el estado inferior se conserva."""

    BINDINGS = [Binding("question_mark", "help", "Ayuda")]

    def compose(self) -> ComposeResult:
        yield Static(texts.STATUS_WARN, id="title")
        yield Static(texts.ERROR_SMALL_TITLE, id="small-title")
        yield Static(texts.ERROR_SMALL_BODY, id="small-body")
        yield ShortcutBar(texts.SHORTCUTS_SMALL)

    def action_help(self) -> None:
        self.app.push_screen(HelpScreen(HelpContext(screen=texts.SCREEN_SMALL)))


class ChoiceDialog(Screen[str | None]):
    """Diálogo modal de acciones con selección por teclado."""

    BINDINGS = [Binding("escape", "cancel", "Volver")]

    def __init__(
        self,
        title: str,
        body: str,
        choices: Sequence[Choice],
        *,
        default: str | None = None,
    ) -> None:
        super().__init__()
        self.title_text = title
        self.body = body
        self.choices = list(choices)
        self.default = default

    def compose(self) -> ComposeResult:
        yield Static(self.title_text, id="title")
        yield Static(self.body, id="dialog-body")
        yield OptionList(
            *(Option(choice.label, id=choice.id) for choice in self.choices),
            Option(texts.BUTTON_BACK, id="cancel"),
            id="dialog-options",
            compact=True,
        )
        yield ShortcutBar(texts.SHORTCUTS_DIALOG)

    def on_mount(self) -> None:
        target = self.default or "cancel"
        options = self.query_one(OptionList)
        options.highlighted = next(
            (index for index, choice in enumerate(self.choices) if choice.id == target),
            len(self.choices),
        )
        options.focus()

    def on_option_list_option_selected(self, event: OptionList.OptionSelected) -> None:
        event.stop()
        self.dismiss(None if event.option.id == "cancel" else event.option.id)

    def action_cancel(self) -> None:
        self.dismiss(None)


class WarningScreen(Screen[bool]):
    """Aviso que permite aplicar una composición igualmente."""

    BINDINGS = [Binding("escape", "back", "Volver")]

    def __init__(self, message: str) -> None:
        super().__init__()
        self.message = message

    def compose(self) -> ComposeResult:
        yield Static(texts.STATUS_WARN, id="title")
        yield Static(
            f"{texts.DEPLOY_VALIDATION_INVALID.format(message=self.message)}. "
            "Puedes aplicar el híbrido igualmente.",
            id="warning",
        )
        yield OptionList(
            Option(texts.BUTTON_BACK_TO_DETAIL, id="back"),
            Option(texts.BUTTON_APPLY_ANYWAY, id="apply"),
            id="warning-options",
            compact=True,
        )
        yield ShortcutBar(texts.SHORTCUTS_DIALOG)

    def on_mount(self) -> None:
        options = self.query_one(OptionList)
        options.highlighted = 0
        options.focus()

    def on_option_list_option_selected(self, event: OptionList.OptionSelected) -> None:
        event.stop()
        self.dismiss(event.option.id == "apply")

    def action_back(self) -> None:
        self.dismiss(False)
