"""Interfaz con el usuario: selección interactiva, tablas y progreso.

Usa `questionary` y `rich`. Es la única capa que imprime o pregunta;
el resto del núcleo no habla con el usuario (diseño técnico §1).
"""

from __future__ import annotations

import sys
from collections.abc import Sequence
from typing import TYPE_CHECKING

import questionary
from rich.console import Console
from rich.table import Table

if TYPE_CHECKING:
    from kamino.backup import BackupResult
    from kamino.restore import RestoreResult

_console = Console()


def select_apps(choices: Sequence[tuple[str, str]]) -> list[str]:
    """Selección múltiple interactiva de apps; devuelve los ids elegidos.

    `choices` son pares `(id, nombre)`. En un entorno sin TTY (o si questionary
    falla) devuelve todos los ids, para no bloquear la CLI.
    """
    if not choices:
        return []
    if not sys.stdin.isatty():
        return [app_id for app_id, _ in choices]
    options = [
        questionary.Choice(title=f"{name} ({app_id})", value=app_id) for app_id, name in choices
    ]
    try:
        selected = questionary.checkbox("Selecciona las apps:", choices=options).ask()
    except (KeyboardInterrupt, EOFError):
        return []
    return selected if selected is not None else []


def print_backup_summary(result: BackupResult) -> None:
    """Imprime el resumen del respaldo, o el plan en modo dry-run."""
    if result.dry_run:
        _console.print("[bold]Plan de respaldo (dry-run)[/bold]")
        for op in result.plan.operations:
            _console.print(f"  + {op.path}")
        for skipped in result.plan.skipped:
            _console.print(f"  ~ {skipped} (omitido)")
        return
    table = Table(title="Respaldo completado")
    table.add_column("App", style="bold")
    table.add_column("Archivos", justify="right")
    for app in result.manifest.apps:
        table.add_row(app.name, str(len(app.files)))
    _console.print(table)
    _console.print(f"Archivo: {result.output}")


def print_restore_summary(result: RestoreResult) -> None:
    """Imprime el resumen de la restauración, o el plan en modo dry-run."""
    if result.dry_run:
        _console.print("[bold]Plan de restauración (dry-run)[/bold]")
        for entry in result.plan.entries:
            _console.print(f"  [{entry.status}] {entry.path}")
        return
    _console.print(f"Restauradas: {len(result.restored)}")
    _console.print(f"Omitidas: {len(result.omitted)}")
    if result.snapshot is not None:
        _console.print(f"Snapshot: {result.snapshot}")
