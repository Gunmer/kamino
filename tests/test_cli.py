"""Pruebas de `cli.py`: cableado de los comandos a los casos de uso."""

from __future__ import annotations

import re
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import MagicMock

import pytest
from rich.text import Text
from typer.testing import CliRunner

import kamino.cli as cli
from kamino.errors import ArchiveError, UpdateError

runner = CliRunner()


def _assert_help_options_visible(output: str, options: list[str]) -> None:
    plain_output = Text.from_ansi(output).plain
    for option in options:
        assert re.search(rf"(?<![\w-]){re.escape(option)}(?![\w-])", plain_output)


class _FakeStdin:
    def __init__(self, is_tty: bool):
        self._is_tty = is_tty

    def isatty(self) -> bool:
        return self._is_tty


class _FakeSys:
    def __init__(self, is_tty: bool):
        self.stdin = _FakeStdin(is_tty)


def _fake_clone_result(*, skipped: tuple[str, ...] = ()) -> SimpleNamespace:
    return SimpleNamespace(
        plan=SimpleNamespace(operations=[], skipped=list(skipped)),
        manifest=SimpleNamespace(apps=[]),
        output=None,
        dry_run=False,
    )


def _fake_deploy_result(
    *, omitted: tuple[str, ...] = (), merged: tuple[str, ...] = (), dry_run: bool = False
) -> SimpleNamespace:
    return SimpleNamespace(
        plan=SimpleNamespace(entries=[], conflicts=[], notices=[], omitted=list(omitted)),
        manifest=SimpleNamespace(apps=[]),
        output=None,
        dry_run=dry_run,
        snapshot=None,
        deployed=[],
        merged=list(merged),
        omitted=list(omitted),
    )


class _FakeDeploySession:
    def __init__(self, *, deployable: tuple[str, ...] = ("git",), omitted: tuple[str, ...] = ()):
        self.deployable_apps = [SimpleNamespace(id=app_id, name=app_id) for app_id in deployable]
        self.omitted = omitted
        self.calls: list[tuple] = []
        self.closed = False

    def plan(self, selected=None, *, dry_run=False):
        self.calls.append(("plan", selected, dry_run))
        return SimpleNamespace(entries=[], conflicts=[], notices=[], omitted=list(self.omitted))

    def dry_run_result(self, plan):
        return _fake_deploy_result(omitted=tuple(plan.omitted), dry_run=True)

    def apply(self, decisions):
        self.calls.append(("apply", decisions))
        return _fake_deploy_result(omitted=self.omitted)

    def close(self):
        self.closed = True


def test_clone_non_interactive(monkeypatch) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli.ui, "print_clone_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_clone(**kwargs):
        calls.update(kwargs)
        return _fake_clone_result()

    monkeypatch.setattr(cli, "run_clone", fake_run_clone)
    result = runner.invoke(cli.app, ["clone", "--apps", "git"])
    assert result.exit_code == 0
    assert calls["selected"] == ["git"]


@pytest.mark.parametrize(
    "args",
    [
        ["clone", "--apps", "git", "--output", "clon.tar.gz"],
        ["clone", "-a", "git", "-o", "clon.tar.gz"],
    ],
)
def test_clone_apps_and_output_aliases(monkeypatch, args: list[str]) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli.ui, "print_clone_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_clone(**kwargs):
        calls.update(kwargs)
        return _fake_clone_result()

    monkeypatch.setattr(cli, "run_clone", fake_run_clone)
    result = runner.invoke(cli.app, args)
    assert result.exit_code == 0
    assert calls["selected"] == ["git"]
    assert calls["output"] == Path("clon.tar.gz")


@pytest.mark.parametrize("flag", ["--dry-run", "-n"])
def test_clone_dry_run_alias(monkeypatch, flag: str) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli.ui, "print_clone_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_clone(**kwargs):
        calls.update(kwargs)
        return _fake_clone_result()

    monkeypatch.setattr(cli, "run_clone", fake_run_clone)
    result = runner.invoke(cli.app, ["clone", "--apps", "git", "--output", "out.tar.gz", flag])
    assert result.exit_code == 0
    assert calls["dry_run"] is True


