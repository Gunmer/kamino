"""Pruebas de integración: ciclo completo respaldo y restauración (§7).

Usa dos HOME falsos y una app `.app` simulada con un `Info.plist` falso para
comprobar que solo se restauran las apps instaladas, sin tocar el sistema real.
"""

from __future__ import annotations

import plistlib
from pathlib import Path

from kamino.backup import run_backup
from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.restore import run_restore


def _app() -> AppDefinition:
    return AppDefinition.model_validate(
        {
            "id": "foo",
            "name": "Foo",
            "bundle_id": "com.example.foo",
            "paths": [".config/foo/config.toml"],
        }
    )


def _fake_app(apps_dir: Path, bundle_id: str) -> None:
    bundle = apps_dir / "Foo.app" / "Contents"
    bundle.mkdir(parents=True)
    with (bundle / "Info.plist").open("wb") as handle:
        plistlib.dump({"CFBundleIdentifier": bundle_id}, handle)


def _write_origin_config(origin: Path) -> None:
    config = origin / ".config" / "foo" / "config.toml"
    config.parent.mkdir(parents=True)
    config.write_text("a = 1\n")


def test_backup_restore_cycle(tmp_path: Path) -> None:
    app = _app()
    catalog = {"foo": app}

    origin = tmp_path / "origin"
    _write_origin_config(origin)
    apps_dir = tmp_path / "Applications"
    _fake_app(apps_dir, "com.example.foo")

    backup = tmp_path / "backup.tar.gz"
    run_backup(
        home=origin,
        catalog=catalog,
        application_dirs=[apps_dir],
        selected=["foo"],
        output=backup,
    )

    target = tmp_path / "target"
    target.mkdir()
    result = run_restore(backup, home=target, catalog=catalog, application_dirs=[apps_dir])

    assert (target / ".config" / "foo" / "config.toml").read_text() == "a = 1\n"
    assert ".config/foo/config.toml" in result.restored


def test_restore_skips_uninstalled_app(tmp_path: Path) -> None:
    app = _app()
    catalog = {"foo": app}

    origin = tmp_path / "origin"
    _write_origin_config(origin)
    apps_dir = tmp_path / "Applications"
    _fake_app(apps_dir, "com.example.foo")

    backup = tmp_path / "backup.tar.gz"
    run_backup(
        home=origin,
        catalog=catalog,
        application_dirs=[apps_dir],
        selected=["foo"],
        output=backup,
    )

    target = tmp_path / "target"
    target.mkdir()
    empty_apps = tmp_path / "empty-apps"
    empty_apps.mkdir()
    result = run_restore(backup, home=target, catalog=catalog, application_dirs=[empty_apps])

    assert not (target / ".config" / "foo" / "config.toml").exists()
    assert result.restored == []


def test_embedded_catalog_has_initial_apps(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("HOME", str(tmp_path))
    catalog = load_catalog()
    assert {"git", "ghostty", "herdr", "yazi"} <= set(catalog)
