"""Pruebas headless de la app única y sus flujos."""

from __future__ import annotations

import asyncio
import time
from pathlib import Path
from types import SimpleNamespace

import pytest
from textual.widgets import Button, DataTable, Input, OptionList, Static

from kamino import texts
from kamino.archive import FileEntry
from kamino.catalog.models import AppDefinition
from kamino.clone import run_clone
from kamino.conflicts import LocalFile, build_conflict
from kamino.deploy import prepare_deploy
from kamino.errors import ArchiveError, OperationCancelled, PartialApplyError
from kamino.tui import clone as clone_tui
from kamino.tui import deploy as deploy_tui
from kamino.tui import update as update_tui
from kamino.tui.app import KaminoApp
from kamino.tui.clone import (
    CloneAppsScreen,
    CloneDestinationScreen,
    CloneReviewScreen,
    CloneState,
    CloneSummaryScreen,
    ReemplazarDialog,
    clone_initial_screen,
)
from kamino.tui.common import ChoiceDialog, ErrorScreen
from kamino.tui.deploy import (
    DeployAppsScreen,
    DeployConfirmScreen,
    DeployConflictsScreen,
    DeployDetailScreen,
    DeployFileScreen,
    DeployPrepareScreen,
    DeployState,
    DeploySummaryScreen,
)
from kamino.tui.menu import MenuScreen
from kamino.tui.update import UpdateScreen
from kamino.update import ReleaseInfo, UpdateOutcome


def _run(coroutine) -> None:
    asyncio.run(coroutine)


def _app_def(app_id: str = "foo", paths: list[str] | None = None) -> AppDefinition:
    return AppDefinition.model_validate(
        {
            "id": app_id,
            "name": app_id.title(),
            "command": app_id,
            "paths": paths if paths is not None else [".config/foo/config.toml"],
        }
    )


def _source_home(tmp_path: Path) -> tuple[Path, dict[str, AppDefinition], str]:
    home = tmp_path / "home"
    (home / ".config/foo").mkdir(parents=True)
    (home / ".config/foo/config.toml").write_text("clone\n", encoding="utf-8")
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    for command in ("foo", "bar"):
        binary = bin_dir / command
        binary.write_text("", encoding="utf-8")
        binary.chmod(0o755)
    catalog = {"foo": _app_def()}
    return home, catalog, str(bin_dir)


def _clone_file(tmp_path: Path) -> tuple[Path, Path, dict[str, AppDefinition], str]:
    home, catalog, bin_path = _source_home(tmp_path)
    output = tmp_path / "clone.tar.gz"
    run_clone(
        home=home,
        output=output,
        catalog=catalog,
        application_dirs=[],
        path=bin_path,
    )
    return output, home, catalog, bin_path


def _text_conflict() -> object:
    local = "".join(f"line {index}\n" for index in range(20))
    clone = local.replace("line 1\n", "LINE ONE\n").replace("line 18\n", "LINE EIGHTEEN\n")
    conflict = build_conflict(
        app_id="foo",
        path=".config/foo/config.yaml",
        destination=Path("/home/user/.config/foo/config.yaml"),
        entry=FileEntry(
            path=".config/foo/config.yaml",
            sha256="0" * 64,
            size=len(clone),
            mode="0644",
        ),
        local=LocalFile(kind="file", data=local.encode(), mode="0600"),
        clone_content=clone.encode(),
    )
    assert conflict is not None
    return conflict


# --- menú y transversales ---


def test_menu_opens_with_wordmark_phrase_and_no_last_mission() -> None:
    app = KaminoApp(mode="menu", phrase="frase fija", show_logo=True)

    async def scenario() -> None:
        async with app.run_test(size=(80, 30)) as pilot:
            await pilot.pause()
            assert isinstance(app.screen, MenuScreen)
            menu = app.screen
            assert "KAMINO" not in str(menu.query_one("#wordmark", Static).content)
            assert "██" in str(menu.query_one("#wordmark", Static).content)
            assert "frase fija" in str(menu.query_one("#phrase", Static).content)
            assert menu.query_one("#last-mission", Static).display is False
            await pilot.press("escape")
            await pilot.pause()
            assert isinstance(app.screen, MenuScreen)
            await pilot.press("q")
            await pilot.pause()
            assert app.return_value == 0

    _run(scenario())


def test_menu_degrades_by_height_and_recovers() -> None:
    app = KaminoApp(mode="menu", phrase="frase", show_logo=True)

    async def scenario() -> None:
        async with app.run_test(size=(80, 24)) as pilot:
            await pilot.pause()
            menu = app.screen
            assert menu.query_one("#subtitle", Static).display is False
            assert menu.query_one("#capsules", Static).display is False
            assert menu.query_one("#menu").display is True
            await pilot.resize_terminal(80, 26)
            await asyncio.sleep(0.3)
            assert menu.query_one("#capsules", Static).display is True
            assert menu.query_one("#subtitle", Static).display is False
            await pilot.resize_terminal(80, 30)
            await asyncio.sleep(0.3)
            assert menu.query_one("#subtitle", Static).display is True
            await pilot.resize_terminal(79, 23)
            await asyncio.sleep(0.3)
            assert list(menu.query("#size-guard"))
            await pilot.resize_terminal(80, 24)
            await asyncio.sleep(0.3)
            assert not list(menu.query("#size-guard"))
            assert app.size == (80, 24)

    _run(scenario())


def test_menu_hides_logo_without_utf8_or_with_no_logo() -> None:
    app = KaminoApp(mode="menu", phrase="frase", show_logo=False)

    async def scenario() -> None:
        async with app.run_test(size=(80, 30)) as pilot:
            await pilot.pause()
            assert app.screen.query_one("#wordmark", Static).display is False

    _run(scenario())


