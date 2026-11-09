"""Pruebas de integración: ciclo completo clonación y despliegue (§7).

Usa dos HOME falsos y una app `.app` simulada con un `Info.plist` falso para
comprobar que solo se despliegan las apps instaladas, sin tocar el sistema real.
"""

from __future__ import annotations

import plistlib
from pathlib import Path

from typer.testing import CliRunner

import kamino.cli as cli
from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.clone import run_clone
from kamino.conflicts import ConflictDecisions, FileAction, FileDecision, RowDecision
from kamino.deploy import prepare_deploy, run_deploy

runner = CliRunner()


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


def test_clone_deploy_cycle(tmp_path: Path) -> None:
    app = _app()
    catalog = {"foo": app}

    origin = tmp_path / "origin"
    _write_origin_config(origin)
    apps_dir = tmp_path / "Applications"
    _fake_app(apps_dir, "com.example.foo")

    clone = tmp_path / "clone.tar.gz"
    run_clone(
        home=origin,
        catalog=catalog,
        application_dirs=[apps_dir],
        selected=["foo"],
        output=clone,
    )

    target = tmp_path / "target"
    target.mkdir()
    result = run_deploy(clone, home=target, catalog=catalog, application_dirs=[apps_dir])

    assert (target / ".config" / "foo" / "config.toml").read_text() == "a = 1\n"
    assert ".config/foo/config.toml" in result.deployed


def test_deploy_skips_uninstalled_app(tmp_path: Path) -> None:
    app = _app()
    catalog = {"foo": app}

    origin = tmp_path / "origin"
    _write_origin_config(origin)
    apps_dir = tmp_path / "Applications"
    _fake_app(apps_dir, "com.example.foo")

    clone = tmp_path / "clone.tar.gz"
    run_clone(
        home=origin,
        catalog=catalog,
        application_dirs=[apps_dir],
        selected=["foo"],
        output=clone,
    )

    target = tmp_path / "target"
    target.mkdir()
    empty_apps = tmp_path / "empty-apps"
    empty_apps.mkdir()
    result = run_deploy(clone, home=target, catalog=catalog, application_dirs=[empty_apps])

    assert not (target / ".config" / "foo" / "config.toml").exists()
    assert result.deployed == []


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


