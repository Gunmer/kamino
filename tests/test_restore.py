"""Pruebas de `restore.py`: plan de restauración, snapshot previo y aplicación.

Cubre el flujo del diseño técnico §5.2 sin interfaz: leer el manifiesto, cruzar
las apps con las instaladas, clasificar cada archivo (nuevo, idéntico o distinto),
verificar el archivo antes de tocar nada, crear el snapshot previo (§6) y aplicar
con reemplazo atómico conservando el modo. Todo se inyecta (home, catálogo,
detección y snapshots_dir) para no tocar el sistema real (§7).
"""

from __future__ import annotations

import hashlib
import io
import os
import re
import stat
import tarfile
from pathlib import Path

import pytest

from kamino.archive import FORMAT_VERSION, MANIFEST_NAME, Manifest, read_manifest, write_archive
from kamino.backup import run_backup
from kamino.catalog.models import AppDefinition
from kamino.errors import IntegrityError, VersionError
from kamino.restore import RestoreResult, RestoreStatus, run_restore


def _app(*, app_id: str, paths: list[str]) -> AppDefinition:
    return AppDefinition.model_validate(
        {"id": app_id, "name": app_id, "command": app_id, "paths": paths}
    )


def _write_file(home: Path, rel: str, content: bytes = b"config\n") -> Path:
    target = home / rel
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_bytes(content)
    return target


def _fake_path(directory: Path, *commands: str) -> str:
    """Crea un PATH falso en `directory` con los binarios indicados (apps "instaladas")."""
    directory.mkdir(parents=True, exist_ok=True)
    for command in commands:
        binary = directory / command
        binary.write_text("", encoding="utf-8")
        binary.chmod(0o755)
    return str(directory)


def _metadata() -> dict[str, str]:
    return {
        "macos_version": "15.1",
        "arch": "arm64",
        "hostname": "mac-test",
        "created_at": "2026-10-28T21:30:00Z",
    }


def _backup(
    tmp_path: Path,
    *,
    apps: dict[str, list[str]],
    files: dict[str, bytes],
    links: dict[str, str] | None = None,
    modes: dict[str, int] | None = None,
) -> tuple[Path, dict[str, AppDefinition]]:
    """Crea un respaldo desde un home de origen falso donde todas las apps están instaladas."""
    source = tmp_path / "source"
    for rel, content in files.items():
        _write_file(source, rel, content)
    for rel, mode in (modes or {}).items():
        (source / rel).chmod(mode)
    for rel, target in (links or {}).items():
        link = source / rel
        link.parent.mkdir(parents=True, exist_ok=True)
        link.symlink_to(target)
    catalog = {app_id: _app(app_id=app_id, paths=paths) for app_id, paths in apps.items()}
    output = tmp_path / "backup.tar.gz"
    run_backup(
        home=source,
        output=output,
        catalog=catalog,
        application_dirs=[],
        path=_fake_path(tmp_path / "source-path", *apps),
        metadata=_metadata(),
    )
    return output, catalog


def _run(
    tmp_path: Path,
    backup: Path,
    home: Path,
    catalog: dict[str, AppDefinition],
    *,
    installed: tuple[str, ...] = ("foo",),
    selected: list[str] | None = None,
    dry_run: bool = False,
) -> RestoreResult:
    return run_restore(
        backup,
        home=home,
        selected=selected,
        dry_run=dry_run,
        catalog=catalog,
        application_dirs=[],
        path=_fake_path(tmp_path / "target-path", *installed),
        snapshots_dir=tmp_path / "snapshots",
    )


def _tamper(archive: Path, member: str, content: bytes) -> Path:
    """Reescribe el archivo con el contenido de `member` alterado (el manifiesto queda igual)."""
    tampered = archive.with_name("tampered.tar.gz")
    with tarfile.open(archive, "r:gz") as src, tarfile.open(tampered, "w:gz") as dst:
        for info in src.getmembers():
            if info.name == member:
                info.size = len(content)
                dst.addfile(info, io.BytesIO(content))
            elif info.isfile():
                handle = src.extractfile(info)
                assert handle is not None
                dst.addfile(info, handle)
            else:
                dst.addfile(info)
    return tampered


