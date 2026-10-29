"""Caso de uso de restauración: construir el plan y ejecutarlo.

Flujo (diseño técnico §5.2): abrir y verificar el archivo, cruzar apps con las
instaladas, construir el plan, verificar checksums, snapshot previo y aplicación
con reemplazo atómico.

Pendiente: paso 6 del plan de implementación.
"""

from __future__ import annotations

from typing import NoReturn


def run_restore() -> NoReturn:
    """Ejecuta el flujo completo de restauración (pendiente)."""
    raise NotImplementedError("restore.run_restore: pendiente de implementar (paso 6).")
