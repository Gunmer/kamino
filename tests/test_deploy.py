"""Pruebas de `deploy.py`: plan de despliegue, snapshot previo y aplicación.

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
import shutil
import stat
import tarfile
from pathlib import Path
from types import SimpleNamespace

import pytest

from kamino import deploy as deploy_module
from kamino.archive import FORMAT_VERSION, MANIFEST_NAME, Manifest, read_manifest, write_archive
from kamino.catalog.models import AppDefinition
from kamino.clone import run_clone
from kamino.conflicts import (
    ConflictDecisions,
    ConflictKind,
    FileAction,
    FileDecision,
    RowDecision,
    apply_all,
    compose,
    omit_all,
    review_all,
)
from kamino.deploy import (
    DeployResult,
    DeployStatus,
    prepare_deploy,
    run_deploy,
)
from kamino.errors import (
    ConcurrentChangeError,
    ConflictError,
    DecisionRequiredError,
    IntegrityError,
    InvalidMergeError,
    PartialApplyError,
    PathError,
    VersionError,
)


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


def _clone(
    tmp_path: Path,
    *,
    apps: dict[str, list[str]],
    files: dict[str, bytes],
    links: dict[str, str] | None = None,
    modes: dict[str, int] | None = None,
) -> tuple[Path, dict[str, AppDefinition]]:
    """Crea un clon desde un home de origen falso donde todas las apps están instaladas."""
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
    output = tmp_path / "clone.tar.gz"
    run_clone(
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
    clone: Path,
    home: Path,
    catalog: dict[str, AppDefinition],
    *,
    installed: tuple[str, ...] = ("foo",),
    selected: list[str] | None = None,
    dry_run: bool = False,
) -> DeployResult:
    return run_deploy(
        clone,
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
    clone, catalog = _clone(
        tmp_path,
        apps={"foo": ["nuevo.toml", "igual.toml", "distinto.toml"]},
        files={"nuevo.toml": b"nuevo\n", "igual.toml": b"igual\n", "distinto.toml": b"nuevo\n"},
    )

    result = _run(tmp_path, clone, home, catalog, dry_run=True)

    assert result.dry_run is True
    assert result.output is None
    assert result.snapshot is None
    assert result.deployed == []
    assert result.omitted == []
    assert result.unchanged == ["igual.toml"]
    assert [(entry.path, entry.status) for entry in result.plan.entries] == [
        ("nuevo.toml", DeployStatus.NEW),
        ("igual.toml", DeployStatus.IDENTICAL),
        ("distinto.toml", DeployStatus.DIFFERENT),
    ]
    # no se escribe nada: ni destinos ni snapshot
    assert (home / "distinto.toml").read_bytes() == b"viejo\n"
    assert not (home / "nuevo.toml").exists()
    assert not (tmp_path / "snapshots").exists()


# --- ejecución real (§5.2.6-8) ---


def test_deploy_creates_new_file_with_content_and_mode(tmp_path: Path) -> None:
    clone, catalog = _clone(
        tmp_path,
        apps={"foo": [".config/foo/config.toml"]},
        files={".config/foo/config.toml": b"[a]\n"},
        modes={".config/foo/config.toml": 0o600},
    )
    home = tmp_path / "home"

    result = _run(tmp_path, clone, home, catalog)

    deployed = home / ".config" / "foo" / "config.toml"
    assert deployed.read_bytes() == b"[a]\n"
    assert stat.S_IMODE(deployed.stat().st_mode) == 0o600
    assert result.deployed == [".config/foo/config.toml"]
    assert result.omitted == []
    assert result.output == clone


def test_deploy_overwrites_different_file(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"viejo\n")
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"nuevo\n"})

    result = _run(tmp_path, clone, home, catalog)

    assert (home / "cfg.toml").read_bytes() == b"nuevo\n"
    assert result.deployed == ["cfg.toml"]
    assert result.omitted == []


def test_identical_file_is_unchanged_and_not_deployed(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"igual\n")
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"igual\n"})

    result = _run(tmp_path, clone, home, catalog)

    assert result.deployed == []
    assert result.omitted == []
    assert result.unchanged == ["cfg.toml"]
    assert (home / "cfg.toml").read_bytes() == b"igual\n"


def test_snapshot_records_previous_state_before_applying(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"viejo\n")
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"nuevo\n"})

    result = _run(tmp_path, clone, home, catalog)

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
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"igual\n"})

    result = _run(tmp_path, clone, home, catalog)

    assert result.snapshot is None
    assert not (tmp_path / "snapshots").exists()


def test_snapshot_is_omitted_when_only_new_files(tmp_path: Path) -> None:
    """Sin destinos existentes no se crea un holocrón vacío."""
    home = tmp_path / "home"
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"nuevo\n"})

    result = _run(tmp_path, clone, home, catalog)

    assert result.deployed == ["cfg.toml"]
    assert result.snapshot is None
    assert result.snapshot_files == 0
    assert not (tmp_path / "snapshots").exists()


# --- cruce de apps instaladas y selección (§5.2.2-3) ---


def test_not_installed_app_and_its_files_are_reported(tmp_path: Path) -> None:
    home = tmp_path / "home"
    clone, catalog = _clone(
        tmp_path,
        apps={"alpha": ["a.toml"], "beta": ["b.toml"]},
        files={"a.toml": b"a\n", "b.toml": b"b\n"},
    )

    result = _run(tmp_path, clone, home, catalog, installed=("alpha",))

    assert result.deployed == ["a.toml"]
    assert result.omitted == []
    assert result.not_installed == ["b.toml"]
    assert (home / "a.toml").read_bytes() == b"a\n"
    assert not (home / "b.toml").exists()


def test_not_installed_apps_are_reported_even_when_not_selected(tmp_path: Path) -> None:
    """Las apps no instaladas se informan sin contar como omitidas."""
    home = tmp_path / "home"
    clone, catalog = _clone(
        tmp_path,
        apps={"alpha": ["a.toml"], "beta": ["b.toml"]},
        files={"a.toml": b"a\n", "b.toml": b"b\n"},
    )

    result = _run(tmp_path, clone, home, catalog, installed=("alpha",), selected=["alpha"])

    assert result.deployed == ["a.toml"]
    assert result.omitted == []
    assert result.not_installed == ["b.toml"]


def test_selected_limits_deploy_and_deselected_apps_are_ignored(tmp_path: Path) -> None:
    home = tmp_path / "home"
    clone, catalog = _clone(
        tmp_path,
        apps={"alpha": ["a.toml"], "beta": ["b.toml"]},
        files={"a.toml": b"a\n", "b.toml": b"b\n"},
    )

    result = _run(tmp_path, clone, home, catalog, installed=("alpha", "beta"), selected=["alpha"])

    assert result.deployed == ["a.toml"]
    assert result.omitted == []
    assert not (home / "b.toml").exists()


def test_unknown_selected_ids_are_ignored(tmp_path: Path) -> None:
    home = tmp_path / "home"
    clone, catalog = _clone(tmp_path, apps={"alpha": ["a.toml"]}, files={"a.toml": b"a\n"})

    result = _run(
        tmp_path, clone, home, catalog, installed=("alpha",), selected=["no-existe", "alpha"]
    )

    assert result.deployed == ["a.toml"]


# --- tipo del destino (clasificación) ---


def test_regular_file_over_symlink_is_different_and_replaced(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "otro.txt", b"otro\n")
    (home / "cfg.toml").symlink_to("otro.txt")
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"nuevo\n"})

    result = _run(tmp_path, clone, home, catalog)

    destination = home / "cfg.toml"
    assert not destination.is_symlink()
    assert destination.read_bytes() == b"nuevo\n"
    assert (home / "otro.txt").read_bytes() == b"otro\n"
    assert result.deployed == ["cfg.toml"]


def test_symlink_over_regular_file_is_different_and_replaced(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "real.txt")
    _write_file(home, "enlace.txt", b"regular\n")
    clone, catalog = _clone(
        tmp_path,
        apps={"foo": ["enlace.txt"]},
        files={},
        links={"enlace.txt": "real.txt"},
    )

    result = _run(tmp_path, clone, home, catalog)

    assert os.readlink(home / "enlace.txt") == "real.txt"
    assert result.deployed == ["enlace.txt"]


def test_symlink_is_recreated_when_missing(tmp_path: Path) -> None:
    home = tmp_path / "home"
    clone, catalog = _clone(
        tmp_path,
        apps={"foo": ["enlace.txt"]},
        files={},
        links={"enlace.txt": "real.txt"},
    )

    result = _run(tmp_path, clone, home, catalog)

    assert os.readlink(home / "enlace.txt") == "real.txt"
    assert result.deployed == ["enlace.txt"]


def test_identical_symlink_is_unchanged(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "real.txt")
    (home / "enlace.txt").symlink_to("real.txt")
    clone, catalog = _clone(
        tmp_path,
        apps={"foo": ["enlace.txt"]},
        files={},
        links={"enlace.txt": "real.txt"},
    )

    result = _run(tmp_path, clone, home, catalog)

    assert result.deployed == []
    assert result.omitted == []
    assert result.unchanged == ["enlace.txt"]


def test_directory_destination_is_omitted(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml/dentro.txt", b"dentro\n")
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"nuevo\n"})

    result = _run(tmp_path, clone, home, catalog)

    assert result.deployed == []
    assert result.omitted == ["cfg.toml"]
    assert (home / "cfg.toml").is_dir()
    assert (home / "cfg.toml" / "dentro.txt").read_bytes() == b"dentro\n"


# --- integridad y seguridad (§7) ---


def test_altered_checksum_aborts_before_touching_anything(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"viejo\n")
    clone, catalog = _clone(
        tmp_path,
        apps={"foo": ["cfg.toml", "nuevo.toml"]},
        files={"cfg.toml": b"nuevo\n", "nuevo.toml": b"nuevo\n"},
    )
    tampered = _tamper(clone, "apps/foo/cfg.toml", b"alterado\n")

    with pytest.raises(IntegrityError):
        _run(tmp_path, tampered, home, catalog)

    assert (home / "cfg.toml").read_bytes() == b"viejo\n"
    assert not (home / "nuevo.toml").exists()
    assert not (tmp_path / "snapshots").exists()


def test_archive_with_escaping_paths_is_rejected(tmp_path: Path) -> None:
    home = tmp_path / "home"
    clone = _malicious_archive(tmp_path)

    with pytest.raises(IntegrityError):
        _run(tmp_path, clone, home, {})

    assert not home.exists()
    assert not (tmp_path / "evil.txt").exists()


def test_future_format_version_is_rejected(tmp_path: Path) -> None:
    home = tmp_path / "home"
    clone = tmp_path / "futuro.tar.gz"
    manifest = Manifest(
        format_version=FORMAT_VERSION + 1,
        tool_version="0.1.0",
        created_at="2026-10-28T21:30:00Z",
        macos_version="15.1",
        arch="arm64",
        hostname="mac-test",
        apps=[],
    )
    write_archive(clone, manifest)

    with pytest.raises(VersionError):
        _run(tmp_path, clone, home, {})

    assert not home.exists()
    assert not (tmp_path / "snapshots").exists()


# --- integración: ciclo completo y undo (§6 y §7) ---


def test_snapshot_can_undo_deploy(tmp_path: Path) -> None:
    """El snapshot tiene el formato de un clon, así `undo` es un `deploy` (§6)."""
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"local\n")
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"origen\n"})

    first = _run(tmp_path, clone, home, catalog)
    assert (home / "cfg.toml").read_bytes() == b"origen\n"
    assert first.snapshot is not None

    second = _run(tmp_path, first.snapshot, home, catalog)

    assert (home / "cfg.toml").read_bytes() == b"local\n"
    assert second.deployed == ["cfg.toml"]


# --- confinamiento y compatibilidad ---


def test_intermediate_symlink_destination_is_rejected_before_snapshot(tmp_path: Path) -> None:
    home = tmp_path / "home"
    external = tmp_path / "external"
    home.mkdir()
    external.mkdir()
    (home / ".config").symlink_to(external)
    clone, catalog = _clone(
        tmp_path,
        apps={"foo": [".config/foo/config.toml"]},
        files={".config/foo/config.toml": b"nuevo\n"},
    )

    with pytest.raises(PathError):
        _run(tmp_path, clone, home, catalog)

    assert not (external / "foo" / "config.toml").exists()
    assert not (tmp_path / "snapshots").exists()


def test_directory_replaced_by_symlink_during_deploy_is_rejected(
    tmp_path: Path, monkeypatch
) -> None:
    home = tmp_path / "home"
    external = tmp_path / "external"
    home.mkdir()
    external.mkdir()
    _write_file(home, ".config/foo/config.toml", b"viejo\n")
    clone, catalog = _clone(
        tmp_path,
        apps={"foo": [".config/foo/config.toml"]},
        files={".config/foo/config.toml": b"nuevo\n"},
    )
    original = deploy_module.create_snapshot

    def swap_after_snapshot(paths, **kwargs):
        snapshot = original(paths, **kwargs)
        shutil.rmtree(home / ".config")
        (home / ".config").symlink_to(external)
        return snapshot

    monkeypatch.setattr(deploy_module, "create_snapshot", swap_after_snapshot)

    with pytest.raises(PartialApplyError):
        _run(tmp_path, clone, home, catalog)

    assert not (external / "foo" / "config.toml").exists()


def test_old_format_clone_snapshots_now_excluded_destination(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, ".config/foo/.env", b"TOKEN=actual\n")
    clone, catalog = _clone(
        tmp_path,
        apps={"foo": [".config/foo/.env"]},
        files={".config/foo/.env": b"TOKEN=nuevo\n"},
    )

    result = _run(tmp_path, clone, home, catalog)

    assert (home / ".config/foo/.env").read_bytes() == b"TOKEN=nuevo\n"
    assert result.snapshot is not None
    snapshot_manifest = read_manifest(result.snapshot)
    (app,) = snapshot_manifest.apps
    by_path = {entry.path: entry for entry in app.files}
    assert by_path[".config/foo/.env"].sha256 == hashlib.sha256(b"TOKEN=actual\n").hexdigest()


def test_deploy_marker_uses_injected_home_not_real_home(tmp_path: Path, monkeypatch) -> None:
    app = AppDefinition.model_validate(
        {
            "id": "oh-my-zsh",
            "name": "Oh My Zsh",
            "command": "zsh",
            "install_marker": ".oh-my-zsh/oh-my-zsh.sh",
            "paths": [".zshrc"],
        }
    )
    catalog = {"oh-my-zsh": app}
    source = tmp_path / "source"
    _write_file(source, ".oh-my-zsh/oh-my-zsh.sh", b"")
    _write_file(source, ".zshrc", b"export ZSH=1\n")
    clone = tmp_path / "clone.tar.gz"
    run_clone(
        home=source,
        output=clone,
        catalog=catalog,
        application_dirs=[],
        path=_fake_path(tmp_path / "source-path", "zsh"),
        metadata=_metadata(),
    )
    target = tmp_path / "target"
    target.mkdir()
    real_home = tmp_path / "real-home"
    _write_file(real_home, ".oh-my-zsh/oh-my-zsh.sh", b"")
    monkeypatch.setenv("HOME", str(real_home))
    bin_path = _fake_path(tmp_path / "target-path", "zsh")

    omitted = run_deploy(
        clone,
        home=target,
        catalog=catalog,
        application_dirs=[],
        path=bin_path,
        snapshots_dir=tmp_path / "snapshots",
    )
    assert omitted.omitted == []
    assert omitted.not_installed == [".zshrc"]
    assert not (target / ".zshrc").exists()

    _write_file(target, ".oh-my-zsh/oh-my-zsh.sh", b"")
    deployed = run_deploy(
        clone,
        home=target,
        catalog=catalog,
        application_dirs=[],
        path=bin_path,
        snapshots_dir=tmp_path / "snapshots",
    )
    assert deployed.deployed == [".zshrc"]


def test_ghostty_absolute_reference_warns_on_deploy_and_dry_run(tmp_path: Path) -> None:
    reference = "/Users/otro/.config/ghostty/extra.conf"
    clone, catalog = _clone(
        tmp_path,
        apps={"ghostty": [".config/ghostty/config"]},
        files={".config/ghostty/config": f"config-file = {reference}\n".encode()},
    )
    home = tmp_path / "home"
    home.mkdir()

    dry = _run(tmp_path, clone, home, catalog, installed=("ghostty",), dry_run=True)

    assert dry.plan.notices
    assert reference not in " ".join(dry.plan.notices)
    assert not (home / ".config").exists()

    result = _run(tmp_path, clone, home, catalog, installed=("ghostty",))

    assert result.plan.notices
    assert (home / ".config/ghostty/config").read_bytes() == (
        f"config-file = {reference}\n".encode()
    )


# --- conflict session API (§5.2 and deploy-conflicts) ---


def _prepare(tmp_path: Path, clone: Path, home: Path, catalog: dict[str, AppDefinition]):
    return prepare_deploy(
        clone,
        home=home,
        catalog=catalog,
        application_dirs=[],
        path=_fake_path(tmp_path / "target-path", "foo"),
        snapshots_dir=tmp_path / "snapshots",
    )


def _two_hunk_clone(tmp_path: Path) -> tuple[Path, dict[str, AppDefinition], str, str]:
    local = "".join(f"line {index}\n" for index in range(20))
    clone = local.replace("line 1\n", "LINE ONE\n").replace("line 18\n", "LINE EIGHTEEN\n")
    path, catalog = _clone(tmp_path, apps={"foo": ["cfg.conf"]}, files={"cfg.conf": clone.encode()})
    return path, catalog, local, clone


def test_plan_reports_content_conflict_and_dry_run_keeps_text(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"a = 1\n")
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"a = 2\n"})
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        plan = session.plan(dry_run=True)

        assert len(plan.conflicts) == 1
        conflict = plan.conflicts[0]
        assert conflict.kind is ConflictKind.TEXT
        assert conflict.local_text == "a = 1\n"
        assert conflict.clone_text == "a = 2\n"
        assert plan.entries[0].status is DeployStatus.DIFFERENT
        assert session.staging is None
        assert (home / "cfg.toml").read_bytes() == b"a = 1\n"
    finally:
        session.close()


def test_apply_omit_keeps_local_without_snapshot(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"a = 1\n")
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"a = 2\n"})
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        plan = session.plan()
        result = session.apply(omit_all(plan.conflicts))

        assert (home / "cfg.toml").read_bytes() == b"a = 1\n"
        assert result.deployed == []
        assert result.merged == []
        assert result.omitted == ["cfg.toml"]
        assert result.snapshot is None
        assert not (tmp_path / "snapshots").exists()
    finally:
        session.close()


def test_apply_review_mixes_lines_and_classifies_merged(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.conf", "".join(f"line {index}\n" for index in range(20)).encode())
    clone, catalog, local, clone_text = _two_hunk_clone(tmp_path)
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        plan = session.plan()
        conflict = plan.conflicts[0]
        decisions = ConflictDecisions(
            {conflict.path: FileDecision(FileAction.REVIEW, (RowDecision.CLONE, RowDecision.LOCAL))}
        )
        result = session.apply(decisions)

        written = (home / "cfg.conf").read_text()
        expected = compose(local, clone_text, (RowDecision.CLONE, RowDecision.LOCAL))
        assert written == expected
        assert "LINE ONE" in written
        assert "line 18" in written
        assert result.merged == ["cfg.conf"]
        assert result.deployed == []
        assert result.omitted == []
        assert result.snapshot is not None
    finally:
        session.close()


def test_apply_review_mixes_lines_inside_one_block(tmp_path: Path) -> None:
    home = tmp_path / "home"
    local = "a\nb\nc\n"
    clone_text = "a\nB\nC\n"
    target = _write_file(home, "cfg.conf", local.encode())
    clone, catalog = _clone(
        tmp_path, apps={"foo": ["cfg.conf"]}, files={"cfg.conf": clone_text.encode()}
    )
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        plan = session.plan()
        conflict = plan.conflicts[0]
        result = session.apply(
            ConflictDecisions(
                {
                    conflict.path: FileDecision(
                        FileAction.REVIEW, (RowDecision.CLONE, RowDecision.LOCAL)
                    )
                }
            )
        )

        assert target.read_text() == "a\nB\nc\n"
        assert result.merged == ["cfg.conf"]
        assert result.deployed == []
        assert result.omitted == []
    finally:
        session.close()


def test_review_rejects_a_decision_list_of_the_wrong_length(tmp_path: Path) -> None:
    home = tmp_path / "home"
    local = "a\nb\nc\n"
    target = _write_file(home, "cfg.conf", local.encode())
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.conf"]}, files={"cfg.conf": b"a\nB\nC\n"})
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        plan = session.plan()
        conflict = plan.conflicts[0]
        decisions = ConflictDecisions(
            {conflict.path: FileDecision(FileAction.REVIEW, (RowDecision.CLONE,))}
        )
        with pytest.raises(ConflictError, match="no cubre todas las líneas"):
            session.apply(decisions)

        assert target.read_text() == local
        assert not (tmp_path / "snapshots").exists()
    finally:
        session.close()


def test_review_all_local_writes_nothing_and_keeps_mode(tmp_path: Path) -> None:
    home = tmp_path / "home"
    target = _write_file(
        home, "cfg.conf", "".join(f"line {index}\n" for index in range(20)).encode()
    )
    target.chmod(0o600)
    clone, catalog, _local, _clone_text = _two_hunk_clone(tmp_path)
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        plan = session.plan()
        conflict = plan.conflicts[0]
        result = session.apply(
            ConflictDecisions({conflict.path: review_all(conflict, RowDecision.LOCAL)})
        )

        assert target.read_text() == "".join(f"line {index}\n" for index in range(20))
        assert stat.S_IMODE(target.stat().st_mode) == 0o600
        assert result.omitted == ["cfg.conf"]
        assert result.snapshot is None
    finally:
        session.close()


def test_review_all_clone_is_deployed_with_clone_mode(tmp_path: Path) -> None:
    home = tmp_path / "home"
    target = _write_file(
        home, "cfg.conf", "".join(f"line {index}\n" for index in range(20)).encode()
    )
    target.chmod(0o600)
    clone, catalog, _local, _clone_text = _two_hunk_clone(tmp_path)
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        plan = session.plan()
        conflict = plan.conflicts[0]
        result = session.apply(ConflictDecisions({conflict.path: review_all(conflict)}))

        assert "LINE ONE" in target.read_text()
        assert "LINE EIGHTEEN" in target.read_text()
        assert stat.S_IMODE(target.stat().st_mode) == 0o644
        assert result.deployed == ["cfg.conf"]
        assert result.merged == []
    finally:
        session.close()


def test_apply_requires_decisions_for_every_conflict(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"a = 1\n")
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"a = 2\n"})
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        session.plan()
        with pytest.raises(DecisionRequiredError):
            session.apply(ConflictDecisions())

        assert (home / "cfg.toml").read_bytes() == b"a = 1\n"
        assert not (tmp_path / "snapshots").exists()
    finally:
        session.close()


def test_core_revalidates_merge_before_writing(tmp_path: Path, monkeypatch) -> None:
    local = "a = 1\n"
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", local.encode())
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"a = 2\n"})
    monkeypatch.setattr(
        deploy_module,
        "validate_merge",
        lambda *args, **kwargs: SimpleNamespace(status="invalid", message="boom"),
    )
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        plan = session.plan()
        conflict = plan.conflicts[0]
        decisions = ConflictDecisions(
            {conflict.path: FileDecision(FileAction.REVIEW, (RowDecision.CLONE,))}
        )
        with pytest.raises(InvalidMergeError):
            session.apply(decisions)

        assert (home / "cfg.toml").read_text() == local
        assert not (tmp_path / "snapshots").exists()
    finally:
        session.close()


def test_destination_changed_between_plan_and_apply_aborts(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"a = 1\n")
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"a = 2\n"})
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        plan = session.plan()
        _write_file(home, "cfg.toml", b"a = 3\n")

        with pytest.raises(ConcurrentChangeError):
            session.apply(apply_all(plan.conflicts))

        assert (home / "cfg.toml").read_bytes() == b"a = 3\n"
        assert not (tmp_path / "snapshots").exists()
    finally:
        session.close()


def test_new_destination_appearing_after_plan_aborts(tmp_path: Path) -> None:
    home = tmp_path / "home"
    home.mkdir()
    clone, catalog = _clone(
        tmp_path, apps={"foo": ["nuevo.toml"]}, files={"nuevo.toml": b"nuevo\n"}
    )
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        session.plan()
        _write_file(home, "nuevo.toml", b"intruso\n")

        with pytest.raises(ConcurrentChangeError):
            session.apply(ConflictDecisions())

        assert (home / "nuevo.toml").read_bytes() == b"intruso\n"
        assert not (tmp_path / "snapshots").exists()
    finally:
        session.close()


def test_symlink_target_changed_after_plan_aborts(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "real.txt")
    (home / "enlace.txt").symlink_to("uno.txt")
    clone, catalog = _clone(
        tmp_path, apps={"foo": ["enlace.txt"]}, files={}, links={"enlace.txt": "dos.txt"}
    )
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        plan = session.plan()
        (home / "enlace.txt").unlink()
        (home / "enlace.txt").symlink_to("tres.txt")

        with pytest.raises(ConcurrentChangeError):
            session.apply(apply_all(plan.conflicts))

        assert os.readlink(home / "enlace.txt") == "tres.txt"
        assert not (tmp_path / "snapshots").exists()
    finally:
        session.close()


def test_snapshot_only_contains_written_destinations(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "a.toml", b"a = 1\n")
    _write_file(home, "b.toml", b"b = 1\n")
    clone, catalog = _clone(
        tmp_path,
        apps={"foo": ["a.toml", "b.toml"]},
        files={"a.toml": b"a = 2\n", "b.toml": b"b = 2\n"},
    )
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        plan = session.plan()
        by_path = {conflict.path: conflict for conflict in plan.conflicts}
        decisions = ConflictDecisions(
            {
                "a.toml": FileDecision(FileAction.APPLY),
                "b.toml": FileDecision(FileAction.OMIT),
            }
        )
        assert by_path.keys() == decisions.decisions.keys()
        result = session.apply(decisions)

        assert (home / "a.toml").read_bytes() == b"a = 2\n"
        assert (home / "b.toml").read_bytes() == b"b = 1\n"
        assert result.snapshot is not None
        snapshot_manifest = read_manifest(result.snapshot)
        (app,) = snapshot_manifest.apps
        assert [entry.path for entry in app.files] == ["a.toml"]
    finally:
        session.close()


def test_type_change_is_not_a_conflict(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "real.txt", b"real\n")
    (home / "cfg.toml").symlink_to("real.txt")
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"regular\n"})
    session = _prepare(tmp_path, clone, home, catalog)
    try:
        plan = session.plan()

        assert plan.conflicts == []
        assert plan.entries[0].status is DeployStatus.DIFFERENT
        result = session.apply(ConflictDecisions())
        assert result.deployed == ["cfg.toml"]
        assert not (home / "cfg.toml").is_symlink()
    finally:
        session.close()


def test_dry_run_does_not_verify_the_archive(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "cfg.toml", b"a = 1\n")
    clone, catalog = _clone(tmp_path, apps={"foo": ["cfg.toml"]}, files={"cfg.toml": b"a = 2\n"})
    tampered = _tamper(clone, "apps/foo/cfg.toml", b"alterado\n")

    result = _run(tmp_path, tampered, home, catalog, dry_run=True)

    assert result.dry_run is True
    assert result.plan.conflicts
    assert (home / "cfg.toml").read_bytes() == b"a = 1\n"
    assert not (tmp_path / "snapshots").exists()


# --- cancelación, parciales y código por categorías ---


def test_partial_apply_reports_applied_failed_and_snapshot(tmp_path: Path, monkeypatch) -> None:
    home = tmp_path / "home"
    _write_file(home, "a.toml", b"viejo a\n")
    _write_file(home, "b.toml", b"viejo b\n")
    clone, catalog = _clone(
        tmp_path,
        apps={"foo": ["a.toml", "b.toml"]},
        files={"a.toml": b"nuevo a\n", "b.toml": b"nuevo b\n"},
    )
    from kamino.paths import SafeHome

    original = SafeHome.write_file

    def fail_second(self, rel, data, *, mode=0o644):
        if str(rel) == "b.toml":
            raise OSError("sin espacio")
        return original(self, rel, data, mode=mode)

    monkeypatch.setattr(SafeHome, "write_file", fail_second)

    with pytest.raises(PartialApplyError) as caught:
        _run(tmp_path, clone, home, catalog)

    error = caught.value
    assert error.applied == ["a.toml"]
    assert error.failed == "b.toml"
    assert error.snapshot is not None
    assert (home / "a.toml").read_bytes() == b"nuevo a\n"
    assert (home / "b.toml").read_bytes() == b"viejo b\n"


def test_missing_requested_app_produces_code_3(tmp_path: Path) -> None:
    from kamino.conflicts import apply_all
    from kamino.deploy import result_code

    home = tmp_path / "home"
    clone, catalog = _clone(tmp_path, apps={"foo": ["a.toml"]}, files={"a.toml": b"a\n"})
    session = prepare_deploy(
        clone,
        home=home,
        catalog=catalog,
        application_dirs=[],
        path=_fake_path(tmp_path / "target-path", "foo"),
        snapshots_dir=tmp_path / "snapshots",
    )
    try:
        plan = session.plan(["foo"], requested=["foo", "no-existe"])
        result = session.apply(apply_all(plan.conflicts))
    finally:
        session.close()

    assert result.missing_requested == ["no-existe"]
    assert result_code(result) == 3


def test_only_perfect_clones_produce_code_0(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_file(home, "a.toml", b"igual\n")
    clone, catalog = _clone(tmp_path, apps={"foo": ["a.toml"]}, files={"a.toml": b"igual\n"})

    result = _run(tmp_path, clone, home, catalog)

    assert result.unchanged == ["a.toml"]
    assert result.omitted == []
    from kamino.deploy import result_code

    assert result_code(result) == 0


def test_not_installed_only_produces_code_0(tmp_path: Path) -> None:
    home = tmp_path / "home"
    clone, catalog = _clone(
        tmp_path,
        apps={"alpha": ["a.toml"], "beta": ["b.toml"]},
        files={"a.toml": b"a\n", "b.toml": b"b\n"},
    )

    result = _run(tmp_path, clone, home, catalog, installed=("alpha",))

    assert result.not_installed == ["b.toml"]
    from kamino.deploy import result_code

    assert result_code(result) == 0


def test_platform_warnings_are_exposed_without_blocking(tmp_path: Path, monkeypatch) -> None:
    import kamino.deploy as deploy_module

    home = tmp_path / "home"
    clone, catalog = _clone(tmp_path, apps={"foo": ["a.toml"]}, files={"a.toml": b"a\n"})
    monkeypatch.setattr(deploy_module.platform, "machine", lambda: "x86_64")
    monkeypatch.setattr(deploy_module.platform, "mac_ver", lambda: ("14.0", "", ""))
    session = prepare_deploy(
        clone,
        home=home,
        catalog=catalog,
        application_dirs=[],
        path=_fake_path(tmp_path / "target-path", "foo"),
        snapshots_dir=tmp_path / "snapshots",
    )
    try:
        assert any("x86_64" in warning for warning in session.warnings)
        assert any("14.0" in warning for warning in session.warnings)
        plan = session.plan(["foo"])
        result = session.apply(apply_all(plan.conflicts))
    finally:
        session.close()

    assert result.deployed == ["a.toml"]
    from kamino.deploy import result_code

    assert result_code(result) == 0


def test_verify_and_extract_cancellation(tmp_path: Path) -> None:
    from kamino.archive import extract_archive, verify_archive
    from kamino.errors import OperationCancelled

    clone, catalog = _clone(tmp_path, apps={"foo": ["a.toml"]}, files={"a.toml": b"a\n"})
    manifest = read_manifest(clone)
    with pytest.raises(OperationCancelled):
        verify_archive(clone, manifest, cancelled=lambda: True)
    staging = tmp_path / "staging"
    with pytest.raises(OperationCancelled):
        extract_archive(clone, staging, cancelled=lambda: True)
    assert not list(staging.rglob("a.toml"))


def test_plan_cancellation_cleans_staging(tmp_path: Path) -> None:
    from kamino.errors import OperationCancelled
    from kamino.progress import Phase

    clone, catalog = _clone(tmp_path, apps={"foo": ["a.toml"]}, files={"a.toml": b"a\n"})
    session = prepare_deploy(
        clone,
        home=tmp_path / "home",
        catalog=catalog,
        application_dirs=[],
        path=_fake_path(tmp_path / "target-path", "foo"),
        snapshots_dir=tmp_path / "snapshots",
    )
    cancelled = {"value": False}

    def progress(event):
        if event.phase is Phase.CLASSIFYING:
            cancelled["value"] = True

    try:
        with pytest.raises(OperationCancelled):
            session.plan(
                ["foo"],
                progress=progress,
                cancelled=lambda: cancelled["value"],
            )
        assert session.staging is None
    finally:
        session.close()


def test_old_snapshot_manifest_with_synthetic_id_is_readable(tmp_path: Path) -> None:
    from kamino.archive import FORMAT_VERSION, AppEntry, FileEntry, Manifest, write_archive
    from kamino.snapshot import SNAPSHOT_APP_ID

    home = tmp_path / "home"
    _write_file(home, "viejo.toml", b"actual\n")
    source = tmp_path / "source"
    _write_file(source, "viejo.toml", b"viejo\n")
    old = tmp_path / "old-snapshot.tar.gz"
    manifest = Manifest(
        format_version=FORMAT_VERSION,
        tool_version="0.1.0",
        created_at="2026-01-01T00:00:00Z",
        macos_version="15.1",
        arch="arm64",
        hostname="mac-test",
        apps=[
            AppEntry(
                id=SNAPSHOT_APP_ID,
                name=SNAPSHOT_APP_ID,
                files=[
                    FileEntry(
                        path="viejo.toml",
                        sha256=hashlib.sha256(b"viejo\n").hexdigest(),
                        size=7,
                        mode="0644",
                    )
                ],
            )
        ],
    )
    write_archive(old, manifest, source_root=source)

    result = _run(tmp_path, old, home, {})

    assert result.deployed == ["viejo.toml"]
    assert result.snapshot is not None
