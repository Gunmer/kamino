"""Carga del catálogo embebido y del directorio del usuario.

El catálogo viene embebido en el paquete (`catalog/apps/*.toml`) y se complementa
con `~/.config/kamino/apps/`, donde se añaden o sobrescriben definiciones sin tocar
el código (diseño técnico §6).

Pendiente: paso 2 del plan de implementación.
"""

from __future__ import annotations

from typing import NoReturn


def load_catalog() -> NoReturn:
    """Carga el catálogo embebido más el del usuario (pendiente)."""
    raise NotImplementedError("loader.load_catalog: pendiente de implementar (paso 2).")
