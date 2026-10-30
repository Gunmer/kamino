"""Pruebas de `backup.py`: orquestación del respaldo sin interfaz.

Cubre el flujo del respaldo (diseño técnico §5.1): cargar catálogo, detectar
instaladas, cruzar con la selección, construir el plan, montar el manifiesto
(§4.2) y escribir el archivo (§4.3). Todo se inyecta (home falso, catálogo,
metadatos, detección) para no tocar el sistema real (§7).
"""

from __future__ import annotations

import plistlib
import re
import tarfile
from pathlib import Path

import pytest

from kamino import __version__
from kamino.archive import FORMAT_VERSION, read_manifest, verify_archive
from kamino.backup import BackupResult, run_backup
from kamino.catalog.models import AppDefinition
from kamino.errors import ArchiveError
from kamino.plan import Plan


def _app(
    *,
    app_id: str = "foo",
    name: str = "Foo",
    bundle_id: str | None = None,
    command: str | None = None,
    paths: list[str] | None = None,
    exclude: list[str] | None = None,
) -> AppDefinition:
    return AppDefinition.model_validate(
        {
            "id": app_id,
            "name": name,
            "bundle_id": bundle_id,
            "command": command if command is not None else (None if bundle_id else app_id),
            "paths": paths if paths is not None else [".config/foo/config.toml"],
            "exclude": exclude or [],
        }
    )


def _catalog(*apps: AppDefinition) -> dict[str, AppDefinition]:
    return {app.id: app for app in apps}


def _write_config(home: Path, rel: str, content: bytes = b"config\n") -> Path:
    target = home / rel
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_bytes(content)
    return target


def _fake_path(tmp_path: Path, *commands: str) -> str:
    """Crea un PATH falso con los binarios indicados (apps "instaladas")."""
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir(parents=True, exist_ok=True)
    for command in commands:
        binary = bin_dir / command
        binary.write_text("", encoding="utf-8")
        binary.chmod(0o755)
    return str(bin_dir)


def _empty_path(tmp_path: Path) -> str:
    """Un PATH vacío: ninguna herramienta de terminal aparece como instalada."""
    empty = tmp_path / "empty-path"
    empty.mkdir(parents=True, exist_ok=True)
    return str(empty)


def _make_app_bundle(
    directory: Path, name: str, *, bundle_id: str, version: str | None = None
) -> Path:
    app = directory / f"{name}.app"
    contents = app / "Contents"
    contents.mkdir(parents=True)
    payload: dict[str, str] = {"CFBundleIdentifier": bundle_id}
    if version is not None:
        payload["CFBundleShortVersionString"] = version
    with (contents / "Info.plist").open("wb") as handle:
        plistlib.dump(payload, handle)
    return app


def _metadata(**overrides: str) -> dict[str, str]:
    data = {
        "macos_version": "15.1",
        "arch": "arm64",
        "hostname": "mac-test",
        "created_at": "2026-10-28T21:30:00Z",
    }
    data.update(overrides)
    return data


# --- dry run ---


