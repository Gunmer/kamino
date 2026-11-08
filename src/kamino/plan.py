"""Plan y Operation, compartidos por `clone` y `deploy`.

Principio (diseño técnico §1): planificar y luego ejecutar. `--dry-run` solo
imprime el plan, así simulación y ejecución comparten el mismo código.

El plan se construye expandiendo las rutas obligatorias, las opcionales y los
patrones acotados de cada app contra el home. Las lecturas usan `SafeHome`, que
rechaza enlaces intermedios. Las rutas obligatorias ausentes se registran como
omitidas; las opcionales ausentes no. Los enlaces finales se guardan como
enlace y no se siguen (§6). Ghostty tiene un descubrimiento acotado de
`config-file` y temas; el resto del catálogo es declarativo.
"""

from __future__ import annotations

import hashlib
import os
import posixpath
import stat
from collections.abc import Sequence
from dataclasses import dataclass, field
from fnmatch import fnmatch
from pathlib import Path, PurePosixPath

from kamino.catalog.models import AppDefinition
from kamino.errors import PathError
from kamino.ghostty import (
    CONFIG_FILENAMES,
    GHOSTTY_APP_ID,
    MAX_CONFIG_BYTES,
    config_reference,
    is_path_reference,
    iter_directives,
    resolve_reference,
    theme_references,
)
from kamino.paths import SafeHome, to_absolute, to_relative

#: Límites del descubrimiento específico de Ghostty.
MAX_GHOSTTY_FILES = 500
MAX_GHOSTTY_DEPTH = 12

#: Nombres que nunca se incluyen en referencias automáticas, aunque no estén
#: en el `exclude` de la definición. La lista no sustituye al cifrado.
SENSITIVE_PATTERNS = (
    "**/.env",
    "**/.env.*",
    "**/auth.json",
    "**/.credentials.json",
    "**/credentials.json",
    "**/secrets.yaml",
    "**/secrets.yml",
    "**/hosts.yml",
    "**/.ssh",
    "**/.ssh/**",
    "**/.aws/credentials",
    "**/id_rsa",
    "**/id_ed25519",
    "**/*.pem",
    "**/*.key",
    "**/sessions",
    "**/sessions/**",
    "**/history",
    "**/history/**",
    "**/*history*",
    "**/.cache",
    "**/.cache/**",
    "**/cache",
    "**/cache/**",
    "**/logs",
    "**/logs/**",
    "**/*.log",
    "**/*.sqlite",
    "**/*.sqlite3",
)

_GHOSTTY_THEME_DIRS = (
    ".config/ghostty/themes",
    "Library/Application Support/com.mitchellh.ghostty/themes",
)


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
    """El plan de una clonación: operaciones, rutas omitidas y avisos.

    `skipped` registra las rutas obligatorias ausentes, las opcionales
    existentes pero inseguras y las referencias que no se pueden incluir.
    `notices` comunica ciclos o límites sin cambiar por sí solo el código de
    salida.
    """

    operations: list[Operation] = field(default_factory=list)
    skipped: list[str] = field(default_factory=list)
    notices: list[str] = field(default_factory=list)


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


@dataclass
class _Builder:
    """Acumula operaciones, omisiones y avisos con deduplicación."""

    safe: SafeHome
    root: Path
    operations: list[Operation] = field(default_factory=list)
    skipped: list[str] = field(default_factory=list)
    notices: list[str] = field(default_factory=list)
    _operations: set[tuple[str, str]] = field(default_factory=set)
    _skipped: set[str] = field(default_factory=set)
    _notices: set[str] = field(default_factory=set)

    def add_operation(self, operation: Operation) -> None:
        key = (operation.app_id, operation.path)
        if key in self._operations:
            return
        self._operations.add(key)
        self.operations.append(operation)

    def skip(self, path: str) -> None:
        if path not in self._skipped:
            self._skipped.add(path)
            self.skipped.append(path)

    def notice(self, message: str) -> None:
        if message not in self._notices:
            self._notices.add(message)
            self.notices.append(message)

    def regular_file(self, app_id: str, path: str) -> Operation | None:
        for operation in self.operations:
            if (
                operation.app_id == app_id
                and operation.path == path
                and operation.sha256 is not None
            ):
                return operation
        return None


def build_plan(apps: Sequence[AppDefinition], *, home: Path | None = None) -> Plan:
    """Construye el plan expandiendo las entradas de cada app contra el home."""
    root = home if home is not None else Path.home()
    with SafeHome(root) as safe:
        builder = _Builder(safe=safe, root=root)
        for app in apps:
            _expand_app(app, builder)
        return Plan(
            operations=builder.operations,
            skipped=builder.skipped,
            notices=builder.notices,
        )


