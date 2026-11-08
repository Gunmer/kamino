"""Headless tests for the interactive restore TUI."""

from __future__ import annotations

import asyncio
from pathlib import Path
from types import SimpleNamespace

from textual.widgets import Button, DataTable, ListView, Static

from kamino.conflicts import (
    Conflict,
    ConflictDecisions,
    ConflictKind,
    FileAction,
    FileDecision,
    HunkRow,
    LocalFile,
    RowDecision,
    RowKind,
    build_conflict,
    build_hunks,
)
from kamino.errors import ConcurrentChangeError
from kamino.restore import (
    RestoreClassification,
    RestoreEntry,
    RestorePlan,
    RestoreStatus,
)
from kamino.tui import app as app_module
from kamino.tui import screens as screens_module
from kamino.tui.app import KaminoRestoreApp
from kamino.tui.screens import (
    AppSelectionScreen,
    ConfirmScreen,
    ConflictDetailScreen,
    ConflictSummaryScreen,
    ErrorScreen,
    HelpScreen,
    SummaryScreen,
)
from kamino.tui.widgets import conflict_detail, context_text, hunk_header, row_text


def _run(coroutine) -> None:
    asyncio.run(coroutine)


def _text_file() -> tuple[str, str]:
    local = "".join(f"line {index}\n" for index in range(20))
    backup = local.replace("line 1\n", "LINE ONE\n").replace("line 18\n", "LINE EIGHTEEN\n")
    return local, backup


def _conflict() -> Conflict:
    local, backup = _text_file()
    conflict = build_conflict(
        app_id="foo",
        path=".config/foo/config.yaml",
        destination=Path("/home/user/.config/foo/config.yaml"),
        entry=__import__("kamino.archive", fromlist=["FileEntry"]).FileEntry(
            path=".config/foo/config.yaml",
            sha256="0" * 64,
            size=len(backup),
            mode="0644",
        ),
        local=LocalFile(kind="file", data=local.encode(), mode="0600"),
        backup_content=backup.encode(),
    )
    assert conflict is not None
    return conflict


def _plan() -> RestorePlan:
    conflict = _conflict()
    entries = [
        RestoreEntry(
            app_id="foo",
            path=conflict.path,
            destination=conflict.destination,
            size=conflict.backup_size,
            sha256=conflict.backup_sha256,
            mode="0644",
            link=None,
            status=RestoreStatus.DIFFERENT,
        ),
        RestoreEntry(
            app_id="foo",
            path="new.toml",
            destination=Path("/home/user/new.toml"),
            size=3,
            sha256="1" * 64,
            mode="0644",
            link=None,
            status=RestoreStatus.NEW,
        ),
        RestoreEntry(
            app_id="foo",
            path="same.toml",
            destination=Path("/home/user/same.toml"),
            size=3,
            sha256="2" * 64,
            mode="0644",
            link=None,
            status=RestoreStatus.IDENTICAL,
        ),
        RestoreEntry(
            app_id="foo",
            path="special",
            destination=Path("/home/user/special"),
            size=0,
            sha256=None,
            mode="0644",
            link=None,
            status=RestoreStatus.UNSUPPORTED,
        ),
    ]
    return RestorePlan(entries=entries, conflicts=[conflict], omitted=["same.toml", "special"])


class FakeSession:
    def __init__(self, plan: RestorePlan | None = None, *, error: Exception | None = None):
        self.restorable_apps = [SimpleNamespace(id="foo", name="Foo")]
        self.result = SimpleNamespace(
            restored=["restored"],
            merged=["merged"],
            omitted=["omitted"],
            snapshot=Path("/tmp/snapshots/snap.tar.gz"),
        )
        self.plan_result = plan or _plan()
        self.selected: list[str] | None = None
        self.applied: ConflictDecisions | None = None
        self.error = error
        self.snapshot_directory = Path("/tmp/snapshots")

    def plan(self, selected=None, *, dry_run=False):
        self.selected = selected
        return self.plan_result

    def preview(self, decisions):
        return RestoreClassification(restored=["restored"], merged=["merged"], omitted=["omitted"])

    def apply(self, decisions):
        if self.error is not None:
            raise self.error
        self.applied = decisions
        return self.result