@pytest.mark.parametrize("flag", ["--verbose", "-v"])
def test_clone_verbose_alias(monkeypatch, flag: str) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    summary: dict = {}

    def fake_print(result, **kwargs):
        summary.update(kwargs)

    monkeypatch.setattr(cli.ui, "print_clone_summary", fake_print)
    monkeypatch.setattr(cli, "run_clone", lambda **kwargs: _fake_clone_result())
    result = runner.invoke(cli.app, ["clone", "--apps", "git", flag])
    assert result.exit_code == 0
    assert summary["verbose"] is True


def test_clone_alias_exit_3_when_skipped(monkeypatch) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli.ui, "print_clone_summary", lambda result, **kw: None)
    monkeypatch.setattr(cli, "run_clone", lambda **kwargs: _fake_clone_result(skipped=("x",)))
    result = runner.invoke(cli.app, ["clone", "-a", "git", "-n"])
    assert result.exit_code == 3


def test_clone_unknown_app_exits_3(monkeypatch) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli.ui, "print_clone_summary", lambda result, **kw: None)
    monkeypatch.setattr(cli, "run_clone", lambda **kwargs: _fake_clone_result())
    result = runner.invoke(cli.app, ["clone", "--apps", "git,nope"])
    assert result.exit_code == 3
    assert "nope" in result.output


def test_clone_non_tty_defaults_to_all(monkeypatch) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli, "sys", _FakeSys(False))
    monkeypatch.setattr(cli.ui, "print_clone_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_clone(**kwargs):
        calls.update(kwargs)
        return _fake_clone_result()

    monkeypatch.setattr(cli, "run_clone", fake_run_clone)
    result = runner.invoke(cli.app, ["clone"])
    assert result.exit_code == 0
    assert calls["selected"] == ["git"]


def test_clone_interactive_selects_all(monkeypatch) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli, "sys", _FakeSys(True))
    monkeypatch.setattr(cli.ui, "select_apps", lambda choices: [c[0] for c in choices])
    monkeypatch.setattr(cli.ui, "print_clone_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_clone(**kwargs):
        calls.update(kwargs)
        return _fake_clone_result()

    monkeypatch.setattr(cli, "run_clone", fake_run_clone)
    result = runner.invoke(cli.app, ["clone"])
    assert result.exit_code == 0
    assert calls["selected"] == ["git"]


@pytest.mark.parametrize("is_tty", [False, True])
@pytest.mark.parametrize("dry_run", [False, True])
def test_clone_prints_privacy_warning_without_confirmation(
    monkeypatch, is_tty: bool, dry_run: bool
) -> None:
    fake_app = SimpleNamespace(id="git", name="Git")
    monkeypatch.setattr(cli, "load_catalog", lambda: {"git": fake_app})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli, "sys", _FakeSys(is_tty))
    monkeypatch.setattr(cli.ui, "select_apps", lambda choices: [c[0] for c in choices])
    monkeypatch.setattr(cli.ui, "print_clone_summary", lambda result, **kw: None)
    monkeypatch.setattr(cli, "run_clone", lambda **kwargs: _fake_clone_result())
    args = ["clone", "--apps", "git"]
    if dry_run:
        args.append("--dry-run")

    result = runner.invoke(cli.app, args)

    assert result.exit_code == 0
    plain = Text.from_ansi(result.output).plain.lower()
    assert "no está cifrado" in plain
    assert "secretos" in plain


def test_user_catalog_definition_takes_precedence_in_cli(tmp_path, monkeypatch) -> None:
    apps_dir = tmp_path / ".config" / "kamino" / "apps"
    apps_dir.mkdir(parents=True)
    (apps_dir / "git.toml").write_text(
        'id = "git"\nname = "Git personalizado"\ncommand = "git"\n'
        'paths = [".gitconfig-personal"]\n',
        encoding="utf-8",
    )
    monkeypatch.setenv("HOME", str(tmp_path))
    monkeypatch.setattr(cli, "is_installed", lambda app: True)
    monkeypatch.setattr(cli.ui, "print_clone_summary", lambda result, **kw: None)
    calls: dict = {}

    def fake_run_clone(**kwargs):
        calls.update(kwargs)
        return _fake_clone_result()

    monkeypatch.setattr(cli, "run_clone", fake_run_clone)
    result = runner.invoke(cli.app, ["clone", "--apps", "git"])

    assert result.exit_code == 0
    assert calls["catalog"]["git"].name == "Git personalizado"
    assert calls["catalog"]["git"].paths == [".gitconfig-personal"]


