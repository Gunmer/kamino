"""Caso de uso de clonación: planificar y escribir son pasos separados.

`plan_clone` construye el plan, el manifiesto y la ruta de destino sin escribir
nada. `write_clone` escribe el temporal con progreso y cancelación cooperativa y
solo entonces lo renombra al destino, de forma atómica. `run_clone` es la
fachada que conserva la vía de la CLI y las pruebas.

Módulo sin interfaz (§1): no imprime ni pregunta. Todas las dependencias se
inyectan (home, catálogo, detección, metadatos y reloj) para que las pruebas
usen un HOME falso y no toquen el sistema real (§7).
"""

from __future__ import annotations

import platform
from collections.abc import Sequence
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path

import kamino
from kamino.archive import FORMAT_VERSION, AppEntry, FileEntry, Manifest, write_archive
from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.detect import detect_version, is_installed
from kamino.errors import ArchiveError
from kamino.plan import Plan, build_plan
from kamino.progress import CancelCallback, ProgressCallback


@dataclass(frozen=True)
class ClonePlan:
    """Plan completo de una clonación, todavía sin escribir en disco."""

    plan: Plan
    manifest: Manifest
    output: Path
    apps: list[AppDefinition] = field(default_factory=list)

    @property
    def file_count(self) -> int:
        return len(self.plan.operations)

    @property
    def size(self) -> int:
        return sum(operation.size for operation in self.plan.operations)

    @property
    def skipped(self):
        return self.plan.skipped


@dataclass(frozen=True)
class CloneResult:
    """Resultado de la clonación: plan, manifiesto, archivo escrito y modo.

    Con `dry_run` el plan y el manifiesto se construyen igual pero no se
    escribe nada y `output` es `None` (§1: planificar y luego ejecutar).
    """

    plan: Plan
    manifest: Manifest
    output: Path | None
    dry_run: bool
    plan_output: Path | None = None


def plan_clone(
    *,
    home: Path | None = None,
    selected: list[str] | None = None,
    output: Path | None = None,
    catalog: dict[str, AppDefinition] | None = None,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
    metadata: dict[str, str] | None = None,
    now: datetime | None = None,
    progress: ProgressCallback | None = None,
    cancelled: CancelCallback | None = None,
) -> ClonePlan:
    """Construye el plan, el manifiesto y la ruta de destino sin escribir.

    - `selected`: ids a clonar; `None` usa todas las instaladas. Los ids
      desconocidos o no instalados se ignoran (cruce de conjuntos).
    - `output`: archivo destino; por defecto `clone-yyyyMMdd.tar.gz` con la
      fecha local en el directorio actual.
    - `now`: reloj inyectable para la fecha del nombre por defecto.
    """
    root = home if home is not None else Path.home()
    definitions = (
        catalog
        if catalog is not None
        else load_catalog(user_dir=root / ".config" / "kamino" / "apps")
    )

    apps = _select_installed(definitions, selected, application_dirs, path, root)
    plan = build_plan(apps, home=root, progress=progress, cancelled=cancelled)
    manifest = _build_manifest(apps, plan, application_dirs, metadata, root)
    target = output if output is not None else _default_output(root, now)
    return ClonePlan(plan=plan, manifest=manifest, output=target, apps=apps)


def write_clone(
    planned: ClonePlan,
    *,
    home: Path | None = None,
    force: bool = False,
    expected_state: tuple[int, int, int] | None = None,
    progress: ProgressCallback | None = None,
    cancelled: CancelCallback | None = None,
) -> CloneResult:
    """Escribe el plan en un temporal y lo renombra al destino.

    Sin `force` nunca sobrescribe. Con `force`, `write_archive` revalida el
    destino antes del renombrado y usa `os.replace`.
    """
    root = home if home is not None else Path.home()
    write_archive(
        planned.output,
        planned.manifest,
        source_root=root,
        overwrite=force,
        expected_state=expected_state,
        progress=progress,
        cancelled=cancelled,
    )
    return CloneResult(
        plan=planned.plan,
        manifest=planned.manifest,
        output=planned.output,
        dry_run=False,
        plan_output=planned.output,
    )


