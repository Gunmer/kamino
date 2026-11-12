"""Pruebas de `paths.py`: normalización relativa al home y validación.

Cubre la misma semántica que `_ensure_relative` en `catalog/models.py`
(diseño técnico §1 y §4.1): todo se guarda relativo a `~` y las rutas fuera
del home no se admiten en la v1. Los tests usan un home falso (`tmp_path`)
y no tocan el home real (diseño técnico §7).
"""

from __future__ import annotations

from pathlib import Path

import pytest

from kamino.errors import PathError
from kamino.paths import SafeHome, to_absolute, to_relative

# --- to_absolute ---


def test_to_absolute_resolves_relative_path(tmp_path: Path) -> None:
    assert to_absolute(".config/foo/config.toml", home=tmp_path) == (
        tmp_path / ".config" / "foo" / "config.toml"
    )


def test_to_absolute_accepts_path_objects(tmp_path: Path) -> None:
    assert to_absolute(Path("archivo.txt"), home=tmp_path) == tmp_path / "archivo.txt"


def test_to_absolute_default_home_uses_environment(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setenv("HOME", str(tmp_path))
    assert to_absolute("archivo.txt") == tmp_path / "archivo.txt"


def test_to_absolute_rejects_empty_path(tmp_path: Path) -> None:
    with pytest.raises(PathError):
        to_absolute("", home=tmp_path)


def test_to_absolute_rejects_absolute_path(tmp_path: Path) -> None:
    with pytest.raises(PathError):
        to_absolute("/etc/hosts", home=tmp_path)


def test_to_absolute_rejects_parent_escape(tmp_path: Path) -> None:
    with pytest.raises(PathError):
        to_absolute("../fuera.txt", home=tmp_path)
    with pytest.raises(PathError):
        to_absolute(".config/../../fuera.txt", home=tmp_path)


def test_to_absolute_rejects_tilde(tmp_path: Path) -> None:
    with pytest.raises(PathError):
        to_absolute("~/archivo.txt", home=tmp_path)
    with pytest.raises(PathError):
        to_absolute("~", home=tmp_path)


# --- to_relative ---


def test_to_relative_returns_relative_form(tmp_path: Path) -> None:
    assert to_relative(tmp_path / ".config" / "foo" / "config.toml", home=tmp_path) == Path(
        ".config/foo/config.toml"
    )


def test_to_relative_accepts_string_paths(tmp_path: Path) -> None:
    assert to_relative(str(tmp_path / "archivo.txt"), home=tmp_path) == Path("archivo.txt")


def test_to_relative_of_home_itself_is_dot(tmp_path: Path) -> None:
    assert to_relative(tmp_path, home=tmp_path) == Path(".")


def test_to_relative_rejects_relative_path(tmp_path: Path) -> None:
    with pytest.raises(PathError):
        to_relative("archivo.txt", home=tmp_path)


def test_to_relative_rejects_path_outside_home(tmp_path: Path) -> None:
    home = tmp_path / "home"
    with pytest.raises(PathError):
        to_relative(tmp_path / "otro" / "archivo.txt", home=home)


def test_to_relative_rejects_sibling_prefix_of_home(tmp_path: Path) -> None:
    """`/home/usuario2` no cae dentro de `/home/usuario` por prefijo de texto."""
    home = tmp_path / "usuario"
    with pytest.raises(PathError):
        to_relative(tmp_path / "usuario2" / "archivo.txt", home=home)


def test_to_relative_rejects_parent_escape_disguised(tmp_path: Path) -> None:
    """Una ruta con `..` que se sale del home se rechaza aunque el prefijo coincida."""
    with pytest.raises(PathError):
        to_relative(tmp_path / "sub" / ".." / ".." / "fuera.txt", home=tmp_path)


def test_to_relative_normalizes_inner_dotdot(tmp_path: Path) -> None:
    """`home/a/../b` sigue dentro del home y se normaliza a `b`."""
    assert to_relative(tmp_path / "a" / ".." / "b", home=tmp_path) == Path("b")


# --- acceso seguro anclado al home ---


def test_safe_home_reads_normal_file(tmp_path: Path) -> None:
    (tmp_path / ".config" / "foo").mkdir(parents=True)
    (tmp_path / ".config" / "foo" / "config.toml").write_text("a = 1\n", encoding="utf-8")

    with SafeHome(tmp_path) as safe:
        assert safe.read_bytes(".config/foo/config.toml") == b"a = 1\n"


def test_safe_home_normalizes_dot_components(tmp_path: Path) -> None:
    (tmp_path / "a").mkdir()
    (tmp_path / "a" / "b.txt").write_text("b", encoding="utf-8")

    with SafeHome(tmp_path) as safe:
        assert safe.read_bytes("./a/./b.txt") == b"b"


@pytest.mark.parametrize("path", ["", "/etc/passwd", "../fuera.txt", "a/../../fuera.txt", "~/x"])
def test_safe_home_rejects_escaping_paths(tmp_path: Path, path: str) -> None:
    with SafeHome(tmp_path) as safe, pytest.raises(PathError):
        safe.read_bytes(path)


def test_safe_home_rejects_intermediate_symlink_without_reading(tmp_path: Path) -> None:
    home = tmp_path / "home"
    external = tmp_path / "external"
    home.mkdir()
    external.mkdir()
    (external / "secret.txt").write_text("secreto", encoding="utf-8")
    (home / ".config").symlink_to(external)

    with SafeHome(home) as safe:
        with pytest.raises(PathError):
            safe.read_bytes(".config/secret.txt")
        with pytest.raises(PathError):
            safe.lstat(".config/secret.txt")


def test_safe_home_preserves_final_symlink_without_following(tmp_path: Path) -> None:
    (tmp_path / "real.txt").write_text("contenido", encoding="utf-8")
    (tmp_path / "enlace.txt").symlink_to("real.txt")

    with SafeHome(tmp_path) as safe:
        assert safe.lstat("enlace.txt").st_mode & 0o170000 == 0o120000
        assert safe.readlink("enlace.txt") == "real.txt"
        with pytest.raises(PathError):
            safe.read_bytes("enlace.txt")


def test_safe_home_rejects_intermediate_symlink_on_write(tmp_path: Path) -> None:
    home = tmp_path / "home"
    external = tmp_path / "external"
    home.mkdir()
    external.mkdir()
    (home / ".config").symlink_to(external)

    with SafeHome(home) as safe, pytest.raises(PathError):
        safe.write_file(".config/nuevo.txt", b"contenido")

    assert not (external / "nuevo.txt").exists()


def test_safe_home_write_replaces_atomically_and_preserves_mode(tmp_path: Path) -> None:
    with SafeHome(tmp_path) as safe:
        safe.write_file(".config/foo/config.toml", b"nuevo", mode=0o600)

    target = tmp_path / ".config" / "foo" / "config.toml"
    assert target.read_bytes() == b"nuevo"
    assert target.stat().st_mode & 0o777 == 0o600
