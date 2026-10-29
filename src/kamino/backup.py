"""Caso de uso de respaldo: construir el plan y ejecutarlo.

Flujo (diseño técnico §5.1): cargar catálogo, detectar apps instaladas, recoger
selección, expandir paths aplicando exclusiones, construir el plan y escribir el
archivo en un temporal para renombrarlo al nombre final.

Pendiente: paso 5 del plan de implementación.
"""

from __future__ import annotations

from typing import NoReturn


def run_backup() -> NoReturn:
    """Ejecuta el flujo completo de respaldo (pendiente)."""
    raise NotImplementedError("backup.run_backup: pendiente de implementar (paso 5).")
