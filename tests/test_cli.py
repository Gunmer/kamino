"""Pruebas de `cli.py`: cableado de los comandos a los casos de uso."""

from __future__ import annotations

from pathlib import Path
from types import SimpleNamespace
from unittest.mock import MagicMock

import pytest
from typer.testing import CliRunner

import kamino.cli as cli
from kamino.errors import ArchiveError

runner = CliRunner()


class _FakeStdin:
    def __init__(self, is_tty: bool):
        self._is_tty = is_tty

    def isatty(self) -> bool:
        return self._is_tty


class _FakeSys:
    def __init__(self, is_tty: bool):
        self.stdin = _FakeStdin(is_tty)


def _fake_backup_result(*, skipped: tuple[str, ...] = ()) -> SimpleNamespace:
    return SimpleNamespace(
        plan=SimpleNamespace(operations=[], skipped=list(skipped)),
        manifest=SimpleNamespace(apps=[]),
        output=None,
        dry_run=False,
    )


def _fake_restore_result(*, omitted: tuple[str, ...] = ()) -> SimpleNamespace:
    return SimpleNamespace(
        plan=SimpleNamespace(entries=[]),
        manifest=SimpleNamespace(apps=[]),
        output=None,
        dry_run=False,
        snapshot=None,
        restored=[],
        omitted=list(omitted),
    )


def test_backup_non_interactive(monkeypatch) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli.ui, "print_backup_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_backup(**kwargs):
        calls.update(kwargs)
        return _fake_backup_result()

    monkeypatch.setattr(cli, "run_backup", fake_run_backup)
    result = runner.invoke(cli.app, ["backup", "--apps", "git"])
    assert result.exit_code == 0
    assert calls["selected"] == ["git"]


@pytest.mark.parametrize(
    "args",
    [
        ["backup", "--apps", "git", "--output", "respaldo.tar.gz"],
        ["backup", "-a", "git", "-o", "respaldo.tar.gz"],
    ],
)
def test_backup_apps_and_output_aliases(monkeypatch, args: list[str]) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli.ui, "print_backup_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_backup(**kwargs):
        calls.update(kwargs)
        return _fake_backup_result()

    monkeypatch.setattr(cli, "run_backup", fake_run_backup)
    result = runner.invoke(cli.app, args)
    assert result.exit_code == 0
    assert calls["selected"] == ["git"]
    assert calls["output"] == Path("respaldo.tar.gz")


@pytest.mark.parametrize("flag", ["--dry-run", "-n"])
def test_backup_dry_run_alias(monkeypatch, flag: str) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli.ui, "print_backup_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_backup(**kwargs):
        calls.update(kwargs)
        return _fake_backup_result()

    monkeypatch.setattr(cli, "run_backup", fake_run_backup)
    result = runner.invoke(cli.app, ["backup", "--apps", "git", "--output", "out.tar.gz", flag])
    assert result.exit_code == 0
    assert calls["dry_run"] is True


@pytest.mark.parametrize("flag", ["--verbose", "-v"])
def test_backup_verbose_alias(monkeypatch, flag: str) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    summary: dict = {}

    def fake_print(result, **kwargs):
        summary.update(kwargs)

    monkeypatch.setattr(cli.ui, "print_backup_summary", fake_print)
    monkeypatch.setattr(cli, "run_backup", lambda **kwargs: _fake_backup_result())
    result = runner.invoke(cli.app, ["backup", "--apps", "git", flag])
    assert result.exit_code == 0
    assert summary["verbose"] is True


def test_backup_alias_exit_3_when_skipped(monkeypatch) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli.ui, "print_backup_summary", lambda result, **kw: None)
    monkeypatch.setattr(cli, "run_backup", lambda **kwargs: _fake_backup_result(skipped=("x",)))
    result = runner.invoke(cli.app, ["backup", "-a", "git", "-n"])
    assert result.exit_code == 3


def test_backup_unknown_app_exits_3(monkeypatch) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli.ui, "print_backup_summary", lambda result, **kw: None)
    monkeypatch.setattr(cli, "run_backup", lambda **kwargs: _fake_backup_result())
    result = runner.invoke(cli.app, ["backup", "--apps", "git,nope"])
    assert result.exit_code == 3
    assert "nope" in result.output