def _malicious_archive(tmp_path: Path) -> Path:
    """Crea un tar.gz con un miembro `../evil.txt` y un manifiesto válido (§7)."""
    manifest = Manifest(
        format_version=FORMAT_VERSION,
        tool_version="0.1.0",
        created_at="2026-10-28T21:30:00Z",
        macos_version="15.1",
        arch="arm64",
        hostname="mac-test",
        apps=[],
    )
    output = tmp_path / "malicioso.tar.gz"
    with tarfile.open(output, "w:gz") as tar:
        data = manifest.model_dump_json(indent=2).encode("utf-8")
        info = tarfile.TarInfo(MANIFEST_NAME)
        info.size = len(data)
        tar.addfile(info, io.BytesIO(data))
        evil = b"malicioso\n"
        info = tarfile.TarInfo("../evil.txt")
        info.size = len(evil)
        tar.addfile(info, io.BytesIO(evil))
    return output


# --- dry run (§5.2.5) ---


def test_dry_run_classifies_plan_without_writing(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "igual.toml", b"igual\n")
    _write_file(home, "distinto.toml", b"viejo\n")
    backup, catalog = _backup(
        tmp_path,
        apps={"foo": ["nuevo.toml", "igual.toml", "distinto.toml"]},
        files={"nuevo.toml": b"nuevo\n", "igual.toml": b"igual\n", "distinto.toml": b"nuevo\n"},
    )

    result = _run(tmp_path, backup, home, catalog, dry_run=True)

    assert result.dry_run is True
    assert result.output is None
    assert result.snapshot is None
    assert result.restored == []
    assert result.omitted == ["igual.toml"]
    assert [(entry.path, entry.status) for entry in result.plan.entries] == [
        ("nuevo.toml", RestoreStatus.NEW),
        ("igual.toml", RestoreStatus.IDENTICAL),
        ("distinto.toml", RestoreStatus.DIFFERENT),
    ]
    # no se escribe nada: ni destinos ni snapshot
    assert (home / "distinto.toml").read_bytes() == b"viejo\n"
    assert not (home / "nuevo.toml").exists()
    assert not (tmp_path / "snapshots").exists()


# --- ejecución real (§5.2.6-8) ---


def test_restore_creates_new_file_with_content_and_mode(tmp_path: Path) -> None:
    backup, catalog = _backup(
        tmp_path,
        apps={"foo": [".config/foo/config.toml"]},
        files={".config/foo/config.toml": b"[a]\n"},
        modes={".config/foo/config.toml": 0o600},
    )
    home = tmp_path / "home"

    result = _run(tmp_path, backup, home, catalog)

    restored = home / ".config" / "foo" / "config.toml"
    assert restored.read_bytes() == b"[a]\n"
    assert stat.S_IMODE(restored.stat().st_mode) == 0o600
    assert result.restored == [".config/foo/config.toml"]
    assert result.omitted == []
    assert result.output == backup


def test_restore_overwrites_different_file(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"viejo\n")
    backup, catalog = _backup(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"nuevo\n"})

    result = _run(tmp_path, backup, home, catalog)

    assert (home / "cfg.toml").read_bytes() == b"nuevo\n"
    assert result.restored == ["cfg.toml"]
    assert result.omitted == []


def test_identical_file_is_omitted_and_not_restored(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"igual\n")
    backup, catalog = _backup(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"igual\n"})

    result = _run(tmp_path, backup, home, catalog)

    assert result.restored == []
    assert result.omitted == ["cfg.toml"]
    assert (home / "cfg.toml").read_bytes() == b"igual\n"


