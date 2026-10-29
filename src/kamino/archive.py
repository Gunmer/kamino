"""Lectura y escritura del archivo `tar.gz`, manifiesto y checksums.

Estructura del archivo (diseño técnico §4.3): `manifest.json` y `apps/<id>/...`.
La extracción nunca usa `extractall` sin control: rechaza rutas absolutas y `..`,
y usa `filter="data"` de `tarfile`.

Pendiente: paso 4 del plan de implementación.
"""

from __future__ import annotations

from typing import NoReturn


def write_archive() -> NoReturn:
    """Escribe el archivo de respaldo y su manifiesto (pendiente)."""
    raise NotImplementedError("archive.write_archive: pendiente de implementar (paso 4).")


def read_archive() -> NoReturn:
    """Lee y verifica un archivo de respaldo (pendiente)."""
    raise NotImplementedError("archive.read_archive: pendiente de implementar (paso 4).")
