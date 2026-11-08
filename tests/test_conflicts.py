"""Unit tests for conflict detection, diffing, composition and validation."""

from __future__ import annotations

from hashlib import sha256
from pathlib import Path

from kamino.archive import FileEntry
from kamino.conflicts import (
    MAX_TEXT_BYTES,
    ConflictDecisions,
    ConflictKind,
    FileAction,
    FileDecision,
    HunkDecision,
    LocalFile,
    apply_all,
    build_conflict,
    build_hunks,
    compose,
    omit_all,
    review_all,
    validate_merge,
)


def _entry(
    path: str = "cfg.toml",
    data: bytes = b"backup\n",
    *,
    mode: str = "0644",
    link: str | None = None,
) -> FileEntry:
    if link is not None:
        return FileEntry(path=path, link=link, mode=mode)
    return FileEntry(path=path, sha256=sha256(data).hexdigest(), size=len(data), mode=mode)


def _conflict(
    local: bytes,
    backup: bytes,
    *,
    path: str = "cfg.toml",
    local_link: str | None = None,
    backup_link: str | None = None,
):
    local_file = (
        LocalFile(kind="symlink", link=local_link, mode="0644")
        if local_link is not None
        else LocalFile(kind="file", data=local, mode="0600")
    )
    entry = _entry(path, backup, link=backup_link)
    return build_conflict(
        app_id="foo",
        path=path,
        destination=Path("/home/user") / path,
        entry=entry,
        local=local_file,
        backup_content=None if backup_link is not None else backup,
    )


# --- build_conflict ---


def test_text_conflict_has_hunks_starting_on_backup() -> None:
    conflict = _conflict(b"a = 1\n", b"a = 2\n")

    assert conflict is not None
    assert conflict.kind is ConflictKind.TEXT
    assert conflict.local_mode == "0600"
    assert conflict.backup_mode == "0644"
    assert len(conflict.hunks) == 1
    assert conflict.hunks[0].decision is HunkDecision.BACKUP


def test_identical_file_is_not_a_conflict() -> None:
    assert _conflict(b"same\n", b"same\n") is None


def test_crlf_only_difference_is_not_a_conflict() -> None:
    assert _conflict(b"same\r\n", b"same\n") is None


def test_mode_only_difference_is_not_a_conflict() -> None:
    local = LocalFile(kind="file", data=b"same\n", mode="0600")
    entry = _entry(data=b"same\n", mode="0644")
    assert (
        build_conflict(
            app_id="foo",
            path="cfg.toml",
            destination=Path("/home/user/cfg.toml"),
            entry=entry,
            local=local,
            backup_content=b"same\n",
        )
        is None
    )


def test_type_change_from_symlink_to_regular_is_not_a_conflict() -> None:
    local = LocalFile(kind="symlink", link="target", mode="0644")
    entry = _entry(data=b"regular\n")
    assert (
        build_conflict(
            app_id="foo",
            path="cfg.toml",
            destination=Path("/home/user/cfg.toml"),
            entry=entry,
            local=local,
            backup_content=b"regular\n",
        )
        is None
    )


def test_type_change_from_regular_to_symlink_is_not_a_conflict() -> None:
    assert _conflict(b"regular\n", b"", backup_link="target") is None


def test_binary_conflict_has_no_hunks() -> None:
    conflict = _conflict(b"\x00\x01", b"\x00\x02")

    assert conflict is not None
    assert conflict.kind is ConflictKind.BINARY
    assert conflict.hunks == ()


def test_text_over_five_megabytes_is_large() -> None:
    local = b"a" * (MAX_TEXT_BYTES + 1)
    backup = b"b" * (MAX_TEXT_BYTES + 1)

    conflict = _conflict(local, backup)

    assert conflict is not None
    assert conflict.kind is ConflictKind.LARGE
    assert conflict.hunks == ()


def test_text_over_twenty_thousand_lines_is_large() -> None:
    local = b"a\n" * 20_001
    backup = b"b\n" * 20_001

    conflict = _conflict(local, backup)

    assert conflict is not None
    assert conflict.kind is ConflictKind.LARGE
    assert conflict.hunks == ()


def test_symlink_conflict_records_both_targets() -> None:
    conflict = _conflict(b"", b"", local_link="a", backup_link="b")

    assert conflict is not None
    assert conflict.kind is ConflictKind.SYMLINK
    assert conflict.local_link == "a"
    assert conflict.backup_link == "b"


def test_identical_symlink_is_not_a_conflict() -> None:
    assert _conflict(b"", b"", local_link="a", backup_link="a") is None


# --- build_hunks ---


def test_hunks_cover_changes_additions_and_deletions() -> None:
    local = "".join(f"line {index}\n" for index in range(20))
    backup = local.replace("line 1\n", "LINE ONE\n") + "line 20\n"

    hunks = build_hunks(local, backup)

    assert len(hunks) == 2
    assert hunks[0].local_start == 1
    assert hunks[0].local_end > hunks[0].local_start
    assert hunks[1].backup_end > hunks[1].backup_start


def test_hunks_use_three_lines_of_context() -> None:
    local = "\n".join(f"line {index}" for index in range(12)) + "\n"
    backup = local.replace("line 6", "line six")

    (hunk,) = build_hunks(local, backup)

    assert hunk.local_start == 4
    assert hunk.local_end == 11
    assert hunk.context_before == ("line 3", "line 4", "line 5")
    assert hunk.context_after == ("line 7", "line 8", "line 9")


# --- compose ---


