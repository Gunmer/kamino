"""Excepciones propias del dominio, sin lógica de interfaz."""

from __future__ import annotations


class KaminoError(Exception):
    """Error base de kamino."""


class CatalogError(KaminoError):
    """Una definición del catálogo es inválida o no se puede cargar."""


class DetectionError(KaminoError):
    """No se pudo detectar si una app está instalada o su versión."""


class ArchiveError(KaminoError):
    """No se pudo leer o escribir un archivo de respaldo."""


class IntegrityError(KaminoError):
    """Un checksum no coincide o el archivo está dañado."""


class VersionError(KaminoError):
    """La versión del formato del respaldo no es soportada."""


class PathError(KaminoError):
    """Una ruta sale de los límites permitidos (home o archivo)."""
