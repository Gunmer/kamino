"""Lectura y escritura del archivo `tar.gz`, manifiesto y checksums.

Estructura del archivo (diseño técnico §4.3): `manifest.json` y `apps/<id>/...`.
La extracción nunca usa `extractall` sin control: rechaza rutas absolutas y `..`,
y usa `filter="data"` de `tarfile` (diseño técnico §6).
"""

from __future__ import annotations

import hashlib
import io
import os
import tarfile
from pathlib import Path, PurePosixPath
from uuid import uuid4

from pydantic import BaseModel, Field, ValidationError, field_validator, model_validator

from kamino.errors import ArchiveError, IntegrityError, VersionError

FORMAT_VERSION = 1
MANIFEST_NAME = "manifest.json"
APPS_DIR = "apps"


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
    """Una app del manifiesto con sus archivos respaldados."""

    id: str
    name: str
    bundle_id: str | None = None
    version: str | None = None
    files: list[FileEntry] = Field(default_factory=list)


class Manifest(BaseModel):
    """Metadatos del respaldo (diseño técnico §4.2)."""

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


def write_archive(
    output: Path,
    manifest: Manifest,
    *,
    source_root: Path | None = None,
    overwrite: bool = False,
) -> None:
    """Escribe `manifest.json` y los archivos del manifiesto en un `tar.gz` atómico.

    Cada `FileEntry.path` se resuelve contra `source_root` (por defecto el home).
    El manifiesto se escribe primero para que pueda leerse sin recorrer todo el
    archivo. Nunca sobrescribe por defecto (commit sin sobrescritura atómico).
    """
    root = source_root if source_root is not None else Path.home()
    output.parent.mkdir(parents=True, exist_ok=True)

    tmp = output.with_name(f".{output.name}.{uuid4().hex}.tmp")
    try:
        with tarfile.open(tmp, "w:gz") as tar:
            _add_manifest(tar, manifest)
            for app in manifest.apps:
                for file in app.files:
                    _add_file(tar, root / file.path, f"{APPS_DIR}/{app.id}/{file.path}", file)
        _commit(tmp, output, overwrite)
    except Exception:
        tmp.unlink(missing_ok=True)
        raise


def _commit(tmp: Path, output: Path, overwrite: bool) -> None:
    if overwrite:
        os.replace(tmp, output)
        return
    try:
        os.link(tmp, output)  # atómico: falla con FileExistsError si output ya existe
    except FileExistsError as exc:
        raise ArchiveError(f"el archivo de respaldo ya existe: {output}") from exc
    tmp.unlink()


def _add_manifest(tar: tarfile.TarFile, manifest: Manifest) -> None:
    data = manifest.model_dump_json(indent=2).encode("utf-8")
    info = tarfile.TarInfo(MANIFEST_NAME)
    info.size = len(data)
    info.mode = 0o644
    tar.addfile(info, io.BytesIO(data))


def _add_file(tar: tarfile.TarFile, source: Path, arcname: str, entry: FileEntry) -> None:
    try:
        info = tar.gettarinfo(str(source), arcname=arcname)
    except OSError as exc:
        raise ArchiveError(f"no se pudo leer {source}: {exc}") from exc
    if info.isfile():
        if entry.link is not None:
            raise ArchiveError(f"{entry.path} es un archivo regular, no un enlace")
        content = source.read_bytes()
        if entry.sha256 is None or hashlib.sha256(content).hexdigest() != entry.sha256:
            raise ArchiveError(f"el archivo cambió durante el respaldo: {entry.path}")
        info.size = len(content)
        tar.addfile(info, io.BytesIO(content))
    elif info.issym():
        if entry.link is None:
            raise ArchiveError(f"{entry.path} es un enlace, no un archivo regular")
        if info.linkname != entry.link:
            raise ArchiveError(
                f"el destino del enlace cambió para {entry.path}: "
                f"{info.linkname!r} != {entry.link!r}"
            )
        tar.addfile(info)
    else:
        raise ArchiveError(f"tipo de archivo no soportado para respaldar: {source}")


def read_manifest(path: Path) -> Manifest:
    """Lee y valida `manifest.json`, rechazando una versión de formato mayor."""
    with tarfile.open(path, "r:gz") as tar:
        for member in tar.getmembers():
            if member.name != MANIFEST_NAME:
                continue
            handle = tar.extractfile(member)
            if handle is None:
                raise ArchiveError(f"no se pudo leer {MANIFEST_NAME}")
            return _parse_manifest(handle.read())
    raise ArchiveError(f"falta {MANIFEST_NAME} en el archivo de respaldo")


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


def verify_archive(path: Path, manifest: Manifest) -> None:
    """Verifica que cada archivo del manifiesto exista y coincida (contenido o enlace)."""
    expected = {
        f"{APPS_DIR}/{app.id}/{file.path}": file for app in manifest.apps for file in app.files
    }
    with tarfile.open(path, "r:gz") as tar:
        for member in tar.getmembers():
            if member.name == MANIFEST_NAME:
                continue
            if member.name not in expected:
                raise IntegrityError(f"archivo inesperado en el respaldo: {member.name}")
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
    if expected:
        raise IntegrityError(f"faltan archivos en el respaldo: {sorted(expected)}")


def _sha256_of(tar: tarfile.TarFile, member: tarfile.TarInfo) -> str:
    handle = tar.extractfile(member)
    if handle is None:
        raise ArchiveError(f"no se pudo leer {member.name}")
    digest = hashlib.sha256()
    for chunk in iter(lambda: handle.read(65536), b""):
        digest.update(chunk)
    return digest.hexdigest()


def extract_archive(path: Path, dest: Path) -> None:
    """Extrae el archivo en `dest` rechazando rutas absolutas, `..` y enlaces externos."""
    dest.mkdir(parents=True, exist_ok=True)
    with tarfile.open(path, "r:gz") as tar:
        _validate_members(tar.getmembers())
        try:
            tar.extractall(dest, filter="data")
        except tarfile.TarError as exc:
            raise ArchiveError(f"extracción insegura o fallida: {exc}") from exc


def _validate_members(members: list[tarfile.TarInfo]) -> None:
    for member in members:
        pure = PurePosixPath(member.name)
        if pure.is_absolute():
            raise ArchiveError(f"ruta absoluta no permitida en el archivo: {member.name!r}")
        if ".." in pure.parts:
            raise ArchiveError(f"ruta con '..' no permitida en el archivo: {member.name!r}")
