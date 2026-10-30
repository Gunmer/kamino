"""Caso de uso de respaldo: construir el plan y ejecutarlo.

Flujo (diseño técnico §5.1): cargar catálogo, detectar apps instaladas, cruzar
la selección, expandir paths aplicando exclusiones, construir el plan y
escribir el archivo en un temporal para renombrarlo al nombre final.

Módulo sin interfaz (§1): no imprime ni pregunta. Todas las dependencias se
inyectan (home, catálogo, detección y metadatos) para que las pruebas usen un
HOME falso y no toquen el sistema real (§7).
"""

from __future__ import annotations

import platform
import re
from collections.abc import Sequence
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

import kamino
from kamino.archive import FORMAT_VERSION, AppEntry, FileEntry, Manifest, write_archive
from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.detect import detect_version, is_installed
from kamino.errors import ArchiveError
from kamino.plan import Plan, build_plan

_DATE_PREFIX = re.compile(r"\d{4}-\d{2}-\d{2}")
_MAX_SUFFIX = 1000


@dataclass(frozen=True)
class BackupResult:
    """Resultado del respaldo: plan, manifiesto, archivo escrito y modo.

    Con `dry_run` el plan y el manifiesto se construyen igual pero no se
    escribe nada y `output` es `None` (§1: planificar y luego ejecutar).
    """

    plan: Plan
    manifest: Manifest
    output: Path | None
    dry_run: bool


def run_backup(
    *,
    home: Path | None = None,
    selected: list[str] | None = None,
    output: Path | None = None,
    dry_run: bool = False,
    catalog: dict[str, AppDefinition] | None = None,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
    metadata: dict[str, str] | None = None,
) -> BackupResult:
    """Ejecuta el flujo completo de respaldo y devuelve el resultado tipado.

    - `home`: home falso para las pruebas; por defecto el del usuario.
    - `selected`: ids a respaldar; `None` usa todas las instaladas. Los ids
      desconocidos o no instalados se ignoran (cruce de conjuntos).
    - `output`: archivo destino; por defecto `kamino-<hostname>-<fecha>.tar.gz`
      en el directorio actual, añadiendo un sufijo numérico si ya existe para
      no sobrescribir nunca (§6).
    - `dry_run`: construye plan y manifiesto sin escribir el archivo.
    - `catalog`: catálogo inyectado; por defecto se carga con `load_catalog`.
    - `application_dirs` y `path`: se pasan a la detección (`is_installed` y
      `detect_version`) para no tocar las ubicaciones reales en las pruebas.
    - `metadata`: sobrescribe `macos_version`, `arch`, `hostname` o
      `created_at`; por defecto salen de `platform` y de la hora actual.
    """
    root = home if home is not None else Path.home()
    definitions = (
        catalog
        if catalog is not None
        else load_catalog(user_dir=root / ".config" / "kamino" / "apps")
    )

    apps = _select_installed(definitions, selected, application_dirs, path)
    plan = build_plan(apps, home=root)
    manifest = _build_manifest(apps, plan, application_dirs, metadata)

    if dry_run:
        return BackupResult(plan=plan, manifest=manifest, output=None, dry_run=True)

    target = output if output is not None else _default_output(manifest)
    write_archive(target, manifest, source_root=root)
    return BackupResult(plan=plan, manifest=manifest, output=target, dry_run=False)


def _select_installed(
    definitions: dict[str, AppDefinition],
    selected: list[str] | None,
    application_dirs: Sequence[Path] | None,
    path: str | None,
) -> list[AppDefinition]:
    """Cruza el catálogo con la selección y las apps instaladas, por orden de id."""
    wanted = set(selected) if selected is not None else None
    apps: list[AppDefinition] = []
    for app_id in sorted(definitions):
        app = definitions[app_id]
        if wanted is not None and app.id not in wanted:
            continue
        if is_installed(app, application_dirs=application_dirs, path=path):
            apps.append(app)
    return apps


def _build_manifest(
    apps: Sequence[AppDefinition],
    plan: Plan,
    application_dirs: Sequence[Path] | None,
    metadata: dict[str, str] | None,
) -> Manifest:
    """Monta el manifiesto del respaldo (§4.2) a partir del plan ya calculado."""
    meta = metadata or {}
    files_by_app: dict[str, list[FileEntry]] = {}
    for operation in plan.operations:
        files_by_app.setdefault(operation.app_id, []).append(
            FileEntry(
                path=operation.path,
                sha256=operation.sha256,
                size=operation.size,
                mode=operation.mode,
                link=operation.link,
            )
        )

    entries = [
        AppEntry(
            id=app.id,
            name=app.name,
            bundle_id=app.bundle_id,
            version=detect_version(app, application_dirs=application_dirs),
            files=files_by_app.get(app.id, []),
        )
        for app in apps
    ]
    return Manifest(
        format_version=FORMAT_VERSION,
        tool_version=kamino.__version__,
        created_at=meta["created_at"] if "created_at" in meta else _now_iso(),
        macos_version=meta["macos_version"] if "macos_version" in meta else platform.mac_ver()[0],
        arch=meta["arch"] if "arch" in meta else platform.machine(),
        hostname=meta["hostname"] if "hostname" in meta else platform.node(),
        apps=entries,
    )


def _now_iso() -> str:
    """Hora actual en ISO 8601 UTC, con sufijo `Z` (§4.2)."""
    return datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def _default_output(manifest: Manifest) -> Path:
    """`kamino-<hostname>-<fecha>.tar.gz` en el cwd, con sufijo si ya existe (§6)."""
    date = _DATE_PREFIX.search(manifest.created_at)
    day = date.group() if date else datetime.now(UTC).strftime("%Y-%m-%d")
    base = f"kamino-{manifest.hostname}-{day}"
    for index in range(_MAX_SUFFIX + 1):
        name = f"{base}.tar.gz" if index == 0 else f"{base}-{index}.tar.gz"
        candidate = Path.cwd() / name
        if not candidate.exists():
            return candidate
    raise ArchiveError(f"no se pudo elegir un nombre libre para {base}.tar.gz")
