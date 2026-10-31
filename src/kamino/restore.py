"""Caso de uso de restauración: construir el plan y ejecutarlo.

Flujo del diseño técnico §5.2 sin interfaz: leer el manifiesto, cruzar las apps
con las instaladas, clasificar cada archivo, verificar el archivo antes de tocar
nada, crear el snapshot previo (§6) y aplicar con reemplazo atómico conservando
el modo. No imprime ni pregunta (§1) y todo se inyecta (home, catálogo, detección
y snapshots_dir) para que las pruebas usen un HOME falso y no toquen el sistema
real (§7).

Clasificación de cada archivo del manifiesto, comparando su destino con la entrada:

- `NEW`: el destino no existe; se crea.
- `IDENTICAL`: existe y coincide (sha256 para archivos regulares, destino del
  enlace para enlaces simbólicos); se omite, no se restaura.
- `DIFFERENT`: existe y difiere; se sobrescribe. Un destino regular donde el
  manifiesto declara un enlace (o viceversa) también es `DIFFERENT`.
- `UNSUPPORTED`: el destino existe con un tipo que no se puede reemplazar de forma
  segura (directorio, socket, tubería...); se omite.
"""

from __future__ import annotations

import hashlib
import os
import shutil
import stat
import tempfile
from collections.abc import Sequence
from dataclasses import dataclass, field
from enum import StrEnum
from pathlib import Path
from uuid import uuid4

from kamino.archive import (
    APPS_DIR,
    AppEntry,
    FileEntry,
    Manifest,
    extract_archive,
    read_manifest,
    verify_archive,
)
from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.detect import is_installed
from kamino.paths import to_absolute
from kamino.snapshot import SNAPSHOT_APP_ID, create_snapshot


class RestoreStatus(StrEnum):
    """Clasificación de un archivo del manifiesto frente a su destino (§5.2.4)."""

    NEW = "new"
    IDENTICAL = "identical"
    DIFFERENT = "different"
    UNSUPPORTED = "unsupported"


@dataclass(frozen=True)
class RestoreEntry:
    """Un archivo del manifiesto clasificado para la restauración."""

    app_id: str
    path: str
    destination: Path
    size: int
    sha256: str | None
    mode: str
    link: str | None
    status: RestoreStatus


@dataclass(frozen=True)
class RestorePlan:
    """El plan de restauración: cada archivo activo del manifiesto, clasificado.

    Las apps que no se restauran (no instaladas o no seleccionadas) no aparecen
    aquí; sus rutas van al resumen de `RestoreResult.omitted`.
    """

    entries: list[RestoreEntry] = field(default_factory=list)


@dataclass(frozen=True)
class RestoreResult:
    """Resultado de la restauración (§5.2.9).

    - `plan`: los archivos del manifiesto activos, clasificados.
    - `manifest`: el manifiesto del respaldo.
    - `output`: el archivo de respaldo restaurado; `None` con `dry_run`, que no
      escribe nada (misma convención que `BackupResult`).
    - `snapshot`: la copia previa creada antes de aplicar; `None` con `dry_run`
      o cuando nada va a cambiar.
    - `restored`: rutas relativas POSIX restauradas; vacío con `dry_run`.
    - `omitted`: rutas relativas POSIX que no se restauran (apps no instaladas,
      destinos idénticos y destinos de tipo no soportado). Se deciden al
      planificar, así que valen igual en `dry_run`.
    """

    plan: RestorePlan
    manifest: Manifest
    output: Path | None
    dry_run: bool
    snapshot: Path | None
    restored: list[str]
    omitted: list[str]