def test_menu_arrows_move_selection_and_enter_accepts() -> None:
    app = KaminoApp(mode="menu", phrase="frase", show_logo=False)

    async def scenario() -> None:
        async with app.run_test(size=(80, 30)) as pilot:
            await pilot.pause()
            options = app.screen.query_one("#menu-options", OptionList)
            assert options.highlighted == 0
            await pilot.press("down")
            await pilot.pause()
            assert options.highlighted == 1
            await pilot.press("up")
            await pilot.pause()
            assert options.highlighted == 0
            await pilot.press("down", "down", "down")
            await pilot.pause()
            assert options.highlighted == 3
            await pilot.press("enter")
            await pilot.pause()
            assert app.return_value == 0

    _run(scenario())


def test_menu_q_returns_last_code_and_shows_last_mission() -> None:
    app = KaminoApp(mode="menu", phrase="frase", show_logo=True)

    async def scenario() -> None:
        async with app.run_test(size=(80, 30)) as pilot:
            await pilot.pause()
            app.session.last_mission = "Clon completado"
            app.session.last_code = 3
            app.screen.refresh_mission()
            await pilot.pause()
            assert "Clon completado" in str(app.screen.query_one("#last-mission", Static).content)
            await pilot.press("q")
            await pilot.pause()
            assert app.return_value == 3

    _run(scenario())


def test_menu_clone_flow_returns_to_menu_with_mission() -> None:
    app = KaminoApp(mode="menu", phrase="frase", show_logo=True)

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await pilot.press("c")
            await pilot.pause()
            assert isinstance(app.screen, CloneAppsScreen)
            app.screen.action_cancel()
            await pilot.pause()
            assert isinstance(app.screen, MenuScreen)
            assert app.session.last_code == 1
            assert app.session.last_mission

    _run(scenario())


