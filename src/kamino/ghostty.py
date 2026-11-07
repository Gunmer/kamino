"""Lectura acotada del formato de configuración de Ghostty.

Ghostty permite repartir la configuración en archivos referenciados con
`config-file` y seleccionar temas por nombre o por ruta. Estas utilidades solo
interpretan texto; nunca ejecutan Ghostty ni un shell. La resolución de rutas
la decide `plan.py` con el acceso seguro al home.
"""

from __future__ import annotations

import posixpath
from collections.abc import Iterator
from pathlib import Path, PurePosixPath

from kamino.errors import PathError
from kamino.paths import to_relative

GHOSTTY_APP_ID = "ghostty"
CONFIG_FILENAMES = frozenset({"config", "config.ghostty"})
MAX_CONFIG_BYTES = 1024 * 1024


def iter_directives(text: str) -> Iterator[tuple[str, str]]:
    """Itera pares `clave = valor` ignorando comentarios y líneas vacías."""
    for raw in text.splitlines():
        line = _strip_comment(raw).strip()
        if not line or "=" not in line:
            continue
        key, value = line.split("=", 1)
        yield key.strip(), value.strip()


def config_reference(value: str) -> tuple[str, bool]:
    """Devuelve la ruta de `config-file` y si estaba marcada como opcional con `?`."""
    raw = _unquote(value.strip())
    optional = raw.startswith("?")
    if optional:
        raw = _unquote(raw[1:].strip())
    return raw, optional


def theme_references(value: str) -> list[str]:
    """Devuelve los temas de un valor simple o de un par claro/oscuro."""
    result: list[str] = []
    for part in value.split(","):
        candidate = part.strip()
        if ":" in candidate:
            prefix, rest = candidate.split(":", 1)
            if prefix.strip() in {"light", "dark"}:
                candidate = rest.strip()
        candidate = _unquote(candidate)
        if candidate:
            result.append(candidate)
    return result


def is_path_reference(value: str) -> bool:
    """Indica si un valor de tema parece una ruta y no un nombre integrado."""
    return PurePosixPath(value).is_absolute() or "/" in value


def resolve_reference(raw: str, base_rel: str, home: Path) -> str | None:
    """Resuelve una referencia a una ruta relativa POSIX dentro del home.

    Devuelve `None` si la referencia es absoluta fuera del home, empieza por
    `~` o intenta escapar con `..`. No comprueba enlaces: el acceso seguro lo
    hará antes de abrir el archivo.
    """
    candidate = _unquote(raw.strip())
    if not candidate or candidate.startswith("~"):
        return None
    if PurePosixPath(candidate).is_absolute():
        try:
            return to_relative(Path(candidate), home=home).as_posix()
        except PathError:
            return None
    base = posixpath.dirname(base_rel)
    resolved = posixpath.normpath(posixpath.join(base, candidate))
    if resolved in {"", ".", ".."} or resolved.startswith("../"):
        return None
    return resolved


def absolute_references(text: str) -> list[str]:
    """Devuelve las referencias absolutas declaradas en un archivo de Ghostty."""
    references: list[str] = []
    for key, value in iter_directives(text):
        if key == "config-file":
            raw, _optional = config_reference(value)
            if PurePosixPath(raw).is_absolute():
                references.append(raw)
        elif key == "theme":
            for raw in theme_references(value):
                if PurePosixPath(raw).is_absolute():
                    references.append(raw)
    return references


def _strip_comment(line: str) -> str:
    quote: str | None = None
    for index, character in enumerate(line):
        if quote is not None:
            if character == quote:
                quote = None
            continue
        if character in {"'", '"'}:
            quote = character
        elif character == "#":
            return line[:index]
    return line


def _unquote(value: str) -> str:
    if len(value) >= 2 and value[0] == value[-1] and value[0] in {"'", '"'}:
        return value[1:-1]
    return value