def test_user_catalog_path_confinement_is_enforced_in_cli(tmp_path, monkeypatch) -> None:
    apps_dir = tmp_path / ".config" / "kamino" / "apps"
    apps_dir.mkdir(parents=True)
    (apps_dir / "git.toml").write_text(
        'id = "git"\nname = "Git personalizado"\ncommand = "git"\npaths = ["/etc/passwd"]\n',
        encoding="utf-8",
    )
    monkeypatch.setenv("HOME", str(tmp_path))
    called: list[str] = []
    monkeypatch.setattr(cli, "run_clone", lambda **kwargs: called.append("clone"))
    result = runner.invoke(cli.app, ["clone", "--apps", "git"])

    assert result.exit_code == 1
    assert called == []
    assert "ruta absoluta" in result.output


def test_clone_error_exit_1(monkeypatch) -> None:
    monkeypatch.setattr(cli, "load_catalog", lambda: {})
    monkeypatch.setattr(cli, "is_installed", lambda app: True)

    def boom(**kwargs):
        raise ArchiveError("fallo de clon")

    monkeypatch.setattr(cli, "run_clone", boom)
    result = runner.invoke(cli.app, ["clone", "--apps", "git"])
    assert result.exit_code == 1
    assert "fallo de clon" in result.output


def _install_deploy(monkeypatch, session: _FakeDeploySession) -> None:
    monkeypatch.setattr(cli, "prepare_deploy", lambda clone_file: session)
    monkeypatch.setattr(cli.ui, "print_deploy_summary", lambda result, **kw: None)


def test_deploy_yes(monkeypatch) -> None:
    session = _FakeDeploySession()
    _install_deploy(monkeypatch, session)

    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", "--yes"])

    assert result.exit_code == 0
    assert session.calls[0] == ("plan", None, False)
    assert session.calls[1][0] == "apply"
    assert session.closed is True


@pytest.mark.parametrize(
    "flags",
    [
        ["--yes", "--verbose"],
        ["-y", "-v"],
    ],
)
def test_deploy_yes_and_verbose_aliases(monkeypatch, flags: list[str]) -> None:
    summary: dict = {}
    session = _FakeDeploySession()
    monkeypatch.setattr(cli, "prepare_deploy", lambda clone_file: session)

    def fake_print(result, **kwargs):
        summary.update(kwargs)

    monkeypatch.setattr(cli.ui, "print_deploy_summary", fake_print)
    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", *flags])
    assert result.exit_code == 0
    assert summary["verbose"] is True


@pytest.mark.parametrize("flag", ["--dry-run", "-n"])
def test_deploy_dry_run_alias(monkeypatch, flag: str) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(False))
    session = _FakeDeploySession()
    _install_deploy(monkeypatch, session)

    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", flag])

    assert result.exit_code == 0
    assert session.calls[0] == ("plan", None, True)
    assert all(call[0] != "apply" for call in session.calls)


@pytest.mark.parametrize("flag", ["--apps", "-a"])
def test_deploy_apps_alias(monkeypatch, flag: str) -> None:
    session = _FakeDeploySession(deployable=("git", "zsh"))
    _install_deploy(monkeypatch, session)

    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", "--yes", flag, "git"])

    assert result.exit_code == 0
    assert session.calls[0] == ("plan", ["git"], False)


def test_deploy_unknown_app_warns_and_exits_3(monkeypatch) -> None:
    session = _FakeDeploySession(deployable=("git",))
    _install_deploy(monkeypatch, session)

    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", "--yes", "--apps", "git,no-existe"])

    assert result.exit_code == 3
    assert "no-existe" in result.output
    assert session.calls[0] == ("plan", ["git"], False)


def test_deploy_interactive_opens_tui_without_apps(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(True))
    session = _FakeDeploySession(deployable=("git",))
    _install_deploy(monkeypatch, session)
    calls: dict = {}

    def fake_tui(current, selected=None):
        calls["session"] = current
        calls["selected"] = selected
        return _fake_deploy_result()

    monkeypatch.setattr(cli, "run_deploy_tui", fake_tui)
    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz"])

    assert result.exit_code == 0
    assert calls["session"] is session
    assert calls["selected"] is None
    assert all(call[0] != "plan" for call in session.calls)