def test_direct_flow_does_not_open_menu() -> None:
    app = KaminoApp(mode="direct", initial=CloneAppsScreen(CloneState()), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            assert isinstance(app.screen, CloneAppsScreen)
            app.screen.action_cancel()
            await pilot.pause()
            assert app.return_value == 1
            assert all(not isinstance(screen, MenuScreen) for screen in app.screen_stack)

    _run(scenario())


def test_error_screen_has_four_parts_and_detail_toggle(monkeypatch) -> None:
    copied: list[str] = []
    app = KaminoApp(mode="direct", phrase="frase")
    monkeypatch.setattr(app, "copy_text", lambda text: copied.append(text) or True)

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            app.push_screen(ErrorScreen("falló", detail="detalle"))
            await pilot.pause()
            screen = app.screen
            assert isinstance(screen, ErrorScreen)
            assert "Qué pasó" in str(screen.query_one("#error-what", Static).content)
            assert "Estado de seguridad" in str(screen.query_one("#error-safety", Static).content)
            assert "Qué hacer" in str(screen.query_one("#error-action", Static).content)
            assert screen.query_one("#error-detail", Static).display is False
            await pilot.press("t")
            await pilot.pause()
            assert screen.query_one("#error-detail", Static).display is True
            await pilot.press("c")
            await pilot.pause()
            assert copied == ["detalle"]

    _run(scenario())


# --- C1 y C2 ---


def test_c1_lists_apps_and_disables_zero_file_apps(tmp_path: Path) -> None:
    home, catalog, bin_path = _source_home(tmp_path)
    empty = _app_def("empty", [".config/empty/config.toml"])
    catalog["empty"] = empty
    catalog["zzz"] = _app_def("zzz")
    (Path(bin_path) / "empty").write_text("", encoding="utf-8")
    (Path(bin_path) / "empty").chmod(0o755)
    state = CloneState(home=home, catalog=catalog, application_dirs=[], path=bin_path)
    app = KaminoApp(mode="direct", initial=CloneAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.3)
            await pilot.pause()
            screen = app.screen
            table = screen.query_one(DataTable)
            rows = [table.get_row_at(i) for i in range(table.row_count)]
            assert any("Foo" in str(row[1]) and "1 archivos" in str(row[2]) for row in rows)
            assert any("Empty" in str(row[1]) and "sin archivos" in str(row[2]) for row in rows)
            assert "zzz" in str(screen.query_one("#not-detected", Static).content)
            await pilot.press("a")
            await pilot.pause()
            assert screen.selected == {"foo"}
            assert screen.query_one("#shortcut-bar", Static).content
            assert not list(screen.query(Button))

    _run(scenario())


def test_c1_filter_and_visible_actions(tmp_path: Path) -> None:
    home, catalog, bin_path = _source_home(tmp_path)
    second = _app_def("bar", [".config/bar/config.toml"])
    (home / ".config/bar").mkdir(parents=True)
    (home / ".config/bar/config.toml").write_text("bar\n")
    catalog["bar"] = second
    state = CloneState(home=home, catalog=catalog, application_dirs=[], path=bin_path)
    app = KaminoApp(mode="direct", initial=CloneAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.3)
            await pilot.pause()
            screen = app.screen
            await pilot.press("slash")
            await pilot.pause()
            filter_input = screen.query_one("#filter", Input)
            assert filter_input.display is True
            filter_input.value = "foo"
            await pilot.pause()
            await pilot.press("a")
            await pilot.pause()
            assert screen.selected == {"foo"}
            filter_input.value = "bar"
            await pilot.pause()
            await pilot.press("n")
            await pilot.pause()
            assert screen.selected == {"foo"}
            await pilot.press("escape")
            await pilot.pause()
            assert filter_input.display is False
            await pilot.press("n")
            await pilot.pause()
            assert screen.selected == set()

    _run(scenario())


def test_c1_space_toggles_without_losing_cursor_or_focus(tmp_path: Path) -> None:
    home, catalog, bin_path = _source_home(tmp_path)
    (home / ".config/bar").mkdir(parents=True)
    (home / ".config/bar/config.toml").write_text("bar\n", encoding="utf-8")
    catalog["bar"] = _app_def("bar", [".config/bar/config.toml"])
    (Path(bin_path) / "bar").write_text("", encoding="utf-8")
    (Path(bin_path) / "bar").chmod(0o755)
    state = CloneState(home=home, catalog=catalog, application_dirs=[], path=bin_path)
    app = KaminoApp(mode="direct", initial=CloneAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.3)
            await pilot.pause()
            screen = app.screen
            table = screen.query_one(DataTable)
            table.focus()
            table.move_cursor(row=1)
            await pilot.pause()
            assert table.cursor_row == 1
            await pilot.press("space")
            await pilot.pause()
            assert screen.selected == {"foo"}
            assert table.cursor_row == 1
            assert app.focused is table
            await pilot.press("space")
            await pilot.pause()
            assert screen.selected == set()
            assert table.cursor_row == 1
            assert app.focused is table

    _run(scenario())


def test_c1_to_c2_preserves_selection_when_going_back(tmp_path: Path) -> None:
    home, catalog, bin_path = _source_home(tmp_path)
    state = CloneState(home=home, catalog=catalog, application_dirs=[], path=bin_path)
    app = KaminoApp(mode="direct", initial=CloneAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.3)
            await pilot.pause()
            await pilot.press("a")
            await pilot.pause()
            await pilot.press("enter")
            await pilot.pause()
            assert isinstance(app.screen, CloneDestinationScreen)
            assert not list(app.screen.query(Button))
            assert app.screen.query_one("#shortcut-bar", Static).content
            await pilot.press("escape")
            await pilot.pause()
            assert isinstance(app.screen, CloneAppsScreen)
            assert app.screen.selected == {"foo"}

    _run(scenario())


def test_pushing_screen_hides_previous_and_pop_restores(tmp_path: Path) -> None:
    home, catalog, bin_path = _source_home(tmp_path)
    state = CloneState(home=home, catalog=catalog, application_dirs=[], path=bin_path)
    app = KaminoApp(mode="direct", initial=CloneAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.3)
            await pilot.pause()
            c1 = app.screen
            app.push_screen(CloneDestinationScreen(state))
            await pilot.pause()
            assert c1.display is False
            app.pop_screen()
            await pilot.pause()
            assert c1.display is True
            assert app.screen is c1

    _run(scenario())


def test_c2_default_name_validation_and_shortcuts(tmp_path: Path, monkeypatch) -> None:
    home, catalog, bin_path = _source_home(tmp_path)
    monkeypatch.chdir(tmp_path)
    state = CloneState(
        home=home,
        catalog=catalog,
        application_dirs=[],
        path=bin_path,
        include_apps=False,
        include_destination=True,
    )
    app = KaminoApp(mode="direct", initial=CloneDestinationScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            path_input = app.screen.query_one("#path", Input)
            assert Path(path_input.value).name.startswith("clone-")
            assert path_input.value.endswith(".tar.gz")
            path_input.value = str(tmp_path / "no-existe" / "clone.tar.gz")
            await pilot.pause()
            assert app.screen.query_one("#status", Static).content
            await pilot.press("f3")
            await pilot.pause()
            assert "Downloads" in path_input.value or path_input.value.endswith(".tar.gz")

    _run(scenario())


def test_c2_existing_clone_opens_dialog_with_focus_on_back(tmp_path: Path) -> None:
    home, catalog, bin_path = _source_home(tmp_path)
    output = tmp_path / "clone.tar.gz"
    output.write_bytes(b"anterior")
    state = CloneState(
        home=home,
        catalog=catalog,
        application_dirs=[],
        path=bin_path,
        output=output,
        include_apps=False,
        include_destination=True,
    )
    app = KaminoApp(mode="direct", initial=CloneDestinationScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await pilot.press("enter")
            await pilot.pause()
            dialog = app.screen
            assert isinstance(dialog, ReemplazarDialog)
            options = dialog.query_one("#dialog-options", OptionList)
            assert app.focused is options
            assert options.highlighted == 2
            await pilot.press("up", "up", "enter")
            await pilot.pause()
            assert isinstance(app.screen, CloneDestinationScreen)
            assert app.screen.query_one("#path", Input).value.endswith("-1.tar.gz")

    _run(scenario())


def test_c2_force_skips_dialog_and_announces_replacement(tmp_path: Path) -> None:
    home, catalog, bin_path = _source_home(tmp_path)
    output = tmp_path / "clone.tar.gz"
    output.write_bytes(b"anterior")
    state = CloneState(
        home=home,
        catalog=catalog,
        application_dirs=[],
        path=bin_path,
        output=output,
        force=True,
        include_apps=False,
        include_destination=True,
    )
    app = KaminoApp(mode="direct", initial=CloneDestinationScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            assert "reemplazará" in str(app.screen.query_one("#force-note", Static).content)
            await pilot.press("enter")
            await pilot.pause()
            assert isinstance(app.screen, CloneReviewScreen)

    _run(scenario())


def test_clone_initial_screen_omits_screens_with_valid_arguments(tmp_path: Path) -> None:
    state = CloneState(
        output=tmp_path / "clone.tar.gz",
        selected=["foo"],
        include_apps=False,
        include_destination=False,
    )
    assert isinstance(clone_initial_screen(state), CloneReviewScreen)

    state = CloneState(output=tmp_path / "clone.tar.gz", include_apps=True)
    assert isinstance(clone_initial_screen(state), CloneAppsScreen)

    state = CloneState(selected=["foo"], include_apps=False)
    assert isinstance(clone_initial_screen(state), CloneDestinationScreen)

    state = CloneState(selected=["foo"], include_apps=False, include_destination=False)
    assert isinstance(clone_initial_screen(state), CloneReviewScreen)


# --- C3 y C4 ---


def _review_app(tmp_path: Path, **state_kwargs) -> KaminoApp:
    home, catalog, bin_path = _source_home(tmp_path)
    state = CloneState(
        home=home,
        catalog=catalog,
        application_dirs=[],
        path=bin_path,
        output=tmp_path / "out.tar.gz",
        selected=["foo"],
        include_apps=False,
        include_destination=False,
        **state_kwargs,
    )
    return KaminoApp(mode="direct", initial=CloneReviewScreen(state), phrase="frase")


def test_c3_shows_plan_skipped_and_included_files(tmp_path: Path) -> None:
    app = _review_app(tmp_path)

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.3)
            await pilot.pause()
            screen = app.screen
            summary = str(screen.query_one("#summary", Static).content)
            assert "Apps: 1" in summary
            assert "Archivos: 1" in summary
            assert "out.tar.gz" in summary
            assert "no está cifrado" in str(screen.query_one("#secrets", Static).content)
            assert screen.query_one("#skipped", Static).display is False
            await pilot.press("f")
            await pilot.pause()
            assert screen.query_one("#files", Static).display is True
            assert ".config/foo/config.toml" in str(screen.query_one("#files", Static).content)
            await pilot.press("d")
            await pilot.pause()
            assert screen.query_one("#skipped", Static).display is True

    _run(scenario())


def test_c3_dry_run_has_no_clone_action_and_exits(tmp_path: Path) -> None:
    app = _review_app(tmp_path, dry_run=True)

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.3)
            await pilot.pause()
            screen = app.screen
            assert not list(screen.query(Button))
            assert "terminar simulación" in str(screen.query_one("#shortcut-bar", Static).content)
            await pilot.press("enter")
            await pilot.pause()
            assert app.return_value == 0
            assert not (tmp_path / "out.tar.gz").exists()

    _run(scenario())


def test_c3_writes_and_opens_summary(tmp_path: Path) -> None:
    app = _review_app(tmp_path)

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.3)
            await pilot.pause()
            await pilot.press("enter")
            for _ in range(30):
                await asyncio.sleep(0.1)
                await pilot.pause()
                if isinstance(app.screen, CloneSummaryScreen):
                    break
            assert isinstance(app.screen, CloneSummaryScreen)
            assert (tmp_path / "out.tar.gz").exists()
            assert app.screen.query_one("#exit-line", Static).display is False
            await pilot.press("enter")
            await pilot.pause()
            assert app.return_value == 0

    _run(scenario())


def test_c3_cancel_removes_temp_and_keeps_screen(tmp_path: Path, monkeypatch) -> None:
    app = _review_app(tmp_path)

    def slow(
        planned,
        *,
        home=None,
        force=False,
        expected_state=None,
        progress=None,
        cancelled=None,
    ):
        while not cancelled():
            time.sleep(0.01)
        raise OperationCancelled("cancelada")

    monkeypatch.setattr(clone_tui, "write_clone", slow)

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.3)
            await pilot.pause()
            await pilot.press("enter")
            await pilot.pause()
            await pilot.press("ctrl+c")
            await asyncio.sleep(0.2)
            await pilot.pause()
            assert isinstance(app.screen, CloneReviewScreen)
            assert "Cancelada" in str(app.screen.query_one("#status", Static).content) or (
                "cancelada" in str(app.screen.query_one("#status", Static).content).lower()
            )
            assert not (tmp_path / "out.tar.gz").exists()

    _run(scenario())


