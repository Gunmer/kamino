"""Detección de apps instaladas y su versión.

API pública:

- `is_installed(app, home=...)`: True si la app está instalada, por `bundle_id`
  o por `command`, y su marcador opcional existe dentro del home.
- `detect_version(app)`: versión de una app `.app` desde su `Info.plist`, o `None`.

Mecanismos (diseño técnico §6): para apps con `.app` se lee `Info.plist` buscando
`CFBundleIdentifier` en `/Applications`, `~/Applications` y `/System/Applications`;
para herramientas de terminal se usa `shutil.which` sobre el PATH y `search_paths`.
Los comandos se localizan, nunca se ejecutan.

La detección es "best-effort": un `Info.plist` ilegible o corrupto, o un directorio
sin permiso de lectura, se trata como "app no detectada", nunca como un error. La
versión solo se obtiene para apps `.app`; para las de terminal se devuelve `None`,
porque obtenerla exigiría ejecutarlas.
"""

from __future__ import annotations

import os
import plistlib
import shutil
import stat
from collections.abc import Iterator, Sequence
from pathlib import Path

from kamino.catalog.models import AppDefinition
from kamino.errors import PathError
from kamino.paths import SafeHome


def is_installed(
    app: AppDefinition,
    *,
    home: Path | None = None,
    application_dirs: Sequence[Path] | None = None,
    path: str | None = None,
) -> bool:
    """Indica si la app está instalada, por `bundle_id` o por `command`.

    `home` se usa para validar `install_marker` sin tocar el HOME real. Las
    pruebas inyectan también `application_dirs` y `path`.
    """
    if app.install_marker is not None and not _marker_exists(app.install_marker, home):
        return False
    has_bundle = (
        app.bundle_id is not None
        and _find_app_bundle(app.bundle_id, _application_dirs(application_dirs, home)) is not None
    )
    commands = [app.command, *app.command_aliases]
    has_command = any(
        command is not None and _find_command(command, app.search_paths, path) is not None
        for command in commands
    )
    return has_bundle or has_command


def detect_version(
    app: AppDefinition,
    *,
    home: Path | None = None,
    application_dirs: Sequence[Path] | None = None,
) -> str | None:
    """Devuelve la versión de una app `.app`, o `None` si no aplica o no aparece."""
    if not app.bundle_id:
        return None
    bundle = _find_app_bundle(app.bundle_id, _application_dirs(application_dirs, home))
    return _read_version(bundle) if bundle is not None else None


def _application_dirs(application_dirs: Sequence[Path] | None, home: Path | None) -> Sequence[Path]:
    if application_dirs is not None:
        return application_dirs
    root = home if home is not None else Path.home()
    return (
        Path("/Applications"),
        root / "Applications",
        Path("/System/Applications"),
    )


def _marker_exists(marker: str, home: Path | None) -> bool:
    """Comprueba un marcador relativo al home sin seguir enlaces intermedios."""
    root = home if home is not None else Path.home()
    try:
        with SafeHome(root) as safe:
            info = safe.lstat(marker)
    except (FileNotFoundError, NotADirectoryError, PathError, OSError):
        return False
    return stat.S_ISREG(info.st_mode)


def _find_app_bundle(bundle_id: str, application_dirs: Sequence[Path]) -> Path | None:
    for directory in application_dirs:
        for candidate in _app_bundles(directory):
            if _bundle_id_of(candidate) == bundle_id:
                return candidate
    return None


def _app_bundles(directory: Path) -> Iterator[Path]:
    if not directory.is_dir():
        return
    try:
        candidates = sorted(path for path in directory.glob("*.app") if path.is_dir())
    except OSError:
        return
    yield from candidates


def _plist_of(app_bundle: Path) -> dict[str, object] | None:
    """Lee `Info.plist` de forma tolerante, devolviendo `None` ante cualquier fallo.

    Los plists son de terceros: pueden ser binarios truncados o contener valores
    que `plistlib` no soporta (fechas inválidas, etc.), que lanzan excepciones
    variadas. La detección es "best-effort": cualquier fallo se descarta en
    silencio a propósito y se trata como "no es esta app", porque el núcleo no
    imprime ni pregunta (diseño técnico §1).
    """
    info_plist = app_bundle / "Contents" / "Info.plist"
    try:
        with info_plist.open("rb") as handle:
            data = plistlib.load(handle)
    except Exception:
        return None
    return data if isinstance(data, dict) else None


def _bundle_id_of(app_bundle: Path) -> str | None:
    data = _plist_of(app_bundle)
    if data is None:
        return None
    value = data.get("CFBundleIdentifier")
    return value if isinstance(value, str) else None


def _read_version(app_bundle: Path) -> str | None:
    data = _plist_of(app_bundle)
    if data is None:
        return None
    value = data.get("CFBundleShortVersionString") or data.get("CFBundleVersion")
    return value if isinstance(value, str) else None


def _find_command(command: str, search_paths: Sequence[str], path: str | None) -> Path | None:
    found = shutil.which(command, path=path)
    if found:
        return Path(found)
    for directory in search_paths:
        candidate = Path(directory) / command
        if candidate.is_file() and os.access(candidate, os.X_OK):
            return candidate
    return None
