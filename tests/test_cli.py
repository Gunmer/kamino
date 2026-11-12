"""Pruebas de `cli.py`: cableado, gating de TUI y códigos de salida."""

from __future__ import annotations

import re
from pathlib import Path
from types import SimpleNamespace

import pytest
from rich.text import Text
from typer.testing import CliRunner

import kamino.cli as cli
from kamino.commands import CloneOutcome, DeployOutcome
from kamino.config import config_path
from kamino.errors import ArchiveError, UpdateError

runner = CliRunner()


def _assert_help_options_visible(output: str, options: list[str]) -> None:
    plain_output = Text.from_ansi(output).plain
    for option in options:
        assert re.search(rf"(?<![\w-]){re.escape(option)}(?![\w-])", plain_output)


def _clone_outcome(*, skipped=(), missing=(), code=0) -> CloneOutcome:
    return CloneOutcome(
        result=SimpleNamespace(
            plan=SimpleNamespace(operations=[], skipped=list(skipped)),
            manifest=SimpleNamespace(apps=[]),
            output=None,
            dry_run=False,
        ),
        code=code,
        missing=list(missing),
    )


def _deploy_outcome(*, code=0, missing=()) -> DeployOutcome:
    return DeployOutcome(
        result=SimpleNamespace(
            plan=SimpleNamespace(entries=[], conflicts=[], notices=[]),
            manifest=SimpleNamespace(apps=[]),
            output=None,
            dry_run=False,
            snapshot=None,
            deployed=[],
            merged=[],
            omitted=[],
            unchanged=[],
            not_installed=[],
            missing_requested=[],
        ),
        code=code,
        missing=list(missing),
    )


@pytest.fixture(autouse=True)
def _text_mode(monkeypatch):
    monkeypatch.setattr(cli, "_interactive", lambda: False)
    monkeypatch.setattr(cli.ui, "print_clone_summary", lambda result, **kw: None)
    monkeypatch.setattr(cli.ui, "print_deploy_summary", lambda result, **kw: None)


def test_clone_non_interactive(monkeypatch) -> None:
    calls: dict = {}

    def fake_execute(**kwargs):
        calls.update(kwargs)
        return _clone_outcome()

    monkeypatch.setattr(cli, "execute_clone", fake_execute)
    result = runner.invoke(cli.app, ["clone", "--apps", "git"])

    assert result.exit_code == 0
    assert calls["requested"] == ["git"]


@pytest.mark.parametrize(
    "args",
    [
        ["clone", "--apps", "git", "--output", "clon.tar.gz"],
        ["clone", "-a", "git", "-o", "clon.tar.gz"],
    ],
)
def test_clone_apps_and_output_aliases(monkeypatch, args: list[str]) -> None:
    calls: dict = {}

    def fake_execute(**kwargs):
        calls.update(kwargs)
        return _clone_outcome()

    monkeypatch.setattr(cli, "execute_clone", fake_execute)
    result = runner.invoke(cli.app, args)

    assert result.exit_code == 0
    assert calls["requested"] == ["git"]
    assert calls["output"] == Path("clon.tar.gz")


@pytest.mark.parametrize("flag", ["--dry-run", "-n"])
def test_clone_dry_run_alias(monkeypatch, flag: str) -> None:
    calls: dict = {}
    monkeypatch.setattr(
        cli,
        "execute_clone",
        lambda **kwargs: calls.update(kwargs) or _clone_outcome(),
    )
    result = runner.invoke(cli.app, ["clone", "--apps", "git", flag])

    assert result.exit_code == 0
    assert calls["dry_run"] is True


@pytest.mark.parametrize("flag", ["--force", "-f"])
def test_clone_force_alias(monkeypatch, flag: str) -> None:
    calls: dict = {}
    monkeypatch.setattr(
        cli,
        "execute_clone",
        lambda **kwargs: calls.update(kwargs) or _clone_outcome(),
    )
    result = runner.invoke(cli.app, ["clone", "--apps", "git", flag])

    assert result.exit_code == 0
    assert calls["force"] is True


def test_clone_alias_exit_3_when_skipped(monkeypatch) -> None:
    monkeypatch.setattr(cli, "execute_clone", lambda **kwargs: _clone_outcome(code=3))
    result = runner.invoke(cli.app, ["clone", "-a", "git", "-n"])

    assert result.exit_code == 3


