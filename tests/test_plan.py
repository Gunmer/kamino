"""Pruebas de `plan.py`: expansión de rutas, exclusiones y operaciones.

Cubre el paso 4 de `clone` (diseño técnico §5.1): expandir `paths` de cada
app, aplicar `exclude` y construir el plan con tamaño y checksum; las rutas
que no existen se registran como omitidas, no como error. Los enlaces
simbólicos se guardan como enlace y no se siguen (§6). Todo se resuelve
contra un home falso (`tmp_path`).
"""

from __future__ import annotations

import hashlib
import os
import stat
from pathlib import Path

import pytest

from kamino.catalog.loader import load_catalog
from kamino.catalog.models import AppDefinition
from kamino.errors import PathError
from kamino.plan import (
    SENSITIVE_PATTERNS,
    Operation,
    Plan,
    build_plan,
    count_files,
    is_excluded,
)


def _app(
    *,
    app_id: str = "foo",
    paths: list[str] | None = None,
    exclude: list[str] | None = None,
) -> AppDefinition:
    return AppDefinition.model_validate(
        {
            "id": app_id,
            "name": app_id,
            "command": app_id,
            "paths": paths if paths is not None else [".config/foo/config.toml"],
            "exclude": exclude or [],
        }
    )


def _sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def _mode_of(path: Path) -> str:
    return f"{stat.S_IMODE(os.lstat(path).st_mode):04o}"


# --- dataclasses ---


def test_operation_is_frozen() -> None:
    operation = Operation(
        app_id="foo",
        path="a.txt",
        source=Path("/tmp/a.txt"),
        size=1,
        sha256="0" * 64,
        mode="0644",
        link=None,
    )
    with pytest.raises(AttributeError):
        operation.size = 2  # type: ignore[misc]


def test_plan_is_frozen_with_defaults() -> None:
    plan = Plan()
    assert plan.operations == []
    assert [item.path for item in plan.skipped] == []
    with pytest.raises(AttributeError):
        plan.skipped = ["x"]  # type: ignore[misc]


# --- expansión de un archivo ---


def test_single_file_operation(tmp_path: Path) -> None:
    content = b"[user]\nname=Test\n"
    source = tmp_path / ".gitconfig"
    source.write_bytes(content)
    source.chmod(0o644)  # el modo depende del umask: se fija para la prueba

    plan = build_plan([_app(paths=[".gitconfig"])], home=tmp_path)

    assert [item.path for item in plan.skipped] == []
    assert len(plan.operations) == 1
    operation = plan.operations[0]
    assert operation.app_id == "foo"
    assert operation.path == ".gitconfig"
    assert operation.source == source
    assert operation.size == len(content)
    assert operation.sha256 == _sha256(content)
    assert operation.mode == "0644"
    assert operation.link is None


def test_file_mode_is_preserved(tmp_path: Path) -> None:
    source = tmp_path / "secreto.txt"
    source.write_text("x", encoding="utf-8")
    source.chmod(0o600)

    plan = build_plan([_app(paths=["secreto.txt"])], home=tmp_path)

    assert plan.operations[0].mode == "0600"


def test_missing_path_is_skipped_not_an_error(tmp_path: Path) -> None:
    plan = build_plan([_app(paths=[".config/foo/no-existe.toml"])], home=tmp_path)

    assert plan.operations == []
    assert [item.path for item in plan.skipped] == [".config/foo/no-existe.toml"]


def test_missing_and_existing_paths_mix(tmp_path: Path) -> None:
    (tmp_path / "existe.txt").write_text("hola", encoding="utf-8")

    plan = build_plan([_app(paths=["falta.txt", "existe.txt"])], home=tmp_path)

    assert [operation.path for operation in plan.operations] == ["existe.txt"]
    assert [item.path for item in plan.skipped] == ["falta.txt"]


# --- expansión de directorios ---


def test_directory_is_walked_recursively(tmp_path: Path) -> None:
    root = tmp_path / ".config" / "foo"
    (root / "sub" / "deep").mkdir(parents=True)
    (root / "config.toml").write_text("a", encoding="utf-8")
    (root / "sub" / "b.toml").write_text("bb", encoding="utf-8")
    (root / "sub" / "deep" / "c.toml").write_text("ccc", encoding="utf-8")

    plan = build_plan([_app(paths=[".config/foo"])], home=tmp_path)

    assert [operation.path for operation in plan.operations] == [
        ".config/foo/config.toml",
        ".config/foo/sub/b.toml",
        ".config/foo/sub/deep/c.toml",
    ]
    assert [item.path for item in plan.skipped] == []
    sizes = {operation.path: operation.size for operation in plan.operations}
    assert sizes[".config/foo/sub/b.toml"] == 2