def run_clone(
    *,
    home: Path | None = None,
    selected: list[str] | None = None,
    output: Path | None = None,
    dry_run: bool = False,
    catalog: dict[str, AppDefinition] | None = None,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
    metadata: dict[str, str] | None = None,
    force: bool = False,
    now: datetime | None = None,
    progress: ProgressCallback | None = None,
    cancelled: CancelCallback | None = None,
) -> CloneResult:
    """Fachada de clonación para la CLI: planifica y, si no es simulación, escribe."""
    planned = plan_clone(
        home=home,
        selected=selected,
        output=output,
        catalog=catalog,
        application_dirs=application_dirs,
        path=path,
        metadata=metadata,
        now=now,
    )
    if dry_run:
        return CloneResult(
            plan=planned.plan,
            manifest=planned.manifest,
            output=None,
            dry_run=True,
            plan_output=planned.output,
        )
    return write_clone(
        planned,
        home=home,
        force=force,
        progress=progress,
        cancelled=cancelled,
    )


def _select_installed(
    definitions: dict[str, AppDefinition],
    selected: list[str] | None,
    application_dirs: Sequence[Path] | None,
    path: str | None,
    home: Path,
) -> list[AppDefinition]:
    """Cruza el catálogo con la selección y las apps instaladas, por orden de id."""
    wanted = set(selected) if selected is not None else None
    apps: list[AppDefinition] = []
    for app_id in sorted(definitions):
        app = definitions[app_id]
        if wanted is not None and app.id not in wanted:
            continue
        if is_installed(app, home=home, application_dirs=application_dirs, path=path):
            apps.append(app)
    return apps


def _build_manifest(
    apps: Sequence[AppDefinition],
    plan: Plan,
    application_dirs: Sequence[Path] | None,
    metadata: dict[str, str] | None,
    home: Path,
) -> Manifest:
    """Monta el manifiesto del clon (§4.2) a partir del plan ya calculado."""
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
            version=detect_version(app, home=home, application_dirs=application_dirs),
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


def default_name(now: datetime | None = None) -> str:
    """Nombre estable del clon con la fecha local: `clone-yyyyMMdd.tar.gz`."""
    moment = now if now is not None else datetime.now()
    return f"clone-{moment.strftime('%Y%m%d')}.tar.gz"


def _default_output(root: Path, now: datetime | None) -> Path:
    """`clone-yyyyMMdd.tar.gz` en el cwd, sin sufijos automáticos."""
    del root  # el destino por defecto es el directorio actual, no el home
    return Path.cwd() / default_name(now)


def free_suffix(output: Path) -> Path:
    """Primer nombre libre para el mismo base: `clone-yyyyMMdd-1.tar.gz`, etc."""
    stem = output.name.removesuffix(".tar.gz")
    for index in range(1, 1001):
        candidate = output.with_name(f"{stem}-{index}.tar.gz")
        if not candidate.exists():
            return candidate
    raise ArchiveError(f"no se pudo elegir un nombre libre para {output.name}")


def validate_destination(output: Path) -> str | None:
    """Valida el destino sin crear nada; devuelve el error visible o `None`."""
    if not output.name or output.name in {".", ".."}:
        return "el nombre del clon no es válido"
    if output.name.endswith("/"):
        return "el nombre del clon no es válido"
    parent = output.parent if output.parent != Path("") else Path.cwd()
    if not parent.exists():
        return f"el directorio no existe: {parent}"
    if not parent.is_dir():
        return f"la ruta del directorio no es válida: {parent}"
    if output.exists() and output.is_dir():
        return f"el destino es un directorio: {output}"
    return None
