"""Interfaz con el usuario: selección interactiva, tablas y progreso.

Usa `questionary` y `rich`. Es la única capa que imprime o pregunta;
el resto del núcleo no habla con el usuario (diseño técnico §1).
"""

from __future__ import annotations

from typing import NoReturn


def select_apps() -> NoReturn:
    """Selección múltiple interactiva de apps (pendiente)."""
    raise NotImplementedError("ui.select_apps: pendiente de implementar.")