def test_compose_mixes_two_blocks() -> None:
    local = "".join(f"line {index}\n" for index in range(20))
    backup = local.replace("line 1\n", "LINE ONE\n").replace("line 19\n", "LINE NINETEEN\n")
    conflict = _conflict(local.encode(), backup.encode())
    assert conflict is not None and len(conflict.hunks) == 2

    merged = compose(
        conflict.local_text or "",
        conflict.backup_text or "",
        [
            HunkDecision.BACKUP,
            HunkDecision.LOCAL,
        ],
    )

    assert "LINE ONE" in merged
    assert "line 19" in merged
    assert "LINE NINETEEN" not in merged


def test_compose_all_backup_equals_backup() -> None:
    local = "a\nb\nc\n"
    backup = "a\nB\nC\n"
    conflict = _conflict(local.encode(), backup.encode())
    assert conflict is not None

    merged = compose(
        conflict.local_text or "",
        conflict.backup_text or "",
        [HunkDecision.BACKUP] * len(conflict.hunks),
    )

    assert merged == backup


def test_compose_all_local_equals_local() -> None:
    local = "a\r\nb\r\nc\r\n"
    backup = "a\nB\nC\n"
    conflict = _conflict(local.encode(), backup.encode())
    assert conflict is not None

    merged = compose(
        conflict.local_text or "",
        conflict.backup_text or "",
        [HunkDecision.LOCAL] * len(conflict.hunks),
    )

    assert merged == local


def test_compose_adapts_backup_lines_to_local_crlf() -> None:
    local = "a\r\nb\r\nc\r\n"
    backup = "a\nB\nC\n"
    conflict = _conflict(local.encode(), backup.encode())
    assert conflict is not None

    merged = compose(
        conflict.local_text or "",
        conflict.backup_text or "",
        [HunkDecision.BACKUP] * len(conflict.hunks),
    )

    assert merged == "a\r\nB\r\nC\r\n"


def test_compose_preserves_missing_final_newline_of_chosen_side() -> None:
    local = "a\nb\n"
    backup = "a\nB"
    conflict = _conflict(local.encode(), backup.encode())
    assert conflict is not None

    merged = compose(
        conflict.local_text or "",
        conflict.backup_text or "",
        [HunkDecision.BACKUP] * len(conflict.hunks),
    )

    assert merged == "a\nB"


# --- validate_merge ---


def test_toml_valid_merge() -> None:
    validation = validate_merge("cfg.toml", "a = 1\n", "a = 2\n", "a = 2\n")

    assert validation.status == "valid"


def test_toml_invalid_merge_with_valid_originals_blocks() -> None:
    validation = validate_merge("cfg.toml", "a = 1\n", "a = 2\n", "a = \n")

    assert validation.status == "invalid"
    assert "TOML" in validation.message


def test_toml_invalid_original_only_warns() -> None:
    validation = validate_merge("cfg.toml", "a = \n", "a = 2\n", "a = \n")

    assert validation.status == "warning"


def test_json_duplicate_keys_are_invalid() -> None:
    validation = validate_merge("cfg.json", '{"a": 1}\n', '{"a": 2}\n', '{"a": 1, "a": 2}\n')

    assert validation.status == "invalid"
    assert "duplicada" in validation.message


def test_json_valid_merge() -> None:
    validation = validate_merge("cfg.json", '{"a": 1}\n', '{"a": 2}\n', '{"a": 2}\n')

    assert validation.status == "valid"


def test_ini_duplicate_keys_are_invalid() -> None:
    validation = validate_merge("config.ini", "[a]\nx = 1\n", "[a]\nx = 2\n", "[a]\nx = 1\nx = 2\n")

    assert validation.status == "invalid"
    assert "INI" in validation.message


def test_ini_valid_merge() -> None:
    validation = validate_merge("config.ini", "[a]\nx = 1\n", "[a]\nx = 2\n", "[a]\nx = 2\n")

    assert validation.status == "valid"


def test_plist_duplicate_keys_are_invalid() -> None:
    merged = (
        '<?xml version="1.0" encoding="UTF-8"?>\n'
        '<plist version="1.0"><dict><key>a</key><string>1</string>'
        "<key>a</key><string>2</string></dict></plist>"
    )
    validation = validate_merge(
        "cfg.plist", "<plist><dict/></plist>", "<plist><dict/></plist>", merged
    )

    assert validation.status == "invalid"
    assert "duplicada" in validation.message


def test_plist_valid_merge() -> None:
    merged = '<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict/></plist>'
    validation = validate_merge("cfg.plist", merged, merged, merged)

    assert validation.status == "valid"


def test_unsupported_format_is_unvalidated() -> None:
    validation = validate_merge("config.yaml", "a: 1\n", "a: 2\n", "a: 2\n")

    assert validation.status == "unvalidated"


def test_shell_is_unvalidated() -> None:
    validation = validate_merge("script.sh", "echo a\n", "echo b\n", "echo b\n")

    assert validation.status == "unvalidated"


# --- decisions ---


def test_apply_all_and_omit_all_cover_every_conflict() -> None:
    first = _conflict(b"a\n", b"b\n", path="a.toml")
    second = _conflict(b"c\n", b"d\n", path="b.toml")
    assert first is not None and second is not None
    conflicts = [first, second]

    applied = apply_all(conflicts)
    omitted = omit_all(conflicts)

    assert applied.missing(conflicts) == []
    assert omitted.missing(conflicts) == []
    assert applied.get("a.toml") == FileDecision(FileAction.APPLY)
    assert omitted.get("b.toml") == FileDecision(FileAction.OMIT)
    assert review_all(first).hunks == (HunkDecision.BACKUP,)


def test_missing_decisions_are_reported() -> None:
    conflict = _conflict(b"a\n", b"b\n")
    assert conflict is not None
    decisions = ConflictDecisions()

    assert decisions.missing([conflict]) == [conflict]
