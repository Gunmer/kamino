"""Pruebas de las rutas por defecto compartidas."""

from __future__ import annotations

from pathlib import Path

from kamino.defaults import catalog_dir, snapshots_dir, vault_dir


def test_default_paths_are_under_config(tmp_path: Path) -> None:
    expected_vault = tmp_path / ".config" / "kamino" / "vault"

    assert catalog_dir(tmp_path) == tmp_path / ".config" / "kamino" / "apps"
    assert vault_dir(tmp_path) == expected_vault
    assert snapshots_dir(tmp_path) == expected_vault / "snapshot"
