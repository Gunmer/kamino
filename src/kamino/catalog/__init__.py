"""Catálogo declarativo de aplicaciones soportadas."""

from __future__ import annotations

from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition

__all__ = ["AppDefinition", "load_catalog"]
