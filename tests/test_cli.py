"""Pruebas de `cli.py`: cableado de los comandos a los casos de uso."""

from __future__ import annotations

from types import SimpleNamespace

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


def test_restore_requires_yes_without_tty(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(False))
    result = runner.invoke(cli.app, ["restore", "backup.tar.gz"])
    assert result.exit_code == 1
    assert "--yes" in result.output


def test_restore_exit_3_when_omitted(monkeypatch) -> None:
    monkeypatch.setattr(cli.ui, "print_restore_summary", lambda result, **kw: None)

    def fake_run_restore(backup_file, **kwargs):
        return _fake_restore_result(omitted=("x",))

    monkeypatch.setattr(cli, "run_restore", fake_run_restore)
    result = runner.invoke(cli.app, ["restore", "backup.tar.gz", "--yes"])
    assert result.exit_code == 3


def test_version() -> None:
    result = runner.invoke(cli.app, ["--version"])
    assert result.exit_code == 0
    assert "kamino" in result.output
