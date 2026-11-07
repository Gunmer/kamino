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


def _fake_executable(directory: Path, name: str) -> None:
    directory.mkdir(parents=True, exist_ok=True)
    binary = directory / name
    binary.write_text("#!/bin/sh\nexit 99\n", encoding="utf-8")
    binary.chmod(0o755)


def _write(home: Path, rel: str, content: bytes | str = b"config\n") -> None:
    path = home / rel
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(content if isinstance(content, bytes) else content.encode())


def test_agent_herdr_zsh_and_ghostty_cycle(tmp_path: Path) -> None:
    catalog = load_catalog(user_dir=tmp_path / "no-user-catalog")
    origin = tmp_path / "origin"
    target = tmp_path / "target"
    target.mkdir()
    bin_dir = tmp_path / "bin"
    apps_dir = tmp_path / "Applications"
    _fake_app(apps_dir, "com.mitchellh.ghostty")
    for command in ("opencode", "herdr", "zsh"):
        _fake_executable(bin_dir, command)

    _write(origin, ".config/opencode/opencode.json", '{"theme": "mi-tema"}\n')
    _write(origin, ".config/opencode/cli.json", '{"keybinds": {}}\n')
    _write(origin, ".config/opencode/themes/mi-tema.json", '{"name": "mi-tema"}\n')
    herdr_config = (
        '[theme]\nname = "oscuro"\n[theme.custom]\nbackground = "#000000"\n'
        '[keys]\nquit = "ctrl+q"\n[[keys.command]]\nkey = "ctrl+t"\ncommand = "new-tab"\n'
    )
    _write(origin, ".config/herdr/config.toml", herdr_config)
    _write(origin, ".config/herdr/plugins/config/settings.toml", "plugin = true\n")
    _write(origin, ".config/herdr/plugins/config/.env", "TOKEN=secreto\n")
    _write(origin, ".oh-my-zsh/oh-my-zsh.sh", "# framework\n")
    _write(origin, ".zshrc", 'export ZSH="$HOME/.oh-my-zsh"\n')
    _write(origin, ".oh-my-zsh/custom/themes/mi.zsh-theme", "PROMPT='%n'\n")
    _write(origin, ".oh-my-zsh/custom/plugins/foo/foo.plugin.zsh", "echo plugin\n")
    _write(origin, ".oh-my-zsh/custom/plugins/foo/.git/config", "[core]\n")
    _write(origin, ".config/ghostty/config", "theme = mi-tema\nconfig-file = extra.conf\n")
    _write(origin, ".config/ghostty/extra.conf", "font-size = 14\n")
    _write(origin, ".config/ghostty/themes/mi-tema", "background = #ffffff\n")
    _write(
        origin,
        "Library/Application Support/com.mitchellh.ghostty/config",
        "font-size = 14\n",
    )
    _write(target, ".oh-my-zsh/oh-my-zsh.sh", "# framework\n")

    backup = tmp_path / "backup.tar.gz"
    backup_result = run_backup(
        home=origin,
        catalog=catalog,
        application_dirs=[apps_dir],
        path=str(bin_dir),
        output=backup,
    )

    assert backup_result.manifest.apps
    manifest_paths = {
        (app.id, entry.path) for app in backup_result.manifest.apps for entry in app.files
    }
    assert ("opencode", ".config/opencode/opencode.json") in manifest_paths
    assert ("herdr", ".config/herdr/config.toml") in manifest_paths
    assert ("ghostty", ".config/ghostty/extra.conf") in manifest_paths
    assert ("ghostty", ".config/ghostty/themes/mi-tema") in manifest_paths
    assert ("oh-my-zsh", ".zshrc") in manifest_paths
    assert ("herdr", ".config/herdr/plugins/config/.env") not in manifest_paths
    assert all("plugins/foo" not in path for _app_id, path in manifest_paths)
    assert not backup_result.plan.skipped

    result = run_restore(
        backup,
        home=target,
        catalog=catalog,
        application_dirs=[apps_dir],
        path=str(bin_dir),
        snapshots_dir=tmp_path / "snapshots",
    )

    assert (target / ".config/herdr/config.toml").read_text() == herdr_config
    assert (target / ".config/opencode/themes/mi-tema.json").exists()
    assert (target / ".config/ghostty/extra.conf").read_text() == "font-size = 14\n"
    assert not (target / ".config/herdr/plugins/config/.env").exists()
    assert not (target / ".oh-my-zsh/custom/plugins/foo/foo.plugin.zsh").exists()
    assert result.restored


def test_uninstalled_new_tool_is_not_backed_up(tmp_path: Path, monkeypatch) -> None:
    catalog = load_catalog(user_dir=tmp_path / "no-user-catalog")
    origin = tmp_path / "origin"
    _write(origin, ".config/navi/config.yaml", "theme: default\n")
    empty_bin = tmp_path / "empty-bin"
    empty_bin.mkdir()

    result = run_backup(
        home=origin,
        catalog=catalog,
        application_dirs=[],
        path=str(empty_bin),
        selected=None,
        output=tmp_path / "backup.tar.gz",
    )

    assert result.manifest.apps == []
