"""Command-line interface: comandos y opciones, capa fina sobre los casos de uso.

La lógica no vive aquí: `backup` y `restore` delegan en los casos de uso
(`kamino.backup` y `kamino.restore`). Solo se imprime o pregunta en `ui`.
"""

from __future__ import annotations

import sys
from pathlib import Path
from typing import Annotated

import typer

from kamino import __version__, ui
from kamino.archive import read_manifest
from kamino.backup import run_backup
from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.detect import is_installed
from kamino.errors import KaminoError
from kamino.restore import run_restore
from kamino.snapshot import SNAPSHOT_APP_ID

app = typer.Typer(
    name="kamino",
    help="Respalda y restaura configuraciones de aplicaciones de macOS.",
    no_args_is_help=True,
)


def _version_callback(value: bool) -> None:
    if value:
        typer.echo(f"kamino {__version__}")
        raise typer.Exit()


@app.callback()
def main(
    version: Annotated[
        bool | None,
        typer.Option("--version", help="Muestra la versión y termina.", callback=_version_callback),
    ] = None,
) -> None:
    """kamino: respaldo y restauración de configuraciones de macOS."""


def _installed_apps(catalog: dict[str, AppDefinition]) -> list[AppDefinition]:
    return [app for app in catalog.values() if is_installed(app)]


@app.command()
def backup(
    apps: Annotated[
        str | None,
        typer.Option("--apps", help="Apps a respaldar, separadas por comas (modo no interactivo)."),
    ] = None,
    output: Annotated[
        str | None,
        typer.Option("--output", help="Ruta del archivo de respaldo."),
    ] = None,
    dry_run: Annotated[
        bool,
        typer.Option("--dry-run", help="Simula sin escribir nada."),
    ] = False,
    verbose: Annotated[
        bool,
        typer.Option("--verbose", help="Salida detallada."),
    ] = False,
) -> None:
    """Crea un archivo de respaldo con las apps elegidas."""
    try:
        catalog = load_catalog()
        installed = _installed_apps(catalog)
        missing: list[str] = []
        if apps:
            requested = [part.strip() for part in apps.split(",") if part.strip()]
            selected = [app.id for app in installed if app.id in requested]
            missing = [rid for rid in requested if rid not in selected]
        elif not sys.stdin.isatty():
            # Sin terminal no hay selección: respaldar todas las instaladas
            # (el respaldo no es destructivo).
            selected = [app.id for app in installed]
        else:
            chosen = ui.select_apps([(app.id, app.name) for app in installed])
            if chosen is None:
                typer.echo("respaldo cancelado", err=True)
                raise typer.Exit(code=1)
            selected = chosen
        for rid in missing:
            typer.echo(f"aviso: app no instalada o desconocida: {rid}", err=True)
        result = run_backup(
            selected=selected,
            output=Path(output) if output else None,
            dry_run=dry_run,
            catalog=catalog,
        )
        ui.print_backup_summary(result, verbose=verbose)
        raise typer.Exit(code=3 if (result.plan.skipped or missing) else 0)
    except KaminoError as exc:
        typer.echo(str(exc), err=True)
        raise typer.Exit(code=1) from None


@app.command()
def restore(
    backup_file: Annotated[
        Path,
        typer.Argument(help="Archivo de respaldo (.tar.gz)."),
    ],
    dry_run: Annotated[
        bool,
        typer.Option("--dry-run", help="Simula sin escribir nada."),
    ] = False,
    yes: Annotated[
        bool,
        typer.Option("--yes", help="Aplica todas las apps restaurables sin confirmar."),
    ] = False,
    verbose: Annotated[
        bool,
        typer.Option("--verbose", help="Salida detallada."),
    ] = False,
) -> None:
    """Restaura desde un archivo, solo las apps instaladas."""
    try:
        selected: list[str] | None = None
        if not yes and sys.stdin.isatty():
            manifest = read_manifest(backup_file)
            catalog = load_catalog()
            installed_ids = {app_id for app_id, app in catalog.items() if is_installed(app)}
            restorable = [
                app for app in manifest.apps if app.id in installed_ids or app.id == SNAPSHOT_APP_ID
            ]
            chosen = ui.select_apps([(app.id, app.name) for app in restorable])
            if chosen is None:
                typer.echo("restauración cancelada", err=True)
                raise typer.Exit(code=1)
            selected = chosen
        elif not yes and not dry_run:
            # Sin --yes y sin terminal (y no es una simulación): no se puede confirmar.
            typer.echo("se requiere --yes para restaurar sin terminal interactiva", err=True)
            raise typer.Exit(code=1)
        result = run_restore(backup_file, selected=selected, dry_run=dry_run)
        ui.print_restore_summary(result, verbose=verbose)
        raise typer.Exit(code=3 if result.omitted else 0)
    except KaminoError as exc:
        typer.echo(str(exc), err=True)
        raise typer.Exit(code=1) from None


if __name__ == "__main__":
    app()
