"""Actualización del binario autocontenido desde las releases de GitHub."""

from __future__ import annotations

import contextlib
import hashlib
import hmac
import http.client
import json
import os
import platform
import re
import ssl
import stat
import sys
import tempfile
import urllib.error
import urllib.request
from dataclasses import dataclass
from enum import StrEnum
from pathlib import Path

import certifi

from kamino import __version__
from kamino.errors import OperationCancelled, UpdateError, UpdateErrorKind
from kamino.progress import CancelCallback, Phase, ProgressCallback, emit, ensure_not_cancelled

REPOSITORY = "Gunmer/kamino"
ASSET_NAME = "kamino-arm64"
API_URL = f"https://api.github.com/repos/{REPOSITORY}/releases/latest"
TIMEOUT_SECONDS = 30
CHUNK_SIZE = 64 * 1024
USER_AGENT = f"kamino/{__version__}"

_VERSION_PATTERN = re.compile(r"^v?(\d+)\.(\d+)\.(\d+)$")
_DIGEST_PATTERN = re.compile(r"^sha256:([0-9a-fA-F]{64})$")


@dataclass(frozen=True)
class ReleaseInfo:
    """Datos del asset de la última release publicada."""

    version: str
    asset_url: str
    asset_size: int
    asset_digest: str


class UpdateOutcome(StrEnum):
    """Resultado estable para que la TUI elija el mensaje sin parsear textos."""

    UPDATED = "updated"
    UP_TO_DATE = "up_to_date"
    NEWER = "newer"


@dataclass(frozen=True)
class UpdateResult:
    """Resultado de una consulta o actualización."""

    current_version: str
    latest_version: str
    target: Path
    updated: bool
    outcome: UpdateOutcome = UpdateOutcome.UP_TO_DATE


@dataclass(frozen=True)
class UpdateCheck:
    """Consulta resuelta: release encontrada y si procede actualizar."""

    current_version: str
    release: ReleaseInfo
    target: Path
    should_update: bool
    outcome: UpdateOutcome


def parse_version(value: str) -> tuple[int, int, int]:
    """Convierte ``vX.Y.Z`` o ``X.Y.Z`` en una tupla comparable."""
    match = _VERSION_PATTERN.fullmatch(value.strip())
    if match is None:
        raise UpdateError(f"versión no válida: {value!r}")
    return (int(match.group(1)), int(match.group(2)), int(match.group(3)))


def normalize_version(value: str) -> str:
    """Devuelve la versión sin el prefijo ``v`` y sin espacios."""
    major, minor, patch = parse_version(value)
    return f"{major}.{minor}.{patch}"


def installed_binary_path() -> Path:
    """Devuelve el binario real en ejecución o falla si no es autocontenido."""
    if not getattr(sys, "frozen", False):
        raise UpdateError(
            "kamino update solo está disponible en el binario autocontenido. "
            "En desarrollo usa `uv sync`; para una instalación con uv, "
            "usa `uv tool upgrade kamino-cli`; con pip, actualiza el paquete "
            "por la vía habitual.",
            kind=UpdateErrorKind.NOT_SELF_CONTAINED,
        )
    return Path(sys.executable).resolve()


def _open_url(request: urllib.request.Request, *, timeout: int = TIMEOUT_SECONDS):
    """Abre una URL con el bundle de CAs de certifi."""
    context = ssl.create_default_context(cafile=certifi.where())
    return urllib.request.urlopen(request, context=context, timeout=timeout)


def _ensure_supported_platform() -> None:
    if platform.system() != "Darwin" or platform.machine() != "arm64":
        raise UpdateError(
            "kamino update solo está disponible para macOS en Apple Silicon (arm64).",
            kind=UpdateErrorKind.PLATFORM,
        )