def test_smoke_app_opens_and_closes() -> None:
    app = KaminoRestoreApp(FakeSession(), selected=[])

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            assert isinstance(app.screen, ConflictSummaryScreen)
            await pilot.press("escape")
            await pilot.pause()
            assert app.return_value is None

    _run(scenario())


def test_app_selection_defaults_to_none_and_can_select_all() -> None:
    session = FakeSession()
    app = KaminoRestoreApp(session, selected=None)

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            assert isinstance(app.screen, AppSelectionScreen)
            await pilot.click("#continue")
            await pilot.pause()
            assert session.selected == []
            assert isinstance(app.screen, ConflictSummaryScreen)

    _run(scenario())


def test_app_selection_select_all_continues() -> None:
    session = FakeSession()
    app = KaminoRestoreApp(session, selected=None)

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            await pilot.click("#all")
            await pilot.click("#continue")
            await pilot.pause()
            assert session.selected == ["foo"]
            assert isinstance(app.screen, ConflictSummaryScreen)

    _run(scenario())


def test_summary_table_lists_conflicts_and_counts() -> None:
    session = FakeSession()
    app = KaminoRestoreApp(session, selected=["foo"])

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            screen = app.screen
            assert isinstance(screen, ConflictSummaryScreen)
            table = screen.query_one(DataTable)
            assert table.row_count == 1
            counts = screen.query_one("#counts", Static).content
            text = str(counts)
            assert "Nuevos: 1" in text
            assert "Idénticos: 1" in text
            assert "No soportados: 1" in text
            assert screen.query_one("#confirm", Button).disabled is True

    _run(scenario())


def test_apply_omit_and_global_actions() -> None:
    session = FakeSession()
    app = KaminoRestoreApp(session, selected=["foo"])
    path = session.plan_result.conflicts[0].path

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            screen = app.screen
            assert isinstance(screen, ConflictSummaryScreen)
            await pilot.click("#apply")
            await pilot.pause()
            assert app.decisions.get(path) == FileDecision(FileAction.APPLY)
            assert screen.query_one("#confirm", Button).disabled is False
            await pilot.click("#omit")
            await pilot.pause()
            assert app.decisions.get(path) == FileDecision(FileAction.OMIT)
            await pilot.click("#apply-all")
            await pilot.pause()
            assert app.decisions.get(path) == FileDecision(FileAction.APPLY)
            await pilot.click("#omit-all")
            await pilot.pause()
            assert app.decisions.get(path) == FileDecision(FileAction.OMIT)

    _run(scenario())


def test_detail_starts_on_the_first_decidable_row_and_skips_context() -> None:
    session = FakeSession()
    app = KaminoRestoreApp(session, selected=["foo"])

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            await pilot.click("#review")
            await pilot.pause()
            detail = app.screen
            assert isinstance(detail, ConflictDetailScreen)
            await pilot.pause()
            list_view = detail.query_one("#hunks", ListView)
            assert list_view.index == 2
            await pilot.press("down")
            await pilot.pause()
            assert list_view.index == 10
            await pilot.press("up")
            await pilot.pause()
            assert list_view.index == 2

            header = list_view.children[0]
            list_view.post_message(ListView.Selected(list_view, header, 0))
            await pilot.pause()
            assert list_view.index == 2

    _run(scenario())


def test_detail_toggles_and_global_shortcuts() -> None:
    session = FakeSession()
    app = KaminoRestoreApp(session, selected=["foo"])
    path = session.plan_result.conflicts[0].path

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            await pilot.click("#review")
            await pilot.pause()
            detail = app.screen
            assert isinstance(detail, ConflictDetailScreen)
            assert detail.decisions == [RowDecision.BACKUP, RowDecision.BACKUP]
            await pilot.press("space")
            await pilot.pause()
            assert detail.decisions[0] is RowDecision.LOCAL
            assert "[local]" in str(detail._row_widgets[0].content)
            assert "[respaldo]" in str(detail._row_widgets[1].content)
            await pilot.press("enter")
            await pilot.pause()
            assert detail.decisions[0] is RowDecision.BACKUP
            await pilot.press("space")
            await pilot.pause()
            assert detail.decisions[0] is RowDecision.LOCAL
            await pilot.press("a")
            await pilot.pause()
            assert detail.decisions == [RowDecision.BACKUP, RowDecision.BACKUP]
            await pilot.press("l")
            await pilot.pause()
            assert detail.decisions == [RowDecision.LOCAL, RowDecision.LOCAL]
            await pilot.click("#back")
            await pilot.pause()
            assert isinstance(app.screen, ConflictSummaryScreen)
            assert app.decisions.get(path) == FileDecision(
                FileAction.REVIEW, (RowDecision.LOCAL, RowDecision.LOCAL)
            )

            await pilot.click("#review")
            await pilot.pause()
            detail = app.screen
            assert isinstance(detail, ConflictDetailScreen)
            assert detail.decisions == [RowDecision.LOCAL, RowDecision.LOCAL]

    _run(scenario())