def test_walk_order_is_deterministic(tmp_path: Path) -> None:
    root = tmp_path / "dir"
    root.mkdir()
    for name in ["z.txt", "a.txt", "m.txt"]:
        (root / name).write_text("x", encoding="utf-8")

    plan = build_plan([_app(paths=["dir"])], home=tmp_path)

    assert [operation.path for operation in plan.operations] == [
        "dir/a.txt",
        "dir/m.txt",
        "dir/z.txt",
    ]


def test_empty_directory_produces_no_operations(tmp_path: Path) -> None:
    (tmp_path / "vacio").mkdir()

    plan = build_plan([_app(paths=["vacio"])], home=tmp_path)

    assert plan.operations == []
    assert [item.path for item in plan.skipped] == []


# --- enlaces simbólicos ---


def test_symlink_stored_as_link_not_followed(tmp_path: Path) -> None:
    (tmp_path / "real.txt").write_text("contenido", encoding="utf-8")
    link = tmp_path / "enlace.txt"
    link.symlink_to("real.txt")

    plan = build_plan([_app(paths=["enlace.txt"])], home=tmp_path)

    operation = plan.operations[0]
    assert operation.link == "real.txt"
    assert operation.sha256 is None
    assert operation.size == 0
    assert operation.mode == _mode_of(link)


def test_symlink_with_nested_relative_target(tmp_path: Path) -> None:
    (tmp_path / "sub").mkdir()
    (tmp_path / "sub" / "real.txt").write_text("x", encoding="utf-8")
    (tmp_path / "enlace.txt").symlink_to("sub/real.txt")

    plan = build_plan([_app(paths=["enlace.txt"])], home=tmp_path)

    assert plan.operations[0].link == "sub/real.txt"


def test_symlink_with_absolute_target_is_skipped(tmp_path: Path) -> None:
    (tmp_path / "enlace.txt").symlink_to("/etc/hosts")

    plan = build_plan([_app(paths=["enlace.txt"])], home=tmp_path)

    assert plan.operations == []
    assert [item.path for item in plan.skipped] == ["enlace.txt"]


def test_symlink_with_parent_escape_is_skipped(tmp_path: Path) -> None:
    (tmp_path / "enlace.txt").symlink_to("../fuera.txt")

    plan = build_plan([_app(paths=["enlace.txt"])], home=tmp_path)

    assert plan.operations == []
    assert [item.path for item in plan.skipped] == ["enlace.txt"]


def test_broken_symlink_still_stored_as_link(tmp_path: Path) -> None:
    """Un enlace roto existe como enlace: se guarda sin seguirlo ni fallar."""
    (tmp_path / "enlace.txt").symlink_to("no-existe.txt")

    plan = build_plan([_app(paths=["enlace.txt"])], home=tmp_path)

    assert plan.operations[0].link == "no-existe.txt"
    assert [item.path for item in plan.skipped] == []


def test_symlinked_directory_inside_walk_is_a_link(tmp_path: Path) -> None:
    (tmp_path / "real-dir").mkdir()
    (tmp_path / "real-dir" / "x.txt").write_text("x", encoding="utf-8")
    (tmp_path / "dir").mkdir()
    (tmp_path / "dir" / "enlace-dir").symlink_to("../real-dir")

    plan = build_plan([_app(paths=["dir"])], home=tmp_path)

    assert plan.operations == []
    assert [item.path for item in plan.skipped] == ["dir/enlace-dir"]


def test_symlinked_directory_with_safe_target_stored_as_link(tmp_path: Path) -> None:
    (tmp_path / "dir").mkdir()
    (tmp_path / "dir" / "otro").mkdir()
    (tmp_path / "dir" / "otro" / "x.txt").write_text("x", encoding="utf-8")
    (tmp_path / "dir" / "enlace-dir").symlink_to("otro")

    plan = build_plan([_app(paths=["dir"])], home=tmp_path)

    assert [(operation.path, operation.link) for operation in plan.operations] == [
        ("dir/enlace-dir", "otro"),
        ("dir/otro/x.txt", None),
    ]


