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
from kamino.backup import run_backup
from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.conflicts import apply_all
from kamino.detect import is_installed
from kamino.errors import KaminoError
from kamino.restore import RestoreSession, prepare_restore
from kamino.tui.app import run_restore_tui
from kamino.update import run_update

app = typer.Typer(
    name="kamino",
    help="Respalda y restaura configuraciones de aplicaciones de macOS.",
    no_args_is_help=False,
    invoke_without_command=True,
    context_settings={"help_option_names": ["-h", "--help"]},
)


def _version_callback(value: bool) -> None:
    if value:
        typer.echo(f"kamino {__version__}")
        raise typer.Exit()


@app.callback()
def main(
    ctx: typer.Context,
    version: Annotated[
        bool | None,
        typer.Option(
            "--version",
            "-V",
            help="Muestra la versión y termina.",
            callback=_version_callback,
        ),
    ] = None,
) -> None:
    """kamino: respaldo y restauración de configuraciones de macOS."""
    if ctx.invoked_subcommand is not None:
        return
    _handle_no_command(ctx)


def _installed_apps(catalog: dict[str, AppDefinition]) -> list[AppDefinition]:
    return [app for app in catalog.values() if is_installed(app)]


def _handle_no_command(ctx: typer.Context) -> None:
    """Sin subcomando: TUI si hay terminal, ayuda si no."""
    if not sys.stdin.isatty():
        # Sin terminal no hay menú: mostrar la ayuda como antes.
        typer.echo(ctx.get_help(), nl=False)
        raise typer.Exit()
    _run_menu(ctx)


def _run_menu(ctx: typer.Context) -> None:
    """TUI: lista de comandos disponibles cuando se ejecuta sin argumentos."""
    command = ui.select_command(
        [
            ("backup", "Respaldo", "Crea un respaldo de las apps elegidas."),
            ("restore", "Restauración", "Restaura desde un archivo de respaldo."),
            ("update", "Actualización", "Actualiza a la última versión publicada."),
        ]
    )
    if command is None:
        typer.echo("cancelado", err=True)
        raise typer.Exit(code=1)
    if command == "backup":
        ctx.invoke(backup)
        return
    if command == "update":
        ctx.invoke(update)
        return
    path = ui.ask_path("Archivo de respaldo (.tar.gz):")
    if path is None or not path.strip():
        typer.echo("cancelado", err=True)
        raise typer.Exit(code=1)
    ctx.invoke(restore, backup_file=Path(path.strip()))


@app.command()
def backup(
    apps: Annotated[
        str | None,
        typer.Option(
            "--apps",
            "-a",
            help="Apps a respaldar, separadas por comas (modo no interactivo).",
        ),
    ] = None,
    output: Annotated[
        str | None,
        typer.Option("--output", "-o", help="Ruta del archivo de respaldo."),
    ] = None,
    dry_run: Annotated[
        bool,
        typer.Option("--dry-run", "-n", help="Simula sin escribir nada."),
    ] = False,
    verbose: Annotated[
        bool,
        typer.Option("--verbose", "-v", help="Salida detallada."),
    ] = False,
) -> None:
    """Crea un archivo de respaldo con las apps elegidas."""
    ui.print_privacy_warning()
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
    apps: Annotated[
        str | None,
        typer.Option(
            "--apps",
            "-a",
            help="Apps a restaurar, separadas por comas (omite la selección interactiva).",
        ),
    ] = None,
    dry_run: Annotated[
        bool,
        typer.Option("--dry-run", "-n", help="Simula sin escribir nada."),
    ] = False,
    yes: Annotated[
        bool,
        typer.Option("--yes", "-y", help="Aplica todas las apps restaurables sin confirmar."),
    ] = False,
    verbose: Annotated[
        bool,
        typer.Option("--verbose", "-v", help="Salida detallada."),
    ] = False,
) -> None:
    """Restaura desde un archivo, solo las apps instaladas."""
    try:
        requested = _parse_apps(apps)
        if not yes and not dry_run and not sys.stdin.isatty():
            typer.echo("se requiere --yes para restaurar sin terminal interactiva", err=True)
            raise typer.Exit(code=1)
        session = prepare_restore(backup_file)
        try:
            selected, missing = _resolve_selection(requested, session)
            for app_id in missing:
                typer.echo(f"aviso: app no restaurable o desconocida: {app_id}", err=True)
            if dry_run:
                plan = session.plan(selected, dry_run=True)
                result = session.dry_run_result(plan)
                ui.print_restore_summary(result, verbose=verbose)
                raise typer.Exit(code=3 if (result.omitted or missing) else 0)
            if yes:
                plan = session.plan(selected)
                result = session.apply(apply_all(plan.conflicts))
                ui.print_restore_summary(result, verbose=verbose)
                raise typer.Exit(code=3 if (result.omitted or missing) else 0)
            result = run_restore_tui(session, selected)
            if result is None:
                typer.echo("restauración cancelada", err=True)
                raise typer.Exit(code=1)
            ui.print_restore_summary(result, verbose=verbose)
            raise typer.Exit(code=3 if (result.omitted or missing) else 0)
        finally:
            session.close()
    except KaminoError as exc:
        typer.echo(str(exc), err=True)
        raise typer.Exit(code=1) from None


@app.command()
def update() -> None:
    """Actualiza kamino a la última versión publicada."""
    ui.print_update_start()
    try:
        result = run_update(__version__)
    except KaminoError as exc:
        typer.echo(str(exc), err=True)
        raise typer.Exit(code=1) from None
    ui.print_update_result(result)


def _parse_apps(value: str | None) -> list[str] | None:
    """Split a comma-separated ``--apps`` value, preserving order and uniqueness."""
    if value is None:
        return None
    return list(dict.fromkeys(part.strip() for part in value.split(",") if part.strip()))


def _resolve_selection(
    requested: list[str] | None, session: RestoreSession
) -> tuple[list[str] | None, list[str]]:
    """Keep requested apps that are restorable and report the rest."""
    if requested is None:
        return None, []
    restorable = {app.id for app in session.restorable_apps}
    selected = [app_id for app_id in requested if app_id in restorable]
    missing = [app_id for app_id in requested if app_id not in restorable]
    return selected, missing


if __name__ == "__main__":
    app()