def test_detail_block_actions_and_navigation() -> None:
    session = FakeSession()
    app = KaminoRestoreApp(session, selected=["foo"])

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            await pilot.click("#review")
            await pilot.pause()
            detail = app.screen
            assert isinstance(detail, ConflictDetailScreen)
            list_view = detail.query_one("#hunks", ListView)

            await pilot.press("l")
            await pilot.pause()
            assert detail.decisions == [RowDecision.LOCAL, RowDecision.LOCAL]
            await pilot.press("A")
            await pilot.pause()
            assert detail.decisions == [RowDecision.BACKUP, RowDecision.LOCAL]
            await pilot.press("L")
            await pilot.pause()
            assert detail.decisions == [RowDecision.LOCAL, RowDecision.LOCAL]
            await pilot.click("#block-backup")
            await pilot.pause()
            assert detail.decisions == [RowDecision.BACKUP, RowDecision.LOCAL]

            await pilot.press("n")
            await pilot.pause()
            assert list_view.index == 10
            await pilot.press("n")
            await pilot.pause()
            assert list_view.index == 10
            await pilot.press("A")
            await pilot.pause()
            assert detail.decisions == [RowDecision.BACKUP, RowDecision.BACKUP]
            await pilot.press("p")
            await pilot.pause()
            assert list_view.index == 2
            await pilot.press("p")
            await pilot.pause()
            assert list_view.index == 2
            await pilot.click("#block-local")
            await pilot.pause()
            assert detail.decisions == [RowDecision.LOCAL, RowDecision.BACKUP]

    _run(scenario())


def test_detail_rejects_an_incomplete_previous_decision() -> None:
    session = FakeSession()
    app = KaminoRestoreApp(session, selected=["foo"])
    conflict = session.plan_result.conflicts[0]

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            app.decisions.set(conflict.path, FileDecision(FileAction.REVIEW, (RowDecision.LOCAL,)))
            await pilot.click("#review")
            await pilot.pause()
            detail = app.screen
            assert isinstance(detail, ConflictDetailScreen)
            assert detail.decisions == [RowDecision.BACKUP, RowDecision.BACKUP]

    _run(scenario())


def test_render_helpers_show_ranges_pairs_and_single_rows() -> None:
    conflict = _conflict()
    first, second = conflict.hunks
    rows = [row for hunk in conflict.hunks for row in hunk.rows]

    assert "local 1-5" in str(hunk_header(first))
    assert "respaldo 1-5" in str(hunk_header(first))
    assert "1 línea" in str(hunk_header(first))
    assert "Bloque 2" in str(hunk_header(second))
    assert "line 0" in str(context_text(first.rows[0]))

    replaced = next(row for row in rows if row.local == "line 1\n")
    assert "* - line 1  [local]" in str(row_text(replaced, RowDecision.LOCAL))
    assert "* + LINE ONE  [respaldo]" in str(row_text(replaced, RowDecision.BACKUP))

    inserted = HunkRow(RowKind.BACKUP_ONLY, None, "extra\n")
    assert "* + extra  [respaldo]" in str(row_text(inserted, RowDecision.BACKUP))
    assert "extra  [omitir]" in str(row_text(inserted, RowDecision.LOCAL))

    removed = HunkRow(RowKind.LOCAL_ONLY, "gone\n", None)
    assert "* - gone  [local]" in str(row_text(removed, RowDecision.LOCAL))
    assert "gone  [omitir]" in str(row_text(removed, RowDecision.BACKUP))