# --- exclusiones ---


def test_star_star_log_pattern_excludes_logs_at_any_depth(tmp_path: Path) -> None:
    root = tmp_path / "app"
    (root / "sub").mkdir(parents=True)
    (root / "config.toml").write_text("c", encoding="utf-8")
    (root / "run.log").write_text("l", encoding="utf-8")
    (root / "sub" / "deep.log").write_text("l", encoding="utf-8")
    (root / "sub" / "data.txt").write_text("d", encoding="utf-8")

    plan = build_plan([_app(paths=["app"], exclude=["**/*.log"])], home=tmp_path)

    assert [operation.path for operation in plan.operations] == [
        "app/config.toml",
        "app/sub/data.txt",
    ]


def test_star_star_dir_pattern_prunes_directory(tmp_path: Path) -> None:
    root = tmp_path / "app"
    (root / "CachedData").mkdir(parents=True)
    (root / "CachedData" / "blob.bin").write_text("b", encoding="utf-8")
    (root / "sub" / "CachedData").mkdir(parents=True)
    (root / "sub" / "CachedData" / "blob.bin").write_text("b", encoding="utf-8")
    (root / "sub" / "keep.toml").write_text("k", encoding="utf-8")

    plan = build_plan([_app(paths=["app"], exclude=["**/CachedData"])], home=tmp_path)

    assert [operation.path for operation in plan.operations] == ["app/sub/keep.toml"]


def test_star_star_name_pattern_excludes_file_at_any_depth(tmp_path: Path) -> None:
    root = tmp_path / "app"
    (root / "sub").mkdir(parents=True)
    (root / "session.json").write_text("s", encoding="utf-8")
    (root / "sub" / "session.json").write_text("s", encoding="utf-8")
    (root / "sub" / "keep.toml").write_text("k", encoding="utf-8")

    plan = build_plan([_app(paths=["app"], exclude=["**/session.json"])], home=tmp_path)

    assert [operation.path for operation in plan.operations] == ["app/sub/keep.toml"]


def test_exclude_applies_to_paths_entry_itself(tmp_path: Path) -> None:
    (tmp_path / "session.json").write_text("s", encoding="utf-8")
    (tmp_path / "keep.toml").write_text("k", encoding="utf-8")

    plan = build_plan(
        [_app(paths=["session.json", "keep.toml"], exclude=["**/session.json"])],
        home=tmp_path,
    )

    assert [operation.path for operation in plan.operations] == ["keep.toml"]
    assert [item.path for item in plan.skipped] == []


def test_excluded_and_skipped_are_different(tmp_path: Path) -> None:
    """Lo excluido se descarta en silencio; lo inexistente se registra como omitido."""
    (tmp_path / "run.log").write_text("l", encoding="utf-8")

    plan = build_plan([_app(paths=["run.log", "falta.toml"], exclude=["**/*.log"])], home=tmp_path)

    assert plan.operations == []
    assert [item.path for item in plan.skipped] == ["falta.toml"]


# --- matriz de patrones de exclusión ---


@pytest.mark.parametrize(
    ("rel", "patterns", "expected"),
    [
        # **/X coincide con X en el nivel superior y a cualquier profundidad
        ("run.log", ["**/*.log"], True),
        ("sub/run.log", ["**/*.log"], True),
        ("sub/deep/run.log", ["**/*.log"], True),
        ("CachedData", ["**/CachedData"], True),
        ("sub/CachedData", ["**/CachedData"], True),
        ("session.json", ["**/session.json"], True),
        ("sub/session.json", ["**/session.json"], True),
        # lo excluido por ser directorio también excluye su contenido
        ("CachedData/blob.bin", ["**/CachedData"], True),
        ("sub/CachedData/blob.bin", ["**/CachedData"], True),
        # patrones que NO deben coincidir
        ("config.toml", ["**/*.log"], False),
        ("run.log.bak", ["**/*.log"], False),
        ("sub/run.log.bak", ["**/*.log"], False),
        ("CachedDataX/blob.bin", ["**/CachedData"], False),
        ("XCachedData/blob.bin", ["**/CachedData"], False),
        ("session.json.bak", ["**/session.json"], False),
        ("sub/session.json.bak", ["**/session.json"], False),
        ("sub/sessionXjson", ["**/session.json"], False),
        # varios patrones a la vez
        ("run.log", ["**/*.log", "**/session.json"], True),
        ("session.json", ["**/*.log", "**/session.json"], True),
        ("config.toml", ["**/*.log", "**/session.json"], False),
        # sin patrones no se excluye nada
        ("run.log", [], False),
    ],
)
def test_is_excluded_pattern_matrix(rel: str, patterns: list[str], expected: bool) -> None:
    assert is_excluded(rel, patterns) is expected


