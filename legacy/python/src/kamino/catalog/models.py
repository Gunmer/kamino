"""Modelos del catálogo: validación de las definiciones TOML.

Esquema de una definición (diseño técnico §4.1):

- `id`: único, en minúsculas y guiones; es el nombre que se usa en `--apps`.
- `name`: nombre legible de la aplicación.
- `bundle_id`: detecta apps con `.app` por su `CFBundleIdentifier`.
- `command` y `command_aliases`: detectan herramientas de terminal sin `.app`.
- `search_paths`: rutas adicionales donde buscar el binario. Opcional.
- `install_marker`: archivo relativo al home que también debe existir para
  considerar instalada la app (por ejemplo, el inicio de Oh My Zsh).
- `paths`: rutas obligatorias relativas al home. Su ausencia se informa como
  omisión para conservar la semántica de las definiciones anteriores.
- `optional_paths`: rutas que solo se incluyen si existen; su ausencia no es
  una omisión.
- `optional_patterns`: patrones acotados a un directorio fijo. Solo el último
  componente puede contener comodines.
- `env_paths`: nombres de variables de entorno con una ruta absoluta que debe
  quedar dentro del home.
- `xdg_paths`: rutas relativas a `XDG_CONFIG_HOME`, admitidas solo si ese
  directorio queda dentro del home.
- `exclude`: patrones glob descartados al expandir las rutas. Opcional.

Una definición necesita al menos una ruta obligatoria, opcional o un patrón.
"""

from __future__ import annotations

import re
from pathlib import PurePosixPath

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

_ID_PATTERN = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
_ENV_PATTERN = re.compile(r"^[A-Z_][A-Z0-9_]*$")
_WILDCARDS = frozenset("*?[")


class AppDefinition(BaseModel):
    """Definición validada de una aplicación del catálogo."""

    model_config = ConfigDict(extra="forbid")

    id: str
    name: str
    bundle_id: str | None = None
    command: str | None = None
    command_aliases: list[str] = Field(default_factory=list)
    search_paths: list[str] = Field(default_factory=list)
    install_marker: str | None = None
    paths: list[str] = Field(default_factory=list)
    optional_paths: list[str] = Field(default_factory=list)
    optional_patterns: list[str] = Field(default_factory=list)
    env_paths: list[str] = Field(default_factory=list)
    xdg_paths: list[str] = Field(default_factory=list)
    exclude: list[str] = Field(default_factory=list)

    @field_validator("id")
    @classmethod
    def _id_must_be_a_slug(cls, value: str) -> str:
        if not _ID_PATTERN.fullmatch(value):
            raise ValueError("'id' debe usar minúsculas y guiones (ej. 'herdr')")
        return value

    @field_validator("name")
    @classmethod
    def _name_must_not_be_blank(cls, value: str) -> str:
        if not value.strip():
            raise ValueError("'name' no puede estar vacío")
        return value

    @field_validator("paths", "optional_paths", "xdg_paths")
    @classmethod
    def _paths_must_be_relative(cls, value: list[str]) -> list[str]:
        for path in value:
            _ensure_relative(path)
        return value

    @field_validator("optional_patterns")
    @classmethod
    def _patterns_must_be_bounded(cls, value: list[str]) -> list[str]:
        for pattern in value:
            _ensure_pattern(pattern)
        return value

    @field_validator("install_marker")
    @classmethod
    def _marker_must_be_relative(cls, value: str | None) -> str | None:
        if value is not None:
            _ensure_relative(value)
        return value

    @field_validator("env_paths")
    @classmethod
    def _env_names_must_be_valid(cls, value: list[str]) -> list[str]:
        for name in value:
            if not _ENV_PATTERN.fullmatch(name):
                raise ValueError(f"nombre de variable de entorno no válido: {name!r}")
        return value

    @field_validator("command_aliases")
    @classmethod
    def _aliases_must_not_be_blank(cls, value: list[str]) -> list[str]:
        for alias in value:
            if not alias.strip():
                raise ValueError("'command_aliases' no admite entradas vacías")
        return value

    @field_validator("bundle_id", "command")
    @classmethod
    def _identifier_must_not_be_blank(cls, value: str | None) -> str | None:
        if value is not None and not value.strip():
            raise ValueError("no puede estar vacío")
        return value

    @model_validator(mode="after")
    def _must_have_detection_and_entries(self) -> AppDefinition:
        if self.bundle_id is None and self.command is None:
            raise ValueError("se requiere 'bundle_id' o 'command' para detectar la app")
        if not (self.paths or self.optional_paths or self.optional_patterns):
            raise ValueError(
                "se requiere al menos una entrada en 'paths', 'optional_paths' "
                "o 'optional_patterns'"
            )
        return self


def _ensure_relative(path: str) -> None:
    """Rechaza rutas que escapan del home en una definición de catálogo."""
    if not path:
        raise ValueError("no se admiten rutas vacías")
    if path.startswith("~"):
        raise ValueError(f"ruta con '~' no admitida, usa una ruta relativa al home: {path!r}")
    pure = PurePosixPath(path)
    if pure.is_absolute():
        raise ValueError(f"ruta absoluta no admitida: {path!r}")
    if ".." in pure.parts:
        raise ValueError(f"ruta fuera del home no admitida: {path!r}")


def _ensure_pattern(pattern: str) -> None:
    """Valida un patrón anclado a un directorio fijo dentro del home."""
    _ensure_relative(pattern)
    if "**" in pattern:
        raise ValueError(f"patrón recursivo no admitido: {pattern!r}")
    parts = PurePosixPath(pattern).parts
    if len(parts) < 2:
        raise ValueError(f"el patrón debe incluir un directorio fijo: {pattern!r}")
    for part in parts[:-1]:
        if any(character in part for character in _WILDCARDS):
            raise ValueError(f"solo el último componente puede contener comodines: {pattern!r}")
    if not any(character in parts[-1] for character in _WILDCARDS):
        raise ValueError(f"el último componente debe contener un comodín: {pattern!r}")