def test_backup_non_tty_defaults_to_all(monkeypatch) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli, "sys", _FakeSys(False))
    monkeypatch.setattr(cli.ui, "print_backup_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_backup(**kwargs):
        calls.update(kwargs)
        return _fake_backup_result()

    monkeypatch.setattr(cli, "run_backup", fake_run_backup)
    result = runner.invoke(cli.app, ["backup"])
    assert result.exit_code == 0
    assert calls["selected"] == ["git"]


def test_backup_interactive_selects_all(monkeypatch) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli, "sys", _FakeSys(True))
    monkeypatch.setattr(cli.ui, "select_apps", lambda choices: [c[0] for c in choices])
    monkeypatch.setattr(cli.ui, "print_backup_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_backup(**kwargs):
        calls.update(kwargs)
        return _fake_backup_result()

    monkeypatch.setattr(cli, "run_backup", fake_run_backup)
    result = runner.invoke(cli.app, ["backup"])
    assert result.exit_code == 0
    assert calls["selected"] == ["git"]


def test_backup_error_exit_1(monkeypatch) -> None:
    monkeypatch.setattr(cli, "load_catalog", lambda: {})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)

    def boom(**kwargs):
        raise ArchiveError("fallo de respaldo")

    monkeypatch.setattr(cli, "run_backup", boom)
    result = runner.invoke(cli.app, ["backup", "--apps", "git"])
    assert result.exit_code == 1
    assert "fallo de respaldo" in result.output


def test_restore_yes(monkeypatch) -> None:
    monkeypatch.setattr(cli.ui, "print_restore_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_restore(backup_file, **kwargs):
        calls["backup_file"] = backup_file
        calls.update(kwargs)
        return _fake_restore_result()

    monkeypatch.setattr(cli, "run_restore", fake_run_restore)
    result = runner.invoke(cli.app, ["restore", "backup.tar.gz", "--yes"])
    assert result.exit_code == 0
    assert calls["selected"] is None


@pytest.mark.parametrize(
    "flags",
    [
        ["--yes", "--verbose"],
        ["-y", "-v"],
    ],
)
def test_restore_yes_and_verbose_aliases(monkeypatch, flags: list[str]) -> None:
    summary: dict = {}
    calls: dict = {}

    def fake_print(result, **kwargs):
        summary.update(kwargs)

    def fake_run_restore(backup_file, **kwargs):
        calls["backup_file"] = backup_file
        calls.update(kwargs)
        return _fake_restore_result()

    monkeypatch.setattr(cli.ui, "print_restore_summary", fake_print)
    monkeypatch.setattr(cli, "run_restore", fake_run_restore)
    result = runner.invoke(cli.app, ["restore", "backup.tar.gz", *flags])
    assert result.exit_code == 0
    assert calls["backup_file"] == Path("backup.tar.gz")
    assert calls["selected"] is None
    assert calls["dry_run"] is False
    assert summary["verbose"] is True


@pytest.mark.parametrize("flag", ["--dry-run", "-n"])
def test_restore_dry_run_alias(monkeypatch, flag: str) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(False))
    calls: dict = {}

    def fake_run_restore(backup_file, **kwargs):
        calls["backup_file"] = backup_file
        calls.update(kwargs)
        return _fake_restore_result()

    monkeypatch.setattr(cli, "run_restore", fake_run_restore)
    result = runner.invoke(cli.app, ["restore", "backup.tar.gz", flag])
    assert result.exit_code == 0
    assert calls["backup_file"] == Path("backup.tar.gz")
    assert calls["dry_run"] is True


