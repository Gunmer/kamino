"""Pruebas de `snapshot.py`: copia de seguridad previa a restaurar.

El snapshot usa el mismo formato que un respaldo (§6: "Snapshots"): una única app
sintética `snapshot`, rutas relativas al home y nombre `<timestamp>.tar.gz` en
`~/Library/Application Support/kamino/snapshots/`, así el futuro `undo` es un
`restore` del snapshot. Todo se inyecta (home, snapshots_dir, timestamp) para no
tocar el sistema real (§7).
"""

from __future__ import annotations

import hashlib
import re
import tarfile
from pathlib import Path

import pytest

from kamino.archive import read_manifest, verify_archive
from kamino.errors import PathError
from kamino.snapshot import create_snapshot

TIMESTAMP = "2026-10-28T21:30:00Z"


def _write_file(home: Path, rel: str, content: bytes = b"config\n") -> Path:
    target = home / rel
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_bytes(content)
    return target


# --- contrato básico ---


def test_empty_paths_returns_none(tmp_path: Path) -> None:
    home = tmp_path / "home"
    snapshots = tmp_path / "snapshots"

    assert create_snapshot([], home=home, snapshots_dir=snapshots) is None
    assert not snapshots.exists()


def test_creates_archive_with_snapshot_app_and_entries(tmp_path: Path) -> None:
    home = tmp_path / "home"
    content = b"abc"
    regular = _write_file(home, ".config/foo/config.toml", content)
    regular.chmod(0o600)
    _write_file(home, "real.txt")
    (home / "enlace.txt").symlink_to("real.txt")
    snapshots = tmp_path / "snapshots"

    snapshot = create_snapshot(
        [home / ".config/foo/config.toml", home / "enlace.txt"],
        home=home,
        snapshots_dir=snapshots,
        timestamp=TIMESTAMP,
    )

    assert snapshot == snapshots / f"{TIMESTAMP}.tar.gz"
    assert snapshot is not None and snapshot.exists()
    manifest = read_manifest(snapshot)
    assert [app.id for app in manifest.apps] == ["snapshot"]
    (app,) = manifest.apps
    by_path = {entry.path: entry for entry in app.files}
    regular_entry = by_path[".config/foo/config.toml"]
    assert regular_entry.sha256 == hashlib.sha256(content).hexdigest()
    assert regular_entry.size == len(content)
    assert regular_entry.mode == "0600"
    assert regular_entry.link is None
    link_entry = by_path["enlace.txt"]
    assert link_entry.link == "real.txt"
    assert link_entry.sha256 is None
    verify_archive(snapshot, manifest)


def test_snapshot_records_current_content(tmp_path: Path) -> None:
    """El snapshot conserva el contenido previo, base del futuro `undo` (§6)."""
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"viejo\n")
    snapshots = tmp_path / "snapshots"

    snapshot = create_snapshot([home / "cfg.toml"], home=home, snapshots_dir=snapshots)

    assert snapshot is not None
    with tarfile.open(snapshot, "r:gz") as tar:
        member = tar.extractfile("apps/snapshot/cfg.toml")
        assert member is not None
        assert member.read() == b"viejo\n"


def test_default_snapshots_dir_is_under_home(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml")

    snapshot = create_snapshot([home / "cfg.toml"], home=home, timestamp=TIMESTAMP)

    expected = home / "Library" / "Application Support" / "kamino" / "snapshots"
    assert snapshot == expected / f"{TIMESTAMP}.tar.gz"
    assert snapshot is not None and snapshot.exists()


def test_default_timestamp_is_iso_utc(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml")
    snapshots = tmp_path / "snapshots"

    snapshot = create_snapshot([home / "cfg.toml"], home=home, snapshots_dir=snapshots)

    assert snapshot is not None
    assert re.fullmatch(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\.tar\.gz", snapshot.name)


# --- rutas que se omiten ---


def test_missing_paths_are_skipped(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "existe.toml")
    snapshots = tmp_path / "snapshots"

    snapshot = create_snapshot(
        [home / "existe.toml", home / "falta.toml"],
        home=home,
        snapshots_dir=snapshots,
        timestamp=TIMESTAMP,
    )

    assert snapshot is not None
    manifest = read_manifest(snapshot)
    (app,) = manifest.apps
    assert [entry.path for entry in app.files] == ["existe.toml"]


def test_unsafe_symlink_is_skipped(tmp_path: Path) -> None:
    """Un enlace con destino absoluto se omite por seguridad (§6)."""
    home = tmp_path / "home"
    _write_file(home, "real.txt")
    (home / "malicioso.txt").symlink_to("/etc/passwd")
    snapshots = tmp_path / "snapshots"

    snapshot = create_snapshot(
        [home / "real.txt", home / "malicioso.txt"],
        home=home,
        snapshots_dir=snapshots,
        timestamp=TIMESTAMP,
    )

    assert snapshot is not None
    manifest = read_manifest(snapshot)
    (app,) = manifest.apps
    assert [entry.path for entry in app.files] == ["real.txt"]


def test_directory_is_expanded_recursively(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "dir/a.toml")
    _write_file(home, "dir/sub/b.toml")
    snapshots = tmp_path / "snapshots"

    snapshot = create_snapshot(
        [home / "dir"], home=home, snapshots_dir=snapshots, timestamp=TIMESTAMP
    )

    assert snapshot is not None
    manifest = read_manifest(snapshot)
    (app,) = manifest.apps
    assert [entry.path for entry in app.files] == ["dir/a.toml", "dir/sub/b.toml"]


def test_overlapping_paths_are_deduplicated(tmp_path: Path) -> None:
    """Una ruta cubierta por otra no se registra dos veces (el tar no admite duplicados)."""
    home = tmp_path / "home"
    _write_file(home, "dir/a.toml")
    snapshots = tmp_path / "snapshots"

    snapshot = create_snapshot(
        [home / "dir", home / "dir" / "a.toml", home / "dir" / "a.toml"],
        home=home,
        snapshots_dir=snapshots,
        timestamp=TIMESTAMP,
    )

    assert snapshot is not None
    manifest = read_manifest(snapshot)
    (app,) = manifest.apps
    assert [entry.path for entry in app.files] == ["dir/a.toml"]


# --- validación de entradas ---


def test_paths_outside_home_raise_error(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml")
    _write_file(tmp_path, "fuera.txt")
    snapshots = tmp_path / "snapshots"

    with pytest.raises(PathError):
        create_snapshot([tmp_path / "fuera.txt"], home=home, snapshots_dir=snapshots)

    assert not snapshots.exists()


def test_relative_paths_are_rejected(tmp_path: Path) -> None:
    home = tmp_path / "home"

    with pytest.raises(PathError):
        create_snapshot([Path("cfg.toml")], home=home, snapshots_dir=tmp_path / "snapshots")


def test_timestamp_with_path_separator_is_rejected(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml")
    snapshots = tmp_path / "snapshots"

    with pytest.raises(PathError):
        create_snapshot(
            [home / "cfg.toml"], home=home, snapshots_dir=snapshots, timestamp="../evil"
        )

    assert not snapshots.exists()
    assert not (tmp_path / "evil.tar.gz").exists()
