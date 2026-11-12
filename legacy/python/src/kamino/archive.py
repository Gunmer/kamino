"""Lectura y escritura del archivo `tar.gz`, manifiesto y checksums.

Estructura del archivo (diseño técnico §4.3): `manifest.json` y `apps/<id>/...`.
La extracción nunca usa `extractall` sin control: rechaza rutas absolutas y `..`,
y usa `filter="data"` de `tarfile` (diseño técnico §6).
"""

from __future__ import annotations

import hashlib
import io
import os
import stat
import tarfile
from pathlib import Path, PurePosixPath
from uuid import uuid4

from pydantic import BaseModel, Field, ValidationError, field_validator, model_validator

from kamino.errors import ArchiveError, IntegrityError, PathError, VersionError
from kamino.paths import SafeHome
from kamino.progress import (
    CancelCallback,
    Phase,
    ProgressCallback,
    emit,
    ensure_not_cancelled,
)

FORMAT_VERSION = 1
MANIFEST_NAME = "manifest.json"
APPS_DIR = "apps"

#: Tope del manifiesto y del número de miembros para acotar archivos hostiles.
MAX_MANIFEST_BYTES = 4 * 1024 * 1024
MAX_ARCHIVE_MEMBERS = 100_000


class FileEntry(BaseModel):
    """Un archivo del manifiesto: regular (con `sha256`) o enlace (con `link`)."""

    path: str
    sha256: str | None = None
    size: int = 0
    mode: str = "0644"
    link: str | None = None

    @field_validator("sha256")
    @classmethod
    def _sha256_must_be_hex(cls, value: str | None) -> str | None:
        if value is None:
            return value
        if len(value) != 64 or any(c not in "0123456789abcdef" for c in value):
            raise ValueError("'sha256' debe ser 64 caracteres hexadecimales")
        return value

    @field_validator("path")
    @classmethod
    def _path_must_be_relative(cls, value: str) -> str:
        _ensure_relative(value)
        return value

    @field_validator("link")
    @classmethod
    def _link_must_be_relative(cls, value: str | None) -> str | None:
        if value is not None:
            _ensure_relative(value)
        return value

    @field_validator("size")
    @classmethod
    def _size_must_be_non_negative(cls, value: int) -> int:
        if value < 0:
            raise ValueError("'size' no puede ser negativo")
        return value

    @model_validator(mode="after")
    def _exactly_one_kind(self) -> FileEntry:
        if self.link is None and self.sha256 is None:
            raise ValueError("se requiere 'sha256' (archivo regular) o 'link' (enlace simbólico)")
        if self.link is not None and self.sha256 is not None:
            raise ValueError("un archivo no puede ser regular y enlace a la vez")
        return self


class AppEntry(BaseModel):
    """Una app del manifiesto con sus archivos clonados."""

    id: str
    name: str
    bundle_id: str | None = None
    version: str | None = None
    files: list[FileEntry] = Field(default_factory=list)


class Manifest(BaseModel):
    """Metadatos del clon (diseño técnico §4.2)."""

    format_version: int
    tool_version: str
    created_at: str
    macos_version: str
    arch: str
    hostname: str
    apps: list[AppEntry] = Field(default_factory=list)

    @field_validator("format_version")
    @classmethod
    def _format_version_must_be_positive(cls, value: int) -> int:
        if value < 1:
            raise ValueError("'format_version' debe ser positivo")
        return value


def _ensure_relative(path: str) -> None:
    """Rechaza rutas vacías, absolutas o con `..` (seguridad del manifiesto)."""
    if not path:
        raise ValueError("ruta vacía no admitida")
    pure = PurePosixPath(path)
    if pure.is_absolute():
        raise ValueError(f"ruta absoluta no admitida: {path!r}")
    if ".." in pure.parts:
        raise ValueError(f"ruta con '..' no admitida: {path!r}")


def destination_state(path: Path) -> tuple[int, int, int] | None:
    """Huella barata del destino para revalidar un reemplazo forzado."""
    try:
        info = path.lstat()
    except FileNotFoundError:
        return None
    return (info.st_ino, info.st_size, info.st_mtime_ns)


