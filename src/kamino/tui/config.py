"""Pantalla de configuración de la TUI."""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path
from typing import TYPE_CHECKING

from textual import events
from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import VerticalScroll
from textual.screen import Screen
from textual.widgets import Input, Static

from kamino import texts
from kamino.config import (
    CATALOG_KEY,
    CLONE_KEY,
    SNAPSHOTS_KEY,
    AppConfig,
    ConfigLoadResult,
    resolve_path,
    save_config,
)
from kamino.errors import ConfigError
from kamino.tui.common import HelpContext, HelpScreen, ShortcutBar

if TYPE_CHECKING:
    from kamino.tui.app import KaminoApp


@dataclass(frozen=True)
class _Field:
    key: str
    input_id: str
    label: str
    hint: str
    warning: str


_FIELDS = (
    _Field(
        CATALOG_KEY,
        "catalog-dir",
        texts.SETTINGS_CATALOG_LABEL,
        texts.SETTINGS_CATALOG_HINT,
        texts.SETTINGS_WARN_CATALOG_MISSING,
    ),
    _Field(
        CLONE_KEY,
        "clone-output-dir",
        texts.SETTINGS_CLONE_LABEL,
        texts.SETTINGS_CLONE_HINT,
        texts.SETTINGS_WARN_CLONE_MISSING,
    ),
    _Field(
        SNAPSHOTS_KEY,
        "snapshots-dir",
        texts.SETTINGS_SNAPSHOTS_LABEL,
        texts.SETTINGS_SNAPSHOTS_HINT,
        texts.SETTINGS_WARN_SNAPSHOTS_MISSING,
    ),
)

_CONFIG_ATTRS = {
    CATALOG_KEY: "catalog_dir",
    CLONE_KEY: "clone_output_dir",
    SNAPSHOTS_KEY: "snapshots_dir",
}