def test_snapshot_records_previous_state_before_applying(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"viejo\n")
    backup, catalog = _backup(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"nuevo\n"})

    result = _run(tmp_path, backup, home, catalog)

    assert (home / "cfg.toml").read_bytes() == b"nuevo\n"
    snapshot = result.snapshot
    assert snapshot is not None
    assert snapshot.parent == tmp_path / "snapshots"
    assert re.fullmatch(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\.tar\.gz", snapshot.name)
    snapshot_manifest = read_manifest(snapshot)
    (app,) = snapshot_manifest.apps
    assert app.id == "snapshot"
    (entry,) = app.files
    assert entry.path == "cfg.toml"
    assert entry.sha256 == hashlib.sha256(b"viejo\n").hexdigest()
    with tarfile.open(snapshot, "r:gz") as tar:
        member = tar.extractfile("apps/snapshot/cfg.toml")
        assert member is not None
        assert member.read() == b"viejo\n"


def test_no_snapshot_when_nothing_changes(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"igual\n")
    backup, catalog = _backup(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"igual\n"})

    result = _run(tmp_path, backup, home, catalog)

    assert result.snapshot is None
    assert not (tmp_path / "snapshots").exists()


def test_snapshot_is_created_even_when_only_new_files(tmp_path: Path) -> None:
    """Con cambios que son solo archivos nuevos el snapshot no tiene nada que conservar.

    `create_snapshot` solo devuelve `None` con `paths` vacío; aquí los destinos que
    van a cambiar no existen todavía y el snapshot queda sin archivos.
    """
    home = tmp_path / "home"
    backup, catalog = _backup(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"nuevo\n"})

    result = _run(tmp_path, backup, home, catalog)

    assert result.restored == ["cfg.toml"]
    snapshot = result.snapshot
    assert snapshot is not None
    snapshot_manifest = read_manifest(snapshot)
    (app,) = snapshot_manifest.apps
    assert app.files == []


# --- cruce de apps instaladas y selección (§5.2.2-3) ---


def test_not_installed_app_and_its_files_are_omitted(tmp_path: Path) -> None:
    home = tmp_path / "home"
    backup, catalog = _backup(
        tmp_path,
        apps={"alpha": ["a.toml"], "beta": ["b.toml"]},
        files={"a.toml": b"a\n", "b.toml": b"b\n"},
    )

    result = _run(tmp_path, backup, home, catalog, installed=("alpha",))

    assert result.restored == ["a.toml"]
    assert result.omitted == ["b.toml"]
    assert (home / "a.toml").read_bytes() == b"a\n"
    assert not (home / "b.toml").exists()


def test_not_installed_apps_are_omitted_even_when_not_selected(tmp_path: Path) -> None:
    """Las apps no instaladas se listan como omitidas antes de la selección (§5.2.2)."""
    home = tmp_path / "home"
    backup, catalog = _backup(
        tmp_path,
        apps={"alpha": ["a.toml"], "beta": ["b.toml"]},
        files={"a.toml": b"a\n", "b.toml": b"b\n"},
    )

    result = _run(tmp_path, backup, home, catalog, installed=("alpha",), selected=["alpha"])

    assert result.restored == ["a.toml"]
    assert result.omitted == ["b.toml"]


def test_selected_limits_restore_and_deselected_apps_are_ignored(tmp_path: Path) -> None:
    home = tmp_path / "home"
    backup, catalog = _backup(
        tmp_path,
        apps={"alpha": ["a.toml"], "beta": ["b.toml"]},
        files={"a.toml": b"a\n", "b.toml": b"b\n"},
    )

    result = _run(tmp_path, backup, home, catalog, installed=("alpha", "beta"), selected=["alpha"])

    assert result.restored == ["a.toml"]
    assert result.omitted == []
    assert not (home / "b.toml").exists()


def test_unknown_selected_ids_are_ignored(tmp_path: Path) -> None:
    home = tmp_path / "home"
    backup, catalog = _backup(tmp_path, apps={"alpha": ["a.toml"]}, files={"a.toml": b"a\n"})

    result = _run(
        tmp_path, backup, home, catalog, installed=("alpha",), selected=["no-existe", "alpha"]
    )

    assert result.restored == ["a.toml"]


# --- tipo del destino (clasificación) ---


def test_regular_file_over_symlink_is_different_and_replaced(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "otro.txt", b"otro\n")
    (home / "cfg.toml").symlink_to("otro.txt")
    backup, catalog = _backup(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"nuevo\n"})

    result = _run(tmp_path, backup, home, catalog)

    destination = home / "cfg.toml"
    assert not destination.is_symlink()
    assert destination.read_bytes() == b"nuevo\n"
    assert (home / "otro.txt").read_bytes() == b"otro\n"
    assert result.restored == ["cfg.toml"]