def test_clone_missing_app_warns_and_exits_3(monkeypatch) -> None:
    monkeypatch.setattr(
        cli, "execute_clone", lambda **kwargs: _clone_outcome(missing=["nope"], code=3)
    )
    result = runner.invoke(cli.app, ["clone", "--apps", "git,nope"])

    assert result.exit_code == 3
    assert "nope" in result.output


def test_clone_prints_privacy_warning(monkeypatch) -> None:
    printed: list[str] = []
    monkeypatch.setattr(cli.ui, "print_privacy_warning", lambda: printed.append("warn"))
    monkeypatch.setattr(cli, "execute_clone", lambda **kwargs: _clone_outcome())

    result = runner.invoke(cli.app, ["clone", "--apps", "git"])

    assert result.exit_code == 0
    assert printed == ["warn"]


def test_clone_error_exit_1(monkeypatch) -> None:
    def boom(**kwargs):
        raise ArchiveError("fallo de clon")

    monkeypatch.setattr(cli, "execute_clone", boom)
    result = runner.invoke(cli.app, ["clone", "--apps", "git"])

    assert result.exit_code == 1
    assert "fallo de clon" in result.output


def test_clone_tui_skips_screens_with_valid_arguments(monkeypatch) -> None:
    monkeypatch.setattr(cli, "_interactive", lambda: True)
    calls: dict = {}

    def fake_flow(**kwargs):
        calls.update(kwargs)
        return 0

    monkeypatch.setattr(cli, "run_clone_flow", fake_flow)
    result = runner.invoke(cli.app, ["clone", "-a", "git", "-o", "out.tar.gz"])

    assert result.exit_code == 0
    assert calls["requested"] == ["git"]
    assert calls["output"] == Path("out.tar.gz").resolve()
    assert calls["output_error"] is None


def test_clone_tui_invalid_output_opens_destination(monkeypatch, tmp_path: Path) -> None:
    monkeypatch.setattr(cli, "_interactive", lambda: True)
    calls: dict = {}

    def fake_flow(**kwargs):
        calls.update(kwargs)
        return 0

    monkeypatch.setattr(cli, "run_clone_flow", fake_flow)
    missing = tmp_path / "no-existe" / "clone.tar.gz"
    result = runner.invoke(cli.app, ["clone", "--output", str(missing)])

    assert result.exit_code == 0
    assert calls["output_error"] is not None


def test_deploy_yes(monkeypatch) -> None:
    calls: dict = {}
    monkeypatch.setattr(
        cli,
        "execute_deploy",
        lambda path, **kwargs: calls.update(kwargs) or _deploy_outcome(),
    )
    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", "--yes"])

    assert result.exit_code == 0
    assert calls["dry_run"] is False


@pytest.mark.parametrize("flag", ["--dry-run", "-n"])
def test_deploy_dry_run_alias(monkeypatch, flag: str) -> None:
    calls: dict = {}
    monkeypatch.setattr(
        cli,
        "execute_deploy",
        lambda path, **kwargs: calls.update(kwargs) or _deploy_outcome(),
    )
    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", flag])

    assert result.exit_code == 0
    assert calls["dry_run"] is True


def test_deploy_requires_yes_without_tty(monkeypatch) -> None:
    called: list[str] = []
    monkeypatch.setattr(cli, "execute_deploy", lambda path, **kwargs: called.append("deploy"))
    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz"])

    assert result.exit_code == 1
    assert "--yes" in result.output
    assert called == []


@pytest.mark.parametrize("flag", ["--yes", "-y"])
def test_deploy_exit_3_when_omitted(monkeypatch, flag: str) -> None:
    monkeypatch.setattr(cli, "execute_deploy", lambda path, **kwargs: _deploy_outcome(code=3))
    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", flag])

    assert result.exit_code == 3


def test_deploy_tui_passes_requested_apps(monkeypatch, tmp_path: Path) -> None:
    monkeypatch.setattr(cli, "_interactive", lambda: True)
    clone = tmp_path / "clone.tar.gz"
    clone.write_bytes(b"x")
    calls: dict = {}

    def fake_flow(**kwargs):
        calls.update(kwargs)
        return 0

    monkeypatch.setattr(cli, "run_deploy_flow", fake_flow)
    result = runner.invoke(cli.app, ["deploy", str(clone), "-a", "git"])

    assert result.exit_code == 0
    assert calls["requested"] == ["git"]
    assert calls.get("file_error") is None