def test_agent_herdr_zsh_and_ghostty_cycle(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.delenv("XDG_CONFIG_HOME", raising=False)
    monkeypatch.delenv("HERDR_CONFIG_PATH", raising=False)
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

    clone = tmp_path / "clone.tar.gz"
    clone_result = run_clone(
        home=origin,
        catalog=catalog,
        application_dirs=[apps_dir],
        path=str(bin_dir),
        output=clone,
    )

    assert clone_result.manifest.apps
    manifest_paths = {
        (app.id, entry.path) for app in clone_result.manifest.apps for entry in app.files
    }
    assert ("opencode", ".config/opencode/opencode.json") in manifest_paths
    assert ("herdr", ".config/herdr/config.toml") in manifest_paths
    assert ("ghostty", ".config/ghostty/extra.conf") in manifest_paths
    assert ("ghostty", ".config/ghostty/themes/mi-tema") in manifest_paths
    assert ("oh-my-zsh", ".zshrc") in manifest_paths
    assert ("herdr", ".config/herdr/plugins/config/.env") not in manifest_paths
    assert all("plugins/foo" not in path for _app_id, path in manifest_paths)
    assert not clone_result.plan.skipped

    result = run_deploy(
        clone,
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
    assert result.deployed


def test_uninstalled_new_tool_is_not_cloned(tmp_path: Path, monkeypatch) -> None:
    catalog = load_catalog(user_dir=tmp_path / "no-user-catalog")
    origin = tmp_path / "origin"
    _write(origin, ".config/navi/config.yaml", "theme: default\n")
    empty_bin = tmp_path / "empty-bin"
    empty_bin.mkdir()

    result = run_clone(
        home=origin,
        catalog=catalog,
        application_dirs=[],
        path=str(empty_bin),
        selected=None,
        output=tmp_path / "clone.tar.gz",
    )

    assert result.manifest.apps == []


# --- deploy conflicts end to end ---


def _conflict_setup(tmp_path: Path) -> tuple[Path, AppDefinition, Path, Path, str, str]:
    catalog = {
        "foo": AppDefinition.model_validate(
            {
                "id": "foo",
                "name": "Foo",
                "command": "foo",
                "paths": [".config/foo/config.conf"],
            }
        )
    }
    local = "".join(f"line {index}\n" for index in range(20))
    clone_text = local.replace("line 1\n", "LINE ONE\n").replace("line 18\n", "LINE EIGHTEEN\n")
    origin = tmp_path / "origin"
    _write(origin, ".config/foo/config.conf", clone_text)
    bin_dir = tmp_path / "bin"
    _fake_executable(bin_dir, "foo")
    clone = tmp_path / "clone.tar.gz"
    run_clone(
        home=origin,
        catalog=catalog,
        application_dirs=[],
        path=str(bin_dir),
        output=clone,
    )
    target = tmp_path / "target"
    _write(target, ".config/foo/config.conf", local)
    return clone, catalog["foo"], target, bin_dir, local, clone_text


def test_conflict_mix_snapshot_and_undo(tmp_path: Path) -> None:
    clone, definition, target, bin_dir, local, _clone_text = _conflict_setup(tmp_path)
    catalog = {"foo": definition}
    session = prepare_deploy(
        clone,
        home=target,
        catalog=catalog,
        application_dirs=[],
        path=str(bin_dir),
        snapshots_dir=tmp_path / "snapshots",
    )
    try:
        plan = session.plan()
        conflict = plan.conflicts[0]
        result = session.apply(
            ConflictDecisions(
                {
                    conflict.path: FileDecision(
                        FileAction.REVIEW, (RowDecision.CLONE, RowDecision.LOCAL)
                    )
                }
            )
        )
    finally:
        session.close()

    merged = (target / ".config/foo/config.conf").read_text()
    assert "LINE ONE" in merged
    assert "line 18" in merged
    assert "LINE EIGHTEEN" not in merged
    assert result.merged == [".config/foo/config.conf"]
    assert result.snapshot is not None

    undone = run_deploy(
        result.snapshot,
        home=target,
        catalog=catalog,
        application_dirs=[],
        path=str(bin_dir),
        snapshots_dir=tmp_path / "snapshots",
    )

    assert (target / ".config/foo/config.conf").read_text() == local
    assert undone.deployed == [".config/foo/config.conf"]


def _cli_environment(tmp_path: Path, target: Path, bin_dir: Path) -> dict[str, str]:
    apps = target / ".config" / "kamino" / "apps"
    apps.mkdir(parents=True, exist_ok=True)
    (apps / "foo.toml").write_text(
        'id = "foo"\nname = "Foo"\ncommand = "foo"\npaths = [".config/foo/config.conf"]\n',
        encoding="utf-8",
    )
    return {"HOME": str(target), "PATH": str(bin_dir)}


def test_cli_yes_applies_conflicts_without_tui(tmp_path: Path) -> None:
    clone, _definition, target, bin_dir, _local, clone_text = _conflict_setup(tmp_path)
    env = _cli_environment(tmp_path, target, bin_dir)

    result = runner.invoke(cli.app, ["deploy", str(clone), "--yes"], env=env)

    assert result.exit_code == 0, result.output
    assert (target / ".config/foo/config.conf").read_text() == clone_text
    snapshots = target / "Library" / "Application Support" / "kamino" / "snapshots"
    assert list(snapshots.glob("*.tar.gz"))


def test_cli_dry_run_prints_diff_without_writing(tmp_path: Path) -> None:
    clone, _definition, target, bin_dir, local, _clone_text = _conflict_setup(tmp_path)
    env = _cli_environment(tmp_path, target, bin_dir)

    result = runner.invoke(cli.app, ["deploy", str(clone), "--dry-run"], env=env)

    assert result.exit_code == 0, result.output
    assert "--- original/.config/foo/config.conf" in result.output
    assert "+++ clon/.config/foo/config.conf" in result.output
    assert "LINE ONE" in result.output
    assert (target / ".config/foo/config.conf").read_text() == local
    snapshots = target / "Library" / "Application Support" / "kamino" / "snapshots"
    assert not snapshots.exists()
