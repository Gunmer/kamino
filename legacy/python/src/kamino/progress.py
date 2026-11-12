"""Contrato de progreso y cancelación compartido por los casos de uso.

El núcleo nunca imprime: emite eventos tipados y consulta un callback de
cancelación en los bucles largos. La TUI traduce esos eventos a widgets y
responde a la cancelación con ``OperationCancelled``.
"""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from enum import StrEnum

from kamino.errors import OperationCancelled


class Phase(StrEnum):
    """Fase visible de una operación larga."""

    PLANNING = "planning"
    VERIFYING = "verifying"
    EXTRACTING = "extracting"
    CLASSIFYING = "classifying"
    WRITING = "writing"
    DOWNLOADING = "downloading"
    REPLACING = "replacing"


@dataclass(frozen=True)
class ProgressEvent:
    """Un avance de la operación: fase, posición, total y ruta opcional."""

    phase: Phase
    current: int
    total: int
    path: str | None = None


ProgressCallback = Callable[[ProgressEvent], None]
CancelCallback = Callable[[], bool]


def emit(
    callback: ProgressCallback | None,
    phase: Phase,
    current: int,
    total: int,
    path: str | None = None,
) -> None:
    """Notifica un avance, si hay interesado."""
    if callback is not None:
        callback(ProgressEvent(phase=phase, current=current, total=total, path=path))


def ensure_not_cancelled(cancelled: CancelCallback | None) -> None:
    """Aborta de forma cooperativa cuando la interfaz lo pidió."""
    if cancelled is not None and cancelled():
        raise OperationCancelled("operación cancelada")
