"""Pruebas de `ui.py`: avisos y resúmenes no interactivos."""

from __future__ import annotations

from pathlib import Path
from types import SimpleNamespace

from rich.console import Console

from kamino import texts, ui
from kamino.conflicts import ConflictKind
from kamino.plan import SkippedPath
from kamino.update import UpdateOutcome


def _record(monkeypatch) -> Console:
    console = Console(record=True)
    monkeypatch.setattr(ui, "_console", console)
    return console


def test_privacy_warning_is_visible(monkeypatch) -> None:
    console = _record(monkeypatch)

    ui.print_privacy_warning()

    text = console.export_text().lower()
    assert "no está cifrado" in text
    assert "secretos" in text


def test_print_clone_summary_dry_run(monkeypatch) -> None:
    console = _record(monkeypatch)
    result = SimpleNamespace(
        dry_run=True,
        plan=SimpleNamespace(
            operations=[SimpleNamespace(path="a"), SimpleNamespace(path="b")],
            skipped=[SkippedPath("c", "no existe")],
            notices=["límite"],
        ),
        manifest=SimpleNamespace(apps=[]),
        output=None,
        plan_output=Path("clone-20261028.tar.gz"),
    )
    ui.print_clone_summary(result)
    text = console.export_text()
    assert "a" in text and "b" in text
    assert "c (no existe)" in text
    assert "clone-20261028.tar.gz" in text


def test_print_clone_summary_verbose(monkeypatch) -> None:
    console = _record(monkeypatch)
    result = SimpleNamespace(
        dry_run=False,
        plan=SimpleNamespace(
            operations=[SimpleNamespace(path="a")],
            skipped=[SkippedPath("s", "enlace no seguro")],
            notices=[],
        ),
        manifest=SimpleNamespace(apps=[SimpleNamespace(name="Git", files=[1, 2])]),
        output="out.tar.gz",
    )
    ui.print_clone_summary(result, verbose=True)
    text = console.export_text()
    assert "Git" in text and "a" in text and "s (enlace no seguro)" in text


def test_print_deploy_summary_uses_new_vocabulary(monkeypatch) -> None:
    console = _record(monkeypatch)
    result = SimpleNamespace(
        dry_run=False,
        plan=SimpleNamespace(entries=[], conflicts=[], notices=[]),
        manifest=SimpleNamespace(apps=[]),
        deployed=["x"],
        merged=["m"],
        omitted=["y"],
        unchanged=["same.toml"],
        not_installed=["otra"],
        missing_requested=[],
        snapshot=Path("snap.tar.gz"),
    )
    ui.print_deploy_summary(result)
    text = console.export_text()
    assert "Desplegados: 1" in text
    assert "Híbridos: 1" in text
    assert "Omitidos: 1" in text
    assert "Sin cambios: 1" in text
    assert "Holocrón: snap.tar.gz" in text
    assert "snapshot" not in text.lower()
    assert "mezclad" not in text.lower()
    assert "idéntic" not in text.lower()


def test_print_deploy_summary_dry_run_with_diffs(monkeypatch) -> None:
    console = _record(monkeypatch)
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
        unchanged=[],
        not_installed=[],
        missing_requested=[],
    )
    ui.print_deploy_summary(result)
    text = console.export_text()
    assert "--- original/cfg.toml" in text
    assert "+++ clon/cfg.toml" in text
    assert "a = 2" in text
    assert "binario" in text
    assert "enlace original -> a" in text
    assert "Omitidos: 1" in text


def test_print_update_start(monkeypatch) -> None:
    console = _record(monkeypatch)

    ui.print_update_start()

    assert "Consultando la última versión" in console.export_text()


def test_print_update_result_updated(monkeypatch) -> None:
    console = _record(monkeypatch)
    result = SimpleNamespace(
        current_version="0.4.0",
        latest_version="0.5.0",
        target=Path("/tmp/kamino"),
        updated=True,
        outcome=UpdateOutcome.UPDATED,
    )

    ui.print_update_result(result)

    text = console.export_text()
    assert "0.4.0 → 0.5.0" in text
    assert "/tmp/kamino" in text
    assert "Reinicia kamino" in text


def test_print_update_result_up_to_date(monkeypatch) -> None:
    console = _record(monkeypatch)
    result = SimpleNamespace(
        current_version="0.5.0",
        latest_version="0.5.0",
        target=Path("/tmp/kamino"),
        updated=False,
        outcome=UpdateOutcome.UP_TO_DATE,
    )

    ui.print_update_result(result)

    assert "ya es la última versión" in console.export_text()


def test_print_update_result_installed_is_newer(monkeypatch) -> None:
    console = _record(monkeypatch)
    result = SimpleNamespace(
        current_version="0.6.0",
        latest_version="0.5.0",
        target=Path("/tmp/kamino"),
        updated=False,
        outcome=UpdateOutcome.NEWER,
    )

    ui.print_update_result(result)

    text = console.export_text()
    assert "más reciente" in text
    assert "0.5.0" in text


def test_visible_glossary_keeps_internal_names_out_of_the_surface() -> None:
    assert texts.ORIGINAL == "original"
    assert texts.SNAPSHOT == "holocrón de seguridad"
    assert texts.HYBRID == "híbrido"
    assert texts.PERFECT_CLONE == "clon perfecto"
    assert texts.NOT_INSTALLED == "sin base en este Mac"
    assert texts.FORCE_DISTURBANCE == "perturbación en la Fuerza"
    assert texts.INTEGRITY_FAILED == "clon con mutaciones"
    assert texts.MANIFEST == "registro de los kaminoanos"
