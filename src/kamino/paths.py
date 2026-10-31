"""Normalización de rutas relativas al home y validación.

Todo se guarda relativo a `~` (diseño técnico §1 y §4.1), lo que resuelve la
migración entre usuarios. Las rutas fuera del home no se admiten en la v1.

La validación sigue la misma semántica que `_ensure_relative` en
`catalog/models.py`: se rechazan rutas vacías, absolutas, con `..` o que
empiezan por `~`.
"""

from __future__ import annotations

import os
from pathlib import Path, PurePosixPath

from kamino.errors import PathError


def to_absolute(path: str | Path, *, home: Path | None = None) -> Path:
    """Resuelve una ruta relativa del home a su forma absoluta.

    `home` por defecto es el home del usuario actual. Eleva `PathError` si la
    ruta está vacía, es absoluta, contiene `..` o empieza por `~`, porque
    cualquiera de esos casos saldría del home.
    """
    _ensure_relative(path)
    root = home if home is not None else Path.home()
    return root / str(path)


def to_relative(path: str | Path, *, home: Path | None = None) -> Path:
    """Convierte una ruta absoluta del home a su forma relativa.

    `home` por defecto es el home del usuario actual. Eleva `PathError` si la
    ruta no es absoluta o queda fuera del home (incluido el caso de un
    prefijo de texto como `/home/usuario2` frente a `/home/usuario`).
    """
    candidate = Path(os.path.normpath(str(path)))
    if not candidate.is_absolute():
        raise PathError(f"se requiere una ruta absoluta: {str(path)!r}")
    root = Path(os.path.normpath(str(home if home is not None else Path.home())))
    try:
        return candidate.relative_to(root)
    except ValueError as exc:
        raise PathError(f"ruta fuera del home {str(root)!r}: {str(path)!r}") from exc


def _ensure_relative(path: str | Path) -> None:
    """Rechaza rutas que escapan del home (misma semántica que el catálogo)."""
    text = str(path)
    if not text:
        raise PathError("ruta vacía no admitida")
    if text.startswith("~"):
        raise PathError(f"ruta con '~' no admitida, usa una ruta relativa al home: {text!r}")
    pure = PurePosixPath(text)
    if pure.is_absolute():
        raise PathError(f"ruta absoluta no admitida: {text!r}")
    if ".." in pure.parts:
        raise PathError(f"ruta fuera del home no admitida: {text!r}")
