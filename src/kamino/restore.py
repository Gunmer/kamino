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
import stat
import tempfile
from collections.abc import Sequence
from dataclasses import dataclass, field
from enum import StrEnum
from pathlib import Path

from kamino.archive import (
    APPS_DIR,
    AppEntry,
    FileEntry,
    Manifest,
    extract_archive,
    read_file_from_archive,
    read_manifest,
    verify_archive,
)
from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.detect import is_installed
from kamino.errors import PathError
from kamino.ghostty import GHOSTTY_APP_ID, MAX_CONFIG_BYTES, absolute_references
from kamino.paths import SafeHome, to_absolute, to_relative
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
    notices: list[str] = field(default_factory=list)


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
    plan, omitted = _build_plan(
        manifest,
        definitions,
        selected,
        root,
        application_dirs,
        path,
        backup_file=backup_file,
    )

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
    root.mkdir(parents=True, exist_ok=True)
    changing = [
        entry
        for entry in plan.entries
        if entry.status in (RestoreStatus.NEW, RestoreStatus.DIFFERENT)
    ]
    _validate_destinations(changing, root)
    with tempfile.TemporaryDirectory(prefix="kamino-restore-") as tmp:
        staging = Path(tmp)
        extract_archive(backup_file, staging)  # staging verificado (§5.2.6)
        snapshot = create_snapshot(
            [entry.destination for entry in changing], home=root, snapshots_dir=snapshots_dir
        )
        restored = _apply(staging, changing, home=root)
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
    *,
    backup_file: Path,
) -> tuple[RestorePlan, list[str]]:
    """Cruza apps instaladas y selección, y clasifica cada archivo (§5.2.2-4).

    Devuelve el plan y las rutas omitidas en orden del manifiesto: archivos de
    apps no instaladas, destinos idénticos y destinos de tipo no soportado.
    Un destino con un enlace intermedio inseguro aborta antes de crear
    instantáneas o escribir nada.
    """
    entries: list[RestoreEntry] = []
    omitted: list[str] = []
    safe = SafeHome(root) if root.exists() else None
    try:
        for app in manifest.apps:
            if not _is_app_installed(app, definitions, application_dirs, path, root):
                omitted.extend(file.path for file in app.files)
                continue
            if selected is not None and app.id not in selected:
                continue  # instalada pero no seleccionada: se ignora
            for file in app.files:
                destination = to_absolute(file.path, home=root)
                status = RestoreStatus.NEW if safe is None else _classify(safe, file.path, file)
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
    finally:
        if safe is not None:
            safe.close()
    notices = _ghostty_portability_notices(manifest, backup_file, root)
    return RestorePlan(entries=entries, notices=notices), list(dict.fromkeys(omitted))


def _is_app_installed(
    app: AppEntry,
    definitions: dict[str, AppDefinition],
    application_dirs: Sequence[Path] | None,
    path: str | None,
    home: Path,
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
        definition, home=home, application_dirs=application_dirs, path=path
    )


def _classify(safe: SafeHome, rel: str, entry: FileEntry) -> RestoreStatus:
    """Compara el destino actual con la entrada del manifiesto y lo clasifica.

    Los enlaces se comparan sin seguirlos (`lstat` y `readlink`): un destino
    regular donde el manifiesto declara un enlace (o viceversa) es `DIFFERENT`,
    y un destino que ni es regular ni es enlace es `UNSUPPORTED`. Un enlace
    intermedio inseguro se rechaza con `PathError`.
    """
    try:
        info = safe.lstat(rel)
    except FileNotFoundError:
        return RestoreStatus.NEW
    if stat.S_ISLNK(info.st_mode):
        if entry.link is None:
            return RestoreStatus.DIFFERENT  # se espera un archivo regular
        return (
            RestoreStatus.IDENTICAL if safe.readlink(rel) == entry.link else RestoreStatus.DIFFERENT
        )
    if stat.S_ISREG(info.st_mode):
        if entry.link is not None:
            return RestoreStatus.DIFFERENT  # se espera un enlace simbólico
        if entry.sha256 is not None and _sha256_of(safe.read_bytes(rel)) == entry.sha256:
            return RestoreStatus.IDENTICAL
        return RestoreStatus.DIFFERENT
    return RestoreStatus.UNSUPPORTED


def _validate_destinations(entries: Sequence[RestoreEntry], home: Path) -> None:
    """Rechaza enlaces intermedios antes de crear la instantánea previa."""
    with SafeHome(home) as safe:
        for entry in entries:
            try:
                safe.lstat(entry.path)
            except FileNotFoundError:
                continue


def _apply(staging: Path, entries: Sequence[RestoreEntry], *, home: Path) -> list[str]:
    """Escribe cada destino en su ruta final con reemplazo atómico (§5.2.8).

    Los archivos regulares conservan su modo y los enlaces simbólicos se
    recrean. Los padres se crean y los destinos se reemplazan mediante
    descriptores relativos al home, de modo que un enlace intermedio aparecido
    después de planificar se rechaza sin escribir fuera del home.
    """
    restored: list[str] = []
    done: set[str] = set()
    with SafeHome(home) as safe:
        for entry in entries:
            if entry.path in done:
                continue
            done.add(entry.path)
            source = staging / APPS_DIR / entry.app_id / entry.path
            if entry.link is not None:
                safe.replace_with_symlink(entry.path, entry.link)
            else:
                safe.write_file(entry.path, source.read_bytes(), mode=int(entry.mode, 8))
            restored.append(entry.path)
    return restored


def _ghostty_portability_notices(manifest: Manifest, backup_file: Path, home: Path) -> list[str]:
    """Avisa de rutas absolutas de Ghostty que no apuntan al home de destino."""
    notices: list[str] = []
    for app in manifest.apps:
        if app.id != GHOSTTY_APP_ID:
            continue
        for file in app.files:
            if file.link is not None:
                continue
            content = read_file_from_archive(backup_file, app.id, file.path, limit=MAX_CONFIG_BYTES)
            if content is None:
                continue
            try:
                text = content.decode("utf-8", errors="replace")
            except Exception:
                continue
            for reference in absolute_references(text):
                try:
                    to_relative(Path(reference), home=home)
                except PathError:
                    notices.append(
                        "aviso: Ghostty conserva una ruta absoluta que puede "
                        "requerir ajuste manual tras la restauración"
                    )
                    break
    return list(dict.fromkeys(notices))


def _sha256_of(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()
