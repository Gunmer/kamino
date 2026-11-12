"""Unit tests for conflict detection, diffing, composition and validation."""

from __future__ import annotations

from hashlib import sha256
from pathlib import Path

import pytest

from kamino.archive import FileEntry
from kamino.conflicts import (
    MAX_TEXT_BYTES,
    ConflictDecisions,
    ConflictKind,
    FileAction,
    FileDecision,
    LocalFile,
    RowDecision,
    RowKind,
    apply_all,
    build_conflict,
    build_hunks,
    compose,
    omit_all,
    review_all,
    row_count,
    validate_merge,
)


def _entry(
    path: str = "cfg.toml",
    data: bytes = b"clone\n",
    *,
    mode: str = "0644",
    link: str | None = None,
) -> FileEntry:
    if link is not None:
        return FileEntry(path=path, link=link, mode=mode)
    return FileEntry(path=path, sha256=sha256(data).hexdigest(), size=len(data), mode=mode)


def _conflict(
    local: bytes,
    clone: bytes,
    *,
    path: str = "cfg.toml",
    local_link: str | None = None,
    clone_link: str | None = None,
):
    local_file = (
        LocalFile(kind="symlink", link=local_link, mode="0644")
        if local_link is not None
        else LocalFile(kind="file", data=local, mode="0600")
    )
    entry = _entry(path, clone, link=clone_link)
    return build_conflict(
        app_id="foo",
        path=path,
        destination=Path("/home/user") / path,
        entry=entry,
        local=local_file,
        clone_content=None if clone_link is not None else clone,
    )


# --- build_conflict ---


def test_text_conflict_has_hunks_starting_on_clone() -> None:
    conflict = _conflict(b"a = 1\n", b"a = 2\n")

    assert conflict is not None
    assert conflict.kind is ConflictKind.TEXT
    assert conflict.local_mode == "0600"
    assert conflict.clone_mode == "0644"
    assert len(conflict.hunks) == 1
    assert [row.kind for row in conflict.hunks[0].rows] == [RowKind.REPLACE]


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
            clone_content=b"same\n",
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
            clone_content=b"regular\n",
        )
        is None
    )


def test_type_change_from_regular_to_symlink_is_not_a_conflict() -> None:
    assert _conflict(b"regular\n", b"", clone_link="target") is None


def test_binary_conflict_has_no_hunks() -> None:
    conflict = _conflict(b"\x00\x01", b"\x00\x02")

    assert conflict is not None
    assert conflict.kind is ConflictKind.BINARY
    assert conflict.hunks == ()


def test_text_over_five_megabytes_is_large() -> None:
    local = b"a" * (MAX_TEXT_BYTES + 1)
    clone = b"b" * (MAX_TEXT_BYTES + 1)

    conflict = _conflict(local, clone)

    assert conflict is not None
    assert conflict.kind is ConflictKind.LARGE
    assert conflict.hunks == ()


def test_text_over_twenty_thousand_lines_is_large() -> None:
    local = b"a\n" * 20_001
    clone = b"b\n" * 20_001

    conflict = _conflict(local, clone)

    assert conflict is not None
    assert conflict.kind is ConflictKind.LARGE
    assert conflict.hunks == ()


def test_symlink_conflict_records_both_targets() -> None:
    conflict = _conflict(b"", b"", local_link="a", clone_link="b")

    assert conflict is not None
    assert conflict.kind is ConflictKind.SYMLINK
    assert conflict.local_link == "a"
    assert conflict.clone_link == "b"


def test_identical_symlink_is_not_a_conflict() -> None:
    assert _conflict(b"", b"", local_link="a", clone_link="a") is None


# --- build_hunks ---


def test_hunks_cover_changes_additions_and_deletions() -> None:
    local = "".join(f"line {index}\n" for index in range(20))
    clone = local.replace("line 1\n", "LINE ONE\n") + "line 20\n"

    hunks = build_hunks(local, clone)

    assert len(hunks) == 2
    assert hunks[0].local_start == 1
    assert hunks[0].local_end > hunks[0].local_start
    assert hunks[1].clone_end > hunks[1].clone_start


def test_hunks_use_three_lines_of_context() -> None:
    local = "\n".join(f"line {index}" for index in range(12)) + "\n"
    clone = local.replace("line 6", "line six")

    (hunk,) = build_hunks(local, clone)

    assert hunk.local_start == 4
    assert hunk.local_end == 11
    context = [row for row in hunk.rows if row.kind is RowKind.EQUAL]
    expected_context = [f"line {index}\n" for index in range(3, 6)]
    expected_context += [f"line {index}\n" for index in range(7, 10)]
    assert [row.local for row in context] == expected_context
    changed = [row for row in hunk.rows if row.kind is RowKind.REPLACE]
    assert [(row.local, row.clone) for row in changed] == [("line 6\n", "line six\n")]


def test_rows_pair_replacements_and_leave_the_extra_clone_line() -> None:
    local = "a\nb\nc\n"
    clone = "a\nX\nY\nZ\n"

    (hunk,) = build_hunks(local, clone)

    assert [(row.kind, row.local, row.clone) for row in hunk.rows] == [
        (RowKind.EQUAL, "a\n", "a\n"),
        (RowKind.REPLACE, "b\n", "X\n"),
        (RowKind.REPLACE, "c\n", "Y\n"),
        (RowKind.CLONE_ONLY, None, "Z\n"),
    ]


