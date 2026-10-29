"""Modelos del catálogo: validación de las definiciones TOML.

Esquema de una definición (diseño técnico §4.1):

- `id`: único, en minúsculas y guiones; es el nombre que se usa en `--apps`.
- `name`: nombre legible de la aplicación.
- `bundle_id`: detecta apps con `.app` por su `CFBundleIdentifier`.
- `command`: detecta herramientas de terminal sin `.app` (ej. `git`).
- `search_paths`: rutas adicionales donde buscar el binario de `command`. Opcional.
- `paths`: rutas relativas al home, archivos o carpetas. No se admiten fuera del home.
- `exclude`: patrones glob descartados al expandir `paths`. Opcional.
"""

from __future__ import annotations

import re
from pathlib import PurePosixPath

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

_ID_PATTERN = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")


class AppDefinition(BaseModel):
    """Definición validada de una aplicación del catálogo."""

    model_config = ConfigDict(extra="forbid")

    id: str
    name: str
    bundle_id: str | None = None
    command: str | None = None
    search_paths: list[str] = Field(default_factory=list)
    paths: list[str]
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

    @field_validator("paths")
    @classmethod
    def _paths_must_be_relative(cls, value: list[str]) -> list[str]:
        if not value:
            raise ValueError("'paths' no puede estar vacío")
        for path in value:
            _ensure_relative(path)
        return value

    @model_validator(mode="after")
    def _must_have_detection_mechanism(self) -> AppDefinition:
        if not self.bundle_id and not self.command:
            raise ValueError("se requiere 'bundle_id' o 'command' para detectar la app")
        return self


def _ensure_relative(path: str) -> None:
    """Rechaza rutas que escapan del home en una definición de catálogo."""
    if not path:
        raise ValueError("'paths' no admite rutas vacías")
    if path.startswith("~"):
        raise ValueError(f"ruta con '~' no admitida, usa una ruta relativa al home: {path!r}")
    pure = PurePosixPath(path)
    if pure.is_absolute():
        raise ValueError(f"ruta absoluta no admitida: {path!r}")
    if ".." in pure.parts:
        raise ValueError(f"ruta fuera del home no admitida: {path!r}")