def test_symlink_over_regular_file_is_different_and_replaced(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "real.txt")
    _write_file(home, "enlace.txt", b"regular\n")
    backup, catalog = _backup(
        tmp_path,
        apps={"foo": ["enlace.txt"]},
        files={},
        links={"enlace.txt": "real.txt"},
    )

    result = _run(tmp_path, backup, home, catalog)

    assert os.readlink(home / "enlace.txt") == "real.txt"
    assert result.restored == ["enlace.txt"]


def test_symlink_is_recreated_when_missing(tmp_path: Path) -> None:
    home = tmp_path / "home"
    backup, catalog = _backup(
        tmp_path,
        apps={"foo": ["enlace.txt"]},
        files={},
        links={"enlace.txt": "real.txt"},
    )

    result = _run(tmp_path, backup, home, catalog)

    assert os.readlink(home / "enlace.txt") == "real.txt"
    assert result.restored == ["enlace.txt"]


def test_identical_symlink_is_omitted(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "real.txt")
    (home / "enlace.txt").symlink_to("real.txt")
    backup, catalog = _backup(
        tmp_path,
        apps={"foo": ["enlace.txt"]},
        files={},
        links={"enlace.txt": "real.txt"},
    )

    result = _run(tmp_path, backup, home, catalog)

    assert result.restored == []
    assert result.omitted == ["enlace.txt"]


def test_directory_destination_is_omitted(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml/dentro.txt", b"dentro\n")
    backup, catalog = _backup(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"nuevo\n"})

    result = _run(tmp_path, backup, home, catalog)

    assert result.restored == []
    assert result.omitted == ["cfg.toml"]
    assert (home / "cfg.toml").is_dir()
    assert (home / "cfg.toml" / "dentro.txt").read_bytes() == b"dentro\n"


# --- integridad y seguridad (§7) ---


def test_altered_checksum_aborts_before_touching_anything(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"viejo\n")
    backup, catalog = _backup(
        tmp_path,
        apps={"foo": ["cfg.toml", "nuevo.toml"]},
        files={"cfg.toml": b"nuevo\n", "nuevo.toml": b"nuevo\n"},
    )
    tampered = _tamper(backup, "apps/foo/cfg.toml", b"alterado\n")

    with pytest.raises(IntegrityError):
        _run(tmp_path, tampered, home, catalog)

    assert (home / "cfg.toml").read_bytes() == b"viejo\n"
    assert not (home / "nuevo.toml").exists()
    assert not (tmp_path / "snapshots").exists()


def test_archive_with_escaping_paths_is_rejected(tmp_path: Path) -> None:
    home = tmp_path / "home"
    backup = _malicious_archive(tmp_path)

    with pytest.raises(IntegrityError):
        _run(tmp_path, backup, home, {})

    assert not home.exists()
    assert not (tmp_path / "evil.txt").exists()


def test_future_format_version_is_rejected(tmp_path: Path) -> None:
    home = tmp_path / "home"
    backup = tmp_path / "futuro.tar.gz"
    manifest = Manifest(
        format_version=FORMAT_VERSION + 1,
        tool_version="0.1.0",
        created_at="2026-10-28T21:30:00Z",
        macos_version="15.1",
        arch="arm64",
        hostname="mac-test",
        apps=[],
    )
    write_archive(backup, manifest)

    with pytest.raises(VersionError):
        _run(tmp_path, backup, home, {})

    assert not home.exists()
    assert not (tmp_path / "snapshots").exists()


# --- integración: ciclo completo y undo (§6 y §7) ---


def test_snapshot_can_undo_restore(tmp_path: Path) -> None:
    """El snapshot tiene el formato de un respaldo, así `undo` es un `restore` (§6)."""
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"local\n")
    backup, catalog = _backup(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"origen\n"})

    first = _run(tmp_path, backup, home, catalog)
    assert (home / "cfg.toml").read_bytes() == b"origen\n"
    assert first.snapshot is not None

    second = _run(tmp_path, first.snapshot, home, catalog)

    assert (home / "cfg.toml").read_bytes() == b"local\n"
    assert second.restored == ["cfg.toml"]