def test_dry_run_returns_plan_and_manifest_without_writing(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_config(home, ".config/foo/config.toml")
    output = tmp_path / "backup.tar.gz"

    result = run_backup(
        home=home,
        output=output,
        dry_run=True,
        catalog=_catalog(_app()),
        application_dirs=[],
        path=_fake_path(tmp_path, "foo"),
        metadata=_metadata(),
    )

    assert isinstance(result, BackupResult)
    assert isinstance(result.plan, Plan)
    assert result.dry_run is True
    assert result.output is None
    assert result.manifest.format_version == FORMAT_VERSION
    assert [entry.id for entry in result.manifest.apps] == ["foo"]
    assert not output.exists()


# --- ejecución real ---


def test_run_writes_archive_with_manifest_and_files(tmp_path: Path) -> None:
    home = tmp_path / "home"
    content = b"[user]\nname=Test\n"
    _write_config(home, ".config/foo/config.toml", content)
    output = tmp_path / "backup.tar.gz"

    result = run_backup(
        home=home,
        output=output,
        catalog=_catalog(_app()),
        application_dirs=[],
        path=_fake_path(tmp_path, "foo"),
        metadata=_metadata(),
    )

    assert result.output == output
    assert result.dry_run is False
    assert output.exists()
    assert read_manifest(output) == result.manifest
    verify_archive(output, result.manifest)

    with tarfile.open(output, "r:gz") as tar:
        extracted = tar.extractfile("apps/foo/.config/foo/config.toml")
        assert extracted is not None
        assert extracted.read() == content


def test_manifest_header_fields(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_config(home, ".config/foo/config.toml")

    result = run_backup(
        home=home,
        output=tmp_path / "b.tar.gz",
        catalog=_catalog(_app()),
        application_dirs=[],
        path=_fake_path(tmp_path, "foo"),
        metadata=_metadata(),
    )

    manifest = result.manifest
    assert manifest.format_version == 1
    assert manifest.tool_version == __version__
    assert manifest.created_at == "2026-10-28T21:30:00Z"
    assert manifest.macos_version == "15.1"
    assert manifest.arch == "arm64"
    assert manifest.hostname == "mac-test"


def test_manifest_defaults_come_from_platform(tmp_path: Path) -> None:
    """Sin `metadata` los valores salen de `platform`; `created_at` es ISO 8601 UTC."""
    home = tmp_path / "home"
    _write_config(home, ".config/foo/config.toml")

    result = run_backup(
        home=home,
        output=tmp_path / "b.tar.gz",
        catalog=_catalog(_app()),
        application_dirs=[],
        path=_fake_path(tmp_path, "foo"),
    )

    manifest = result.manifest
    assert re.fullmatch(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z", manifest.created_at)
    assert manifest.arch != ""
    assert manifest.hostname != ""


# --- entradas de app y archivos del manifiesto ---


def test_app_entry_includes_detected_version(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_config(home, ".config/foo/config.toml")
    apps_dir = tmp_path / "Applications"
    _make_app_bundle(apps_dir, "Foo", bundle_id="com.example.foo", version="1.95.0")

    result = run_backup(
        home=home,
        output=tmp_path / "b.tar.gz",
        catalog=_catalog(_app(bundle_id="com.example.foo")),
        application_dirs=[apps_dir],
        path=_empty_path(tmp_path),
        metadata=_metadata(),
    )

    entry = result.manifest.apps[0]
    assert entry.id == "foo"
    assert entry.name == "Foo"
    assert entry.bundle_id == "com.example.foo"
    assert entry.version == "1.95.0"


def test_command_app_version_is_none(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_config(home, ".config/foo/config.toml")

    result = run_backup(
        home=home,
        output=tmp_path / "b.tar.gz",
        catalog=_catalog(_app(command="foo")),
        application_dirs=[],
        path=_fake_path(tmp_path, "foo"),
        metadata=_metadata(),
    )

    assert result.manifest.apps[0].version is None


def test_manifest_files_carry_checksum_size_and_mode(tmp_path: Path) -> None:
    home = tmp_path / "home"
    content = b"abc"
    _write_config(home, ".config/foo/config.toml", content)

    result = run_backup(
        home=home,
        output=tmp_path / "b.tar.gz",
        catalog=_catalog(_app()),
        application_dirs=[],
        path=_fake_path(tmp_path, "foo"),
        metadata=_metadata(),
    )

    (file_entry,) = result.manifest.apps[0].files
    assert file_entry.path == ".config/foo/config.toml"
    assert file_entry.size == len(content)
    assert file_entry.link is None
    assert file_entry.sha256 is not None and len(file_entry.sha256) == 64


def test_manifest_includes_link_entries_for_symlinks(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_config(home, "real.txt")
    (home / "enlace.txt").symlink_to("real.txt")

    result = run_backup(
        home=home,
        output=tmp_path / "b.tar.gz",
        catalog=_catalog(_app(paths=["real.txt", "enlace.txt"])),
        application_dirs=[],
        path=_fake_path(tmp_path, "foo"),
        metadata=_metadata(),
    )

    by_path = {entry.path: entry for entry in result.manifest.apps[0].files}
    assert by_path["enlace.txt"].link == "real.txt"
    assert by_path["enlace.txt"].sha256 is None
    assert by_path["real.txt"].sha256 is not None


def test_missing_paths_are_skipped_and_absent_from_manifest(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_config(home, "existe.toml")

    result = run_backup(
        home=home,
        output=tmp_path / "b.tar.gz",
        catalog=_catalog(_app(paths=["existe.toml", "falta.toml"])),
        application_dirs=[],
        path=_fake_path(tmp_path, "foo"),
        metadata=_metadata(),
    )

    assert result.plan.skipped == ["falta.toml"]
    assert [entry.path for entry in result.manifest.apps[0].files] == ["existe.toml"]


# --- selección de apps ---


def test_selected_none_backs_up_all_installed(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_config(home, "a.toml")
    _write_config(home, "b.toml")

    result = run_backup(
        home=home,
        output=tmp_path / "b.tar.gz",
        selected=None,
        catalog=_catalog(
            _app(app_id="alpha", paths=["a.toml"]),
            _app(app_id="beta", paths=["b.toml"]),
        ),
        application_dirs=[],
        path=_fake_path(tmp_path, "alpha", "beta"),
        metadata=_metadata(),
    )

    assert [entry.id for entry in result.manifest.apps] == ["alpha", "beta"]


def test_selected_filters_installed_apps(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_config(home, "a.toml")
    _write_config(home, "b.toml")

    result = run_backup(
        home=home,
        output=tmp_path / "b.tar.gz",
        selected=["beta"],
        catalog=_catalog(
            _app(app_id="alpha", paths=["a.toml"]),
            _app(app_id="beta", paths=["b.toml"]),
        ),
        application_dirs=[],
        path=_fake_path(tmp_path, "alpha", "beta"),
        metadata=_metadata(),
    )

    assert [entry.id for entry in result.manifest.apps] == ["beta"]


def test_not_installed_apps_are_excluded(tmp_path: Path) -> None:
    """Una app seleccionada pero no instalada no se respalda (§5.1)."""
    home = tmp_path / "home"
    _write_config(home, "a.toml")
    _write_config(home, "b.toml")

    result = run_backup(
        home=home,
        output=tmp_path / "b.tar.gz",
        selected=["alpha", "beta"],
        catalog=_catalog(
            _app(app_id="alpha", paths=["a.toml"]),
            _app(app_id="beta", paths=["b.toml"]),
        ),
        application_dirs=[],
        path=_fake_path(tmp_path, "alpha"),  # beta no está instalada
        metadata=_metadata(),
    )

    assert [entry.id for entry in result.manifest.apps] == ["alpha"]


def test_unknown_selected_ids_are_ignored(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_config(home, "a.toml")

    result = run_backup(
        home=home,
        output=tmp_path / "b.tar.gz",
        selected=["no-existe", "alpha"],
        catalog=_catalog(_app(app_id="alpha", paths=["a.toml"])),
        application_dirs=[],
        path=_fake_path(tmp_path, "alpha"),
        metadata=_metadata(),
    )

    assert [entry.id for entry in result.manifest.apps] == ["alpha"]


def test_apps_are_backed_up_in_id_order(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_config(home, "a.toml")
    _write_config(home, "b.toml")

    result = run_backup(
        home=home,
        output=tmp_path / "b.tar.gz",
        catalog=_catalog(
            _app(app_id="zeta", paths=["a.toml"]),
            _app(app_id="alfa", paths=["b.toml"]),
        ),
        application_dirs=[],
        path=_fake_path(tmp_path, "zeta", "alfa"),
        metadata=_metadata(),
    )

    assert [entry.id for entry in result.manifest.apps] == ["alfa", "zeta"]


# --- ubicación y nombre del respaldo (§6) ---


def test_default_output_name_in_cwd(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    workdir = tmp_path / "workdir"
    workdir.mkdir()
    monkeypatch.chdir(workdir)
    home = tmp_path / "home"
    _write_config(home, "a.toml")

    result = run_backup(
        home=home,
        output=None,
        catalog=_catalog(_app(paths=["a.toml"])),
        application_dirs=[],
        path=_fake_path(tmp_path, "foo"),
        metadata=_metadata(),
    )

    expected = workdir / "kamino-mac-test-2026-10-28.tar.gz"
    assert result.output == expected
    assert expected.exists()


def test_default_output_name_adds_suffix_if_it_exists(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    workdir = tmp_path / "workdir"
    workdir.mkdir()
    monkeypatch.chdir(workdir)
    home = tmp_path / "home"
    _write_config(home, "a.toml")
    kwargs = {
        "home": home,
        "output": None,
        "catalog": _catalog(_app(paths=["a.toml"])),
        "application_dirs": [],
        "path": _fake_path(tmp_path, "foo"),
        "metadata": _metadata(),
    }

    first = run_backup(**kwargs)
    second = run_backup(**kwargs)

    assert first.output == workdir / "kamino-mac-test-2026-10-28.tar.gz"
    assert second.output == workdir / "kamino-mac-test-2026-10-28-1.tar.gz"
    assert first.output is not None and first.output.exists()
    assert second.output is not None and second.output.exists()


def test_explicit_output_is_never_overwritten(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write_config(home, "a.toml")
    output = tmp_path / "b.tar.gz"
    kwargs = {
        "home": home,
        "output": output,
        "catalog": _catalog(_app(paths=["a.toml"])),
        "application_dirs": [],
        "path": _fake_path(tmp_path, "foo"),
        "metadata": _metadata(),
    }

    run_backup(**kwargs)
    with pytest.raises(ArchiveError):
        run_backup(**kwargs)
