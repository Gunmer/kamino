"""Pruebas del paso 2 del plan: modelos y carga del catálogo.

Cubre definiciones válidas e inválidas (diseño técnico §7) y la composición del
catálogo embebido con el directorio del usuario (§6).
"""

from __future__ import annotations

import pytest
from pydantic import ValidationError

from kamino.catalog import AppDefinition, load_catalog
from kamino.errors import CatalogError


def _definition(**overrides: object) -> dict[str, object]:
    base: dict[str, object] = {
        "id": "herdr",
        "name": "herdr",
        "command": "herdr",
        "paths": [".config/herdr/config.toml"],
    }
    base.update(overrides)
    return base


# --- modelos: definiciones válidas ---


def test_minimal_valid_definition() -> None:
    definition = AppDefinition.model_validate(_definition())
    assert definition.id == "herdr"
    assert definition.name == "herdr"
    assert definition.command == "herdr"
    assert definition.bundle_id is None
    assert definition.search_paths == []
    assert definition.paths == [".config/herdr/config.toml"]
    assert definition.exclude == []


def test_bundle_id_detection() -> None:
    definition = AppDefinition.model_validate(
        _definition(bundle_id="com.microsoft.VSCode", command=None)
    )
    assert definition.bundle_id == "com.microsoft.VSCode"
    assert definition.command is None


def test_optional_fields() -> None:
    definition = AppDefinition.model_validate(
        _definition(
            exclude=["**/*.log", "**/session.json"],
            search_paths=["/opt/homebrew/bin"],
        )
    )
    assert definition.exclude == ["**/*.log", "**/session.json"]
    assert definition.search_paths == ["/opt/homebrew/bin"]


# --- modelos: definiciones inválidas ---


def test_id_must_be_a_slug() -> None:
    with pytest.raises(ValidationError):
        AppDefinition.model_validate(_definition(id="Herdr"))
    with pytest.raises(ValidationError):
        AppDefinition.model_validate(_definition(id="herdr_app"))


def test_empty_name_rejected() -> None:
    with pytest.raises(ValidationError):
        AppDefinition.model_validate(_definition(name="   "))


def test_empty_paths_rejected() -> None:
    with pytest.raises(ValidationError):
        AppDefinition.model_validate(_definition(paths=[]))


def test_absolute_path_rejected() -> None:
    with pytest.raises(ValidationError):
        AppDefinition.model_validate(_definition(paths=["/Users/alguien/.config/foo"]))


def test_dotdot_path_rejected() -> None:
    with pytest.raises(ValidationError):
        AppDefinition.model_validate(_definition(paths=["../.ssh/config"]))


def test_tilde_path_rejected() -> None:
    with pytest.raises(ValidationError):
        AppDefinition.model_validate(_definition(paths=["~/.config/foo"]))


def test_missing_detection_mechanism_rejected() -> None:
    with pytest.raises(ValidationError):
        AppDefinition.model_validate(_definition(command=None))


def test_extra_field_rejected() -> None:
    with pytest.raises(ValidationError):
        AppDefinition.model_validate(_definition(unknown="campo"))


# --- loader: composición del catálogo ---


def test_embedded_catalog_has_example(tmp_path, monkeypatch) -> None:
    # HOME sin catálogo del usuario: solo el embebido.
    monkeypatch.setenv("HOME", str(tmp_path))
    catalog = load_catalog()
    assert "git" in catalog
    assert catalog["git"].command == "git"


def test_default_user_catalog_from_home(tmp_path, monkeypatch) -> None:
    monkeypatch.setenv("HOME", str(tmp_path))
    apps_dir = tmp_path / ".config" / "kamino" / "apps"
    apps_dir.mkdir(parents=True)
    (apps_dir / "custom.toml").write_text(
        'id = "custom"\nname = "Custom"\ncommand = "custom"\n'
        'paths = [".config/custom/config.toml"]\n',
        encoding="utf-8",
    )
    catalog = load_catalog()
    assert "git" in catalog
    assert "custom" in catalog