def _expand_app(app: AppDefinition, builder: _Builder) -> None:
    for entry in app.paths:
        _expand(app, entry, builder, required=True)
    for entry in app.optional_paths:
        _expand(app, entry, builder, required=False)
    for pattern in app.optional_patterns:
        _expand_pattern(app, pattern, builder)
    for name in app.env_paths:
        _expand_env(app, name, builder)
    for entry in app.xdg_paths:
        _expand_xdg(app, entry, builder)
    if app.id == GHOSTTY_APP_ID:
        _expand_ghostty(app, builder)


def _expand(
    app: AppDefinition,
    entry: str,
    builder: _Builder,
    *,
    required: bool,
    automatic: bool = False,
) -> None:
    """Expande una entrada: archivo, enlace o directorio recursivo."""
    rel = PurePosixPath(str(entry)).as_posix()
    SafeHome.parts(rel)
    if is_excluded(rel, app.exclude) or (automatic and is_excluded(rel, SENSITIVE_PATTERNS)):
        return
    try:
        info = builder.safe.lstat(rel)
    except FileNotFoundError:
        if required:
            builder.skip(rel)
        return
    except (PathError, NotADirectoryError, OSError):
        builder.skip(rel)
        return
    if stat.S_ISLNK(info.st_mode):
        _add_link(app.id, rel, info, builder)
    elif stat.S_ISDIR(info.st_mode):
        _walk(app, rel, builder)
    elif stat.S_ISREG(info.st_mode):
        _add_file(app.id, rel, info, builder)
    else:
        builder.skip(rel)


def _expand_pattern(app: AppDefinition, pattern: str, builder: _Builder) -> None:
    """Expande un patrón acotado listando su directorio fijo sin seguir enlaces."""
    parent = PurePosixPath(pattern).parent.as_posix()
    try:
        names = builder.safe.listdir(parent)
    except FileNotFoundError:
        return
    except (PathError, NotADirectoryError, OSError):
        builder.skip(parent)
        return
    for name in names:
        if fnmatch(name, PurePosixPath(pattern).name):
            _expand(app, f"{parent}/{name}", builder, required=False)


def _expand_env(app: AppDefinition, name: str, builder: _Builder) -> None:
    """Expande una ruta absoluta tomada de una variable de entorno, dentro del home."""
    value = os.environ.get(name)
    if value is None:
        return
    try:
        rel = to_relative(Path(value), home=builder.root).as_posix()
    except PathError:
        builder.skip(f"{name} fuera del home")
        return
    _expand(app, rel, builder, required=False)


def _expand_xdg(app: AppDefinition, entry: str, builder: _Builder) -> None:
    """Expande una ruta relativa a XDG_CONFIG_HOME cuando este queda en el home."""
    value = os.environ.get("XDG_CONFIG_HOME")
    if not value:
        return
    try:
        xdg = to_relative(Path(value), home=builder.root).as_posix()
    except PathError:
        builder.skip("XDG_CONFIG_HOME fuera del home")
        return
    _expand(app, f"{xdg}/{entry}", builder, required=False)


def _walk(app: AppDefinition, rel: str, builder: _Builder) -> None:
    """Recorre un directorio en orden determinista sin seguir enlaces."""

    def prune(child: str) -> bool:
        return is_excluded(child, app.exclude)

    for item in builder.safe.walk(rel, prune=prune):
        if item.kind == "file" and item.info is not None:
            _add_file(app.id, item.path, item.info, builder)
        elif item.kind == "link" and item.info is not None:
            _add_link(app.id, item.path, item.info, builder)
        else:
            builder.skip(item.path)


def _add_file(app_id: str, rel: str, info: os.stat_result, builder: _Builder) -> None:
    """Crea la operación de un archivo regular leyéndolo de forma segura."""
    try:
        data = builder.safe.read_bytes(rel)
    except (PathError, OSError):
        builder.skip(rel)
        return
    builder.add_operation(
        Operation(
            app_id=app_id,
            path=rel,
            source=to_absolute(rel, home=builder.root),
            size=len(data),
            sha256=hashlib.sha256(data).hexdigest(),
            mode=f"{stat.S_IMODE(info.st_mode):04o}",
            link=None,
        )
    )