# --- varias apps ---


def test_operations_carry_their_app_id(tmp_path: Path) -> None:
    (tmp_path / "a.txt").write_text("a", encoding="utf-8")
    (tmp_path / "b.txt").write_text("b", encoding="utf-8")

    plan = build_plan(
        [
            _app(app_id="alpha", paths=["a.txt"]),
            _app(app_id="beta", paths=["b.txt", "falta.txt"]),
        ],
        home=tmp_path,
    )

    assert [(operation.app_id, operation.path) for operation in plan.operations] == [
        ("alpha", "a.txt"),
        ("beta", "b.txt"),
    ]
    assert [item.path for item in plan.skipped] == ["falta.txt"]


def test_build_plan_rejects_catalog_path_outside_home(tmp_path: Path) -> None:
    """Una definición con ruta fuera del home no debería llegar aquí, pero si
    llega, se rechaza con `PathError` en lugar de expandir fuera del home."""
    definition = AppDefinition.model_construct(
        id="foo", name="foo", command="foo", paths=["../fuera.toml"]
    )

    with pytest.raises(PathError):
        build_plan([definition], home=tmp_path)


# --- rutas opcionales, patrones y deduplicación ---


def _definition(**overrides: object) -> AppDefinition:
    data: dict[str, object] = {
        "id": "foo",
        "name": "foo",
        "command": "foo",
        "paths": [],
    }
    data.update(overrides)
    return AppDefinition.model_validate(data)


def test_missing_optional_path_is_not_skipped(tmp_path: Path) -> None:
    (tmp_path / "config.toml").write_text("x", encoding="utf-8")
    app = _definition(paths=["config.toml"], optional_paths=["themes"])

    plan = build_plan([app], home=tmp_path)

    assert [operation.path for operation in plan.operations] == ["config.toml"]
    assert [item.path for item in plan.skipped] == []


def test_missing_required_path_is_still_skipped_with_optional_paths(tmp_path: Path) -> None:
    app = _definition(paths=["falta.toml"], optional_paths=["themes"])

    plan = build_plan([app], home=tmp_path)

    assert plan.operations == []
    assert [item.path for item in plan.skipped] == ["falta.toml"]


def test_optional_pattern_is_expanded_without_following_symlinks(tmp_path: Path) -> None:
    (tmp_path / ".codex").mkdir()
    (tmp_path / ".codex" / "uno.config.toml").write_text("a", encoding="utf-8")
    (tmp_path / ".codex" / "dos.config.toml").write_text("b", encoding="utf-8")
    (tmp_path / ".codex" / "otro.toml").write_text("c", encoding="utf-8")
    app = _definition(optional_patterns=[".codex/*.config.toml"])

    plan = build_plan([app], home=tmp_path)

    assert [operation.path for operation in plan.operations] == [
        ".codex/dos.config.toml",
        ".codex/uno.config.toml",
    ]
    assert [item.path for item in plan.skipped] == []


def test_optional_pattern_parent_symlink_is_skipped(tmp_path: Path) -> None:
    external = tmp_path / "external"
    external.mkdir()
    (external / "uno.config.toml").write_text("a", encoding="utf-8")
    (tmp_path / ".codex").symlink_to(external)
    app = _definition(optional_patterns=[".codex/*.config.toml"])

    plan = build_plan([app], home=tmp_path)

    assert plan.operations == []
    assert [item.path for item in plan.skipped] == [".codex"]


def test_duplicate_paths_are_deduplicated_per_app(tmp_path: Path) -> None:
    (tmp_path / "config.toml").write_text("x", encoding="utf-8")
    app = _definition(
        paths=["config.toml"],
        optional_paths=["config.toml", "./config.toml"],
    )

    plan = build_plan([app], home=tmp_path)

    assert [operation.path for operation in plan.operations] == ["config.toml"]


