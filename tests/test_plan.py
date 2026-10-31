"""Pruebas de `plan.py`: expansión de rutas, exclusiones y operaciones.

Cubre el paso 4 de `backup` (diseño técnico §5.1): expandir `paths` de cada
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

from kamino.catalog.models import AppDefinition
from kamino.errors import PathError
from kamino.plan import Operation, Plan, build_plan, is_excluded


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
    assert plan.skipped == []
    with pytest.raises(AttributeError):
        plan.skipped = ["x"]  # type: ignore[misc]


# --- expansión de un archivo ---


def test_single_file_operation(tmp_path: Path) -> None:
    content = b"[user]\nname=Test\n"
    source = tmp_path / ".gitconfig"
    source.write_bytes(content)
    source.chmod(0o644)  # el modo depende del umask: se fija para la prueba

    plan = build_plan([_app(paths=[".gitconfig"])], home=tmp_path)

    assert plan.skipped == []
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
    assert plan.skipped == [".config/foo/no-existe.toml"]


def test_missing_and_existing_paths_mix(tmp_path: Path) -> None:
    (tmp_path / "existe.txt").write_text("hola", encoding="utf-8")

    plan = build_plan([_app(paths=["falta.txt", "existe.txt"])], home=tmp_path)

    assert [operation.path for operation in plan.operations] == ["existe.txt"]
    assert plan.skipped == ["falta.txt"]


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
    assert plan.skipped == []
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
    assert plan.skipped == []


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
    assert plan.skipped == ["enlace.txt"]


def test_symlink_with_parent_escape_is_skipped(tmp_path: Path) -> None:
    (tmp_path / "enlace.txt").symlink_to("../fuera.txt")

    plan = build_plan([_app(paths=["enlace.txt"])], home=tmp_path)

    assert plan.operations == []
    assert plan.skipped == ["enlace.txt"]


def test_broken_symlink_still_stored_as_link(tmp_path: Path) -> None:
    """Un enlace roto existe como enlace: se guarda sin seguirlo ni fallar."""
    (tmp_path / "enlace.txt").symlink_to("no-existe.txt")

    plan = build_plan([_app(paths=["enlace.txt"])], home=tmp_path)

    assert plan.operations[0].link == "no-existe.txt"
    assert plan.skipped == []


def test_symlinked_directory_inside_walk_is_a_link(tmp_path: Path) -> None:
    (tmp_path / "real-dir").mkdir()
    (tmp_path / "real-dir" / "x.txt").write_text("x", encoding="utf-8")
    (tmp_path / "dir").mkdir()
    (tmp_path / "dir" / "enlace-dir").symlink_to("../real-dir")

    plan = build_plan([_app(paths=["dir"])], home=tmp_path)

    assert plan.operations == []
    assert plan.skipped == ["dir/enlace-dir"]


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
    assert plan.skipped == []


def test_excluded_and_skipped_are_different(tmp_path: Path) -> None:
    """Lo excluido se descarta en silencio; lo inexistente se registra como omitido."""
    (tmp_path / "run.log").write_text("l", encoding="utf-8")

    plan = build_plan([_app(paths=["run.log", "falta.toml"], exclude=["**/*.log"])], home=tmp_path)

    assert plan.operations == []
    assert plan.skipped == ["falta.toml"]


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
    assert plan.skipped == ["falta.txt"]


def test_build_plan_rejects_catalog_path_outside_home(tmp_path: Path) -> None:
    """Una definición con ruta fuera del home no debería llegar aquí, pero si
    llega, se rechaza con `PathError` en lugar de expandir fuera del home."""
    definition = AppDefinition.model_construct(
        id="foo", name="foo", command="foo", paths=["../fuera.toml"]
    )

    with pytest.raises(PathError):
        build_plan([definition], home=tmp_path)


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
        assert ".config/app/secret" in plan.skipped
        assert ".config/app/ok.txt" in [op.path for op in plan.operations]
    finally:
        secret.chmod(0o755)