def test_deploy_tui_missing_file_opens_d0(monkeypatch, tmp_path: Path) -> None:
    monkeypatch.setattr(cli, "_interactive", lambda: True)
    calls: dict = {}
    monkeypatch.setattr(
        cli,
        "run_deploy_flow",
        lambda **kwargs: calls.update(kwargs) or 0,
    )
    result = runner.invoke(cli.app, ["deploy", str(tmp_path / "no-existe.tar.gz")])

    assert result.exit_code == 0
    assert calls["file_error"] is not None


@pytest.mark.parametrize(
    ("args", "expected_names"),
    [
        (["-h"], ["--help", "-h", "--version", "-V"]),
        (["--help"], ["--help", "-h", "--version", "-V"]),
        (["clone", "-h"], ["--help", "-h", "--force", "-f"]),
        (["deploy", "-h"], ["--help", "-h", "--yes", "-y"]),
    ],
)
def test_help_aliases(monkeypatch, args: list[str], expected_names: list[str]) -> None:
    result = runner.invoke(cli.app, args)
    assert result.exit_code == 0
    assert "Usage:" in Text.from_ansi(result.output).plain
    _assert_help_options_visible(result.output, expected_names)


def test_main_help_keeps_completion_option_names() -> None:
    result = runner.invoke(cli.app, ["-h"])
    assert result.exit_code == 0
    _assert_help_options_visible(result.output, ["--install-completion", "--show-completion"])


@pytest.mark.parametrize("flag", ["--version", "-V"])
def test_version(flag: str) -> None:
    result = runner.invoke(cli.app, [flag])
    assert result.exit_code == 0
    assert result.output == f"kamino {cli.__version__}\n"


def test_no_args_non_tty_shows_help() -> None:
    result = runner.invoke(cli.app, [])
    assert result.exit_code == 0
    assert "clone" in result.output
    assert "deploy" in result.output


def test_no_args_tty_opens_menu(monkeypatch) -> None:
    monkeypatch.setattr(cli, "_interactive", lambda: True)
    monkeypatch.setattr(cli, "_run_menu", lambda: 3)
    result = runner.invoke(cli.app, [])
    assert result.exit_code == 3


def test_kamino_plain_forces_text_mode(monkeypatch) -> None:
    monkeypatch.setenv("KAMINO_PLAIN", "1")
    monkeypatch.setattr(cli.sys.stdin, "isatty", lambda: True)
    monkeypatch.setattr(cli.sys.stdout, "isatty", lambda: True)
    assert cli._interactive() is False


def test_update_command_runs_use_case(monkeypatch) -> None:
    update_result = SimpleNamespace(
        current_version="0.4.0",
        latest_version="0.5.0",
        target=Path("/tmp/kamino"),
        updated=True,
    )
    calls: list[object] = []
    monkeypatch.setattr(cli, "execute_update", lambda version: update_result)
    monkeypatch.setattr(cli.ui, "print_update_start", lambda: calls.append("start"))
    monkeypatch.setattr(cli.ui, "print_update_result", lambda result: calls.append(result))

    result = runner.invoke(cli.app, ["update"])

    assert result.exit_code == 0
    assert calls == ["start", update_result]


def test_update_command_error_exit_1(monkeypatch) -> None:
    def boom(version: str):
        raise UpdateError("fallo de actualización")

    monkeypatch.setattr(cli, "execute_update", boom)
    monkeypatch.setattr(cli.ui, "print_update_start", lambda: None)
    result = runner.invoke(cli.app, ["update"])

    assert result.exit_code == 1
    assert "fallo de actualización" in result.output


def test_update_tui_when_interactive(monkeypatch) -> None:
    monkeypatch.setattr(cli, "_interactive", lambda: True)
    monkeypatch.setattr(cli, "run_update_flow", lambda: 0)
    result = runner.invoke(cli.app, ["update"])
    assert result.exit_code == 0


def test_deploy_yes_never_opens_tui(monkeypatch) -> None:
    monkeypatch.setattr(cli, "_interactive", lambda: True)
    monkeypatch.setattr(
        cli, "run_deploy_flow", lambda **kwargs: pytest.fail("no debe abrir la TUI")
    )
    monkeypatch.setattr(cli, "execute_deploy", lambda path, **kwargs: _deploy_outcome())

    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", "--yes"])

    assert result.exit_code == 0