def latest_release() -> ReleaseInfo:
    """Consulta la última release publicada y valida su asset."""
    request = urllib.request.Request(
        API_URL,
        headers={"Accept": "application/vnd.github+json", "User-Agent": USER_AGENT},
    )
    try:
        with _open_url(request) as response:
            raw = response.read()
    except urllib.error.HTTPError as exc:
        if exc.code == 403:
            raise UpdateError(
                "GitHub ha limitado las peticiones de la API; inténtalo más tarde "
                "o descarga la release desde https://github.com/Gunmer/kamino/releases.",
                kind=UpdateErrorKind.NETWORK,
            ) from exc
        raise UpdateError(
            f"no se pudo consultar la última versión: HTTP {exc.code}",
            kind=UpdateErrorKind.NETWORK,
        ) from exc
    except urllib.error.URLError as exc:
        raise UpdateError(
            f"no se pudo consultar la última versión: {exc.reason}",
            kind=UpdateErrorKind.NETWORK,
        ) from exc
    except (OSError, http.client.HTTPException) as exc:
        raise UpdateError(
            f"no se pudo consultar la última versión: {exc}",
            kind=UpdateErrorKind.NETWORK,
        ) from exc
    try:
        payload = json.loads(raw)
    except (json.JSONDecodeError, UnicodeDecodeError) as exc:
        raise UpdateError(
            "la respuesta de GitHub no es JSON válido", kind=UpdateErrorKind.NETWORK
        ) from exc
    if not isinstance(payload, dict):
        raise UpdateError(
            "la respuesta de GitHub no tiene el formato esperado",
            kind=UpdateErrorKind.NETWORK,
        )
    tag = payload.get("tag_name")
    if not isinstance(tag, str) or not tag.strip():
        raise UpdateError(
            "la release no incluye un tag de versión válido", kind=UpdateErrorKind.NETWORK
        )
    try:
        version = normalize_version(tag)
    except UpdateError as exc:
        raise UpdateError(
            f"la release no tiene una versión válida: {tag!r}",
            kind=UpdateErrorKind.NETWORK,
        ) from exc
    assets = payload.get("assets")
    if not isinstance(assets, list):
        raise UpdateError(
            "la release no incluye una lista de assets válida", kind=UpdateErrorKind.NETWORK
        )
    asset = next(
        (item for item in assets if isinstance(item, dict) and item.get("name") == ASSET_NAME),
        None,
    )
    if asset is None:
        raise UpdateError(
            f"la release {tag} no incluye el asset {ASSET_NAME}", kind=UpdateErrorKind.NETWORK
        )
    url = asset.get("browser_download_url")
    size = asset.get("size")
    digest = asset.get("digest")
    if not isinstance(url, str) or not url.startswith("https://"):
        raise UpdateError(
            f"el asset {ASSET_NAME} no tiene una URL de descarga válida",
            kind=UpdateErrorKind.NETWORK,
        )
    if not isinstance(size, int) or isinstance(size, bool) or size <= 0:
        raise UpdateError(
            f"el asset {ASSET_NAME} no tiene un tamaño válido", kind=UpdateErrorKind.NETWORK
        )
    if not isinstance(digest, str) or _DIGEST_PATTERN.fullmatch(digest) is None:
        raise UpdateError(
            f"el asset {ASSET_NAME} no publica un SHA-256 válido",
            kind=UpdateErrorKind.INTEGRITY,
        )
    return ReleaseInfo(
        version=version,
        asset_url=url,
        asset_size=size,
        asset_digest=digest.lower(),
    )


def _digest_hex(digest: str) -> str:
    match = _DIGEST_PATTERN.fullmatch(digest)
    if match is None:
        raise UpdateError(f"SHA-256 no válido: {digest!r}", kind=UpdateErrorKind.INTEGRITY)
    return match.group(1).lower()


def query_update(
    current_version: str,
    *,
    target: Path | None = None,
    release: ReleaseInfo | None = None,
) -> UpdateCheck:
    """Comprueba plataforma, binario y última release sin descargar nada."""
    target_path = Path(target).resolve() if target is not None else installed_binary_path()
    _ensure_supported_platform()
    current = normalize_version(current_version)
    if not target_path.is_file():
        raise UpdateError(
            f"no se encontró el binario instalado en {target_path}",
            kind=UpdateErrorKind.NOT_SELF_CONTAINED,
        )
    found = release if release is not None else latest_release()
    if parse_version(found.version) > parse_version(current):
        outcome = UpdateOutcome.UPDATED
        should_update = True
    elif parse_version(found.version) == parse_version(current):
        outcome = UpdateOutcome.UP_TO_DATE
        should_update = False
    else:
        outcome = UpdateOutcome.NEWER
        should_update = False
    return UpdateCheck(
        current_version=current,
        release=found,
        target=target_path,
        should_update=should_update,
        outcome=outcome,
    )