def run_restore(
    backup_file: Path,
    *,
    home: Path | None = None,
    selected: list[str] | None = None,
    dry_run: bool = False,
    catalog: dict[str, AppDefinition] | None = None,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
    snapshots_dir: Path | None = None,
) -> RestoreResult:
    """Ejecuta el flujo completo de restauración y devuelve el resultado tipado.

    - `backup_file`: archivo de respaldo a restaurar.
    - `home`: home falso para las pruebas; por defecto el del usuario.
    - `selected`: ids a aplicar; `None` usa todas las apps del manifiesto que
      estén instaladas. Las apps no instaladas van a `omitted` con sus archivos,
      estén o no seleccionadas (§5.2.2); las instaladas pero no seleccionadas se
      ignoran.
    - `dry_run`: construye el plan clasificado sin verificar, extraer,
      snapshotar ni escribir nada (§5.2.5).
    - `catalog`: catálogo con el que detectar las apps instaladas; por defecto se
      carga con `load_catalog`. La app sintética `snapshot` siempre se considera
      instalada, porque un snapshot es el estado previo de esta misma máquina
      (§6: el futuro `undo` es un `restore` del snapshot).
    - `application_dirs` y `path`: se pasan a `is_installed` para no tocar las
      ubicaciones reales en las pruebas.
    - `snapshots_dir`: destino del snapshot previo; por defecto
      `~/Library/Application Support/kamino/snapshots/`.
    """
    root = home if home is not None else Path.home()
    definitions = (
        catalog
        if catalog is not None
        else load_catalog(user_dir=root / ".config" / "kamino" / "apps")
    )
    manifest = read_manifest(backup_file)  # valida format_version (§5.2.1)
    plan, omitted = _build_plan(manifest, definitions, selected, root, application_dirs, path)

    if dry_run:
        return RestoreResult(
            plan=plan,
            manifest=manifest,
            output=None,
            dry_run=True,
            snapshot=None,
            restored=[],
            omitted=omitted,
        )

    verify_archive(backup_file, manifest)  # aborta antes de tocar nada (§5.2.6)
    changing = [
        entry
        for entry in plan.entries
        if entry.status in (RestoreStatus.NEW, RestoreStatus.DIFFERENT)
    ]
    with tempfile.TemporaryDirectory(prefix="kamino-restore-") as tmp:
        staging = Path(tmp)
        extract_archive(backup_file, staging)  # staging verificado (§5.2.6)
        snapshot = create_snapshot(
            [entry.destination for entry in changing], home=root, snapshots_dir=snapshots_dir
        )
        restored = _apply(staging, changing)
    return RestoreResult(
        plan=plan,
        manifest=manifest,
        output=backup_file,
        dry_run=False,
        snapshot=snapshot,
        restored=restored,
        omitted=omitted,
    )


def _build_plan(
    manifest: Manifest,
    definitions: dict[str, AppDefinition],
    selected: list[str] | None,
    root: Path,
    application_dirs: Sequence[Path] | None,
    path: str | None,
) -> tuple[RestorePlan, list[str]]:
    """Cruza apps instaladas y selección, y clasifica cada archivo (§5.2.2-4).

    Devuelve el plan y las rutas omitidas en orden del manifiesto: archivos de
    apps no instaladas, destinos idénticos y destinos de tipo no soportado.
    """
    entries: list[RestoreEntry] = []
    omitted: list[str] = []
    for app in manifest.apps:
        if not _is_app_installed(app, definitions, application_dirs, path):
            omitted.extend(file.path for file in app.files)
            continue
        if selected is not None and app.id not in selected:
            continue  # instalada pero no seleccionada: se ignora
        for file in app.files:
            destination = to_absolute(file.path, home=root)
            status = _classify(destination, file)
            entries.append(
                RestoreEntry(
                    app_id=app.id,
                    path=file.path,
                    destination=destination,
                    size=file.size,
                    sha256=file.sha256,
                    mode=file.mode,
                    link=file.link,
                    status=status,
                )
            )
            if status in (RestoreStatus.IDENTICAL, RestoreStatus.UNSUPPORTED):
                omitted.append(file.path)
    return RestorePlan(entries=entries), list(dict.fromkeys(omitted))