def write_archive(
    output: Path,
    manifest: Manifest,
    *,
    source_root: Path | None = None,
    overwrite: bool = False,
    expected_state: tuple[int, int, int] | None = None,
    progress: ProgressCallback | None = None,
    cancelled: CancelCallback | None = None,
) -> None:
    """Escribe `manifest.json` y los archivos del manifiesto en un `tar.gz` atómico.

    Cada `FileEntry.path` se resuelve contra `source_root` (por defecto el home).
    El manifiesto se escribe primero para que pueda leerse sin recorrer todo el
    archivo. Nunca sobrescribe por defecto (commit sin sobrescritura atómico).
    Con `overwrite`, revalida el destino justo antes del renombrado y usa
    `os.replace`; cualquier error o cancelación elimina el temporal.
    """
    root = source_root if source_root is not None else Path.home()
    output.parent.mkdir(parents=True, exist_ok=True)

    initial_state = destination_state(output)
    if expected_state is not None and initial_state != expected_state:
        raise ArchiveError(f"el archivo del clon cambió antes de escribir: {output}")

    entries = [(app.id, file) for app in manifest.apps for file in app.files]
    tmp = output.with_name(f".{output.name}.{uuid4().hex}.tmp")
    try:
        with SafeHome(root) as safe, tarfile.open(tmp, "w:gz") as tar:
            _add_manifest(tar, manifest)
            for index, (app_id, file) in enumerate(entries, start=1):
                ensure_not_cancelled(cancelled)
                _add_file(tar, safe, file.path, f"{APPS_DIR}/{app_id}/{file.path}", file)
                emit(progress, Phase.WRITING, index, len(entries), file.path)
        ensure_not_cancelled(cancelled)
        _commit(tmp, output, overwrite, initial_state)
    except Exception:
        tmp.unlink(missing_ok=True)
        raise


def _commit(
    tmp: Path,
    output: Path,
    overwrite: bool,
    initial_state: tuple[int, int, int] | None,
) -> None:
    if overwrite:
        if destination_state(output) != initial_state:
            raise ArchiveError(f"el archivo del clon cambió antes del renombrado: {output}")
        os.replace(tmp, output)
        return
    try:
        os.link(tmp, output)  # atómico: falla con FileExistsError si output ya existe
    except FileExistsError as exc:
        raise ArchiveError(f"el archivo del clon ya existe: {output}") from exc
    tmp.unlink()


def _add_manifest(tar: tarfile.TarFile, manifest: Manifest) -> None:
    data = manifest.model_dump_json(indent=2).encode("utf-8")
    info = tarfile.TarInfo(MANIFEST_NAME)
    info.size = len(data)
    info.mode = 0o644
    tar.addfile(info, io.BytesIO(data))


def _add_file(
    tar: tarfile.TarFile, safe: SafeHome, rel: str, arcname: str, entry: FileEntry
) -> None:
    """Añade un archivo reabriéndolo de forma segura, sin seguir enlaces intermedios."""
    try:
        info = safe.lstat(rel)
    except (FileNotFoundError, PathError, OSError) as exc:
        raise ArchiveError(f"no se pudo leer {rel}: {exc}") from exc
    if stat.S_ISREG(info.st_mode):
        if entry.link is not None:
            raise ArchiveError(f"{entry.path} es un archivo regular, no un enlace")
        try:
            content = safe.read_bytes(rel)
        except (PathError, OSError) as exc:
            raise ArchiveError(f"no se pudo leer {rel}: {exc}") from exc
        if entry.sha256 is None or hashlib.sha256(content).hexdigest() != entry.sha256:
            raise ArchiveError(f"el archivo cambió durante la clonación: {entry.path}")
        tar.addfile(_tar_info(arcname, info, size=len(content)), io.BytesIO(content))
    elif stat.S_ISLNK(info.st_mode):
        if entry.link is None:
            raise ArchiveError(f"{entry.path} es un enlace, no un archivo regular")
        try:
            target = safe.readlink(rel)
        except (PathError, OSError) as exc:
            raise ArchiveError(f"no se pudo leer el enlace {rel}: {exc}") from exc
        if target != entry.link:
            raise ArchiveError(
                f"el destino del enlace cambió para {entry.path}: {target!r} != {entry.link!r}"
            )
        tar_info = _tar_info(arcname, info)
        tar_info.type = tarfile.SYMTYPE
        tar_info.linkname = target
        tar_info.size = 0
        tar.addfile(tar_info)
    else:
        raise ArchiveError(f"tipo de archivo no soportado para clonar: {rel}")