def test_c4_shows_code_3_and_copies_path(tmp_path: Path) -> None:
    home, catalog, bin_path = _source_home(tmp_path)
    (home / "falta.toml").unlink(missing_ok=True)
    state = CloneState(
        home=home,
        catalog={"foo": _app_def("foo", [".config/foo/config.toml", "falta.toml"])},
        application_dirs=[],
        path=bin_path,
        output=tmp_path / "out.tar.gz",
        selected=["foo"],
        include_apps=False,
        include_destination=False,
    )
    app = KaminoApp(
        mode="direct",
        initial=CloneReviewScreen(state),
        phrase="frase",
    )
    copied: list[str] = []
    monkeypatch = pytest.MonkeyPatch()
    monkeypatch.setattr(app, "copy_text", lambda text: copied.append(text) or False)

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.3)
            await pilot.pause()
            await pilot.press("enter")
            for _ in range(30):
                await asyncio.sleep(0.1)
                await pilot.pause()
                if isinstance(app.screen, CloneSummaryScreen):
                    break
            screen = app.screen
            assert isinstance(screen, CloneSummaryScreen)
            assert screen.code == 3
            assert screen.query_one("#exit-line", Static).display is True
            await pilot.press("d")
            await pilot.pause()
            assert "falta.toml" in str(screen.query_one("#skipped", Static).content)
            await pilot.press("c")
            await pilot.pause()
            assert copied == [str(tmp_path / "out.tar.gz")]
            assert "OSC 52" in str(screen.query_one("#status", Static).content)

    try:
        _run(scenario())
    finally:
        monkeypatch.undo()


# --- D0-D5 ---


def _deploy_state(tmp_path: Path, *, conflict: bool = True) -> DeployState:
    clone, origin, catalog, bin_path = _clone_file(tmp_path)
    target = tmp_path / "target"
    (target / ".config/foo").mkdir(parents=True)
    target_file = target / ".config/foo/config.toml"
    if conflict:
        target_file.write_text("local\n", encoding="utf-8")
    else:
        target_file.write_text("clone\n", encoding="utf-8")
    session = prepare_deploy(
        clone,
        home=target,
        catalog=catalog,
        application_dirs=[],
        path=bin_path,
        snapshots_dir=tmp_path / "snapshots",
    )
    return DeployState(
        home=target,
        catalog=catalog,
        application_dirs=[],
        path=bin_path,
        snapshots_dir=tmp_path / "snapshots",
        clone_file=clone,
        session=session,
        include_file=False,
        include_apps=True,
    )