def _is_app_installed(
    app: AppEntry,
    definitions: dict[str, AppDefinition],
    application_dirs: Sequence[Path] | None,
    path: str | None,
) -> bool:
    """Indica si la app del manifiesto está instalada en esta máquina.

    La app sintética `snapshot` siempre lo está: un snapshot es el estado previo
    de esta misma máquina. El resto se detecta con `is_installed` usando su
    definición del catálogo; sin definición no hay forma de detectarla y se trata
    como no instalada.
    """
    if app.id == SNAPSHOT_APP_ID:
        return True
    definition = definitions.get(app.id)
    return definition is not None and is_installed(
        definition, application_dirs=application_dirs, path=path
    )


def _classify(destination: Path, entry: FileEntry) -> RestoreStatus:
    """Compara el destino actual con la entrada del manifiesto y lo clasifica.

    Los enlaces se comparan sin seguirlos (`os.lstat` y `os.readlink`): un
    destino regular donde el manifiesto declara un enlace (o viceversa) es
    `DIFFERENT`, y un destino que ni es regular ni es enlace es `UNSUPPORTED`.
    """
    try:
        info = os.lstat(destination)
    except FileNotFoundError:
        return RestoreStatus.NEW
    if stat.S_ISLNK(info.st_mode):
        if entry.link is None:
            return RestoreStatus.DIFFERENT  # se espera un archivo regular
        return (
            RestoreStatus.IDENTICAL
            if os.readlink(destination) == entry.link
            else RestoreStatus.DIFFERENT
        )
    if stat.S_ISREG(info.st_mode):
        if entry.link is not None:
            return RestoreStatus.DIFFERENT  # se espera un enlace simbólico
        if entry.sha256 is not None and _sha256_of(destination) == entry.sha256:
            return RestoreStatus.IDENTICAL
        return RestoreStatus.DIFFERENT
    return RestoreStatus.UNSUPPORTED


def _apply(staging: Path, entries: Sequence[RestoreEntry]) -> list[str]:
    """Escribe cada destino en su ruta final con reemplazo atómico (§5.2.8).

    Los archivos regulares conservan su modo con `os.chmod` y los enlaces
    simbólicos se recrean. Si dos apps del manifiesto comparten la misma ruta,
    solo se escribe una vez.
    """
    restored: list[str] = []
    done: set[Path] = set()
    for entry in entries:
        if entry.destination in done:
            continue
        done.add(entry.destination)
        source = staging / APPS_DIR / entry.app_id / entry.path
        entry.destination.parent.mkdir(parents=True, exist_ok=True)
        if entry.link is not None:
            _replace_link(entry.destination, entry.link)
        else:
            _replace_file(entry.destination, source, entry.mode)
        restored.append(entry.path)
    return restored


def _replace_file(destination: Path, source: Path, mode: str) -> None:
    """Copia el archivo extraído en un temporal del destino y lo renombra (§6)."""
    tmp = destination.with_name(f".{destination.name}.{uuid4().hex}.tmp")
    try:
        shutil.copyfile(source, tmp)
        os.chmod(tmp, int(mode, 8))
        os.replace(tmp, destination)
    except Exception:
        tmp.unlink(missing_ok=True)
        raise


def _replace_link(destination: Path, link: str) -> None:
    """Recrea el enlace simbólico en un temporal del destino y lo renombra (§6)."""
    tmp = destination.with_name(f".{destination.name}.{uuid4().hex}.tmp")
    try:
        os.symlink(link, tmp)
        os.replace(tmp, destination)
    except Exception:
        tmp.unlink(missing_ok=True)
        raise


def _sha256_of(source: Path) -> str:
    digest = hashlib.sha256()
    with source.open("rb") as handle:
        for chunk in iter(lambda: handle.read(65536), b""):
            digest.update(chunk)
    return digest.hexdigest()
