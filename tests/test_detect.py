"""Pruebas del paso 3: detección de apps instaladas y su versión.

Simula apps `.app` con un `Info.plist` falso y binarios de terminal en un PATH
falso, sin tocar el home ni las carpetas reales (diseño técnico §7).
"""

from __future__ import annotations

import plistlib
from pathlib import Path

from kamino.catalog.models import AppDefinition
from kamino.detect import detect_version, is_installed


def _definition(
    *,
    bundle_id: str | None = None,
    command: str | None = None,
    search_paths: list[str] | None = None,
) -> AppDefinition:
    return AppDefinition.model_validate(
        {
            "id": "foo",
            "name": "Foo",
            "bundle_id": bundle_id,
            "command": command,
            "search_paths": search_paths or [],
            "paths": [".config/foo/config.toml"],
        }
    )


def _make_app_bundle(
    directory: Path,
    name: str,
    *,
    bundle_id: str,
    info: dict[str, str] | None = None,
) -> Path:
    app = directory / f"{name}.app"
    contents = app / "Contents"
    contents.mkdir(parents=True)
    payload: dict[str, str] = {"CFBundleIdentifier": bundle_id}
    if info:
        payload.update(info)
    with (contents / "Info.plist").open("wb") as handle:
        plistlib.dump(payload, handle)
    return app


def _make_executable(directory: Path, name: str) -> Path:
    binary = directory / name
    binary.write_text("", encoding="utf-8")
    binary.chmod(0o755)
    return binary


# --- instalada: por bundle_id ---


def test_installed_via_bundle_id(tmp_path: Path) -> None:
    apps_dir = tmp_path / "Applications"
    _make_app_bundle(apps_dir, "Foo", bundle_id="com.example.foo")
    app = _definition(bundle_id="com.example.foo")
    assert is_installed(app, application_dirs=[apps_dir]) is True


def test_not_installed_via_bundle_id(tmp_path: Path) -> None:
    apps_dir = tmp_path / "Applications"
    app = _definition(bundle_id="com.example.foo")
    assert is_installed(app, application_dirs=[apps_dir]) is False


def test_bundle_id_ignores_wrong_identifier(tmp_path: Path) -> None:
    apps_dir = tmp_path / "Applications"
    _make_app_bundle(apps_dir, "Foo", bundle_id="com.example.otro")
    app = _definition(bundle_id="com.example.foo")
    assert is_installed(app, application_dirs=[apps_dir]) is False


# --- instalada: por command ---


def test_installed_via_command_in_path(tmp_path: Path) -> None:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    _make_executable(bin_dir, "foo")
    app = _definition(command="foo")
    assert is_installed(app, path=str(bin_dir)) is True


def test_installed_via_search_paths(tmp_path: Path) -> None:
    bin_dir = tmp_path / "opt-bin"
    bin_dir.mkdir()
    _make_executable(bin_dir, "foo")
    empty_path = tmp_path / "empty"
    empty_path.mkdir()
    app = _definition(command="foo", search_paths=[str(bin_dir)])
    assert is_installed(app, path=str(empty_path)) is True


def test_command_not_installed(tmp_path: Path) -> None:
    empty_path = tmp_path / "empty"
    empty_path.mkdir()
    app = _definition(command="foo")
    assert is_installed(app, path=str(empty_path)) is False


# --- versión ---


def test_detect_version_from_plist(tmp_path: Path) -> None:
    apps_dir = tmp_path / "Applications"
    _make_app_bundle(
        apps_dir,
        "Foo",
        bundle_id="com.example.foo",
        info={"CFBundleShortVersionString": "1.2.3"},
    )
    app = _definition(bundle_id="com.example.foo")
    assert detect_version(app, application_dirs=[apps_dir]) == "1.2.3"


def test_detect_version_falls_back_to_bundle_version(tmp_path: Path) -> None:
    apps_dir = tmp_path / "Applications"
    _make_app_bundle(
        apps_dir,
        "Foo",
        bundle_id="com.example.foo",
        info={"CFBundleVersion": "99"},
    )
    app = _definition(bundle_id="com.example.foo")
    assert detect_version(app, application_dirs=[apps_dir]) == "99"


def test_detect_version_command_is_none() -> None:
    app = _definition(command="foo")
    assert detect_version(app) is None


def test_detect_version_missing_app_is_none(tmp_path: Path) -> None:
    app = _definition(bundle_id="com.example.foo")
    assert detect_version(app, application_dirs=[tmp_path / "none"]) is None


# --- plist inválido no rompe la detección ---


def test_invalid_plist_ignored(tmp_path: Path) -> None:
    apps_dir = tmp_path / "Applications"
    app_bundle = apps_dir / "Foo.app"
    contents = app_bundle / "Contents"
    contents.mkdir(parents=True)
    (contents / "Info.plist").write_bytes(b"esto no es un plist")
    app = _definition(bundle_id="com.example.foo")
    assert is_installed(app, application_dirs=[apps_dir]) is False
    assert detect_version(app, application_dirs=[apps_dir]) is None
