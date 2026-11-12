"""Excepciones propias del dominio, sin lógica de interfaz."""

from __future__ import annotations

from enum import StrEnum


class KaminoError(Exception):
    """Error base de kamino."""


class OperationCancelled(KaminoError):
    """La persona usuaria canceló una operación cooperativa."""


class CatalogError(KaminoError):
    """Una definición del catálogo es inválida o no se puede cargar."""


class ConfigError(KaminoError):
    """La configuración de la TUI es inválida o no se puede leer/escribir."""


class DetectionError(KaminoError):
    """No se pudo detectar si una app está instalada o su versión."""


class ArchiveError(KaminoError):
    """No se pudo leer o escribir un archivo del clon."""


class IntegrityError(KaminoError):
    """Un checksum no coincide o el archivo está dañado."""


class VersionError(KaminoError):
    """La versión del formato del clon no es soportada."""


class PathError(KaminoError):
    """Una ruta sale de los límites permitidos (home o archivo)."""


class UpdateErrorKind(StrEnum):
    """Categoría de un fallo de actualización, para elegir la acción contextual."""

    NOT_SELF_CONTAINED = "not_self_contained"
    PLATFORM = "platform"
    NETWORK = "network"
    INTEGRITY = "integrity"
    REPLACE = "replace"
    OTHER = "other"


class UpdateError(KaminoError):
    """La actualización del binario no se pudo completar."""

    def __init__(self, message: str, *, kind: UpdateErrorKind = UpdateErrorKind.OTHER):
        super().__init__(message)
        self.kind = kind


class ConflictError(KaminoError):
    """A deploy conflict cannot be resolved with the given decisions."""


class DecisionRequiredError(ConflictError):
    """At least one conflict has no decision."""


class InvalidMergeError(ConflictError):
    """A composed file is invalid and both originals were valid."""


class ConcurrentChangeError(ConflictError):
    """A destination changed between planning and applying."""


class PartialApplyError(KaminoError):
    """Algunas escrituras se aplicaron y otra falló después del holocrón."""

    def __init__(self, message: str, *, applied: list[str], failed: str, snapshot=None):
        super().__init__(message)
        self.applied = applied
        self.failed = failed
        self.snapshot = snapshot