def _add_link(app_id: str, rel: str, info: os.stat_result, builder: _Builder) -> None:
    """Registra un enlace simbólico sin seguirlo; si su destino es inseguro, lo omite."""
    try:
        target = builder.safe.readlink(rel)
    except (PathError, OSError):
        builder.skip(rel)
        return
    if not _is_safe_link(target):
        builder.skip(rel)
        return
    builder.add_operation(
        Operation(
            app_id=app_id,
            path=rel,
            source=to_absolute(rel, home=builder.root),
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


def _expand_ghostty(app: AppDefinition, builder: _Builder) -> None:
    """Descubre `config-file` y temas de Ghostty sin ejecutar su contenido."""
    theme_dirs = _ghostty_theme_dirs(builder)
    visited: set[str] = set()
    stack: list[str] = []
    processed = 0

    initial = [
        operation.path
        for operation in builder.operations
        if operation.app_id == app.id
        and operation.sha256 is not None
        and PurePosixPath(operation.path).name in CONFIG_FILENAMES
    ]

    def parse(rel: str, depth: int) -> None:
        nonlocal processed
        if rel in visited:
            return
        visited.add(rel)
        if processed >= MAX_GHOSTTY_FILES:
            builder.notice("se alcanzó el límite de archivos de configuración de Ghostty")
            return
        if depth > MAX_GHOSTTY_DEPTH:
            builder.notice(f"se alcanzó el límite de profundidad en {rel}")
            return
        try:
            text = builder.safe.read_text(rel, limit=MAX_CONFIG_BYTES)
        except (PathError, OSError):
            return
        processed += 1
        stack.append(rel)
        try:
            for key, value in iter_directives(text):
                if key == "config-file":
                    raw, optional = config_reference(value)
                    _follow_config_file(
                        app, raw, rel, depth, optional, builder, visited, stack, parse
                    )
                elif key == "theme":
                    for raw in theme_references(value):
                        _follow_theme(app, raw, rel, builder, theme_dirs)
        finally:
            stack.pop()

    for rel in initial:
        parse(rel, 0)


def _follow_config_file(
    app: AppDefinition,
    raw: str,
    base_rel: str,
    depth: int,
    optional: bool,
    builder: _Builder,
    visited: set[str],
    stack: list[str],
    parse,
) -> None:
    if not raw:
        return
    resolved = resolve_reference(raw, base_rel, builder.root)
    if resolved is None:
        builder.skip(raw)
        builder.notice(f"referencia de Ghostty fuera del home: {raw}")
        return
    if _blocked_reference(app, resolved):
        builder.skip(resolved)
        builder.notice(f"referencia de Ghostty excluida por seguridad: {resolved}")
        return
    if resolved in stack:
        builder.notice(f"referencia circular de Ghostty en {resolved}")
        return
    _expand(app, resolved, builder, required=not optional, automatic=True)
    included = builder.regular_file(app.id, resolved) is not None
    if included:
        parse(resolved, depth + 1)
    elif not optional:
        builder.skip(resolved)


def _follow_theme(
    app: AppDefinition,
    raw: str,
    base_rel: str,
    builder: _Builder,
    theme_dirs: Sequence[str],
) -> None:
    if not raw:
        return
    if is_path_reference(raw):
        resolved = resolve_reference(raw, base_rel, builder.root)
        if resolved is None:
            builder.skip(raw)
            builder.notice(f"tema de Ghostty fuera del home: {raw}")
            return
        if _blocked_reference(app, resolved):
            builder.skip(resolved)
            builder.notice(f"tema de Ghostty excluido por seguridad: {resolved}")
            return
        _expand(app, resolved, builder, required=True, automatic=True)
        return
    for directory in theme_dirs:
        candidate = posixpath.join(directory, raw)
        try:
            info = builder.safe.lstat(candidate)
        except FileNotFoundError:
            continue
        except (PathError, NotADirectoryError, OSError):
            builder.skip(candidate)
            builder.notice(f"tema de Ghostty inseguro: {candidate}")
            return
        if stat.S_ISREG(info.st_mode):
            _expand(app, candidate, builder, required=False, automatic=True)
        return


def _blocked_reference(app: AppDefinition, rel: str) -> bool:
    return is_excluded(rel, app.exclude) or is_excluded(rel, SENSITIVE_PATTERNS)


def _ghostty_theme_dirs(builder: _Builder) -> list[str]:
    directories: list[str] = []
    xdg = os.environ.get("XDG_CONFIG_HOME")
    if xdg:
        try:
            rel = to_relative(Path(xdg), home=builder.root).as_posix()
        except PathError:
            rel = ""
        if rel:
            directories.append(posixpath.join(rel, "ghostty", "themes"))
    for directory in _GHOSTTY_THEME_DIRS:
        if directory not in directories:
            directories.append(directory)
    return directories
