"""Carga del catálogo embebido y del directorio del usuario.

El catálogo viene embebido en el paquete (`catalog/apps/*.toml`) y se complementa
con `~/.config/kamino/apps/`, donde se añaden o sobrescriben definiciones sin tocar
el código (diseño técnico §6).
"""

from __future__ import annotations

import tomllib
from importlib.resources import files as resource_files
from pathlib import Path
from typing import TYPE_CHECKING

from pydantic import ValidationError

from kamino.catalog.models import AppDefinition
from kamino.errors import CatalogError

if TYPE_CHECKING:
    from importlib.abc import Traversable


def load_catalog(*, user_dir: Path | None = None) -> dict[str, AppDefinition]:
    """Devuelve el catálogo embebido complementado con las definiciones del usuario.

    Las definiciones del usuario sobrescriben las embebidas por `id` o añaden
    aplicaciones nuevas. Un archivo TOML inválido o con una definición inválida
    lanza `CatalogError` nombrando el archivo.
    """
    catalog = _load_embedded()

    directory = user_dir if user_dir is not None else _default_user_dir()
    user_catalog: dict[str, AppDefinition] = {}
    for path in _iter_toml_files(directory):
        definition = _load_definition(path)
        if definition.id in user_catalog:
            raise CatalogError(f"id duplicado {definition.id!r} en el catálogo del usuario")
        user_catalog[definition.id] = definition

    catalog.update(user_catalog)
    return catalog


def _default_user_dir() -> Path:
    return Path.home() / ".config" / "kamino" / "apps"


def _load_embedded() -> dict[str, AppDefinition]:
    apps_dir = resource_files("kamino.catalog").joinpath("apps")
    try:
        entries = list(apps_dir.iterdir())
    except OSError as exc:
        raise CatalogError(f"no se pudo leer el catálogo embebido: {exc}") from exc
    catalog: dict[str, AppDefinition] = {}
    for path in entries:
        if not path.name.endswith(".toml"):
            continue
        definition = _load_definition(path)
        if definition.id in catalog:
            raise CatalogError(f"id duplicado {definition.id!r} en el catálogo embebido")
        catalog[definition.id] = definition
    return catalog


def _iter_toml_files(directory: Path) -> list[Path]:
    try:
        entries = list(directory.iterdir())
    except FileNotFoundError:
        return []
    except OSError as exc:
        raise CatalogError(
            f"no se pudo leer el catálogo del usuario {str(directory)!r}: {exc}"
        ) from exc
    return sorted(path for path in entries if path.is_file() and path.suffix == ".toml")


def _load_definition(path: Traversable | Path) -> AppDefinition:
    try:
        text = path.read_text(encoding="utf-8")
    except OSError as exc:
        raise CatalogError(f"no se pudo leer {path.name!r}: {exc}") from exc
    except UnicodeDecodeError as exc:
        raise CatalogError(f"{path.name!r} no es UTF-8 válido: {exc}") from exc

    try:
        data = tomllib.loads(text)
    except tomllib.TOMLDecodeError as exc:
        raise CatalogError(f"TOML inválido en {path.name!r}: {exc}") from exc

    try:
        return AppDefinition.model_validate(data)
    except ValidationError as exc:
        raise CatalogError(f"definición inválida en {path.name!r}: {exc}") from exc