def test_deploy_interactive_skips_selection_with_apps(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(True))
    session = _FakeDeploySession(deployable=("git", "zsh"))
    _install_deploy(monkeypatch, session)
    calls: dict = {}

    def fake_tui(current, selected=None):
        calls["selected"] = selected
        return _fake_deploy_result()

    monkeypatch.setattr(cli, "run_deploy_tui", fake_tui)
    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", "-a", "zsh"])

    assert result.exit_code == 0
    assert calls["selected"] == ["zsh"]


def test_deploy_tui_cancel_exits_1(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(True))
    session = _FakeDeploySession()
    _install_deploy(monkeypatch, session)
    monkeypatch.setattr(cli, "run_deploy_tui", lambda current, selected=None: None)

    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz"])

    assert result.exit_code == 1
    assert "cancelado" in result.output


def test_deploy_requires_yes_without_tty(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(False))
    prepared: list[str] = []
    monkeypatch.setattr(cli, "prepare_deploy", lambda path: prepared.append("prepared"))
    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz"])
    assert result.exit_code == 1
    assert "--yes" in result.output
    assert prepared == []


@pytest.mark.parametrize("flag", ["--yes", "-y"])
def test_deploy_exit_3_when_omitted(monkeypatch, flag: str) -> None:
    session = _FakeDeploySession(omitted=("x",))
    _install_deploy(monkeypatch, session)
    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", flag])
    assert result.exit_code == 3


@pytest.mark.parametrize(
    ("args", "expected_names"),
    [
        (["-h"], ["--help", "-h", "--version", "-V"]),
        (["--help"], ["--help", "-h", "--version", "-V"]),
        (["clone", "-h"], ["--help", "-h"]),
        (["clone", "--help"], ["--help", "-h"]),
        (["deploy", "-h"], ["--help", "-h"]),
        (["deploy", "--help"], ["--help", "-h"]),
    ],
)
def test_help_aliases(monkeypatch, args: list[str], expected_names: list[str]) -> None:
    dispatched: list[str] = []
    monkeypatch.setattr(cli, "_run_menu", lambda ctx: dispatched.append("run_menu"))
    monkeypatch.setattr(cli, "run_clone", lambda **kwargs: dispatched.append("clone"))
    result = runner.invoke(cli.app, args)
    assert result.exit_code == 0
    assert "Usage:" in Text.from_ansi(result.output).plain
    _assert_help_options_visible(result.output, expected_names)
    assert dispatched == []


@pytest.mark.parametrize(
    ("command", "expected_options"),
    [
        ([], ["--help", "-h", "--version", "-V", "--install-completion", "--show-completion"]),
        (
            ["clone"],
            [
                "--help",
                "-h",
                "--apps",
                "-a",
                "--output",
                "-o",
                "--dry-run",
                "-n",
                "--verbose",
                "-v",
            ],
        ),
        (
            ["deploy"],
            [
                "--help",
                "-h",
                "--apps",
                "-a",
                "--dry-run",
                "-n",
                "--yes",
                "-y",
                "--verbose",
                "-v",
            ],
        ),
    ],
)
def test_help_options_are_visible_with_ansi_color(
    monkeypatch, command: list[str], expected_options: list[str]
) -> None:
    dispatched: list[str] = []
    monkeypatch.setattr(cli, "_run_menu", lambda ctx: dispatched.append("run_menu"))
    monkeypatch.setattr(cli, "run_clone", lambda **kwargs: dispatched.append("clone"))
    result = runner.invoke(cli.app, [*command, "-h"], env={"FORCE_COLOR": "1"})
    assert result.exit_code == 0
    assert "\x1b[" in result.output
    assert "Usage:" in Text.from_ansi(result.output).plain
    _assert_help_options_visible(result.output, expected_options)
    assert dispatched == []


def test_main_help_keeps_completion_option_names(monkeypatch) -> None:
    installed: list[str] = []
    monkeypatch.setattr(cli, "_run_menu", lambda ctx: installed.append("run_menu"))
    result = runner.invoke(cli.app, ["-h"])
    assert result.exit_code == 0
    _assert_help_options_visible(result.output, ["--install-completion", "--show-completion"])
    assert installed == []


