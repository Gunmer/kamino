"""Rutas por defecto compartidas por la CLI y la TUI."""

from __future__ import annotations

from pathlib import Path


def catalog_dir(home: Path | None = None) -> Path:
    """Directorio por defecto de las definiciones de apps del usuario."""
    return _home(home) / ".config" / "kamino" / "apps"


def vault_dir(home: Path | None = None) -> Path:
    """Directorio por defecto de clones y holocrones."""
    return _home(home) / ".config" / "kamino" / "vault"


def snapshots_dir(home: Path | None = None) -> Path:
    """Directorio por defecto de los holocrones de seguridad."""
    return vault_dir(home) / "snapshot"


def _home(home: Path | None) -> Path:
    return home if home is not None else Path.home()