def download_update(
    release: ReleaseInfo,
    target_path: Path,
    *,
    progress: ProgressCallback | None = None,
    cancelled: CancelCallback | None = None,
) -> Path:
    """Descarga y verifica el asset en un temporal del mismo directorio."""
    try:
        descriptor, name = tempfile.mkstemp(prefix=".kamino-update-", dir=target_path.parent)
        temp = Path(name)
        os.close(descriptor)
    except OSError as exc:
        raise UpdateError(
            f"no se pudo crear un archivo temporal en {target_path.parent}: {exc}",
            kind=UpdateErrorKind.REPLACE,
        ) from exc
    try:
        _download_asset(release, temp, progress=progress, cancelled=cancelled)
    except BaseException:
        with contextlib.suppress(OSError):
            temp.unlink(missing_ok=True)
        raise
    return temp


def replace_binary(temp: Path, target: Path) -> None:
    """Chmod y `os.replace` atómico; el temporal ya está verificado."""
    mode = 0o755
    try:
        current_mode = stat.S_IMODE(target.stat().st_mode)
    except OSError:
        current_mode = 0
    if current_mode & 0o111:
        mode = current_mode
    try:
        os.chmod(temp, mode)
        os.replace(temp, target)
    except OSError as exc:
        raise UpdateError(
            f"no se pudo reemplazar {target}: {exc}", kind=UpdateErrorKind.REPLACE
        ) from exc


#: Alias interno conservado para pruebas y llamadas existentes.
_replace_binary = replace_binary


def run_update(
    current_version: str,
    *,
    target: Path | None = None,
    release: ReleaseInfo | None = None,
    progress: ProgressCallback | None = None,
    cancelled: CancelCallback | None = None,
) -> UpdateResult:
    """Fachada: consulta, descarga/verifica, reemplaza y devuelve el resultado."""
    check = query_update(current_version, target=target, release=release)
    if not check.should_update:
        return UpdateResult(
            check.current_version,
            check.release.version,
            check.target,
            updated=False,
            outcome=check.outcome,
        )
    temp: Path | None = download_update(
        check.release,
        check.target,
        progress=progress,
        cancelled=cancelled,
    )
    try:
        ensure_not_cancelled(cancelled)
        emit(progress, Phase.REPLACING, 0, 1, str(check.target))
        replace_binary(temp, check.target)
        temp = None
    finally:
        if temp is not None:
            with contextlib.suppress(OSError):
                temp.unlink(missing_ok=True)
    return UpdateResult(
        check.current_version,
        check.release.version,
        check.target,
        updated=True,
        outcome=UpdateOutcome.UPDATED,
    )


def _download_asset(
    release: ReleaseInfo,
    destination: Path,
    *,
    progress: ProgressCallback | None = None,
    cancelled: CancelCallback | None = None,
) -> None:
    request = urllib.request.Request(
        release.asset_url,
        headers={"User-Agent": USER_AGENT},
    )
    digest = hashlib.sha256()
    size = 0
    try:
        with _open_url(request) as response, destination.open("wb") as handle:
            while chunk := response.read(CHUNK_SIZE):
                ensure_not_cancelled(cancelled)
                handle.write(chunk)
                digest.update(chunk)
                size += len(chunk)
                emit(progress, Phase.DOWNLOADING, size, release.asset_size)
    except (OperationCancelled, UpdateError):
        raise
    except urllib.error.HTTPError as exc:
        raise UpdateError(
            f"no se pudo descargar {ASSET_NAME}: HTTP {exc.code}",
            kind=UpdateErrorKind.NETWORK,
        ) from exc
    except urllib.error.URLError as exc:
        raise UpdateError(
            f"no se pudo descargar {ASSET_NAME}: {exc.reason}",
            kind=UpdateErrorKind.NETWORK,
        ) from exc
    except (OSError, http.client.HTTPException) as exc:
        raise UpdateError(
            f"no se pudo descargar {ASSET_NAME}: {exc}", kind=UpdateErrorKind.NETWORK
        ) from exc
    if size != release.asset_size:
        raise UpdateError(
            f"la descarga de {ASSET_NAME} está incompleta: "
            f"se esperaban {release.asset_size} bytes y se recibieron {size}",
            kind=UpdateErrorKind.INTEGRITY,
        )
    if not hmac.compare_digest(digest.hexdigest(), _digest_hex(release.asset_digest)):
        raise UpdateError(
            f"el SHA-256 de {ASSET_NAME} no coincide con el publicado",
            kind=UpdateErrorKind.INTEGRITY,
        )
