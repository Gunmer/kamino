"""Copia de seguridad previa a restaurar (snapshot).

Se guarda en `~/Library/Application Support/kamino/snapshots/<fecha>.tar.gz` con
el mismo formato que un respaldo (diseño técnico §6): una única app sintética
`snapshot` con las rutas relativas al home, así el futuro `undo` es un `restore`
del snapshot.

Módulo sin interfaz (§1): no imprime ni pregunta. `home`, `snapshots_dir` y
`timestamp` se inyectan para que las pruebas no toquen el sistema real (§7).
"""

from __future__ import annotations

import platform
from collections.abc import Sequence
from datetime import UTC, datetime
from pathlib import Path, PurePosixPath

import kamino
from kamino.archive import FORMAT_VERSION, AppEntry, FileEntry, Manifest, write_archive
from kamino.catalog.models import AppDefinition
from kamino.errors import ArchiveError, PathError
from kamino.paths import to_relative
from kamino.plan import build_plan

#: Id de la app sintética que agrupa los archivos de un snapshot (§6).
SNAPSHOT_APP_ID = "snapshot"

_DATE_FORMAT = "%Y-%m-%dT%H:%M:%SZ"
_MAX_SUFFIX = 1000


def create_snapshot(
    paths: Sequence[Path],
    *,
    home: Path | None = None,
    snapshots_dir: Path | None = None,
    timestamp: str | None = None,
) -> Path | None:
    """Guarda el estado actual de `paths` antes de que la restauración lo cambie.

    Crea un archivo con el mismo formato que un respaldo (§6) bajo una única app
    sintética `snapshot`, con las rutas relativas al home. Los directorios se
    expanden recursivamente; lo que no existe o no cabe en el manifiesto (enlaces
    inseguros, tipos especiales) se omite, igual que en un respaldo.

    Se guarda en `snapshots_dir` (por defecto
    `~/Library/Application Support/kamino/snapshots/`) con nombre
    `<timestamp>.tar.gz`, usando la hora actual en ISO 8601 UTC si no se indica;
    si el nombre ya existe se añade un sufijo numérico y nunca se sobrescribe (§6).

    Devuelve la ruta del snapshot, o `None` si `paths` está vacío.
    """
    root = home if home is not None else Path.home()
    if not paths:
        return None
    when = timestamp if timestamp is not None else datetime.now(UTC).strftime(_DATE_FORMAT)
    _check_timestamp(when)

    definition = AppDefinition(
        id=SNAPSHOT_APP_ID,
        name=SNAPSHOT_APP_ID,
        command=SNAPSHOT_APP_ID,
        paths=_relative_targets(paths, root),
    )
    plan = build_plan([definition], home=root)
    files = [
        FileEntry(path=op.path, sha256=op.sha256, size=op.size, mode=op.mode, link=op.link)
        for op in plan.operations
    ]
    manifest = Manifest(
        format_version=FORMAT_VERSION,
        tool_version=kamino.__version__,
        created_at=when,
        macos_version=platform.mac_ver()[0],
        arch=platform.machine(),
        hostname=platform.node(),
        apps=[AppEntry(id=SNAPSHOT_APP_ID, name=SNAPSHOT_APP_ID, files=files)],
    )

    directory = snapshots_dir if snapshots_dir is not None else _default_dir(root)
    output = _output_path(directory, when)
    write_archive(output, manifest, source_root=root)
    return output


def _default_dir(root: Path) -> Path:
    """Directorio por defecto de los snapshots bajo el home (§6)."""
    return root / "Library" / "Application Support" / "kamino" / "snapshots"


def _check_timestamp(timestamp: str) -> None:
    """Rechaza timestamps vacíos o con separadores, que saldrían del directorio."""
    if not timestamp or Path(timestamp).name != timestamp:
        raise PathError(f"timestamp no admitido como nombre de archivo: {timestamp!r}")


def _output_path(directory: Path, timestamp: str) -> Path:
    """`<timestamp>.tar.gz` en `directory`, con sufijo numérico si ya existe (§6)."""
    for index in range(_MAX_SUFFIX + 1):
        suffix = "" if index == 0 else f"-{index}"
        candidate = directory / f"{timestamp}{suffix}.tar.gz"
        if not candidate.exists():
            return candidate
    raise ArchiveError(f"no se pudo elegir un nombre libre para {timestamp}.tar.gz")


def _relative_targets(paths: Sequence[Path], root: Path) -> list[str]:
    """Normaliza las rutas a relativas POSIX, sin duplicados ni solapamientos.

    Una ruta cubierta por otra se descarta: `build_plan` la expandiría dos veces y
    el archivo no admite dos miembros con el mismo nombre.
    """
    rels = list(dict.fromkeys(to_relative(path, home=root).as_posix() for path in paths))
    return [rel for rel in rels if not any(_is_ancestor(other, rel) for other in rels)]


def _is_ancestor(other: str, rel: str) -> bool:
    """Indica si `other` es un ancestro estricto de `rel`."""
    parts = PurePosixPath(rel).parts
    other_parts = PurePosixPath(other).parts
    return len(other_parts) < len(parts) and parts[: len(other_parts)] == other_parts
