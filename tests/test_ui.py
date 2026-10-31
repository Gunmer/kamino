"""Pruebas de `ui.py`: selección interactiva y resúmenes."""

from __future__ import annotations

from types import SimpleNamespace

from rich.console import Console

from kamino import ui


def test_select_apps_non_tty_returns_all(monkeypatch) -> None:
    class FakeStdin:
        def isatty(self) -> bool:
            return False

    monkeypatch.setattr("sys.stdin", FakeStdin())
    assert ui.select_apps([("git", "Git"), ("ghostty", "Ghostty")]) == ["git", "ghostty"]


def test_select_apps_empty() -> None:
    assert ui.select_apps([]) == []


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


def test_print_backup_summary_normal(monkeypatch) -> None:
    console = Console(record=True)
    monkeypatch.setattr(ui, "_console", console)
    result = SimpleNamespace(
        dry_run=False,
        plan=SimpleNamespace(operations=[], skipped=[]),
        manifest=SimpleNamespace(apps=[SimpleNamespace(name="Git", files=[1, 2])]),
        output="out.tar.gz",
    )
    ui.print_backup_summary(result)
    text = console.export_text()
    assert "Git" in text and "out.tar.gz" in text


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