def test_two_apps_keep_their_own_entry_for_the_same_path(tmp_path: Path) -> None:
    (tmp_path / "config.toml").write_text("x", encoding="utf-8")
    plan = build_plan(
        [
            _definition(id="alpha", command="alpha", paths=["config.toml"]),
            _definition(id="beta", command="beta", paths=["config.toml"]),
        ],
        home=tmp_path,
    )

    assert [(operation.app_id, operation.path) for operation in plan.operations] == [
        ("alpha", "config.toml"),
        ("beta", "config.toml"),
    ]


def test_intermediate_symlink_path_is_skipped_and_not_read(tmp_path: Path) -> None:
    external = tmp_path / "external"
    external.mkdir()
    (external / "config.toml").write_text("secreto", encoding="utf-8")
    (tmp_path / ".config").symlink_to(external)
    app = _definition(paths=[".config/config.toml"])

    plan = build_plan([app], home=tmp_path)

    assert plan.operations == []
    assert [item.path for item in plan.skipped] == [".config/config.toml"]


# --- referencias de Ghostty ---


def _ghostty(**overrides: object) -> AppDefinition:
    data: dict[str, object] = {
        "id": "ghostty",
        "name": "Ghostty",
        "bundle_id": "com.mitchellh.ghostty",
        "paths": [".config/ghostty/config"],
    }
    data.update(overrides)
    return AppDefinition.model_validate(data)


def test_ghostty_follows_nested_relative_config_file(tmp_path: Path) -> None:
    (tmp_path / ".config" / "ghostty" / "sub").mkdir(parents=True)
    (tmp_path / ".config" / "ghostty" / "config").write_text(
        "config-file = sub/uno.conf\n", encoding="utf-8"
    )
    (tmp_path / ".config" / "ghostty" / "sub" / "uno.conf").write_text(
        "config-file = dos.conf\n", encoding="utf-8"
    )
    (tmp_path / ".config" / "ghostty" / "sub" / "dos.conf").write_text(
        "font-size = 13\n", encoding="utf-8"
    )

    plan = build_plan([_ghostty()], home=tmp_path)

    assert [operation.path for operation in plan.operations] == [
        ".config/ghostty/config",
        ".config/ghostty/sub/uno.conf",
        ".config/ghostty/sub/dos.conf",
    ]


def test_ghostty_optional_missing_reference_is_not_skipped(tmp_path: Path) -> None:
    (tmp_path / ".config" / "ghostty").mkdir(parents=True)
    (tmp_path / ".config" / "ghostty" / "config").write_text(
        "config-file = ?no-existe.conf\n", encoding="utf-8"
    )

    plan = build_plan([_ghostty()], home=tmp_path)

    assert [operation.path for operation in plan.operations] == [".config/ghostty/config"]
    assert [item.path for item in plan.skipped] == []


def test_ghostty_cycle_is_reported_once_and_cut(tmp_path: Path) -> None:
    (tmp_path / ".config" / "ghostty").mkdir(parents=True)
    (tmp_path / ".config" / "ghostty" / "config").write_text(
        "config-file = a.conf\n", encoding="utf-8"
    )
    (tmp_path / ".config" / "ghostty" / "a.conf").write_text(
        "config-file = config\n", encoding="utf-8"
    )

    plan = build_plan([_ghostty()], home=tmp_path)

    assert [operation.path for operation in plan.operations] == [
        ".config/ghostty/config",
        ".config/ghostty/a.conf",
    ]
    assert any("circular" in notice for notice in plan.notices)


def test_ghostty_theme_by_name_light_and_dark(tmp_path: Path) -> None:
    themes = tmp_path / ".config" / "ghostty" / "themes"
    themes.mkdir(parents=True)
    (tmp_path / ".config" / "ghostty" / "config").write_text(
        "theme = light:claro,dark:oscuro\n", encoding="utf-8"
    )
    (themes / "claro").write_text("background = #fff\n", encoding="utf-8")
    (themes / "oscuro").write_text("background = #000\n", encoding="utf-8")

    plan = build_plan([_ghostty()], home=tmp_path)

    assert [operation.path for operation in plan.operations] == [
        ".config/ghostty/config",
        ".config/ghostty/themes/claro",
        ".config/ghostty/themes/oscuro",
    ]


def test_ghostty_builtin_theme_is_not_an_omission(tmp_path: Path) -> None:
    (tmp_path / ".config" / "ghostty").mkdir(parents=True)
    (tmp_path / ".config" / "ghostty" / "config").write_text(
        "theme = builtin-name\n", encoding="utf-8"
    )

    plan = build_plan([_ghostty()], home=tmp_path)

    assert [operation.path for operation in plan.operations] == [".config/ghostty/config"]
    assert [item.path for item in plan.skipped] == []