def test_embedded_catalog_has_example_with_missing_user_dir(tmp_path) -> None:
    catalog = load_catalog(user_dir=tmp_path / "no-existe")
    assert "git" in catalog


def test_user_catalog_adds_definition(tmp_path) -> None:
    user_dir = tmp_path / "apps"
    user_dir.mkdir()
    (user_dir / "custom.toml").write_text(
        'id = "custom"\nname = "Custom"\ncommand = "custom"\n'
        'paths = [".config/custom/config.toml"]\n',
        encoding="utf-8",
    )
    catalog = load_catalog(user_dir=user_dir)
    assert "git" in catalog
    assert "custom" in catalog
    assert catalog["custom"].name == "Custom"


def test_user_catalog_overrides_embedded(tmp_path) -> None:
    user_dir = tmp_path / "apps"
    user_dir.mkdir()
    (user_dir / "git.toml").write_text(
        'id = "git"\nname = "Git personalizado"\ncommand = "git"\npaths = [".gitconfig"]\n',
        encoding="utf-8",
    )
    catalog = load_catalog(user_dir=user_dir)
    assert catalog["git"].name == "Git personalizado"
    assert catalog["git"].paths == [".gitconfig"]


def test_invalid_user_definition_raises(tmp_path) -> None:
    user_dir = tmp_path / "apps"
    user_dir.mkdir()
    (user_dir / "bad.toml").write_text(
        'id = "Bad Id"\nname = "bad"\ncommand = "bad"\npaths = ["x"]\n',
        encoding="utf-8",
    )
    with pytest.raises(CatalogError, match="bad.toml"):
        load_catalog(user_dir=user_dir)


def test_invalid_toml_raises(tmp_path) -> None:
    user_dir = tmp_path / "apps"
    user_dir.mkdir()
    (user_dir / "broken.toml").write_text("esto no es toml = [", encoding="utf-8")
    with pytest.raises(CatalogError, match="broken.toml"):
        load_catalog(user_dir=user_dir)


def test_duplicate_user_id_raises(tmp_path) -> None:
    user_dir = tmp_path / "apps"
    user_dir.mkdir()
    content = 'id = "dup"\nname = "Dup"\ncommand = "dup"\npaths = ["x"]\n'
    (user_dir / "a.toml").write_text(content, encoding="utf-8")
    (user_dir / "b.toml").write_text(content, encoding="utf-8")
    with pytest.raises(CatalogError, match="dup"):
        load_catalog(user_dir=user_dir)


def test_non_toml_files_ignored(tmp_path) -> None:
    user_dir = tmp_path / "apps"
    user_dir.mkdir()
    (user_dir / "README.md").write_text("nada", encoding="utf-8")
    catalog = load_catalog(user_dir=user_dir)
    assert "git" in catalog


def test_blank_bundle_id_rejected() -> None:
    with pytest.raises(ValidationError):
        AppDefinition.model_validate(_definition(bundle_id=""))


def test_blank_command_rejected() -> None:
    with pytest.raises(ValidationError):
        AppDefinition.model_validate(_definition(command="   "))


def test_unreadable_user_dir_raises(tmp_path) -> None:
    user_dir = tmp_path / "apps"
    user_dir.mkdir()
    user_dir.chmod(0o000)
    try:
        try:
            next(user_dir.iterdir(), None)
        except OSError:
            pass
        else:
            pytest.skip("el entorno permite listar directorios sin permiso")
        with pytest.raises(CatalogError):
            load_catalog(user_dir=user_dir)
    finally:
        user_dir.chmod(0o755)


def test_user_dir_is_file_raises(tmp_path) -> None:
    user_file = tmp_path / "apps"
    user_file.write_text("no es un directorio", encoding="utf-8")
    with pytest.raises(CatalogError):
        load_catalog(user_dir=user_file)
