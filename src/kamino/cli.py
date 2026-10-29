"""Command-line interface: comandos y opciones, capa fina sobre los casos de uso.

La lógica no vive aquí: `backup` y `restore` delegarán en los módulos de caso de uso
(`kamino.backup` y `kamino.restore`) cuando se implementen en los pasos 5 y 6.
"""

from __future__ import annotations

from pathlib import Path
from typing import Annotated

import typer

from kamino import __version__

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
    typer.echo("backup: aún no implementado (esqueleto del proyecto).", err=True)
    raise typer.Exit(code=1)


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
    verbose: Annotated[
        bool,
        typer.Option("--verbose", help="Salida detallada."),
    ] = False,
) -> None:
    """Restaura desde un archivo, solo las apps instaladas."""
    typer.echo("restore: aún no implementado (esqueleto del proyecto).", err=True)
    raise typer.Exit(code=1)


if __name__ == "__main__":
    app()