def test_render_header_shows_a_dash_for_an_empty_side() -> None:
    (inserted,) = build_hunks("", "a\n")
    (deleted,) = build_hunks("a\n", "")

    inserted_header = str(hunk_header(inserted))
    assert "local -" in inserted_header
    assert "respaldo 1-1" in inserted_header
    deleted_header = str(hunk_header(deleted))
    assert "local 1-1" in deleted_header
    assert "respaldo -" in deleted_header


def test_summary_detail_counts_blocks_and_lines() -> None:
    conflict = _conflict()

    assert conflict_detail(conflict) == "2 bloques · 2 líneas"


def test_summary_review_decision_counts_backup_rows() -> None:
    session = FakeSession()
    app = KaminoRestoreApp(session, selected=["foo"])
    conflict = session.plan_result.conflicts[0]

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            screen = app.screen
            assert isinstance(screen, ConflictSummaryScreen)
            app.decisions.set(
                conflict.path,
                FileDecision(FileAction.REVIEW, (RowDecision.BACKUP, RowDecision.LOCAL)),
            )
            screen.refresh_table()
            await pilot.pause()
            row = screen.query_one(DataTable).get_row_at(0)
            assert row[3] == "2 bloques · 2 líneas"
            assert row[4] == "revisión (1/2 respaldo)"

    _run(scenario())


def test_detail_blocks_invalid_merge(monkeypatch) -> None:
    session = FakeSession()
    app = KaminoRestoreApp(session, selected=["foo"])

    def invalid(*args, **kwargs):
        return SimpleNamespace(status="invalid", message="boom")

    monkeypatch.setattr(screens_module, "validate_merge", invalid)

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            await pilot.click("#review")
            await pilot.pause()
            detail = app.screen
            assert isinstance(detail, ConflictDetailScreen)
            await pilot.click("#back")
            await pilot.pause()
            assert app.screen is detail
            assert "Mezcla inválida" in str(detail.query_one("#hunk-error", Static).content)

    _run(scenario())


def test_confirm_screen_and_summary() -> None:
    session = FakeSession()
    app = KaminoRestoreApp(session, selected=["foo"])

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            await pilot.click("#apply-all")
            await pilot.click("#confirm")
            await pilot.pause()
            confirm = app.screen
            assert isinstance(confirm, ConfirmScreen)
            assert "Restaurados: 1" in str(confirm.query_one("#counts", Static).content)
            assert "/tmp/snapshots" in str(confirm.query_one("#snapshot-dir", Static).content)
            await pilot.click("#apply")
            await pilot.pause()
            assert session.applied is not None
            assert isinstance(app.screen, SummaryScreen)
            text = str(app.screen.query_one("#counts", Static).content)
            assert "Restaurados: 1" in text
            assert "Mezclados: 1" in text
            assert "Omitidos: 1" in text
            await pilot.click("#exit")
            await pilot.pause()
            assert app.return_value is session.result

    _run(scenario())


def test_confirm_cancel_does_not_apply() -> None:
    session = FakeSession()
    app = KaminoRestoreApp(session, selected=["foo"])

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            await pilot.click("#apply-all")
            await pilot.click("#confirm")
            await pilot.pause()
            await pilot.click("#cancel")
            await pilot.pause()
            assert session.applied is None
            assert app.return_value is None

    _run(scenario())


def test_confirm_apply_handles_concurrent_change() -> None:
    session = FakeSession(error=ConcurrentChangeError("cambió"))
    app = KaminoRestoreApp(session, selected=["foo"])

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            await pilot.click("#apply-all")
            await pilot.click("#confirm")
            await pilot.pause()
            await pilot.click("#apply")
            await pilot.pause()
            assert isinstance(app.screen, ErrorScreen)
            assert isinstance(app.error, ConcurrentChangeError)
            await pilot.click("#exit")
            await pilot.pause()
            assert app.return_value is None

    _run(scenario())


def test_help_screen_is_available() -> None:
    app = KaminoRestoreApp(FakeSession(), selected=["foo"])

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            await pilot.click("#help")
            await pilot.pause()
            help_screen = app.screen
            assert isinstance(help_screen, HelpScreen)
            body = str(help_screen.query_one("#help-body", Static).content)
            assert "secretos" in body
            await pilot.click("#close")
            await pilot.pause()
            assert isinstance(app.screen, ConflictSummaryScreen)

    _run(scenario())