def test_d0_lists_tar_gz_and_rejects_invalid_file(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.chdir(tmp_path)
    clone, _, _, _ = _clone_file(tmp_path)
    app = KaminoApp(mode="direct", initial=DeployFileScreen(DeployState()), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            table = app.screen.query_one(DataTable)
            assert table.row_count == 1
            path_input = app.screen.query_one("#path", Input)
            path_input.value = str(clone)
            await pilot.pause()
            await pilot.press("enter")
            await pilot.pause()
            assert isinstance(app.screen, DeployAppsScreen)
            assert app.screen.state.session is not None

    _run(scenario())


def test_d0_blocks_invalid_file(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.chdir(tmp_path)
    bad = tmp_path / "bad.tar.gz"
    bad.write_bytes(b"no tar")
    app = KaminoApp(mode="direct", initial=DeployFileScreen(DeployState()), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            app.screen.query_one("#path", Input).value = str(bad)
            await pilot.pause()
            await pilot.press("enter")
            await pilot.pause()
            assert isinstance(app.screen, DeployFileScreen)
            assert "no es un clon válido" in str(app.screen.query_one("#status", Static).content)

    _run(scenario())


def test_d1_lists_apps_and_not_installed(tmp_path: Path) -> None:
    state = _deploy_state(tmp_path)
    app = KaminoApp(mode="direct", initial=DeployAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            assert app.screen.query_one(DataTable).row_count == 1
            assert "sin base" in str(app.screen.query_one("#not-installed", Static).content) or (
                app.screen.query_one("#not-installed", Static).content == ""
            )
            await pilot.press("a")
            await pilot.pause()
            assert app.screen.selected == {"foo"}
            await pilot.press("n")
            await pilot.pause()
            assert app.screen.selected == set()

    _run(scenario())


def test_d1_plan_opens_conflicts(tmp_path: Path) -> None:
    state = _deploy_state(tmp_path)
    app = KaminoApp(mode="direct", initial=DeployAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await pilot.press("a")
            await pilot.press("enter")
            for _ in range(30):
                await asyncio.sleep(0.1)
                await pilot.pause()
                if isinstance(app.screen, DeployConflictsScreen):
                    break
            assert isinstance(app.screen, DeployConflictsScreen)
            assert state.plan is not None
            assert state.plan.conflicts

    _run(scenario())


def test_d1_checksum_failure_returns_to_d0(tmp_path: Path, monkeypatch) -> None:

    state = _deploy_state(tmp_path)
    app = KaminoApp(mode="direct", initial=DeployAppsScreen(state), phrase="frase")

    def corrupt(path, manifest, **kwargs):
        raise ArchiveError("clon con mutaciones")

    monkeypatch.setattr(deploy_tui, "DeploySession", deploy_tui.DeploySession)
    monkeypatch.setattr("kamino.deploy.verify_archive", corrupt)
    monkeypatch.setattr("kamino.archive.verify_archive", corrupt)

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await pilot.press("a")
            await pilot.press("enter")
            for _ in range(30):
                await asyncio.sleep(0.1)
                await pilot.pause()
                if isinstance(app.screen, ErrorScreen):
                    break
            assert isinstance(app.screen, ErrorScreen)
            await pilot.press("d")
            await pilot.pause()
            assert isinstance(app.screen, DeployFileScreen)
            assert (state.home / ".config/foo/config.toml").read_text() != "clone\n"

    _run(scenario())


def test_d2_counts_actions_and_confirm(tmp_path: Path) -> None:
    state = _deploy_state(tmp_path)
    app = KaminoApp(mode="direct", initial=DeployAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await pilot.press("a")
            await pilot.press("enter")
            for _ in range(30):
                await asyncio.sleep(0.1)
                await pilot.pause()
                if isinstance(app.screen, DeployConflictsScreen):
                    break
            screen = app.screen
            assert isinstance(screen, DeployConflictsScreen)
            assert screen.query_one(DataTable).row_count == 1
            assert "Falta decidir" in str(screen.query_one("#status", Static).content)
            await pilot.press("c")
            await pilot.pause()
            assert state.decisions.get(state.plan.conflicts[0].path).action.value == "apply"
            assert "Todos los conflictos" in str(screen.query_one("#status", Static).content)
            await pilot.press("o")
            await pilot.pause()
            assert state.decisions.get(state.plan.conflicts[0].path).action.value == "omit"
            await pilot.press("C")
            await pilot.press("enter")
            await pilot.pause()
            assert isinstance(app.screen, DeployConfirmScreen)

    _run(scenario())


def test_d2_back_discards_after_confirmation(tmp_path: Path) -> None:
    state = _deploy_state(tmp_path)
    app = KaminoApp(mode="direct", initial=DeployAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await pilot.press("a")
            await pilot.press("enter")
            for _ in range(30):
                await asyncio.sleep(0.1)
                await pilot.pause()
                if isinstance(app.screen, DeployConflictsScreen):
                    break
            await pilot.press("C")
            await pilot.pause()
            await pilot.press("escape")
            await pilot.pause()
            assert isinstance(app.screen, ChoiceDialog)
            assert app.screen.query_one("#dialog-options", OptionList).highlighted == 1
            await pilot.press("up", "enter")
            await pilot.pause()
            assert isinstance(app.screen, DeployAppsScreen)
            assert state.decisions.decisions == {}

    _run(scenario())


def test_d3_block_view_toggles_and_validates(tmp_path: Path) -> None:
    from textual.app import ComposeResult
    from textual.screen import Screen

    conflict = _text_conflict()
    results: list[object] = []

    class Host(Screen[None]):
        BINDINGS = []

        def compose(self) -> ComposeResult:
            yield Static("host", id="host")

        def on_mount(self) -> None:
            self.app.push_screen(
                DeployDetailScreen(conflict, None), lambda decision: results.append(decision)
            )

    app = KaminoApp(mode="direct", initial=Host(), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await pilot.pause()
            screen = app.screen
            assert screen.block == 0
            assert screen.cursor == 0
            body = str(screen.query_one("#block-body", Static).content)
            assert "● clon" in body and "○ original" in body
            assert "Clon: 20 líneas" in str(screen.query_one("#block-footer", Static).content)
            await pilot.press("space")
            await pilot.pause()
            assert screen.decisions[0].value == "local"
            await pilot.press("tab")
            await pilot.pause()
            assert screen.block == 1
            await pilot.press("c")
            await pilot.pause()
            assert screen.decisions[1].value == "clone"
            await pilot.press("shift+tab")
            await pilot.pause()
            assert screen.block == 0
            await pilot.press("O")
            await pilot.pause()
            assert all(decision.value == "local" for decision in screen.decisions)
            await pilot.press("escape")
            await pilot.pause()
            assert isinstance(app.screen, Host)
            assert results and results[0].action.value == "review"

    _run(scenario())


def test_d3_handles_20k_lines_without_mounting_thousands_of_widgets() -> None:
    local = "".join(f"line {index}\n" for index in range(20_000))
    clone = local.replace("line 10000\n", "LINE TEN THOUSAND\n")
    conflict = build_conflict(
        app_id="foo",
        path="big.txt",
        destination=Path("/tmp/big.txt"),
        entry=FileEntry(path="big.txt", sha256="0" * 64, size=len(clone), mode="0644"),
        local=LocalFile(kind="file", data=local.encode(), mode="0644"),
        clone_content=clone.encode(),
    )
    assert conflict is not None
    app = KaminoApp(mode="direct", initial=DeployDetailScreen(conflict, None), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            screen = app.screen
            assert len(screen.query(Static)) < 20
            assert screen.conflict is conflict

    _run(scenario())


def test_d4_confirm_counts_and_back_does_not_write(tmp_path: Path) -> None:
    state = _deploy_state(tmp_path)
    app = KaminoApp(mode="direct", initial=DeployAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await pilot.press("a")
            await pilot.press("enter")
            for _ in range(30):
                await asyncio.sleep(0.1)
                await pilot.pause()
                if isinstance(app.screen, DeployConflictsScreen):
                    break
            await pilot.press("C")
            await pilot.press("enter")
            await pilot.pause()
            confirm = app.screen
            assert isinstance(confirm, DeployConfirmScreen)
            assert "Archivos protegidos: 1" in str(confirm.query_one("#protected", Static).content)
            options = confirm.query_one("#confirm-options", OptionList)
            assert app.focused is options
            assert options.highlighted == 1
            counts = str(confirm.query_one("#counts", Static).content)
            assert "1 reemplazos" in counts
            await pilot.press("escape")
            await pilot.pause()
            assert isinstance(app.screen, DeployConflictsScreen)
            assert (state.home / ".config/foo/config.toml").read_text() == "local\n"
            assert state.plan is not None

    _run(scenario())


def test_d4_apply_opens_summary_with_snapshot(tmp_path: Path) -> None:
    state = _deploy_state(tmp_path)
    app = KaminoApp(mode="direct", initial=DeployAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await pilot.press("a")
            await pilot.press("enter")
            for _ in range(30):
                await asyncio.sleep(0.1)
                await pilot.pause()
                if isinstance(app.screen, DeployConflictsScreen):
                    break
            await pilot.press("C")
            await pilot.press("enter")
            await pilot.pause()
            await pilot.press("down", "enter")
            for _ in range(30):
                await asyncio.sleep(0.1)
                await pilot.pause()
                if isinstance(app.screen, DeploySummaryScreen):
                    break
            summary = app.screen
            assert isinstance(summary, DeploySummaryScreen)
            assert (state.home / ".config/foo/config.toml").read_text() == "clone\n"
            assert "Holocrón:" in str(summary.query_one("#snapshot", Static).content)
            assert "kamino deploy" in str(summary.query_one("#undo", Static).content)
            await pilot.press("enter")
            await pilot.pause()
            assert app.return_value == 0

    _run(scenario())


def test_d5_partial_summary_uses_incomplete_variant() -> None:
    state = DeployState()
    partial = PartialApplyError(
        "incompleto", applied=["a.toml"], failed="b.toml", snapshot=Path("/tmp/snap.tar.gz")
    )
    app = KaminoApp(
        mode="direct",
        initial=DeploySummaryScreen(state, partial=partial),
        phrase="frase",
    )

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            screen = app.screen
            assert "Despliegue incompleto" in str(screen.query_one("#title", Static).content)
            assert "1 aplicados" in str(screen.query_one("#counts", Static).content)
            assert screen.code == 1
            await pilot.press("enter")
            await pilot.pause()
            assert app.return_value == 1

    _run(scenario())


def test_deploy_prepare_screen_can_cancel() -> None:
    class SlowSession:
        deployable_apps: list = []
        not_installed_apps: list = []
        notices: list = []
        snapshot_directory = Path("/tmp")
        closed = False

        def plan(
            self, selected=None, *, dry_run=False, requested=None, progress=None, cancelled=None
        ):
            while not cancelled():
                time.sleep(0.01)
            raise OperationCancelled("cancelada")

        def preview(self, decisions):
            raise AssertionError("no debería preparar")

        def close(self):
            self.closed = True

    state = DeployState(session=SlowSession())  # type: ignore[arg-type]
    app = KaminoApp(mode="direct", initial=DeployPrepareScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            assert isinstance(app.screen, DeployPrepareScreen)
            await pilot.press("escape")
            for _ in range(20):
                await asyncio.sleep(0.05)
                await pilot.pause()
                if app.return_value == 1:
                    break
            assert app.return_value == 1

    _run(scenario())


# --- U1 ---


def _release() -> ReleaseInfo:
    return ReleaseInfo(
        version="0.5.0",
        asset_url="https://example.test/kamino-arm64",
        asset_size=4,
        asset_digest="sha256:" + "0" * 64,
    )


def test_u1_shows_up_to_date_result(tmp_path: Path, monkeypatch) -> None:
    target = tmp_path / "kamino"
    target.write_bytes(b"actual")
    screen = UpdateScreen(current_version="0.5.0", target=target, release=_release())
    app = KaminoApp(mode="direct", initial=screen, phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            for _ in range(30):
                await asyncio.sleep(0.1)
                await pilot.pause()
                if "ya es la última versión" in str(
                    app.screen.query_one("#result", Static).content
                ):
                    break
            assert "ya es la última versión" in str(app.screen.query_one("#result", Static).content)
            await pilot.press("enter")
            await pilot.pause()
            assert app.return_value == 0

    _run(scenario())


def test_u1_updated_requests_restart(monkeypatch, tmp_path: Path) -> None:
    target = tmp_path / "kamino"
    target.write_bytes(b"actual")
    monkeypatch.setattr(
        update_tui,
        "query_update",
        lambda *a, **k: SimpleNamespace(
            should_update=True,
            current_version="0.4.0",
            release=_release(),
            target=target,
            outcome=UpdateOutcome.UPDATED,
        ),
    )
    monkeypatch.setattr(
        update_tui, "download_update", lambda release, target, **kwargs: tmp_path / "tmp"
    )
    (tmp_path / "tmp").write_bytes(b"nuevo")
    replaced: list[Path] = []
    monkeypatch.setattr(update_tui, "replace_binary", lambda temp, target: replaced.append(target))
    app = KaminoApp(
        mode="direct",
        initial=UpdateScreen(current_version="0.4.0", target=target),
        phrase="frase",
    )

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            for _ in range(30):
                await asyncio.sleep(0.1)
                await pilot.pause()
                if "Reinicia" in str(app.screen.query_one("#result", Static).content):
                    break
            assert "Reinicia kamino" in str(app.screen.query_one("#result", Static).content)
            assert replaced == [target]
            await pilot.press("enter")
            await pilot.pause()
            assert app.return_value == 0

    _run(scenario())


def test_u1_cancel_before_replace_shows_cancelled(tmp_path: Path, monkeypatch) -> None:
    target = tmp_path / "kamino"
    target.write_bytes(b"actual")

    def slow(release, target, **kwargs):
        while not kwargs["cancelled"]():
            time.sleep(0.01)
        raise OperationCancelled("cancelada")

    monkeypatch.setattr(
        update_tui,
        "query_update",
        lambda *a, **k: SimpleNamespace(
            should_update=True,
            current_version="0.4.0",
            release=_release(),
            target=target,
            outcome=UpdateOutcome.UPDATED,
        ),
    )
    monkeypatch.setattr(update_tui, "download_update", slow)
    app = KaminoApp(
        mode="direct",
        initial=UpdateScreen(current_version="0.4.0", target=target),
        phrase="frase",
    )

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await pilot.press("escape")
            await asyncio.sleep(0.2)
            await pilot.pause()
            assert target.read_bytes() == b"actual"
            await pilot.press("enter")
            await pilot.pause()
            assert app.return_value == 1

    _run(scenario())


# --- Ctrl+C ---


def test_ctrl_c_cancels_before_writing(tmp_path: Path) -> None:
    home, catalog, bin_path = _source_home(tmp_path)
    state = CloneState(home=home, catalog=catalog, application_dirs=[], path=bin_path)
    app = KaminoApp(mode="direct", initial=CloneAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await pilot.press("ctrl+c")
            await pilot.pause()
            assert app.return_value == 1

    _run(scenario())


def test_ctrl_c_during_write_requests_cancel(tmp_path: Path, monkeypatch) -> None:
    app = _review_app(tmp_path)

    def slow(
        planned,
        *,
        home=None,
        force=False,
        expected_state=None,
        progress=None,
        cancelled=None,
    ):
        while not cancelled():
            time.sleep(0.01)
        raise OperationCancelled("cancelada")

    monkeypatch.setattr(clone_tui, "write_clone", slow)

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.3)
            await pilot.pause()
            await pilot.press("enter")
            await pilot.pause()
            await pilot.press("ctrl+c")
            await asyncio.sleep(0.2)
            await pilot.pause()
            assert isinstance(app.screen, CloneReviewScreen)
            assert "cancelada" in str(app.screen.query_one("#status", Static).content).lower()

    _run(scenario())


def test_ctrl_c_after_snapshot_shows_cannot_cancel() -> None:
    from kamino.deploy import DeployClassification

    state = DeployState()
    state.plan = SimpleNamespace(entries=[])
    state.classification = DeployClassification(deployed=[], merged=[], omitted=[], unchanged=[])
    state.session = SimpleNamespace(snapshot_directory=Path("/tmp"))
    app = KaminoApp(mode="direct", initial=DeployConfirmScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            screen = app.screen
            screen._applying = True
            await pilot.press("ctrl+c")
            await pilot.pause()
            assert texts.DEPLOY_AFTER_SNAPSHOT in str(screen.query_one("#status", Static).content)

    _run(scenario())


def test_resize_guard_preserves_flow_state(tmp_path: Path) -> None:
    home, catalog, bin_path = _source_home(tmp_path)
    state = CloneState(home=home, catalog=catalog, application_dirs=[], path=bin_path)
    app = KaminoApp(mode="direct", initial=CloneAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(80, 24)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.3)
            await pilot.pause()
            screen = app.screen
            await pilot.press("a")
            await pilot.pause()
            assert screen.selected == {"foo"}
            await pilot.resize_terminal(79, 23)
            await asyncio.sleep(0.3)
            assert list(app.screen.query("#size-guard"))
            await pilot.resize_terminal(80, 24)
            await asyncio.sleep(0.3)
            assert app.screen is screen
            assert screen.selected == {"foo"}

    _run(scenario())


def test_d3_handles_long_lines_and_final_newline() -> None:
    long_line = "x" * 500 + "\n"
    local = "a\n" + long_line + "sin salto final"
    clone = "a\n" + "y" * 500 + "\n" + "sin salto final"
    conflict = build_conflict(
        app_id="foo",
        path="long.txt",
        destination=Path("/tmp/long.txt"),
        entry=FileEntry(path="long.txt", sha256="0" * 64, size=len(clone), mode="0644"),
        local=LocalFile(kind="file", data=local.encode(), mode="0644"),
        clone_content=clone.encode(),
    )
    assert conflict is not None
    app = KaminoApp(mode="direct", initial=DeployDetailScreen(conflict, None), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            body = str(app.screen.query_one("#block-body", Static).content)
            assert "x" * 500 in body or "y" * 500 in body
            await pilot.press("space")
            await pilot.pause()
            assert app.screen.decisions[0].value == "local"
            assert "Clon:" in str(app.screen.query_one("#block-footer", Static).content)

    _run(scenario())


def test_u1_error_uses_classified_advice(monkeypatch, tmp_path: Path) -> None:
    from kamino.errors import UpdateError, UpdateErrorKind

    def fail(*args, **kwargs):
        raise UpdateError("no autocontenido", kind=UpdateErrorKind.NOT_SELF_CONTAINED)

    monkeypatch.setattr(update_tui, "query_update", fail)
    app = KaminoApp(
        mode="direct",
        initial=UpdateScreen(current_version="0.4.0", target=tmp_path / "kamino"),
        phrase="frase",
    )

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await asyncio.sleep(0.2)
            await pilot.pause()
            screen = app.screen
            assert isinstance(screen, ErrorScreen)
            assert "uv tool upgrade" in str(screen.query_one("#error-action", Static).content)

    _run(scenario())


def test_u1_newer_result_is_success(monkeypatch, tmp_path: Path) -> None:
    target = tmp_path / "kamino"
    target.write_bytes(b"actual")
    monkeypatch.setattr(
        update_tui,
        "query_update",
        lambda *a, **k: SimpleNamespace(
            should_update=False,
            current_version="0.6.0",
            release=_release(),
            target=target,
            outcome=UpdateOutcome.NEWER,
        ),
    )
    app = KaminoApp(
        mode="direct",
        initial=UpdateScreen(current_version="0.6.0", target=target),
        phrase="frase",
    )

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            for _ in range(20):
                await asyncio.sleep(0.05)
                await pilot.pause()
                if "más reciente" in str(app.screen.query_one("#result", Static).content):
                    break
            assert "más reciente" in str(app.screen.query_one("#result", Static).content)
            await pilot.press("enter")
            await pilot.pause()
            assert app.return_value == 0

    _run(scenario())


def test_ascii_glyphs_without_utf8() -> None:
    app = KaminoApp(mode="menu", phrase="frase", show_logo=False, unicode=False)

    async def scenario() -> None:
        async with app.run_test(size=(80, 30)) as pilot:
            await pilot.pause()
            capsules = str(app.screen.query_one("#capsules", Static).content)
            assert "o" in capsules and "○" not in capsules

    _run(scenario())


def test_d3_ascii_markers_without_utf8() -> None:
    conflict = _text_conflict()
    app = KaminoApp(
        mode="direct",
        initial=DeployDetailScreen(conflict, None),
        phrase="frase",
        unicode=False,
    )

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            body = str(app.screen.query_one("#block-body", Static).content)
            assert "*" in body and "." in body
            assert "●" not in body and "○" not in body

    _run(scenario())


def test_no_color_keeps_status_text(monkeypatch) -> None:
    monkeypatch.setenv("NO_COLOR", "1")
    app = KaminoApp(mode="direct", phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            assert app.no_color is True
            app.push_screen(ErrorScreen("falló"))
            await pilot.pause()
            assert "✖" in str(app.screen.query_one("#title", Static).content)

    _run(scenario())


def test_default_theme_is_ansi_dark_and_overridable(monkeypatch) -> None:
    from kamino.tui.app import default_theme

    monkeypatch.delenv("TEXTUAL_THEME", raising=False)
    assert default_theme() == "ansi-dark"
    monkeypatch.setenv("TEXTUAL_THEME", "ansi-light")
    assert default_theme() == "ansi-light"


def test_app_uses_terminal_ansi_colors(monkeypatch) -> None:
    monkeypatch.setattr(KaminoApp, "theme", "ansi-dark")
    app = KaminoApp(mode="menu", phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(80, 30)):
            assert app.theme == "ansi-dark"
            assert app.current_theme.ansi is True
            assert app.native_ansi_color is True

    _run(scenario())


def test_textual_theme_override_applies_to_new_app(monkeypatch) -> None:
    monkeypatch.setenv("TEXTUAL_THEME", "ansi-light")
    app = KaminoApp(mode="menu", phrase="frase")
    assert app.theme == "ansi-light"


def test_d2_uppercase_shortcuts_apply_globally(tmp_path: Path) -> None:
    state = _deploy_state(tmp_path)
    app = KaminoApp(mode="direct", initial=DeployAppsScreen(state), phrase="frase")

    async def scenario() -> None:
        async with app.run_test(size=(100, 40)) as pilot:
            await pilot.pause()
            await pilot.press("a")
            await pilot.press("enter")
            for _ in range(30):
                await asyncio.sleep(0.1)
                await pilot.pause()
                if isinstance(app.screen, DeployConflictsScreen):
                    break
            await pilot.press("C")
            await pilot.pause()
            assert all(
                decision.action.value == "apply" for decision in state.decisions.decisions.values()
            )
            await pilot.press("O")
            await pilot.pause()
            assert all(
                decision.action.value == "omit" for decision in state.decisions.decisions.values()
            )

    _run(scenario())