def _tar_info(arcname: str, info: os.stat_result, *, size: int = 0) -> tarfile.TarInfo:
    """Crea un `TarInfo` a partir de un `lstat` ya validado."""
    tar_info = tarfile.TarInfo(arcname)
    tar_info.mode = stat.S_IMODE(info.st_mode)
    tar_info.uid = info.st_uid
    tar_info.gid = info.st_gid
    tar_info.mtime = int(info.st_mtime)
    tar_info.size = size
    return tar_info


def read_manifest(path: Path) -> Manifest:
    """Lee y valida `manifest.json`, rechazando un archivo hostil o ambiguo.

    Exige un único miembro regular llamado `manifest.json`, acota el número de
    miembros y su tamaño a :data:`MAX_MANIFEST_BYTES`, y envuelve los errores
    de `tarfile`/sistema en :class:`ArchiveError`.
    """
    manifest_member: tarfile.TarInfo | None = None
    try:
        with tarfile.open(path, "r:gz") as tar:
            for index, member in enumerate(tar, start=1):
                if index > MAX_ARCHIVE_MEMBERS:
                    raise ArchiveError(
                        f"el archivo del clon supera el tope de {MAX_ARCHIVE_MEMBERS} miembros"
                    )
                if member.name != MANIFEST_NAME:
                    continue
                if manifest_member is not None:
                    raise ArchiveError(f"hay más de un {MANIFEST_NAME} en el archivo del clon")
                manifest_member = member
            if manifest_member is None:
                raise ArchiveError(f"falta {MANIFEST_NAME} en el archivo del clon")
            if not manifest_member.isfile():
                raise ArchiveError(f"{MANIFEST_NAME} no es un archivo regular")
            if manifest_member.size > MAX_MANIFEST_BYTES:
                raise ArchiveError(f"{MANIFEST_NAME} supera el tope de {MAX_MANIFEST_BYTES} bytes")
            handle = tar.extractfile(manifest_member)
            if handle is None:
                raise ArchiveError(f"no se pudo leer {MANIFEST_NAME}")
            raw = handle.read(MAX_MANIFEST_BYTES + 1)
    except ArchiveError:
        raise
    except (tarfile.TarError, OSError) as exc:
        raise ArchiveError(f"no se pudo leer el archivo del clon: {exc}") from exc
    if len(raw) > MAX_MANIFEST_BYTES:
        raise ArchiveError(f"{MANIFEST_NAME} supera el tope de {MAX_MANIFEST_BYTES} bytes")
    return _parse_manifest(raw)


def read_file_from_archive(path: Path, app_id: str, rel: str, *, limit: int) -> bytes | None:
    """Lee de forma acotada un archivo regular del clon, sin extraerlo.

    Devuelve `None` si el miembro no existe, no es un archivo regular o supera
    `limit`. Se usa para avisos de portabilidad de Ghostty durante la
    planificación del despliegue.
    """
    name = f"{APPS_DIR}/{app_id}/{rel}"
    with tarfile.open(path, "r:gz") as tar:
        try:
            member = tar.getmember(name)
        except KeyError:
            return None
        if not member.isfile() or member.size > limit:
            return None
        handle = tar.extractfile(member)
        if handle is None:
            return None
        return handle.read(limit + 1)


def _parse_manifest(raw: bytes) -> Manifest:
    try:
        manifest = Manifest.model_validate_json(raw)
    except ValidationError as exc:
        raise ArchiveError(f"manifiesto inválido: {exc}") from exc
    if manifest.format_version > FORMAT_VERSION:
        raise VersionError(
            f"format_version {manifest.format_version} no soportado (máximo {FORMAT_VERSION})"
        )
    return manifest