@pytest.mark.parametrize("flag", ["--version", "-V"])
def test_version(monkeypatch, flag: str) -> None:
    dispatched: list[str] = []
    monkeypatch.setattr(cli, "_run_menu", lambda ctx: dispatched.append("run_menu"))
    monkeypatch.setattr(cli, "run_clone", lambda **kwargs: dispatched.append("clone"))
    result = runner.invoke(cli.app, [flag])
    assert result.exit_code == 0
    assert result.output == f"kamino {cli.__version__}\n"
    assert dispatched == []


def test_no_args_non_tty_shows_help(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(False))
    result = runner.invoke(cli.app, [])
    assert result.exit_code == 0
    assert "clone" in result.output
    assert "deploy" in result.output


def test_no_args_tty_menu_dispatches_to_run_menu(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(True))
    dispatched: list[str] = []
    monkeypatch.setattr(cli, "_run_menu", lambda ctx: dispatched.append("run_menu"))
    result = runner.invoke(cli.app, [])
    assert result.exit_code == 0
    assert dispatched == ["run_menu"]


def test_run_menu_invokes_clone_via_ctx(monkeypatch) -> None:
    monkeypatch.setattr(cli.ui, "select_command", lambda choices: "clone")
    ctx = MagicMock()
    cli._run_menu(ctx)
    ctx.invoke.assert_called_once_with(cli.clone)


def test_run_menu_invokes_deploy_via_ctx(monkeypatch) -> None:
    monkeypatch.setattr(cli.ui, "select_command", lambda choices: "deploy")
    monkeypatch.setattr(cli.ui, "ask_path", lambda prompt: "clone.tar.gz")
    ctx = MagicMock()
    cli._run_menu(ctx)
    ctx.invoke.assert_called_once_with(cli.deploy, clone_file=Path("clone.tar.gz"))


def test_no_args_tty_menu_cancel(monkeypatch) -> None:
    monkeypatch.setattr(cli, "sys", _FakeSys(True))
    monkeypatch.setattr(cli.ui, "select_command", lambda choices: None)
    result = runner.invoke(cli.app, [])
    assert result.exit_code == 1
    assert "cancelado" in result.output


def test_update_command_runs_use_case(monkeypatch) -> None:
    update_result = SimpleNamespace(
        current_version="0.4.0",
        latest_version="0.5.0",
        target=Path("/tmp/kamino"),
        updated=True,
    )
    calls: list[object] = []

    def fake_run(version: str):
        calls.append(("run", version))
        return update_result

    monkeypatch.setattr(cli, "run_update", fake_run)
    monkeypatch.setattr(cli.ui, "print_update_start", lambda: calls.append(("start", None)))
    monkeypatch.setattr(
        cli.ui, "print_update_result", lambda result: calls.append(("result", result))
    )

    result = runner.invoke(cli.app, ["update"])

    assert result.exit_code == 0
    assert calls == [("start", None), ("run", cli.__version__), ("result", update_result)]


def test_update_command_error_exit_1(monkeypatch) -> None:
    def boom(version: str):
        raise UpdateError("fallo de actualización")

    monkeypatch.setattr(cli, "run_update", boom)
    monkeypatch.setattr(cli.ui, "print_update_start", lambda: None)

    result = runner.invoke(cli.app, ["update"])

    assert result.exit_code == 1
    assert "fallo de actualización" in result.output


def test_run_menu_invokes_update_via_ctx(monkeypatch) -> None:
    monkeypatch.setattr(cli.ui, "select_command", lambda choices: "update")
    ctx = MagicMock()
    cli._run_menu(ctx)
    ctx.invoke.assert_called_once_with(cli.update)


def test_run_menu_lists_update_option(monkeypatch) -> None:
    choices: list[tuple[str, str, str]] = []

    def fake_select(options):
        choices.extend(options)
        return "update"

    monkeypatch.setattr(cli.ui, "select_command", fake_select)
    ctx = MagicMock()
    cli._run_menu(ctx)

    assert (
        "update",
        "Actualización",
        "Actualiza a la última versión publicada.",
    ) in choices