class ConfigScreen(Screen[None]):
    """Vista y edición de las tres rutas de la TUI."""

    BINDINGS = [
        Binding("ctrl+s", "save", "Guardar", priority=True),
        Binding("enter", "next_field", "Siguiente", priority=True),
        Binding("escape", "back", "Volver"),
        Binding("f1", "help", "Ayuda"),
        Binding("question_mark", "help", "Ayuda", priority=True),
    ]

    def __init__(
        self,
        result: ConfigLoadResult,
        *,
        home: Path | None = None,
        error_mode: bool = False,
        on_saved: Callable[[AppConfig], None] | None = None,
    ) -> None:
        super().__init__()
        self.result = result
        self.home = home if home is not None else Path.home()
        self.error_mode = error_mode
        self.on_saved = on_saved
        self._save_error: str | None = None

    @property
    def kamino(self) -> KaminoApp:
        return self.app  # type: ignore[return-value]

    def compose(self) -> ComposeResult:
        yield Static(texts.SETTINGS_TITLE, id="settings-title")
        yield Static(texts.SETTINGS_SUBTITLE, id="settings-subtitle")
        yield Static(texts.SETTINGS_CLI_NOTE, id="settings-note")
        yield Static("", id="settings-error")
        with VerticalScroll(id="settings-body"):
            for field in _FIELDS:
                yield Static(field.label, classes="settings-label")
                yield Input(
                    placeholder=texts.SETTINGS_PATH_PLACEHOLDER,
                    id=field.input_id,
                )
        yield Static("", id="settings-hint")
        yield Static("", id="settings-status")
        yield ShortcutBar(self._shortcuts())

    def on_mount(self) -> None:
        error = self.query_one("#settings-error", Static)
        error.display = False
        if self.result.error is not None:
            error.update(f"{texts.STATUS_ERROR} {self.result.error}")
            error.display = True
        for field in _FIELDS:
            value = self.result.raw.get(
                field.key,
                str(getattr(self.result.config, _CONFIG_ATTRS[field.key])),
            )
            self._input(field).value = value
        self._refresh_hint()
        self._refresh_status()
        self._input(_FIELDS[0]).focus()

    def on_input_changed(self, _event: Input.Changed) -> None:
        self._save_error = None
        self._refresh_status()

    def on_descendant_focus(self, _event: events.DescendantFocus) -> None:
        self._refresh_hint()
        self._refresh_status()

    def action_next_field(self) -> None:
        inputs = [self._input(field) for field in _FIELDS]
        focused = self.app.focused
        try:
            index = inputs.index(focused) if focused is not None else -1
        except ValueError:
            index = -1
        inputs[(index + 1) % len(inputs)].focus()

    def action_save(self) -> None:
        values: dict[str, Path] = {}
        errors: list[tuple[_Field, str]] = []
        for field in _FIELDS:
            message, is_error = self._validate_field(field)
            if is_error:
                errors.append((field, message))
                continue
            values[field.key] = resolve_path(
                self._input(field).value,
                home=self.home,
                key=field.label,
            )

        if errors:
            field, message = errors[0]
            self._save_error = None
            self._show_status(message, kind="error")
            self._input(field).focus()
            return

        config = AppConfig(
            catalog_dir=values[CATALOG_KEY],
            clone_output_dir=values[CLONE_KEY],
            snapshots_dir=values[SNAPSHOTS_KEY],
        )
        try:
            save_config(config, home=self.home)
        except ConfigError as exc:
            self._save_error = str(exc)
            self._show_status(f"{texts.SETTINGS_SAVE_ERROR} {exc}", kind="error")
            return

        self.kamino.config = config
        self.kamino.config_result = ConfigLoadResult(config=config, exists=True)
        self.kamino.config_error = None
        if self.on_saved is not None:
            self.on_saved(config)
            return
        self.app.pop_screen()

    def action_back(self) -> None:
        if self.error_mode:
            self.app.exit(1)
            return
        self.app.pop_screen()

    def action_interrupt(self) -> None:
        self.action_back()

    def action_help(self) -> None:
        self.app.push_screen(
            HelpScreen(
                HelpContext(
                    screen=texts.SCREEN_SETTINGS,
                    shortcuts=texts.HELP_SETTINGS,
                )
            )
        )

    def _input(self, field: _Field) -> Input:
        return self.query_one(f"#{field.input_id}", Input)

    def _focused_field(self) -> _Field:
        focused = self.app.focused
        for field in _FIELDS:
            if self._input(field) is focused:
                return field
        return _FIELDS[0]

    def _refresh_hint(self) -> None:
        self.query_one("#settings-hint", Static).update(self._focused_field().hint)

    def _refresh_status(self) -> None:
        field = self._focused_field()
        message, is_error = self._validate_field(field)
        if message and is_error:
            self._show_status(message, kind="error")
            return
        if self._save_error is not None:
            self._show_status(self._save_error, kind="error")
            return
        if self.result.error is not None:
            self._show_status(texts.STATUS_ERROR, kind="error")
            return
        if message:
            self._show_status(message, kind="warning")
            return
        if not self.result.exists:
            self._show_status(texts.SETTINGS_WARN_NO_FILE, kind="warning")
            return
        self._show_status(texts.STATUS_OK, kind="success")

    def _validate_field(self, field: _Field) -> tuple[str | None, bool]:
        try:
            path = resolve_path(
                self._input(field).value,
                home=self.home,
                key=field.label,
            )
        except ConfigError as exc:
            return str(exc), True
        if not path.exists():
            return f"{field.label}: {field.warning}", False
        return None, False

    def _show_status(self, message: str, *, kind: str) -> None:
        status = self.query_one("#settings-status", Static)
        status.remove_class("settings-error", "settings-warning", "settings-success")
        if kind == "error":
            status.add_class("settings-error")
            text = f"{texts.STATUS_ERROR} {message}"
        elif kind == "warning":
            status.add_class("settings-warning")
            text = f"{texts.STATUS_WARN} {message}"
        else:
            status.add_class("settings-success")
            text = message
        status.update(text)

    def _shortcuts(self) -> str:
        if self.error_mode:
            return texts.SHORTCUTS_SETTINGS_ERROR
        return texts.SHORTCUTS_SETTINGS