def test_ghostty_external_reference_is_not_read_or_included(tmp_path: Path) -> None:
    home = tmp_path / "home"
    (home / ".config" / "ghostty").mkdir(parents=True)
    external = tmp_path / "external.conf"
    external.write_text("secret = x\n", encoding="utf-8")
    (home / ".config" / "ghostty" / "config").write_text(
        f"config-file = {external}\n", encoding="utf-8"
    )

    plan = build_plan([_ghostty()], home=home)

    assert [operation.path for operation in plan.operations] == [".config/ghostty/config"]
    assert str(external) in [item.path for item in plan.skipped]
    assert any("fuera del home" in notice for notice in plan.notices)


def test_ghostty_sensitive_reference_is_reported_without_reading(tmp_path: Path) -> None:
    (tmp_path / ".config" / "ghostty").mkdir(parents=True)
    (tmp_path / ".config" / "ghostty" / ".env").write_text("TOKEN=x\n", encoding="utf-8")
    (tmp_path / ".config" / "ghostty" / "config").write_text(
        "config-file = .env\n", encoding="utf-8"
    )

    plan = build_plan([_ghostty()], home=tmp_path)

    assert [operation.path for operation in plan.operations] == [".config/ghostty/config"]
    assert ".config/ghostty/.env" in [item.path for item in plan.skipped]
    assert any("excluida por seguridad" in notice for notice in plan.notices)


