"""Detección de apps instaladas y su versión.

Mecanismos (diseño técnico §6): leer `Info.plist` por `CFBundleIdentifier` en
`/Applications`, `~/Applications` y `/System/Applications` para apps con `.app`;
`shutil.which` y `search_paths` para herramientas de terminal.

Pendiente: paso 3 del plan de implementación.
"""

from __future__ import annotations

from typing import NoReturn


def is_installed() -> NoReturn:
    """Indica si una app está instalada (pendiente)."""
    raise NotImplementedError("detect.is_installed: pendiente de implementar (paso 3).")
