"""Pruebas de `ui.py`: selección interactiva y resúmenes."""

from __future__ import annotations

from types import SimpleNamespace

from rich.console import Console

from kamino import ui


class _FakeCheckbox:
    def __init__(self, result):
        self._result = result

    def ask(self):
        return self._result


def test_select_apps_empty() -> None:
    assert ui.select_apps([]) == []


def test_select_apps_returns_selection(monkeypatch) -> None:
    monkeypatch.setattr(ui.questionary, "checkbox", lambda *a, **kw: _FakeCheckbox(["git"]))
    assert ui.select_apps([("git", "Git")]) == ["git"]


def test_select_apps_cancel_returns_none(monkeypatch) -> None:
    monkeypatch.setattr(ui.questionary, "checkbox", lambda *a, **kw: _FakeCheckbox(None))
    assert ui.select_apps([("git", "Git")]) is None


def test_print_backup_summary_dry_run(monkeypatch) -> None:
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
    ui.print_backup_summary(result)
    text = console.export_text()
    assert "a" in text and "b" in text and "c" in text


def test_print_backup_summary_verbose(monkeypatch) -> None:
    console = Console(record=True)
    monkeypatch.setattr(ui, "_console", console)
    result = SimpleNamespace(
        dry_run=False,
        plan=SimpleNamespace(operations=[SimpleNamespace(path="a")], skipped=["s"]),
        manifest=SimpleNamespace(apps=[SimpleNamespace(name="Git", files=[1, 2])]),
        output="out.tar.gz",
    )
    ui.print_backup_summary(result, verbose=True)
    text = console.export_text()
    assert "Git" in text and "a" in text and "s" in text


def test_print_restore_summary(monkeypatch) -> None:
    console = Console(record=True)
    monkeypatch.setattr(ui, "_console", console)
    result = SimpleNamespace(
        dry_run=False,
        plan=SimpleNamespace(entries=[]),
        manifest=SimpleNamespace(apps=[]),
        restored=["x"],
        omitted=["y"],
        snapshot="snap.tar.gz",
    )
    ui.print_restore_summary(result)
    text = console.export_text()
    assert "Restauradas: 1" in text
    assert "Omitidas: 1" in text
    assert "snap.tar.gz" in text