def test_ghostty_depth_limit_is_reported(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setattr("kamino.plan.MAX_GHOSTTY_DEPTH", 1)
    (tmp_path / ".config" / "ghostty").mkdir(parents=True)
    (tmp_path / ".config" / "ghostty" / "config").write_text(
        "config-file = a.conf\n", encoding="utf-8"
    )
    (tmp_path / ".config" / "ghostty" / "a.conf").write_text(
        "config-file = b.conf\n", encoding="utf-8"
    )
    (tmp_path / ".config" / "ghostty" / "b.conf").write_text("font-size = 14\n", encoding="utf-8")

    plan = build_plan([_ghostty()], home=tmp_path)

    assert any("profundidad" in notice for notice in plan.notices)


def test_ghostty_reference_already_declared_is_deduplicated(tmp_path: Path) -> None:
    (tmp_path / ".config" / "ghostty").mkdir(parents=True)
    (tmp_path / ".config" / "ghostty" / "config").write_text(
        "config-file = extra.conf\n", encoding="utf-8"
    )
    (tmp_path / ".config" / "ghostty" / "extra.conf").write_text("x = 1\n", encoding="utf-8")

    app = _ghostty(optional_paths=[".config/ghostty/extra.conf"])
    plan = build_plan([app], home=tmp_path)

    assert [operation.path for operation in plan.operations].count(
        ".config/ghostty/extra.conf"
    ) == 1


def test_sensitive_names_are_excluded_before_reading(tmp_path: Path) -> None:
    root = tmp_path / ".config" / "foo"
    (root / ".ssh").mkdir(parents=True)
    (root / ".aws").mkdir()
    for name in (".env", ".env.local", ".ssh/config", ".aws/credentials", "hosts.yml"):
        (root / name).write_text("secreto\n", encoding="utf-8")
    (root / "config.toml").write_text("allowed = true\n", encoding="utf-8")
    app = _definition(paths=[".config/foo"], exclude=list(SENSITIVE_PATTERNS))

    plan = build_plan([app], home=tmp_path)

    assert [operation.path for operation in plan.operations] == [".config/foo/config.toml"]


def test_ghostty_four_global_variants_and_xdg(tmp_path: Path, monkeypatch) -> None:
    home = tmp_path / "home"
    for base in (".config/ghostty", "Library/Application Support/com.mitchellh.ghostty"):
        for name in ("config", "config.ghostty"):
            target = home / base / name
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text("font-size = 14\n", encoding="utf-8")
    xdg = home / "xdg"
    for name in ("config", "config.ghostty"):
        target = xdg / "ghostty" / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text("font-size = 15\n", encoding="utf-8")
    (xdg / "ghostty" / "themes").mkdir()
    (xdg / "ghostty" / "themes" / "mi-tema").write_text("background = #fff\n")
    monkeypatch.setenv("XDG_CONFIG_HOME", str(xdg))
    ghostty = load_catalog(user_dir=tmp_path / "no-user-catalog")["ghostty"]

    plan = build_plan([ghostty], home=home)

    paths = {operation.path for operation in plan.operations}
    assert ".config/ghostty/config" in paths
    assert ".config/ghostty/config.ghostty" in paths
    assert "Library/Application Support/com.mitchellh.ghostty/config" in paths
    assert "Library/Application Support/com.mitchellh.ghostty/config.ghostty" in paths
    assert "xdg/ghostty/config" in paths
    assert "xdg/ghostty/config.ghostty" in paths
    assert "xdg/ghostty/themes/mi-tema" in paths


def test_ghostty_historical_required_paths_keep_omission_semantics(tmp_path: Path) -> None:
    home = tmp_path / "home"
    target = home / ".config" / "ghostty" / "config.ghostty"
    target.parent.mkdir(parents=True)
    target.write_text("font-size = 14\n", encoding="utf-8")
    ghostty = load_catalog(user_dir=tmp_path / "no-user-catalog")["ghostty"]

    plan = build_plan([ghostty], home=home)

    assert [operation.path for operation in plan.operations] == [".config/ghostty/config.ghostty"]
    assert ".config/ghostty/config" in [item.path for item in plan.skipped]
    assert "Library/Application Support/com.mitchellh.ghostty/config" in [
        item.path for item in plan.skipped
    ]


def test_ghostty_reference_with_intermediate_symlink_is_not_read(tmp_path: Path) -> None:
    home = tmp_path / "home"
    external = tmp_path / "external"
    home.mkdir()
    external.mkdir()
    (external / "secreto.conf").write_text("secret = x\n", encoding="utf-8")
    (home / ".config" / "ghostty").mkdir(parents=True)
    (home / ".config" / "ghostty" / "config").write_text(
        "config-file = sub/secreto.conf\n", encoding="utf-8"
    )
    (home / ".config" / "ghostty" / "sub").symlink_to(external)

    plan = build_plan([_ghostty()], home=home)

    assert [operation.path for operation in plan.operations] == [".config/ghostty/config"]
    assert ".config/ghostty/sub/secreto.conf" in [item.path for item in plan.skipped]


def test_herdr_env_path_inside_home_is_included(tmp_path: Path, monkeypatch) -> None:
    home = tmp_path / "home"
    home.mkdir()
    (home / ".config" / "herdr").mkdir(parents=True)
    (home / ".config" / "herdr" / "config.toml").write_text("[theme]\n")
    (home / ".config" / "herdr" / "plugins" / "config").mkdir(parents=True)
    custom = home / "custom" / "herdr.toml"
    custom.parent.mkdir()
    custom.write_text("[keys]\n")
    monkeypatch.setenv("HERDR_CONFIG_PATH", str(custom))
    app = _definition(
        id="herdr",
        command="herdr",
        paths=[".config/herdr/config.toml", ".config/herdr/plugins/config"],
        env_paths=["HERDR_CONFIG_PATH"],
    )

    plan = build_plan([app], home=home)

    assert "custom/herdr.toml" in [operation.path for operation in plan.operations]


def test_herdr_external_env_path_is_reported_without_reading(tmp_path: Path, monkeypatch) -> None:
    home = tmp_path / "home"
    home.mkdir()
    (home / ".config" / "herdr").mkdir(parents=True)
    (home / ".config" / "herdr" / "config.toml").write_text("[theme]\n")
    (home / ".config" / "herdr" / "plugins" / "config").mkdir(parents=True)
    external = tmp_path / "external.toml"
    external.write_text("[keys]\n")
    monkeypatch.setenv("HERDR_CONFIG_PATH", str(external))
    app = _definition(
        id="herdr",
        command="herdr",
        paths=[".config/herdr/config.toml", ".config/herdr/plugins/config"],
        env_paths=["HERDR_CONFIG_PATH"],
    )

    plan = build_plan([app], home=home)

    assert "custom" not in " ".join(operation.path for operation in plan.operations)
    assert "HERDR_CONFIG_PATH fuera del home" in [item.path for item in plan.skipped]


def test_herdr_xdg_path_inside_and_outside_home(tmp_path: Path, monkeypatch) -> None:
    home = tmp_path / "home"
    home.mkdir()
    (home / ".config" / "herdr").mkdir(parents=True)
    (home / ".config" / "herdr" / "config.toml").write_text("[theme]\n")
    (home / ".config" / "herdr" / "plugins" / "config").mkdir(parents=True)
    xdg = home / "xdg"
    target = xdg / "herdr" / "config.toml"
    target.parent.mkdir(parents=True)
    target.write_text("[theme]\n")
    monkeypatch.setenv("XDG_CONFIG_HOME", str(xdg))
    app = _definition(
        id="herdr",
        command="herdr",
        paths=[".config/herdr/config.toml", ".config/herdr/plugins/config"],
        xdg_paths=["herdr/config.toml"],
    )

    plan = build_plan([app], home=home)
    assert "xdg/herdr/config.toml" in [operation.path for operation in plan.operations]

    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "external-xdg"))
    plan = build_plan([app], home=home)
    assert "XDG_CONFIG_HOME fuera del home" in [item.path for item in plan.skipped]