def test_rows_include_insertions_and_deletions() -> None:
    inserted = build_hunks("a\nc\n", "a\nb\nc\n")
    deleted = build_hunks("a\nb\nc\n", "a\nc\n")

    assert [row.kind for row in inserted[0].rows] == [
        RowKind.EQUAL,
        RowKind.CLONE_ONLY,
        RowKind.EQUAL,
    ]
    assert [row.kind for row in deleted[0].rows] == [
        RowKind.EQUAL,
        RowKind.LOCAL_ONLY,
        RowKind.EQUAL,
    ]
    assert inserted[0].rows[1].clone == "b\n"
    assert deleted[0].rows[1].local == "b\n"


def test_row_count_ignores_context_rows() -> None:
    local = "".join(f"line {index}\n" for index in range(20))
    clone = local.replace("line 1\n", "LINE ONE\n").replace("line 19\n", "LINE NINETEEN\n")
    conflict = _conflict(local.encode(), clone.encode())
    assert conflict is not None

    assert len(conflict.hunks) == 2
    assert row_count(conflict) == 2


# --- compose ---


def test_compose_mixes_two_blocks() -> None:
    local = "".join(f"line {index}\n" for index in range(20))
    clone = local.replace("line 1\n", "LINE ONE\n").replace("line 19\n", "LINE NINETEEN\n")
    conflict = _conflict(local.encode(), clone.encode())
    assert conflict is not None and len(conflict.hunks) == 2

    merged = compose(
        conflict.local_text or "",
        conflict.clone_text or "",
        [
            RowDecision.CLONE,
            RowDecision.LOCAL,
        ],
    )

    assert "LINE ONE" in merged
    assert "line 19" in merged
    assert "LINE NINETEEN" not in merged


def test_compose_mixes_lines_inside_one_block() -> None:
    local = "a\nb\nc\n"
    clone = "a\nB\nC\n"
    conflict = _conflict(local.encode(), clone.encode())
    assert conflict is not None
    assert row_count(conflict) == 2

    merged = compose(
        conflict.local_text or "",
        conflict.clone_text or "",
        [RowDecision.CLONE, RowDecision.LOCAL],
    )

    assert merged == "a\nB\nc\n"


def test_compose_all_clone_equals_clone() -> None:
    local = "a\nb\nc\n"
    clone = "a\nB\nC\n"
    conflict = _conflict(local.encode(), clone.encode())
    assert conflict is not None

    merged = compose(
        conflict.local_text or "",
        conflict.clone_text or "",
        [RowDecision.CLONE] * row_count(conflict),
    )

    assert merged == clone


def test_compose_all_local_equals_local() -> None:
    local = "a\r\nb\r\nc\r\n"
    clone = "a\nB\nC\n"
    conflict = _conflict(local.encode(), clone.encode())
    assert conflict is not None

    merged = compose(
        conflict.local_text or "",
        conflict.clone_text or "",
        [RowDecision.LOCAL] * row_count(conflict),
    )

    assert merged == local


def test_compose_adapts_clone_lines_to_local_crlf() -> None:
    local = "a\r\nb\r\nc\r\n"
    clone = "a\nB\nC\n"
    conflict = _conflict(local.encode(), clone.encode())
    assert conflict is not None

    merged = compose(
        conflict.local_text or "",
        conflict.clone_text or "",
        [RowDecision.CLONE] * row_count(conflict),
    )

    assert merged == "a\r\nB\r\nC\r\n"


def test_compose_preserves_missing_final_newline_of_chosen_side() -> None:
    local = "a\nb\n"
    clone = "a\nB"
    conflict = _conflict(local.encode(), clone.encode())
    assert conflict is not None

    merged = compose(
        conflict.local_text or "",
        conflict.clone_text or "",
        [RowDecision.CLONE] * row_count(conflict),
    )

    assert merged == "a\nB"


def test_compose_adds_local_eol_to_a_mid_file_line_without_final_newline() -> None:
    local = "L1\nL2\nL3"
    clone = "B1\nB2"
    conflict = _conflict(local.encode(), clone.encode())
    assert conflict is not None
    assert row_count(conflict) == 3

    merged = compose(
        conflict.local_text or "",
        conflict.clone_text or "",
        [RowDecision.CLONE, RowDecision.CLONE, RowDecision.LOCAL],
    )

    assert merged == "B1\nB2\nL3"


def test_compose_rejects_a_decision_list_that_does_not_match_the_rows() -> None:
    conflict = _conflict(b"a\nb\nc\n", b"a\nB\nC\n")
    assert conflict is not None

    with pytest.raises(ValueError, match="decision list"):
        compose(conflict.local_text or "", conflict.clone_text or "", [RowDecision.CLONE])


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
    assert review_all(first).rows == (RowDecision.CLONE,)


def test_review_all_builds_one_decision_per_row() -> None:
    conflict = _conflict(b"a\nb\nc\n", b"a\nB\nC\n")
    assert conflict is not None

    decision = review_all(conflict, RowDecision.LOCAL)

    assert row_count(conflict) == 2
    assert decision.rows == (RowDecision.LOCAL, RowDecision.LOCAL)


def test_missing_decisions_are_reported() -> None:
    conflict = _conflict(b"a\n", b"b\n")
    assert conflict is not None
    decisions = ConflictDecisions()

    assert decisions.missing([conflict]) == [conflict]