def test_deploy_dry_run_never_opens_tui(monkeypatch) -> None:
    monkeypatch.setattr(cli, "_interactive", lambda: True)
    monkeypatch.setattr(
        cli, "run_deploy_flow", lambda **kwargs: pytest.fail("no debe abrir la TUI")
    )
    monkeypatch.setattr(cli, "execute_deploy", lambda path, **kwargs: _deploy_outcome())

    result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", "--dry-run"])

    assert result.exit_code == 0


def test_clone_dry_run_with_tty_opens_review(monkeypatch) -> None:
    monkeypatch.setattr(cli, "_interactive", lambda: True)
    called: list[dict] = []

    def fake_flow(**kwargs):
        called.append(kwargs)
        return 0

    monkeypatch.setattr(cli, "run_clone_flow", fake_flow)
    result = runner.invoke(cli.app, ["clone", "--dry-run"])

    assert result.exit_code == 0
    assert called and called[0]["dry_run"] is True


def test_kamino_plain_keeps_commands_non_interactive(monkeypatch) -> None:
    monkeypatch.setenv("KAMINO_PLAIN", "1")
    monkeypatch.setattr(cli.sys.stdin, "isatty", lambda: True)
    monkeypatch.setattr(cli.sys.stdout, "isatty", lambda: True)
    monkeypatch.setattr(cli, "execute_clone", lambda **kwargs: _clone_outcome())
    monkeypatch.setattr(cli, "run_clone_flow", lambda **kwargs: pytest.fail("no debe abrir la TUI"))

    result = runner.invoke(cli.app, ["clone", "--apps", "git"])

    assert result.exit_code == 0


def test_cli_clone_ignores_config_and_preserves_it(tmp_path: Path, monkeypatch) -> None:
    home = tmp_path / "home"
    target = config_path(home)
    target.parent.mkdir(parents=True, exist_ok=True)
    original = (
        f'[catalog]\ndir = "{home / "catalog"}"\n\n[clone]\noutput_dir = "{home / "clones"}"\n'
    )
    target.write_text(original, encoding="utf-8")
    monkeypatch.setenv("HOME", str(home))
    calls: dict = {}
    monkeypatch.setattr(
        cli,
        "execute_clone",
        lambda **kwargs: calls.update(kwargs) or _clone_outcome(),
    )

    result = runner.invoke(cli.app, ["clone"])

    assert result.exit_code == 0
    assert calls["output"] is None
    assert target.read_text(encoding="utf-8") == original


def test_cli_does_not_create_config(tmp_path: Path, monkeypatch) -> None:
    home = tmp_path / "home"
    monkeypatch.setenv("HOME", str(home))
    monkeypatch.setattr(cli, "execute_clone", lambda **kwargs: _clone_outcome())

    result = runner.invoke(cli.app, ["clone"])

    assert result.exit_code == 0
    assert not config_path(home).exists()


def test_cli_invalid_config_does_not_affect_commands(tmp_path: Path, monkeypatch) -> None:
    home = tmp_path / "home"
    target = config_path(home)
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text("[ui]\nplain = false\n", encoding="utf-8")
    monkeypatch.setenv("HOME", str(home))
    monkeypatch.setattr(cli, "execute_clone", lambda **kwargs: _clone_outcome())
    monkeypatch.setattr(cli, "execute_deploy", lambda path, **kwargs: _deploy_outcome())
    update_result = SimpleNamespace(
        current_version="0.4.0",
        latest_version="0.5.0",
        target=Path("/tmp/kamino"),
        updated=True,
    )
    monkeypatch.setattr(cli, "execute_update", lambda version: update_result)
    monkeypatch.setattr(cli.ui, "print_update_start", lambda: None)
    monkeypatch.setattr(cli.ui, "print_update_result", lambda result: None)

    clone_result = runner.invoke(cli.app, ["clone"])
    deploy_result = runner.invoke(cli.app, ["deploy", "clone.tar.gz", "--yes"])
    update_result_invocation = runner.invoke(cli.app, ["update"])

    assert clone_result.exit_code == 0
    assert deploy_result.exit_code == 0
    assert update_result_invocation.exit_code == 0