def test_non_text_conflict_does_not_open_detail() -> None:
    plan = _plan()
    plan.conflicts[0] = Conflict(
        app_id="foo",
        path="blob.bin",
        destination=Path("/home/user/blob.bin"),
        kind=ConflictKind.BINARY,
        local_size=1,
        backup_size=1,
        local_mode="0644",
        backup_mode="0644",
        local_sha256=None,
        backup_sha256=None,
        local_link=None,
        backup_link=None,
        local_text=None,
        backup_text=None,
    )
    session = FakeSession(plan)
    app = KaminoRestoreApp(session, selected=["foo"])

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            await pilot.click("#review")
            await pilot.pause()
            assert isinstance(app.screen, ConflictSummaryScreen)
            assert "archivo entero" in str(app.screen.query_one("#status", Static).content)

    _run(scenario())


def _real_session(tmp_path: Path):
    from kamino.backup import run_backup
    from kamino.catalog.models import AppDefinition
    from kamino.restore import prepare_restore

    definition = AppDefinition.model_validate(
        {
            "id": "foo",
            "name": "Foo",
            "command": "foo",
            "paths": [".config/foo/config.yaml"],
        }
    )
    catalog = {"foo": definition}
    local, backup_text = _text_file()
    origin = tmp_path / "origin"
    (origin / ".config/foo").mkdir(parents=True)
    (origin / ".config/foo/config.yaml").write_text(backup_text)
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    binary = bin_dir / "foo"
    binary.write_text("")
    binary.chmod(0o755)
    backup = tmp_path / "backup.tar.gz"
    run_backup(
        home=origin,
        catalog=catalog,
        application_dirs=[],
        path=str(bin_dir),
        output=backup,
    )
    target = tmp_path / "target"
    (target / ".config/foo").mkdir(parents=True)
    (target / ".config/foo/config.yaml").write_text(local)
    session = prepare_restore(
        backup,
        home=target,
        catalog=catalog,
        application_dirs=[],
        path=str(bin_dir),
        snapshots_dir=tmp_path / "snapshots",
    )
    return session, target, backup_text


def test_real_session_full_tui_flow(tmp_path: Path) -> None:
    session, target, backup_text = _real_session(tmp_path)
    app = KaminoRestoreApp(session, selected=None)

    async def scenario() -> None:
        async with app.run_test() as pilot:
            await pilot.pause()
            assert isinstance(app.screen, AppSelectionScreen)
            await pilot.click("#all")
            await pilot.pause()
            await pilot.click("#continue")
            await pilot.pause()
            assert isinstance(app.screen, ConflictSummaryScreen)
            assert app.screen.plan.conflicts
            await pilot.click("#apply-all")
            await pilot.pause()
            assert app.decisions.missing(app.screen.plan.conflicts) == []
            await pilot.click("#confirm")
            await pilot.pause()
            assert isinstance(app.screen, ConfirmScreen)
            await pilot.click("#apply")
            await pilot.pause()
            assert isinstance(app.screen, SummaryScreen)
            await pilot.click("#exit")
            await pilot.pause()
            assert app.return_value is not None

    try:
        _run(scenario())
        assert (target / ".config/foo/config.yaml").read_text() == backup_text
        assert list((tmp_path / "snapshots").glob("*.tar.gz"))
    finally:
        session.close()


def test_default_theme_is_ansi_dark(monkeypatch) -> None:
    monkeypatch.delenv("TEXTUAL_THEME", raising=False)

    assert app_module._default_theme() == "ansi-dark"


def test_default_theme_can_be_overridden(monkeypatch) -> None:
    monkeypatch.setenv("TEXTUAL_THEME", "ansi-light")

    assert app_module._default_theme() == "ansi-light"


def test_app_uses_terminal_ansi_colors(monkeypatch) -> None:
    monkeypatch.setattr(KaminoRestoreApp, "theme", "ansi-dark")
    app = KaminoRestoreApp(FakeSession(), selected=[])

    async def scenario() -> None:
        async with app.run_test():
            assert app.theme == "ansi-dark"
            assert app.current_theme.ansi is True
            assert app.native_ansi_color is True

    _run(scenario())
