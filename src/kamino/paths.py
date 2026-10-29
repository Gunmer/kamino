"""Normalización de rutas relativas al home y validación.

Todo se guarda relativo a `~` (diseño técnico §1 y §4.1), lo que resuelve la
migración entre usuarios. Las rutas fuera del home no se admiten en la v1.
"""

from __future__ import annotations

from typing import NoReturn


def to_relative() -> NoReturn:
    """Convierte una ruta absoluta del home a su forma relativa (pendiente)."""
    raise NotImplementedError("paths.to_relative: pendiente de implementar.")


def to_absolute() -> NoReturn:
    """Resuelve una ruta relativa al home del usuario actual (pendiente)."""
    raise NotImplementedError("paths.to_absolute: pendiente de implementar.")
