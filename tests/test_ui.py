"""Pruebas de `ui.py`: selección interactiva y resúmenes."""

from __future__ import annotations

from pathlib import Path
from types import SimpleNamespace

from rich.console import Console

from kamino import ui
from kamino.conflicts import ConflictKind


class _FakeCheckbox:
    def __init__(self, result):
        self._result = result

    def ask(self):
        return self._result


def test_select_command_returns_id(monkeypatch) -> None:
    monkeypatch.setattr(ui.questionary, "select", lambda *a, **kw: _FakeCheckbox("clone"))
    assert ui.select_command([("clone", "Clon", "desc")]) == "clone"


def test_select_command_cancel_returns_none(monkeypatch) -> None:
    monkeypatch.setattr(ui.questionary, "select", lambda *a, **kw: _FakeCheckbox(None))
    assert ui.select_command([("clone", "Clon", "desc")]) is None


def test_ask_path_returns_value(monkeypatch) -> None:
    monkeypatch.setattr(ui.questionary, "text", lambda *a, **kw: _FakeCheckbox("x.tar.gz"))
    assert ui.ask_path("Archivo:") == "x.tar.gz"


def test_ask_path_cancel_returns_none(monkeypatch) -> None:
    monkeypatch.setattr(ui.questionary, "text", lambda *a, **kw: _FakeCheckbox(None))
    assert ui.ask_path("Archivo:") is None


def test_select_apps_empty() -> None:
    assert ui.select_apps([]) == []


def test_select_apps_returns_selection(monkeypatch) -> None:
    monkeypatch.setattr(ui.questionary, "checkbox", lambda *a, **kw: _FakeCheckbox(["git"]))
    assert ui.select_apps([("git", "Git")]) == ["git"]


def test_select_apps_cancel_returns_none(monkeypatch) -> None:
    monkeypatch.setattr(ui.questionary, "checkbox", lambda *a, **kw: _FakeCheckbox(None))
    assert ui.select_apps([("git", "Git")]) is None


def test_print_clone_summary_dry_run(monkeypatch) -> None:
    console = Console(record=True)
    monkeypatch.setattr(ui, "_console", console)
    result = SimpleNamespace(
        dry_run=True,
        plan=SimpleNamespace(
            operations=[SimpleNamespace(path="a"), SimpleNamespace(path="b")],
            skipped=["c"],
        ),
        manifest=SimpleNamespace(apps=[]),
        output=None,
    )
    ui.print_clone_summary(result)
    text = console.export_text()
    assert "a" in text and "b" in text and "c" in text


def test_print_clone_summary_verbose(monkeypatch) -> None:
    console = Console(record=True)
    monkeypatch.setattr(ui, "_console", console)
    result = SimpleNamespace(
        dry_run=False,
        plan=SimpleNamespace(operations=[SimpleNamespace(path="a")], skipped=["s"]),
        manifest=SimpleNamespace(apps=[SimpleNamespace(name="Git", files=[1, 2])]),
        output="out.tar.gz",
    )
    ui.print_clone_summary(result, verbose=True)
    text = console.export_text()
    assert "Git" in text and "a" in text and "s" in text


def test_print_deploy_summary(monkeypatch) -> None:
    console = Console(record=True)
    monkeypatch.setattr(ui, "_console", console)
    result = SimpleNamespace(
        dry_run=False,
        plan=SimpleNamespace(entries=[], conflicts=[]),
        manifest=SimpleNamespace(apps=[]),
        deployed=["x"],
        merged=["m"],
        omitted=["y"],
        snapshot="snap.tar.gz",
    )
    ui.print_deploy_summary(result)
    text = console.export_text()
    assert "Desplegadas: 1" in text
    assert "Mezcladas: 1" in text
    assert "Omitidas: 1" in text
    assert "snap.tar.gz" in text


def test_print_deploy_summary_dry_run_with_diffs(monkeypatch) -> None:
    console = Console(record=True)
    monkeypatch.setattr(ui, "_console", console)
    text_conflict = SimpleNamespace(
        kind=ConflictKind.TEXT,
        path="cfg.toml",
        local_text="a = 1\n",
        clone_text="a = 2\n",
    )
    binary_conflict = SimpleNamespace(
        kind=ConflictKind.BINARY,
        path="blob.bin",
        local_size=4,
        clone_size=5,
    )
    link_conflict = SimpleNamespace(
        kind=ConflictKind.SYMLINK,
        path="link",
        local_link="a",
        clone_link="b",
    )
    result = SimpleNamespace(
        dry_run=True,
        plan=SimpleNamespace(
            entries=[],
            conflicts=[text_conflict, binary_conflict, link_conflict],
            notices=[],
        ),
        manifest=SimpleNamespace(apps=[]),
        deployed=[],
        merged=[],
        omitted=["same.toml"],
    )
    ui.print_deploy_summary(result)
    text = console.export_text()
    assert "--- local/cfg.toml" in text
    assert "+++ clon/cfg.toml" in text
    assert "a = 2" in text
    assert "binario" in text
    assert "enlace local -> a" in text
    assert "Omitidas: 1" in text


def test_print_update_start(monkeypatch) -> None:
    console = Console(record=True)
    monkeypatch.setattr(ui, "_console", console)

    ui.print_update_start()

    assert "Comprobando la última versión" in console.export_text()


def test_print_update_result_updated(monkeypatch) -> None:
    console = Console(record=True)
    monkeypatch.setattr(ui, "_console", console)
    result = SimpleNamespace(
        current_version="0.4.0",
        latest_version="0.5.0",
        target=Path("/tmp/kamino"),
        updated=True,
    )

    ui.print_update_result(result)

    text = console.export_text()
    assert "0.4.0 -> 0.5.0" in text
    assert "/tmp/kamino" in text


def test_print_update_result_up_to_date(monkeypatch) -> None:
    console = Console(record=True)
    monkeypatch.setattr(ui, "_console", console)
    result = SimpleNamespace(
        current_version="0.5.0",
        latest_version="0.5.0",
        target=Path("/tmp/kamino"),
        updated=False,
    )

    ui.print_update_result(result)

    assert "ya es la última versión" in console.export_text()


def test_print_update_result_local_is_newer(monkeypatch) -> None:
    console = Console(record=True)
    monkeypatch.setattr(ui, "_console", console)
    result = SimpleNamespace(
        current_version="0.6.0",
        latest_version="0.5.0",
        target=Path("/tmp/kamino"),
        updated=False,
    )

    ui.print_update_result(result)

    text = console.export_text()
    assert "más reciente" in text
    assert "0.5.0" in text
