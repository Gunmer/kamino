"""Flujos de comando compartidos por la CLI y el menú, sin Typer ni salida.

Devuelven resultados tipados y un código de salida. La presentación queda en
`ui.py` y la traducción a `typer.Exit`, en `cli.py`.
"""

from __future__ import annotations

from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path

from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.clone import CloneResult, run_clone
from kamino.conflicts import apply_all
from kamino.defaults import catalog_dir as default_catalog_dir
from kamino.deploy import DeployResult, DeploySession, prepare_deploy, result_code
from kamino.detect import is_installed
from kamino.update import UpdateResult, run_update


@dataclass(frozen=True)
class CloneOutcome:
    result: CloneResult
    code: int
    missing: list[str]


@dataclass(frozen=True)
class DeployOutcome:
    result: DeployResult
    code: int
    missing: list[str]


def installed_apps(
    catalog: dict[str, AppDefinition],
    *,
    home: Path | None = None,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
) -> list[AppDefinition]:
    """Apps del catálogo instaladas en esta máquina, por id."""
    return [
        catalog[app_id]
        for app_id in sorted(catalog)
        if is_installed(
            catalog[app_id],
            home=home,
            application_dirs=application_dirs,
            path=path,
        )
    ]


def execute_clone(
    *,
    requested: list[str] | None,
    output: Path | None,
    dry_run: bool,
    force: bool,
    catalog: dict[str, AppDefinition] | None = None,
    home: Path | None = None,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
) -> CloneOutcome:
    """Clonación de texto: selecciona, planifica y escribe sin interfaz."""
    definitions = (
        catalog
        if catalog is not None
        else load_catalog(user_dir=default_catalog_dir(home or Path.home()))
    )
    installed = installed_apps(definitions, home=home, application_dirs=application_dirs, path=path)
    if requested is None:
        selected = [app.id for app in installed]
        missing: list[str] = []
    else:
        available = {app.id for app in installed}
        selected = [app_id for app_id in requested if app_id in available]
        missing = [app_id for app_id in requested if app_id not in available]
    result = run_clone(
        selected=selected,
        output=output,
        dry_run=dry_run,
        force=force,
        catalog=definitions,
        home=home,
        application_dirs=application_dirs,
        path=path,
    )
    code = 3 if (result.plan.skipped or missing) else 0
    return CloneOutcome(result=result, code=code, missing=missing)


def execute_deploy(
    clone_file: Path,
    *,
    requested: list[str] | None,
    dry_run: bool,
    catalog: dict[str, AppDefinition] | None = None,
    home: Path | None = None,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
    snapshots_dir: Path | None = None,
) -> DeployOutcome:
    """Despliegue de texto: conflictos aplicados al clon y resumen tipado."""
    session = prepare_deploy(
        clone_file,
        home=home,
        catalog=catalog,
        application_dirs=application_dirs,
        path=path,
        snapshots_dir=snapshots_dir,
    )
    try:
        deployable = {app.id for app in session.deployable_apps}
        selected = (
            None if requested is None else [app_id for app_id in requested if app_id in deployable]
        )
        missing = (
            []
            if requested is None
            else [app_id for app_id in requested if app_id not in deployable]
        )
        plan = session.plan(
            selected,
            dry_run=dry_run,
            requested=requested,
        )
        if dry_run:
            result = session.dry_run_result(plan)
        else:
            result = session.apply(apply_all(plan.conflicts))
        return DeployOutcome(result=result, code=result_code(result), missing=missing)
    finally:
        session.close()


def execute_update(
    current_version: str,
    *,
    target: Path | None = None,
) -> UpdateResult:
    """Actualización de texto."""
    return run_update(current_version, target=target)


def deployable_ids(session: DeploySession) -> set[str]:
    """Ids que el manifiesto permite desplegar en esta máquina."""
    return {app.id for app in session.deployable_apps}
