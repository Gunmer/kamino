"""Copia de seguridad previa a restaurar (snapshot).

Se guarda en `~/Library/Application Support/kamino/snapshots/<fecha>.tar.gz` con
el mismo formato que un respaldo, así el futuro `undo` es un `restore` del snapshot
(diseño técnico §6).

Pendiente: paso 6 del plan de implementación.
"""

from __future__ import annotations

from typing import NoReturn


def create_snapshot() -> NoReturn:
    """Guarda la configuración actual antes de sobrescribir (pendiente)."""
    raise NotImplementedError("snapshot.create_snapshot: pendiente de implementar (paso 6).")
