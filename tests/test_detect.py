"""Pruebas del paso 3: detección de apps instaladas y su versión.

Simula apps `.app` con un `Info.plist` falso y binarios de terminal en un PATH
falso, sin tocar el home ni las carpetas reales (diseño técnico §7).
"""

from __future__ import annotations

import plistlib
from pathlib import Path

import pytest

from kamino.catalog.models import AppDefinition
from kamino.detect import detect_version, is_installed


def _definition(
    *,
    bundle_id: str | None = None,
    command: str | None = None,
    command_aliases: list[str] | None = None,
    install_marker: str | None = None,
    search_paths: list[str] | None = None,
) -> AppDefinition:
    return AppDefinition.model_validate(
        {
            "id": "foo",
            "name": "Foo",
            "bundle_id": bundle_id,
            "command": command,
            "command_aliases": command_aliases or [],
            "install_marker": install_marker,
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


def test_plist_with_invalid_value_does_not_crash(tmp_path: Path) -> None:
    apps_dir = tmp_path / "Applications"
    app_bundle = apps_dir / "Foo.app"
    contents = app_bundle / "Contents"
    contents.mkdir(parents=True)
    (contents / "Info.plist").write_bytes(
        b'<?xml version="1.0" encoding="UTF-8"?>'
        b'<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" '
        b'"http://www.apple.com/DTDs/PropertyList-1.0.dtd">'
        b'<plist version="1.0"><dict>'
        b"<key>CFBundleIdentifier</key><string>com.example.foo</string>"
        b"<key>d</key><date>not-a-real-date</date>"
        b"</dict></plist>"
    )
    app = _definition(bundle_id="com.example.foo")
    assert is_installed(app, application_dirs=[apps_dir]) is False
    assert detect_version(app, application_dirs=[apps_dir]) is None


# --- marcador de instalación y alias ---


def test_marker_required_alongside_command(tmp_path: Path) -> None:
    bin_dir = tmp_path / "bin"
    home = tmp_path / "home"
    bin_dir.mkdir()
    home.mkdir()
    _make_executable(bin_dir, "zsh")
    app = _definition(command="zsh", install_marker=".oh-my-zsh/oh-my-zsh.sh")

    assert is_installed(app, home=home, path=str(bin_dir)) is False

    marker = home / ".oh-my-zsh" / "oh-my-zsh.sh"
    marker.parent.mkdir(parents=True)
    marker.write_text("", encoding="utf-8")
    assert is_installed(app, home=home, path=str(bin_dir)) is True


def test_marker_in_intermediate_symlink_is_not_installed(tmp_path: Path) -> None:
    bin_dir = tmp_path / "bin"
    home = tmp_path / "home"
    external = tmp_path / "external"
    bin_dir.mkdir()
    home.mkdir()
    external.mkdir()
    _make_executable(bin_dir, "zsh")
    (external / "oh-my-zsh.sh").write_text("", encoding="utf-8")
    (home / ".oh-my-zsh").symlink_to(external)
    app = _definition(command="zsh", install_marker=".oh-my-zsh/oh-my-zsh.sh")

    assert is_installed(app, home=home, path=str(bin_dir)) is False


def test_command_alias_is_detected_without_executing(tmp_path: Path) -> None:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    binary = _make_executable(bin_dir, "cave")
    binary.write_text("#!/bin/sh\nexit 99\n", encoding="utf-8")
    binary.chmod(0o755)
    app = _definition(command="caveman", command_aliases=["cave"])

    assert is_installed(app, path=str(bin_dir)) is True


def test_marker_uses_injected_home_not_real_home(tmp_path: Path, monkeypatch) -> None:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    _make_executable(bin_dir, "zsh")
    real_home = tmp_path / "real-home"
    real_home.mkdir()
    marker = real_home / ".oh-my-zsh" / "oh-my-zsh.sh"
    marker.parent.mkdir(parents=True)
    marker.write_text("", encoding="utf-8")
    monkeypatch.setenv("HOME", str(real_home))
    app = _definition(command="zsh", install_marker=".oh-my-zsh/oh-my-zsh.sh")

    assert is_installed(app, home=tmp_path / "fake-home", path=str(bin_dir)) is False
    assert is_installed(app, home=real_home, path=str(bin_dir)) is True


def test_unreadable_application_dir_is_ignored(tmp_path: Path) -> None:
    apps_dir = tmp_path / "Applications"
    apps_dir.mkdir()
    apps_dir.chmod(0o000)
    try:
        try:
            next(apps_dir.iterdir(), None)
        except OSError:
            pass
        else:
            pytest.skip("el entorno permite listar directorios sin permiso")
        app = _definition(bundle_id="com.example.foo")
        assert is_installed(app, application_dirs=[apps_dir]) is False
    finally:
        apps_dir.chmod(0o755)
