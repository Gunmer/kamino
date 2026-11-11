"""Command-line interface: argumentos, salida y códigos de terminación.

La lógica no vive aquí: los flujos de texto están en `commands.py`, los casos
de uso en `kamino.clone`, `kamino.deploy` y `kamino.update`, y la interfaz
interactiva en `kamino.tui`. `cli.py` traduce resultados a `typer.Exit`.
"""

from __future__ import annotations

import os
import sys
from pathlib import Path
from typing import Annotated

import typer

from kamino import __version__, texts, ui
from kamino.clone import validate_destination
from kamino.commands import execute_clone, execute_deploy, execute_update
from kamino.errors import KaminoError, PartialApplyError
from kamino.tui.app import run_app
from kamino.tui.clone import run_clone_flow
from kamino.tui.deploy import run_deploy_flow
from kamino.tui.update import run_update_flow

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


def _interactive() -> bool:
    """Hay terminal en entrada y salida y no se pidió la vía de texto."""
    if os.environ.get("KAMINO_PLAIN"):
        return False
    return bool(sys.stdin.isatty() and sys.stdout.isatty())


def _handle_no_command(ctx: typer.Context) -> None:
    """Sin subcomando: menú si hay terminal; ayuda si no."""
    if not _interactive():
        typer.echo(ctx.get_help(), nl=False)
        raise typer.Exit()
    code = _run_menu()
    raise typer.Exit(code=code)


def _run_menu() -> int:
    """Abre la app única en modo menú."""
    return run_app(mode="menu")


@app.command()
def clone(
    apps: Annotated[
        str | None,
        typer.Option(
            "--apps",
            "-a",
            help="Apps a clonar, separadas por comas (omite C1 en la TUI).",
        ),
    ] = None,
    output: Annotated[
        str | None,
        typer.Option("--output", "-o", help="Ruta del clon (omite C2 en la TUI)."),
    ] = None,
    dry_run: Annotated[
        bool,
        typer.Option("--dry-run", "-n", help="Simula sin escribir nada."),
    ] = False,
    verbose: Annotated[
        bool,
        typer.Option("--verbose", "-v", help="Salida detallada."),
    ] = False,
    force: Annotated[
        bool,
        typer.Option(
            "--force",
            "-f",
            help=texts.CLONE_FORCE_HELP,
        ),
    ] = False,
) -> None:
    """Crea un clon con las apps elegidas."""
    requested = _parse_apps(apps)
    try:
        if _interactive():
            code = _clone_tui(
                requested=requested,
                output=Path(output) if output else None,
                dry_run=dry_run,
                verbose=verbose,
                force=force,
            )
            raise typer.Exit(code=code)
        ui.print_privacy_warning()
        outcome = execute_clone(
            requested=requested,
            output=Path(output) if output else None,
            dry_run=dry_run,
            force=force,
        )
        for app_id in outcome.missing:
            typer.echo(f"aviso: app no instalada o desconocida: {app_id}", err=True)
        ui.print_clone_summary(outcome.result, verbose=verbose)
        raise typer.Exit(code=outcome.code)
    except KaminoError as exc:
        typer.echo(str(exc), err=True)
        raise typer.Exit(code=1) from None


def _clone_tui(
    *,
    requested: list[str] | None,
    output: Path | None,
    dry_run: bool,
    verbose: bool,
    force: bool,
) -> int:
    """Valida la ruta y abre el flujo directo de clonación."""
    output_path: Path | None = None
    output_error: str | None = None
    if output is not None:
        output_path = output.expanduser().resolve()
        output_error = validate_destination(output_path)
    return run_clone_flow(
        requested=requested,
        output=output_path,
        force=force,
        dry_run=dry_run,
        verbose=verbose,
        output_error=output_error,
    )


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
            help="Apps a desplegar, separadas por comas (omite D1 en la TUI).",
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
    requested = _parse_apps(apps)
    interactive = _interactive()
    try:
        if interactive and not dry_run and not yes:
            code = _deploy_tui(clone_file, requested=requested, verbose=verbose)
            raise typer.Exit(code=code)
        if not yes and not dry_run and not interactive:
            typer.echo(texts.REQUIRES_YES, err=True)
            raise typer.Exit(code=1)
        outcome = execute_deploy(
            clone_file,
            requested=requested,
            dry_run=dry_run,
        )
        for app_id in outcome.missing:
            typer.echo(f"aviso: app no desplegable o desconocida: {app_id}", err=True)
        ui.print_deploy_summary(outcome.result, verbose=verbose)
        raise typer.Exit(code=outcome.code)
    except PartialApplyError as exc:
        typer.echo(
            texts.DEPLOY_PARTIAL.format(applied=len(exc.applied), failed=exc.failed),
            err=True,
        )
        if exc.snapshot is not None:
            typer.echo(texts.DEPLOY_PARTIAL_SNAPSHOT.format(path=exc.snapshot), err=True)
        raise typer.Exit(code=1) from None
    except KaminoError as exc:
        typer.echo(str(exc), err=True)
        raise typer.Exit(code=1) from None


def _deploy_tui(
    clone_file: Path,
    *,
    requested: list[str] | None,
    verbose: bool,
) -> int:
    """Valida el archivo y abre el flujo directo de despliegue."""
    if not clone_file.is_file():
        return run_deploy_flow(
            clone_file=clone_file,
            requested=requested,
            verbose=verbose,
            file_error=f"no existe el archivo: {clone_file}",
        )
    return run_deploy_flow(
        clone_file=clone_file,
        requested=requested,
        verbose=verbose,
    )


@app.command()
def update() -> None:
    """Actualiza kamino a la última versión publicada."""
    if _interactive():
        raise typer.Exit(code=run_update_flow())
    ui.print_update_start()
    try:
        result = execute_update(__version__)
    except KaminoError as exc:
        typer.echo(str(exc), err=True)
        raise typer.Exit(code=1) from None
    ui.print_update_result(result)


def _parse_apps(value: str | None) -> list[str] | None:
    """Divide y deduplica un valor de `--apps`, conservando el orden."""
    if value is None:
        return None
    return list(dict.fromkeys(part.strip() for part in value.split(",") if part.strip()))


if __name__ == "__main__":
    app()
