"""Command-line interface: comandos y opciones, capa fina sobre los casos de uso.

La lógica no vive aquí: `clone` y `deploy` delegan en los casos de uso
(`kamino.clone` y `kamino.deploy`). Solo se imprime o pregunta en `ui`.
"""

from __future__ import annotations

import sys
from pathlib import Path
from typing import Annotated

import typer

from kamino import __version__, ui
from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.clone import run_clone
from kamino.conflicts import apply_all
from kamino.deploy import DeploySession, prepare_deploy
from kamino.detect import is_installed
from kamino.errors import KaminoError
from kamino.tui.app import run_deploy_tui
from kamino.update import run_update

app = typer.Typer(
    name="kamino",
    help="Clona y despliega configuraciones de aplicaciones de macOS.",
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
    """kamino: clonación y despliegue de configuraciones de macOS."""
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
            ("clone", "Clon", "Crea un clon de las apps elegidas."),
            ("deploy", "Despliegue", "Despliega desde un archivo del clon."),
            ("update", "Actualización", "Actualiza a la última versión publicada."),
        ]
    )
    if command is None:
        typer.echo("cancelado", err=True)
        raise typer.Exit(code=1)
    if command == "clone":
        ctx.invoke(clone)
        return
    if command == "update":
        ctx.invoke(update)
        return
    path = ui.ask_path("Archivo del clon (.tar.gz):")
    if path is None or not path.strip():
        typer.echo("cancelado", err=True)
        raise typer.Exit(code=1)
    ctx.invoke(deploy, clone_file=Path(path.strip()))


@app.command()
def clone(
    apps: Annotated[
        str | None,
        typer.Option(
            "--apps",
            "-a",
            help="Apps a clonar, separadas por comas (modo no interactivo).",
        ),
    ] = None,
    output: Annotated[
        str | None,
        typer.Option("--output", "-o", help="Ruta del clon."),
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
    """Crea un clon con las apps elegidas."""
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
            # Sin terminal no hay selección: clonar todas las instaladas
            # (el clon no es destructivo).
            selected = [app.id for app in installed]
        else:
            chosen = ui.select_apps([(app.id, app.name) for app in installed])
            if chosen is None:
                typer.echo("clonación cancelada", err=True)
                raise typer.Exit(code=1)
            selected = chosen
        for rid in missing:
            typer.echo(f"aviso: app no instalada o desconocida: {rid}", err=True)
        result = run_clone(
            selected=selected,
            output=Path(output) if output else None,
            dry_run=dry_run,
            catalog=catalog,
        )
        ui.print_clone_summary(result, verbose=verbose)
        raise typer.Exit(code=3 if (result.plan.skipped or missing) else 0)
    except KaminoError as exc:
        typer.echo(str(exc), err=True)
        raise typer.Exit(code=1) from None


@app.command()
def deploy(
    clone_file: Annotated[
        Path,
        typer.Argument(help="Archivo del clon (.tar.gz)."),
    ],
    apps: Annotated[
        str | None,
        typer.Option(
            "--apps",
            "-a",
            help="Apps a desplegar, separadas por comas (omite la selección interactiva).",
        ),
    ] = None,
    dry_run: Annotated[
        bool,
        typer.Option("--dry-run", "-n", help="Simula sin escribir nada."),
    ] = False,
    yes: Annotated[
        bool,
        typer.Option("--yes", "-y", help="Aplica todas las apps desplegables sin confirmar."),
    ] = False,
    verbose: Annotated[
        bool,
        typer.Option("--verbose", "-v", help="Salida detallada."),
    ] = False,
) -> None:
    """Despliega desde un archivo, solo las apps instaladas."""
    try:
        requested = _parse_apps(apps)
        if not yes and not dry_run and not sys.stdin.isatty():
            typer.echo("se requiere --yes para desplegar sin terminal interactiva", err=True)
            raise typer.Exit(code=1)
        session = prepare_deploy(clone_file)
        try:
            selected, missing = _resolve_selection(requested, session)
            for app_id in missing:
                typer.echo(f"aviso: app no desplegable o desconocida: {app_id}", err=True)
            if dry_run:
                plan = session.plan(selected, dry_run=True)
                result = session.dry_run_result(plan)
                ui.print_deploy_summary(result, verbose=verbose)
                raise typer.Exit(code=3 if (result.omitted or missing) else 0)
            if yes:
                plan = session.plan(selected)
                result = session.apply(apply_all(plan.conflicts))
                ui.print_deploy_summary(result, verbose=verbose)
                raise typer.Exit(code=3 if (result.omitted or missing) else 0)
            result = run_deploy_tui(session, selected)
            if result is None:
                typer.echo("despliegue cancelado", err=True)
                raise typer.Exit(code=1)
            ui.print_deploy_summary(result, verbose=verbose)
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
    requested: list[str] | None, session: DeploySession
) -> tuple[list[str] | None, list[str]]:
    """Keep requested apps that are deployable and report the rest."""
    if requested is None:
        return None, []
    deployable = {app.id for app in session.deployable_apps}
    selected = [app_id for app_id in requested if app_id in deployable]
    missing = [app_id for app_id in requested if app_id not in deployable]
    return selected, missing


if __name__ == "__main__":
    app()
