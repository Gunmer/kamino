"""Plan y Operation, compartidos por `backup` y `restore`.

Principio (diseño técnico §1): planificar y luego ejecutar. `--dry-run` solo
imprime el plan, así simulación y ejecución comparten el mismo código.

El plan se construye al estilo del paso 4 de `backup` (§5.1): expandir cada
`paths` de las apps (archivos y carpetas, recursivo), aplicar `exclude` y
registrar tamaño, checksum y modo. Las rutas que no existen se registran como
omitidas, no como error. Los enlaces simbólicos se guardan como enlace y no se
siguen (§6): los de destino absoluto o con `..` se omiten por seguridad.
"""

from __future__ import annotations

import hashlib
import os
import stat
from collections.abc import Sequence
from dataclasses import dataclass, field
from fnmatch import fnmatch
from pathlib import Path, PurePosixPath

from kamino.catalog.models import AppDefinition
from kamino.paths import to_absolute, to_relative


@dataclass(frozen=True)
class Operation:
    """Una operación del plan: un archivo regular o un enlace simbólico.

    Para archivos regulares `sha256` es el digest del contenido y `link` es
    `None`; para enlaces simbólicos `link` es el destino relativo seguro,
    `sha256` es `None` y `size` es 0.
    """

    app_id: str
    path: str
    source: Path
    size: int
    sha256: str | None
    mode: str
    link: str | None


@dataclass(frozen=True)
class Plan:
    """El plan de un respaldo: operaciones y rutas relativas omitidas.

    `skipped` registra las rutas que no existen y los enlaces inseguros
    (destino absoluto o con `..`), que se omiten en lugar de fallar.
    """

    operations: list[Operation] = field(default_factory=list)
    skipped: list[str] = field(default_factory=list)


def is_excluded(path: str, patterns: Sequence[str]) -> bool:
    """Indica si una ruta relativa POSIX está descartada por algún patrón.

    Se aplica `fnmatch.fnmatch` sobre la ruta completa y también sobre cada
    uno de sus ancestros, de modo que `**/CachedData` descarta el directorio y
    todo su contenido a cualquier profundidad (§6: "Estado frente a
    configuración"). Un patrón `**/X` también coincide con `X` en el nivel
    superior, porque `fnmatch` no interpreta `**` de forma especial.
    """
    pure = PurePosixPath(path)
    candidates = [path]
    candidates.extend("/".join(pure.parts[:depth]) for depth in range(1, len(pure.parts)))
    return any(_matches(candidate, pattern) for candidate in candidates for pattern in patterns)


def _matches(path: str, pattern: str) -> bool:
    if fnmatch(path, pattern):
        return True
    # `**/X` debe coincidir también con `X` suelto en el nivel superior.
    return pattern.startswith("**/") and fnmatch(path, pattern[3:])


def build_plan(apps: Sequence[AppDefinition], *, home: Path | None = None) -> Plan:
    """Construye el plan expandiendo los `paths` de cada app contra el home.

    Los directorios se recorren recursivamente en orden determinista, los
    enlaces simbólicos se guardan como enlace sin seguirlos, lo excluido se
    descarta en silencio y las rutas inexistentes van a `skipped`.
    """
    root = home if home is not None else Path.home()
    operations: list[Operation] = []
    skipped: list[str] = []
    for app in apps:
        for entry in app.paths:
            source = to_absolute(entry, home=root)
            _expand(app.id, source, root, app.exclude, operations, skipped)
    return Plan(operations=operations, skipped=skipped)


def _expand(
    app_id: str,
    source: Path,
    root: Path,
    exclude: Sequence[str],
    operations: list[Operation],
    skipped: list[str],
) -> None:
    """Expande una entrada de `paths`: archivo, enlace o directorio recursivo."""
    rel = to_relative(source, home=root).as_posix()
    if is_excluded(rel, exclude):
        return
    try:
        info = os.lstat(source)
    except FileNotFoundError:
        skipped.append(rel)
        return
    if stat.S_ISLNK(info.st_mode):
        _add_link(app_id, source, rel, info, operations, skipped)
    elif stat.S_ISDIR(info.st_mode):
        _walk(app_id, source, root, exclude, operations, skipped)
    elif stat.S_ISREG(info.st_mode):
        operations.append(_file_operation(app_id, source, rel, info))
    else:
        # tipos especiales (sockets, pipes...) no se pueden respaldar
        skipped.append(rel)


def _walk(
    app_id: str,
    source: Path,
    root: Path,
    exclude: Sequence[str],
    operations: list[Operation],
    skipped: list[str],
) -> None:
    """Recorre un directorio en orden determinista sin seguir enlaces."""
    for dirpath, dirnames, filenames in os.walk(source, followlinks=False):
        dirnames.sort()
        filenames.sort()
        base = Path(dirpath)
        rel_dir = to_relative(base, home=root).as_posix()
        subdirs: list[str] = []
        for name in dirnames:
            child = base / name
            child_rel = (PurePosixPath(rel_dir) / name).as_posix()
            if is_excluded(child_rel, exclude):
                continue
            if child.is_symlink():
                _add_link(app_id, child, child_rel, os.lstat(child), operations, skipped)
            else:
                subdirs.append(name)
        dirnames[:] = subdirs
        for name in filenames:
            child = base / name
            child_rel = (PurePosixPath(rel_dir) / name).as_posix()
            if is_excluded(child_rel, exclude):
                continue
            info = os.lstat(child)
            if stat.S_ISLNK(info.st_mode):
                _add_link(app_id, child, child_rel, info, operations, skipped)
            elif stat.S_ISREG(info.st_mode):
                operations.append(_file_operation(app_id, child, child_rel, info))
            else:
                skipped.append(child_rel)


def _file_operation(app_id: str, source: Path, rel: str, info: os.stat_result) -> Operation:
    """Crea la operación de un archivo regular con su checksum y tamaño."""
    return Operation(
        app_id=app_id,
        path=rel,
        source=source,
        size=info.st_size,
        sha256=_sha256_of(source),
        mode=f"{stat.S_IMODE(info.st_mode):04o}",
        link=None,
    )


def _add_link(
    app_id: str,
    source: Path,
    rel: str,
    info: os.stat_result,
    operations: list[Operation],
    skipped: list[str],
) -> None:
    """Registra un enlace simbólico sin seguirlo; si su destino es inseguro, lo omite."""
    target = os.readlink(source)
    if not _is_safe_link(target):
        skipped.append(rel)
        return
    operations.append(
        Operation(
            app_id=app_id,
            path=rel,
            source=source,
            size=0,
            sha256=None,
            mode=f"{stat.S_IMODE(info.st_mode):04o}",
            link=target,
        )
    )


def _is_safe_link(target: str) -> bool:
    """Un destino es seguro si es relativo y no contiene `..` (§6)."""
    if not target:
        return False
    pure = PurePosixPath(target)
    return not pure.is_absolute() and ".." not in pure.parts


def _sha256_of(source: Path) -> str:
    digest = hashlib.sha256()
    with source.open("rb") as handle:
        for chunk in iter(lambda: handle.read(65536), b""):
            digest.update(chunk)
    return digest.hexdigest()