def verify_archive(
    path: Path,
    manifest: Manifest,
    *,
    progress: ProgressCallback | None = None,
    cancelled: CancelCallback | None = None,
) -> None:
    """Verifica que cada archivo del manifiesto exista y coincida (contenido o enlace)."""
    expected = {
        f"{APPS_DIR}/{app.id}/{file.path}": file for app in manifest.apps for file in app.files
    }
    total = len(expected)
    current = 0
    try:
        with tarfile.open(path, "r:gz") as tar:
            for index, member in enumerate(tar, start=1):
                ensure_not_cancelled(cancelled)
                if index > MAX_ARCHIVE_MEMBERS + 1:
                    raise ArchiveError(
                        f"el archivo del clon supera el tope de {MAX_ARCHIVE_MEMBERS} miembros"
                    )
                if member.name == MANIFEST_NAME:
                    continue
                if member.name not in expected:
                    raise IntegrityError(f"archivo inesperado en el clon: {member.name}")
                entry = expected[member.name]
                if member.isfile():
                    if entry.link is not None:
                        raise IntegrityError(f"esperaba un enlace en {member.name}, no un archivo")
                    if entry.sha256 is None or _sha256_of(tar, member) != entry.sha256:
                        raise IntegrityError(f"checksum no coincide para {member.name}")
                elif member.issym():
                    if entry.link is None:
                        raise IntegrityError(f"esperaba un archivo en {member.name}, no un enlace")
                    if member.linkname != entry.link:
                        raise IntegrityError(f"destino de enlace no coincide para {member.name}")
                else:
                    raise IntegrityError(f"tipo de miembro no soportado: {member.name}")
                del expected[member.name]
                current += 1
                emit(progress, Phase.VERIFYING, current, total, member.name)
    except (IntegrityError, ArchiveError):
        raise
    except (tarfile.TarError, OSError) as exc:
        raise ArchiveError(f"no se pudo verificar el archivo del clon: {exc}") from exc
    if expected:
        raise IntegrityError(f"faltan archivos en el clon: {sorted(expected)}")


def _sha256_of(tar: tarfile.TarFile, member: tarfile.TarInfo) -> str:
    handle = tar.extractfile(member)
    if handle is None:
        raise ArchiveError(f"no se pudo leer {member.name}")
    digest = hashlib.sha256()
    for chunk in iter(lambda: handle.read(65536), b""):
        digest.update(chunk)
    return digest.hexdigest()


def extract_archive(
    path: Path,
    dest: Path,
    *,
    progress: ProgressCallback | None = None,
    cancelled: CancelCallback | None = None,
) -> None:
    """Extrae el archivo en `dest` rechazando rutas absolutas, `..` y enlaces externos."""
    dest.mkdir(parents=True, exist_ok=True)
    try:
        with tarfile.open(path, "r:gz") as tar:
            members: list[tarfile.TarInfo] = []
            for index, member in enumerate(tar, start=1):
                if index > MAX_ARCHIVE_MEMBERS:
                    raise ArchiveError(
                        f"el archivo del clon supera el tope de {MAX_ARCHIVE_MEMBERS} miembros"
                    )
                members.append(member)
            _validate_members(members)
            total = len(members)
            for index, member in enumerate(members, start=1):
                ensure_not_cancelled(cancelled)
                tar.extract(member, dest, filter="data")
                emit(progress, Phase.EXTRACTING, index, total, member.name)
    except ArchiveError:
        raise
    except (tarfile.TarError, OSError) as exc:
        raise ArchiveError(f"extracción insegura o fallida: {exc}") from exc


def _validate_members(members: list[tarfile.TarInfo]) -> None:
    for member in members:
        pure = PurePosixPath(member.name)
        if pure.is_absolute():
            raise ArchiveError(f"ruta absoluta no permitida en el archivo: {member.name!r}")
        if ".." in pure.parts:
            raise ArchiveError(f"ruta con '..' no permitida en el archivo: {member.name!r}")