def test_restore_interactive_selection_with_dry_run_alias(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(True))
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(
        cli,
        "read_manifest",
        lambda path: SimpleNamespace(apps=[SimpleNamespace(id="git", name="Git")]),
    )
    monkeypatch.setattr(cli.ui, "select_apps", lambda choices: ["git"])
    monkeypatch.setattr(cli.ui, "print_restore_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_restore(backup_file, **kwargs):
        calls["backup_file"] = backup_file
        calls.update(kwargs)
        return _fake_restore_result()

    monkeypatch.setattr(cli, "run_restore", fake_run_restore)
    result = runner.invoke(cli.app, ["restore", "backup.tar.gz", "-n"])
    assert result.exit_code == 0
    assert calls["selected"] == ["git"]
    assert calls["dry_run"] is True


def test_restore_requires_yes_without_tty(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(False))
    called: list[str] = []
    monkeypatch.setattr(cli, "run_restore", lambda *args, **kwargs: called.append("restore"))
    result = runner.invoke(cli.app, ["restore", "backup.tar.gz"])
    assert result.exit_code == 1
    assert "--yes" in result.output
    assert called == []


@pytest.mark.parametrize("flag", ["--yes", "-y"])
def test_restore_exit_3_when_omitted(monkeypatch, flag: str) -> None:
    monkeypatch.setattr(cli.ui, "print_restore_summary", lambda result, **kw: None)

    def fake_run_restore(backup_file, **kwargs):
        return _fake_restore_result(omitted=("x",))

    monkeypatch.setattr(cli, "run_restore", fake_run_restore)
    result = runner.invoke(cli.app, ["restore", "backup.tar.gz", flag])
    assert result.exit_code == 3


@pytest.mark.parametrize(
    ("args", "expected_names"),
    [
        (["-h"], ["--help", "-h", "--version", "-V"]),
        (["--help"], ["--help", "-h", "--version", "-V"]),
        (["backup", "-h"], ["--help", "-h"]),
        (["backup", "--help"], ["--help", "-h"]),
        (["restore", "-h"], ["--help", "-h"]),
        (["restore", "--help"], ["--help", "-h"]),
    ],
)
def test_help_aliases(monkeypatch, args: list[str], expected_names: list[str]) -> None:
    dispatched: list[str] = []
    monkeypatch.setattr(cli, "_run_menu", lambda ctx: dispatched.append("run_menu"))
    monkeypatch.setattr(cli, "run_backup", lambda **kwargs: dispatched.append("backup"))
    monkeypatch.setattr(cli, "run_restore", lambda *args, **kwargs: dispatched.append("restore"))
    result = runner.invoke(cli.app, args)
    assert result.exit_code == 0
    assert "Usage:" in result.output
    for name in expected_names:
        assert name in result.output
    assert dispatched == []


def test_main_help_keeps_completion_option_names(monkeypatch) -> None:
    installed: list[str] = []
    monkeypatch.setattr(cli, "_run_menu", lambda ctx: installed.append("run_menu"))
    result = runner.invoke(cli.app, ["-h"])
    assert result.exit_code == 0
    assert "--install-completion" in result.output
    assert "--show-completion" in result.output
    assert installed == []


@pytest.mark.parametrize("flag", ["--version", "-V"])
def test_version(monkeypatch, flag: str) -> None:
    dispatched: list[str] = []
    monkeypatch.setattr(cli, "_run_menu", lambda ctx: dispatched.append("run_menu"))
    monkeypatch.setattr(cli, "run_backup", lambda **kwargs: dispatched.append("backup"))
    monkeypatch.setattr(cli, "run_restore", lambda *args, **kwargs: dispatched.append("restore"))
    result = runner.invoke(cli.app, [flag])
    assert result.exit_code == 0
    assert result.output == f"kamino {cli.__version__}\n"
    assert dispatched == []


def test_no_args_non_tty_shows_help(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(False))
    result = runner.invoke(cli.app, [])
    assert result.exit_code == 0
    assert "backup" in result.output
    assert "restore" in result.output


def test_no_args_tty_menu_dispatches_to_run_menu(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(True))
    dispatched: list[str] = []
    monkeypatch.setattr(cli, "_run_menu", lambda ctx: dispatched.append("run_menu"))
    result = runner.invoke(cli.app, [])
    assert result.exit_code == 0
    assert dispatched == ["run_menu"]


def test_run_menu_invokes_backup_via_ctx(monkeypatch) -> None:
    monkeypatch.setattr(cli.ui, "select_command", lambda choices: "backup")
    ctx = MagicMock()
    cli._run_menu(ctx)
    ctx.invoke.assert_called_once_with(cli.backup)


def test_run_menu_invokes_restore_via_ctx(monkeypatch) -> None:
    monkeypatch.setattr(cli.ui, "select_command", lambda choices: "restore")
    monkeypatch.setattr(cli.ui, "ask_path", lambda prompt: "backup.tar.gz")
    ctx = MagicMock()
    cli._run_menu(ctx)
    ctx.invoke.assert_called_once_with(cli.restore, backup_file=Path("backup.tar.gz"))


def test_no_args_tty_menu_cancel(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(True))
    monkeypatch.setattr(cli.ui, "select_command", lambda choices: None)
    result = runner.invoke(cli.app, [])
    assert result.exit_code == 1
    assert "cancelado" in result.output
