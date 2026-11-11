"""Pruebas del archivo de configuración exclusivo de la TUI."""

from __future__ import annotations

import tomllib
from pathlib import Path

import pytest

from kamino.config import (
    CATALOG_KEY,
    CLONE_KEY,
    AppConfig,
    ConfigLoadResult,
    config_path,
    load_config,
    resolve_path,
    save_config,
)
from kamino.errors import ConfigError


def _write(path: Path, text: str) -> Path:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")
    return path


def _full_config(home: Path) -> str:
    return (
        "[catalog]\n"
        f'dir = "{home / "apps"}"\n'
        "\n[clone]\n"
        f'output_dir = "{home / "clones"}"\n'
        "\n[deploy]\n"
        f'snapshots_dir = "{home / "snapshots"}"\n'
    )


def test_absent_file_uses_defaults(tmp_path: Path) -> None:
    home = tmp_path / "home"

    result = load_config(home=home)

    assert result.error is None
    assert result.exists is False
    assert result.config.catalog_dir == home / ".config" / "kamino" / "apps"
    assert result.config.clone_output_dir == home / ".config" / "kamino" / "vault"
    assert result.config.snapshots_dir == home / ".config" / "kamino" / "vault" / "snapshot"


def test_valid_file_loads_all_paths(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write(config_path(home), _full_config(home))

    result = load_config(home=home)

    assert result.error is None
    assert result.exists is True
    assert result.config.catalog_dir == home / "apps"
    assert result.config.clone_output_dir == home / "clones"
    assert result.config.snapshots_dir == home / "snapshots"
    assert result.raw[CATALOG_KEY] == str(home / "apps")


def test_omitted_keys_use_defaults(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write(
        config_path(home),
        f'[catalog]\ndir = "{home / "apps"}"\n',
    )

    result = load_config(home=home)

    assert result.error is None
    assert result.config.catalog_dir == home / "apps"
    assert result.config.clone_output_dir == home / ".config" / "kamino" / "vault"
    assert result.config.snapshots_dir == home / ".config" / "kamino" / "vault" / "snapshot"


def test_tilde_expands_against_injected_home(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write(
        config_path(home),
        "[catalog]\n"
        'dir = "~/apps"\n'
        "\n[clone]\n"
        'output_dir = "~/clones"\n'
        "\n[deploy]\n"
        'snapshots_dir = "~/snapshots"\n',
    )

    result = load_config(home=home)

    assert result.error is None
    assert result.config.catalog_dir == home / "apps"
    assert result.config.clone_output_dir == home / "clones"
    assert result.config.snapshots_dir == home / "snapshots"


def test_invalid_toml_returns_defaults_and_error(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write(config_path(home), "[catalog\ndir = ")

    result = load_config(home=home)

    assert result.exists is True
    assert result.error is not None
    assert "TOML inválido" in str(result.error)
    assert result.config.catalog_dir == home / ".config" / "kamino" / "apps"


def test_unknown_section_is_rejected(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write(config_path(home), "[ui]\nplain = false\n")

    result = load_config(home=home)

    assert result.error is not None
    assert "ui" in str(result.error)


def test_unknown_key_is_rejected(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write(config_path(home), "[clone]\nforce = true\n")

    result = load_config(home=home)

    assert result.error is not None
    assert "clone.force" in str(result.error)


def test_wrong_type_is_rejected(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write(config_path(home), "[clone]\noutput_dir = 42\n")

    result = load_config(home=home)

    assert result.error is not None
    assert "cadena" in str(result.error)
    assert result.config.clone_output_dir == home / ".config" / "kamino" / "vault"


def test_empty_relative_and_tilde_user_paths_are_rejected(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write(
        config_path(home),
        "[catalog]\n"
        'dir = ""\n'
        "\n[clone]\n"
        'output_dir = "Downloads"\n'
        "\n[deploy]\n"
        'snapshots_dir = "~otro/vault"\n',
    )

    result = load_config(home=home)

    assert result.error is not None
    message = str(result.error)
    assert "no puede estar vacía" in message
    assert "ruta absoluta" in message
    assert "~ o ~/" in message
    assert result.raw[CATALOG_KEY] == ""
    assert result.raw[CLONE_KEY] == "Downloads"


def test_existing_file_that_is_not_a_directory_is_rejected(tmp_path: Path) -> None:
    home = tmp_path / "home"
    (home / "apps").parent.mkdir(parents=True, exist_ok=True)
    (home / "apps").write_text("not a dir", encoding="utf-8")
    _write(config_path(home), f'[catalog]\ndir = "{home / "apps"}"\n')

    result = load_config(home=home)

    assert result.error is not None
    assert "no es un directorio" in str(result.error)


def test_partial_values_are_kept_for_correction(tmp_path: Path) -> None:
    home = tmp_path / "home"
    _write(
        config_path(home),
        f'[catalog]\ndir = "{home / "apps"}"\n\n[clone]\noutput_dir = "Downloads"\n',
    )

    result = load_config(home=home)

    assert result.error is not None
    assert result.config.catalog_dir == home / "apps"
    assert result.config.clone_output_dir == home / ".config" / "kamino" / "vault"
    assert result.raw[CLONE_KEY] == "Downloads"


def test_resolve_path_rejects_relative_and_tilde_user(tmp_path: Path) -> None:
    with pytest.raises(ConfigError, match="ruta absoluta"):
        resolve_path("relative", home=tmp_path)
    with pytest.raises(ConfigError, match="~ o ~/"):
        resolve_path("~otro/path", home=tmp_path)


def test_save_config_writes_all_keys_and_creates_only_config_dir(tmp_path: Path) -> None:
    home = tmp_path / "home"
    config = AppConfig(
        catalog_dir=home / "apps",
        clone_output_dir=home / "clones",
        snapshots_dir=home / "snapshots",
    )

    target = save_config(config, home=home)

    assert target == config_path(home)
    assert target.exists()
    assert not (home / "apps").exists()
    assert not (home / "clones").exists()
    assert not (home / "snapshots").exists()
    data = tomllib.loads(target.read_text(encoding="utf-8"))
    assert data["catalog"]["dir"] == str(home / "apps")
    assert data["clone"]["output_dir"] == str(home / "clones")
    assert data["deploy"]["snapshots_dir"] == str(home / "snapshots")


def test_save_config_replaces_previous_file(tmp_path: Path) -> None:
    home = tmp_path / "home"
    target = _write(config_path(home), "old content")
    config = AppConfig(
        catalog_dir=home / "apps",
        clone_output_dir=home / "clones",
        snapshots_dir=home / "snapshots",
    )

    save_config(config, home=home)

    assert "old content" not in target.read_text(encoding="utf-8")
    assert not list(target.parent.glob(".config-*.tmp"))


def test_save_config_failure_keeps_previous_file(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    home = tmp_path / "home"
    target = _write(config_path(home), "previous")
    config = AppConfig(
        catalog_dir=home / "apps",
        clone_output_dir=home / "clones",
        snapshots_dir=home / "snapshots",
    )

    def fail_replace(src, dst):
        raise OSError("disk full")

    monkeypatch.setattr("kamino.config.os.replace", fail_replace)

    with pytest.raises(ConfigError, match="no se pudo guardar"):
        save_config(config, home=home)

    assert target.read_text(encoding="utf-8") == "previous"
    assert not list(target.parent.glob(".config-*.tmp"))


def test_save_config_rejects_relative_path(tmp_path: Path) -> None:
    config = AppConfig(
        catalog_dir=Path("apps"),
        clone_output_dir=tmp_path / "clones",
        snapshots_dir=tmp_path / "snapshots",
    )

    with pytest.raises(ConfigError, match="ruta absoluta"):
        save_config(config, home=tmp_path)


def test_load_result_type_is_stable() -> None:
    result = ConfigLoadResult(
        config=AppConfig(Path("/a"), Path("/b"), Path("/c")),
        error=None,
        exists=False,
        raw={},
    )
    assert result.config.catalog_dir == Path("/a")