def test_walk_unreadable_dir_recorded(tmp_path: Path) -> None:
    home = tmp_path / "home"
    app_dir = home / ".config" / "app"
    app_dir.mkdir(parents=True)
    (app_dir / "ok.txt").write_text("x")
    secret = app_dir / "secret"
    secret.mkdir()
    secret.chmod(0o000)
    app = _app(paths=[".config/app"])
    try:
        try:
            next(secret.iterdir(), None)
        except OSError:
            pass
        else:
            pytest.skip("el entorno permite listar directorios sin permiso")
        plan = build_plan([app], home=home)
        assert ".config/app/secret" in [item.path for item in plan.skipped]
        assert ".config/app/ok.txt" in [op.path for op in plan.operations]
    finally:
        secret.chmod(0o755)


# --- recuento para C1 ---


def test_count_files_matches_plan_with_expansion_and_exclusions(tmp_path: Path) -> None:
    home = tmp_path / "home"
    (home / ".config/foo").mkdir(parents=True)
    (home / ".config/foo/a.toml").write_text("a\n", encoding="utf-8")
    (home / ".config/foo/b.toml").write_text("b\n", encoding="utf-8")
    (home / ".config/foo/.env").write_text("secreto\n", encoding="utf-8")
    (home / "enlace.toml").symlink_to(".config/foo/a.toml")
    app = _app(paths=[".config/foo", "falta.toml", "enlace.toml"], exclude=["**/.env"])

    plan = build_plan([app], home=home)
    counted = count_files([app], home=home)

    assert counted.counts["foo"] == len(plan.operations) == 3
    assert [item.path for item in counted.skipped] == ["falta.toml"]


def test_count_files_parity_for_ghostty_references(tmp_path: Path) -> None:
    home = tmp_path / "home"
    ghostty = home / ".config/ghostty"
    (ghostty / "themes").mkdir(parents=True)
    (ghostty / "config").write_text("config-file = extra.conf\ntheme = mi-tema\n", encoding="utf-8")
    (ghostty / "extra.conf").write_text("font-size = 12\n", encoding="utf-8")
    (ghostty / "themes" / "mi-tema").write_text("background = 000000\n", encoding="utf-8")
    app = AppDefinition.model_validate(
        {
            "id": "ghostty",
            "name": "Ghostty",
            "command": "ghostty",
            "paths": [".config/ghostty/config"],
        }
    )

    plan = build_plan([app], home=home)
    counted = count_files([app], home=home)

    assert counted.counts["ghostty"] == len(plan.operations)
    assert {operation.path for operation in plan.operations} == {
        ".config/ghostty/config",
        ".config/ghostty/extra.conf",
        ".config/ghostty/themes/mi-tema",
    }


def test_count_files_does_not_hash_regular_files(tmp_path: Path, monkeypatch) -> None:
    home = tmp_path / "home"
    (home / ".config/foo").mkdir(parents=True)
    (home / ".config/foo/a.toml").write_text("a\n", encoding="utf-8")
    app = _app(paths=[".config/foo"])
    called: list[str] = []
    from kamino.paths import SafeHome

    original = SafeHome.read_bytes

    def read_bytes(self, rel):
        called.append(str(rel))
        return original(self, rel)

    monkeypatch.setattr(SafeHome, "read_bytes", read_bytes)

    counted = count_files([app], home=home)

    assert counted.counts["foo"] == 1
    assert called == []
