"""Plan y Operation, compartidos por `backup` y `restore`.

Principio (diseño técnico §1): planificar y luego ejecutar. `--dry-run` solo
imprime el plan, así simulación y ejecución comparten el mismo código.

Pendiente: paso 5 del plan de implementación.
"""

from __future__ import annotations

from typing import NoReturn


def build_plan() -> NoReturn:
    """Construye el plan a partir de una selección (pendiente)."""
    raise NotImplementedError("plan.build_plan: pendiente de implementar (paso 5).")
