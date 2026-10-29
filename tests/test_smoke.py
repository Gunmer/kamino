"""Pruebas de humo del esqueleto: el paquete importa y la versión existe."""

from __future__ import annotations

import importlib

import kamino


def test_version_is_defined() -> None:
    assert kamino.__version__


def test_errors_hierarchy() -> None:
    from kamino.errors import CatalogError, KaminoError

    assert issubclass(CatalogError, KaminoError)


def test_all_modules_import() -> None:
    modules = [
        "kamino.cli",
        "kamino.ui",
        "kamino.backup",
        "kamino.restore",
        "kamino.plan",
        "kamino.detect",
        "kamino.archive",
        "kamino.paths",
        "kamino.snapshot",
        "kamino.errors",
        "kamino.catalog",
        "kamino.catalog.models",
        "kamino.catalog.loader",
    ]
    for name in modules:
        importlib.import_module(name)
