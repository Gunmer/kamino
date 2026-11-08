"""Interfaz con el usuario: selección interactiva, tablas y progreso.

Usa `questionary` y `rich`. Es la única capa que imprime o pregunta;
el resto del núcleo no habla con el usuario (diseño técnico §1).
"""

from __future__ import annotations

import difflib
from collections.abc import Sequence
from typing import TYPE_CHECKING

import questionary
from rich.console import Console
from rich.table import Table
from rich.text import Text

if TYPE_CHECKING:
    from kamino.clone import CloneResult
    from kamino.conflicts import Conflict
    from kamino.deploy import DeployResult
    from kamino.update import UpdateResult

_console = Console()

#: Texto del aviso previo al clon. No se interpola ningún valor de archivo.
PRIVACY_WARNING = (
    "aviso: el archivo del clon no está cifrado y las configuraciones incluidas "
    "pueden contener secretos. Los archivos de credenciales conocidos se excluyen, "
    "pero los ajustes permitidos se conservan íntegros."
)


#: Tripletas ``(id, title, description)`` para el menú de comandos.
CommandChoice = tuple[str, str, str]


def print_privacy_warning() -> None:
    """Muestra el aviso de contenido sensible antes de clonar."""
    _console.print(f"[yellow]{PRIVACY_WARNING}[/yellow]")


def print_update_start() -> None:
    """Muestra que comienza la comprobación de la última versión."""
    _console.print("Comprobando la última versión publicada...")


def print_update_result(result: UpdateResult) -> None:
    """Imprime el resultado de la actualización o de la consulta."""
    if result.updated:
        _console.print(f"kamino {result.current_version} -> {result.latest_version}")
        _console.print(f"Actualizado: {result.target}")
        return
    if result.current_version == result.latest_version:
        _console.print(f"kamino {result.current_version} ya es la última versión disponible.")
        return
    _console.print(
        f"kamino {result.current_version} es más reciente que la última publicada "
        f"({result.latest_version}); no se modifica."
    )


def select_command(choices: Sequence[CommandChoice]) -> str | None:
    """Selección simple de un comando.

    :param choices: Tripletas ``(id, title, description)``.
    :returns: El ``id`` elegido o ``None`` si se cancela.
    """
    options = [
        questionary.Choice(title=f"{title}: {description}", value=command_id)
        for command_id, title, description in choices
    ]
    try:
        return questionary.select("¿Qué quieres hacer?", choices=options).ask()
    except (KeyboardInterrupt, EOFError):
        return None


def ask_path(prompt: str) -> str | None:
    """Pide una ruta de archivo; devuelve `None` si el usuario cancela."""
    try:
        return questionary.text(prompt).ask()
    except (KeyboardInterrupt, EOFError):
        return None


def select_apps(choices: Sequence[tuple[str, str]]) -> list[str] | None:
    """Selección múltiple interactiva; devuelve `None` si el usuario cancela.

    `choices` son pares `(id, nombre)`. Con lista vacía devuelve `[]`. La
    decisión sobre entorno sin TTY la toma la capa CLI, no esta función.
    """
    if not choices:
        return []
    options = [
        questionary.Choice(title=f"{name} ({app_id})", value=app_id) for app_id, name in choices
    ]
    try:
        selected = questionary.checkbox("Selecciona las apps:", choices=options).ask()
    except (KeyboardInterrupt, EOFError):
        return None
    return selected


def print_clone_summary(result: CloneResult, *, verbose: bool = False) -> None:
    """Imprime el resumen de la clonación, o el plan en modo dry-run."""
    if result.dry_run:
        _console.print("[bold]Plan de clonación (dry-run)[/bold]")
        for op in result.plan.operations:
            _console.print(f"  + {op.path}")
        for skipped in result.plan.skipped:
            _console.print(f"  ~ {skipped} (omitido)")
        for notice in getattr(result.plan, "notices", []):
            _console.print(f"  ! {notice}")
        return
    table = Table(title="Clonación completada")
    table.add_column("App", style="bold")
    table.add_column("Archivos", justify="right")
    for app in result.manifest.apps:
        table.add_row(app.name, str(len(app.files)))
    _console.print(table)
    _console.print(f"Archivo: {result.output}")
    if verbose:
        _console.print("[bold]Archivos clonados[/bold]")
        for op in result.plan.operations:
            _console.print(f"  + {op.path}")
        for skipped in result.plan.skipped:
            _console.print(f"  ~ {skipped} (omitido)")
        for notice in getattr(result.plan, "notices", []):
            _console.print(f"  ! {notice}")


def print_deploy_summary(result: DeployResult, *, verbose: bool = False) -> None:
    """Print the deploy summary, or the plan and diffs in dry-run mode."""
    if result.dry_run:
        _console.print("[bold]Plan de despliegue (dry-run)[/bold]")
        for entry in result.plan.entries:
            _console.print(f"  [{entry.status}] {entry.path}")
        for conflict in result.plan.conflicts:
            _print_conflict_diff(conflict)
        _console.print(f"Omitidas: {len(result.omitted)}")
        for notice in getattr(result.plan, "notices", []):
            _console.print(f"  ! {notice}")
        return
    _console.print(f"Desplegadas: {len(result.deployed)}")
    _console.print(f"Mezcladas: {len(result.merged)}")
    _console.print(f"Omitidas: {len(result.omitted)}")
    if result.snapshot is not None:
        _console.print(f"Snapshot: {result.snapshot}")
    for notice in getattr(result.plan, "notices", []):
        _console.print(f"[yellow]! {notice}[/yellow]")
    if verbose:
        _console.print("[bold]Detalle[/bold]")
        for path in result.deployed:
            _console.print(f"  + {path}")
        for path in result.merged:
            _console.print(f"  * {path}")
        for path in result.omitted:
            _console.print(f"  ~ {path}")


def _print_conflict_diff(conflict: Conflict) -> None:
    """Write one conflict diff or a whole-file note to stdout."""
    if conflict.kind.value == "text":
        if conflict.local_text is None or conflict.clone_text is None:
            return
        diff = difflib.unified_diff(
            conflict.local_text.splitlines(),
            conflict.clone_text.splitlines(),
            fromfile=f"local/{conflict.path}",
            tofile=f"clon/{conflict.path}",
            lineterm="",
        )
        _console.print(Text("\n".join(diff)), markup=False, highlight=False)
        return
    if conflict.kind.value == "symlink":
        _console.print(
            f"  ! {conflict.path}: enlace local -> {conflict.local_link}, "
            f"clon -> {conflict.clone_link}"
        )
        return
    reason = "binario" if conflict.kind.value == "binary" else "grande (>5 MB)"
    _console.print(f"  ! {conflict.path}: conflicto {reason}; se resuelve como archivo entero")
