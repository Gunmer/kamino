"""Pruebas del paso 4: archive.py (tar.gz, manifiesto y checksums)."""

from __future__ import annotations

import hashlib
import io
import os
import tarfile
from pathlib import Path

import pytest
from pydantic import ValidationError

from kamino.archive import (
    FORMAT_VERSION,
    AppEntry,
    FileEntry,
    Manifest,
    extract_archive,
    read_manifest,
    verify_archive,
    write_archive,
)
from kamino.catalog.models import AppDefinition
from kamino.errors import ArchiveError, IntegrityError, VersionError
from kamino.plan import build_plan


def _sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def _app_with_files(root: Path, app_id: str, files: dict[str, bytes]) -> dict:
    """Crea los archivos en `root` y devuelve la entrada de app del manifiesto."""
    entries = []
    for rel, content in files.items():
        path = root / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(content)
        entries.append(
            {
                "path": rel,
                "sha256": _sha256(content),
                "size": len(content),
                "mode": "0644",
            }
        )
    return {"id": app_id, "name": app_id, "files": entries}


def _manifest(apps: list[dict], **overrides) -> Manifest:
    data = {
        "format_version": FORMAT_VERSION,
        "tool_version": "0.1.0",
        "created_at": "2026-10-28T21:30:00Z",
        "macos_version": "15.1",
        "arch": "arm64",
        "hostname": "mac-origen",
        "apps": apps,
    }
    data.update(overrides)
    return Manifest.model_validate(data)


def _raw_archive(path: Path, members: list[tuple[str, bytes | str, int]]) -> None:
    """Escribe un tar.gz con miembros arbitrarios (para pruebas de seguridad)."""
    with tarfile.open(path, "w:gz") as tar:
        for name, content, typ in members:
            info = tarfile.TarInfo(name)
            info.type = typ
            if typ == tarfile.REGTYPE:
                info.size = len(content)
                tar.addfile(info, io.BytesIO(content))
            elif typ == tarfile.SYMTYPE:
                info.linkname = content if isinstance(content, str) else content.decode()
                tar.addfile(info)


# --- escritura y lectura ---


def test_roundtrip(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    apps = [
        _app_with_files(
            source_root,
            "git",
            {
                ".gitconfig": b"[user]\nname=Cristiam\n",
                ".config/git/ignore": b"*.log\n",
            },
        )
    ]
    manifest = _manifest(apps)
    output = tmp_path / "backup.tar.gz"

    write_archive(output, manifest, source_root=source_root)

    assert output.exists()
    assert read_manifest(output) == manifest
    verify_archive(output, manifest)

    with tarfile.open(output, "r:gz") as tar:
        names = sorted(m.name for m in tar.getmembers())
    assert names == [
        "apps/git/.config/git/ignore",
        "apps/git/.gitconfig",
        "manifest.json",
    ]


def test_write_refuses_overwrite(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    manifest = _manifest([_app_with_files(source_root, "git", {".gitconfig": b"x"})])
    output = tmp_path / "b.tar.gz"
    write_archive(output, manifest, source_root=source_root)

    with pytest.raises(ArchiveError):
        write_archive(output, manifest, source_root=source_root)
    write_archive(output, manifest, source_root=source_root, overwrite=True)


def test_write_preserves_symlink(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    (source_root / ".config").mkdir(parents=True)
    target = source_root / ".config" / "real.toml"
    target.write_text("x=1\n")
    link = source_root / "link.toml"
    link.symlink_to(target.name)

    manifest = _manifest(
        [
            {
                "id": "app",
                "name": "app",
                "files": [{"path": "link.toml", "link": "real.toml", "size": 0, "mode": "0644"}],
            }
        ]
    )
    output = tmp_path / "b.tar.gz"
    write_archive(output, manifest, source_root=source_root)

    with tarfile.open(output, "r:gz") as tar:
        link_member = next(m for m in tar.getmembers() if m.name.endswith("link.toml"))
    assert link_member.issym()
    assert link_member.linkname == "real.toml"


def _symlink_manifest(root: Path, link_target: str) -> Manifest:
    root.mkdir(parents=True, exist_ok=True)
    link = root / "link.toml"
    if link.exists() or link.is_symlink():
        link.unlink()
    link.symlink_to(link_target)
    return _manifest(
        [
            {
                "id": "app",
                "name": "app",
                "files": [{"path": "link.toml", "link": link_target, "size": 0, "mode": "0644"}],
            }
        ]
    )


def test_verify_symlink_ok(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    manifest = _symlink_manifest(source_root, "real.toml")
    (source_root / "real.toml").write_text("x=1\n")
    output = tmp_path / "b.tar.gz"
    write_archive(output, manifest, source_root=source_root)
    verify_archive(output, manifest)


def test_verify_symlink_target_mismatch(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    manifest = _symlink_manifest(source_root, "real.toml")
    (source_root / "real.toml").write_text("x=1\n")
    output = tmp_path / "b.tar.gz"
    write_archive(output, manifest, source_root=source_root)
    manifest.apps[0].files[0].link = "otro.toml"
    with pytest.raises(IntegrityError):
        verify_archive(output, manifest)


def test_extract_symlink_ok(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    manifest = _symlink_manifest(source_root, "real.toml")
    (source_root / "real.toml").write_text("x=1\n")
    output = tmp_path / "b.tar.gz"
    write_archive(output, manifest, source_root=source_root)
    dest = tmp_path / "out"
    extract_archive(output, dest)
    link_path = dest / "apps" / "app" / "link.toml"
    assert link_path.is_symlink()
    assert os.readlink(link_path) == "real.toml"


def test_write_rejects_symlink_target_mismatch(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    source_root.mkdir(parents=True, exist_ok=True)
    (source_root / "link.toml").symlink_to("/etc/passwd")  # destino absoluto en disco
    manifest = _manifest(
        [
            {
                "id": "app",
                "name": "app",
                "files": [{"path": "link.toml", "link": "real.toml", "size": 0, "mode": "0644"}],
            }
        ]
    )
    output = tmp_path / "b.tar.gz"
    with pytest.raises(ArchiveError):
        write_archive(output, manifest, source_root=source_root)


def test_file_entry_link_must_be_relative() -> None:
    with pytest.raises(ValidationError):
        FileEntry(path="link", link="../fuera")
    with pytest.raises(ValidationError):
        FileEntry(path="link", link="/etc/passwd")


# --- lectura del manifiesto ---


def test_read_manifest_rejects_newer_format(tmp_path: Path) -> None:
    manifest_json = (
        '{"format_version": 99, "tool_version": "0.1.0", "created_at": "x",'
        ' "macos_version": "15", "arch": "arm64", "hostname": "h", "apps": []}'
    )
    output = tmp_path / "b.tar.gz"
    with tarfile.open(output, "w:gz") as tar:
        info = tarfile.TarInfo("manifest.json")
        data = manifest_json.encode()
        info.size = len(data)
        tar.addfile(info, io.BytesIO(data))

    with pytest.raises(VersionError):
        read_manifest(output)


def test_read_manifest_missing(tmp_path: Path) -> None:
    output = tmp_path / "b.tar.gz"
    with tarfile.open(output, "w:gz") as tar:
        info = tarfile.TarInfo("otro.txt")
        info.size = 0
        tar.addfile(info, io.BytesIO(b""))

    with pytest.raises(ArchiveError):
        read_manifest(output)


def test_read_manifest_invalid_json(tmp_path: Path) -> None:
    output = tmp_path / "b.tar.gz"
    with tarfile.open(output, "w:gz") as tar:
        info = tarfile.TarInfo("manifest.json")
        data = b"esto no es json"
        info.size = len(data)
        tar.addfile(info, io.BytesIO(data))

    with pytest.raises(ArchiveError):
        read_manifest(output)


# --- verificación de checksums ---


def test_verify_ok(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    manifest = _manifest([_app_with_files(source_root, "git", {".gitconfig": b"contenido"})])
    output = tmp_path / "b.tar.gz"
    write_archive(output, manifest, source_root=source_root)
    verify_archive(output, manifest)


def test_verify_checksum_mismatch(tmp_path: Path) -> None:
    app = {
        "id": "git",
        "name": "git",
        "files": [
            {
                "path": ".gitconfig",
                "sha256": _sha256(b"contenido esperado"),
                "size": 18,
                "mode": "0644",
            }
        ],
    }
    manifest = _manifest([app])
    output = tmp_path / "b.tar.gz"
    with tarfile.open(output, "w:gz") as tar:
        info = tarfile.TarInfo("manifest.json")
        data = manifest.model_dump_json().encode()
        info.size = len(data)
        tar.addfile(info, io.BytesIO(data))
        minfo = tarfile.TarInfo("apps/git/.gitconfig")
        minfo.size = len(b"contenido real")
        tar.addfile(minfo, io.BytesIO(b"contenido real"))

    with pytest.raises(IntegrityError):
        verify_archive(output, manifest)


def test_write_archive_rejects_changed_file(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    app = _app_with_files(source_root, "git", {".gitconfig": b"contenido real"})
    app["files"][0]["sha256"] = _sha256(b"contenido distinto")
    manifest = _manifest([app])
    output = tmp_path / "b.tar.gz"
    with pytest.raises(ArchiveError):
        write_archive(output, manifest, source_root=source_root)


def test_verify_missing_file(tmp_path: Path) -> None:
    app = {
        "id": "git",
        "name": "git",
        "files": [
            {"path": ".gitconfig", "sha256": _sha256(b"contenido"), "size": 9, "mode": "0644"},
            {"path": ".config/git/ignore", "sha256": _sha256(b"x"), "size": 1, "mode": "0644"},
        ],
    }
    manifest = _manifest([app])
    output = tmp_path / "b.tar.gz"
    with tarfile.open(output, "w:gz") as tar:
        info = tarfile.TarInfo("manifest.json")
        data = manifest.model_dump_json().encode()
        info.size = len(data)
        tar.addfile(info, io.BytesIO(data))
        # solo .gitconfig presente; falta .config/git/ignore
        minfo = tarfile.TarInfo("apps/git/.gitconfig")
        minfo.size = len(b"contenido")
        tar.addfile(minfo, io.BytesIO(b"contenido"))

    with pytest.raises(IntegrityError):
        verify_archive(output, manifest)


def test_verify_unexpected_file(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    manifest = _manifest([_app_with_files(source_root, "git", {".gitconfig": b"contenido"})])
    output = tmp_path / "b.tar.gz"
    write_archive(output, manifest, source_root=source_root)

    # Reconstruir un archivo con un miembro inesperado.
    extra = tmp_path / "b2.tar.gz"
    with tarfile.open(extra, "w:gz") as tar:
        info = tarfile.TarInfo("manifest.json")
        data = manifest.model_dump_json().encode()
        info.size = len(data)
        tar.addfile(info, io.BytesIO(data))
        for name, content in [("apps/git/.gitconfig", b"contenido"), ("apps/git/extra", b"x")]:
            minfo = tarfile.TarInfo(name)
            minfo.size = len(content)
            tar.addfile(minfo, io.BytesIO(content))

    with pytest.raises(IntegrityError):
        verify_archive(extra, manifest)


# --- extracción segura ---


def test_extract_ok(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    manifest = _manifest([_app_with_files(source_root, "git", {".gitconfig": b"contenido"})])
    output = tmp_path / "b.tar.gz"
    write_archive(output, manifest, source_root=source_root)

    dest = tmp_path / "out"
    extract_archive(output, dest)
    assert (dest / "apps" / "git" / ".gitconfig").read_bytes() == b"contenido"


def test_extract_rejects_dotdot(tmp_path: Path) -> None:
    output = tmp_path / "b.tar.gz"
    _raw_archive(output, [("../evil", b"x", tarfile.REGTYPE)])

    with pytest.raises(ArchiveError):
        extract_archive(output, tmp_path / "out")


def test_extract_rejects_absolute(tmp_path: Path) -> None:
    output = tmp_path / "b.tar.gz"
    _raw_archive(output, [("/etc/evil", b"x", tarfile.REGTYPE)])

    with pytest.raises(ArchiveError):
        extract_archive(output, tmp_path / "out")


def test_extract_rejects_malicious_symlink(tmp_path: Path) -> None:
    output = tmp_path / "b.tar.gz"
    _raw_archive(output, [("link", "/etc/passwd", tarfile.SYMTYPE)])

    with pytest.raises(ArchiveError):
        extract_archive(output, tmp_path / "out")


# --- validación del modelo ---


def test_file_entry_rejects_bad_sha256() -> None:
    with pytest.raises(ValidationError):
        FileEntry(path=".gitconfig", sha256="no-es-hex", size=1, mode="0644")


def test_file_entry_rejects_absolute_path() -> None:
    with pytest.raises(ValidationError):
        FileEntry(path="/etc/passwd", sha256=_sha256(b"x"), size=1, mode="0644")


def test_file_entry_rejects_dotdot_path() -> None:
    with pytest.raises(ValidationError):
        FileEntry(path="../.ssh", sha256=_sha256(b"x"), size=1, mode="0644")


def test_write_archive_rejects_intermediate_symlink_without_reading(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    external = tmp_path / "external"
    source_root.mkdir()
    external.mkdir()
    (external / "config.toml").write_text("secreto\n", encoding="utf-8")
    (source_root / ".config").symlink_to(external)
    manifest = _manifest(
        [
            {
                "id": "app",
                "name": "app",
                "files": [
                    {
                        "path": ".config/config.toml",
                        "sha256": _sha256(b"secreto\n"),
                        "size": 8,
                        "mode": "0644",
                    }
                ],
            }
        ]
    )
    output = tmp_path / "b.tar.gz"

    with pytest.raises(ArchiveError):
        write_archive(output, manifest, source_root=source_root)

    assert not output.exists()


def test_write_archive_preserves_allowed_config_with_embedded_secret(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    content = b'api_key = "secreto-incrustado"\n'
    manifest = _manifest([_app_with_files(source_root, "app", {"config.toml": content})])
    output = tmp_path / "b.tar.gz"

    write_archive(output, manifest, source_root=source_root)

    with tarfile.open(output, "r:gz") as tar:
        member = tar.extractfile("apps/app/config.toml")
        assert member is not None
        assert member.read() == content


def test_read_file_from_archive_is_bounded(tmp_path: Path) -> None:
    from kamino.archive import read_file_from_archive

    source_root = tmp_path / "home"
    manifest = _manifest([_app_with_files(source_root, "app", {"config.toml": b"1234"})])
    output = tmp_path / "b.tar.gz"
    write_archive(output, manifest, source_root=source_root)

    assert read_file_from_archive(output, "app", "config.toml", limit=4) == b"1234"
    assert read_file_from_archive(output, "app", "config.toml", limit=3) is None
    assert read_file_from_archive(output, "app", "falta", limit=4) is None


def test_deduplicated_plan_writes_one_verifiable_member(tmp_path: Path) -> None:
    source_root = tmp_path / "home"
    (source_root / ".config" / "app").mkdir(parents=True)
    content = b"config\n"
    (source_root / ".config" / "app" / "config.toml").write_bytes(content)
    app = AppDefinition.model_validate(
        {
            "id": "app",
            "name": "app",
            "command": "app",
            "paths": [".config/app", ".config/app/config.toml"],
        }
    )
    plan = build_plan([app], home=source_root)
    (operation,) = plan.operations
    manifest = _manifest(
        [
            {
                "id": "app",
                "name": "app",
                "files": [
                    {
                        "path": operation.path,
                        "sha256": operation.sha256,
                        "size": operation.size,
                        "mode": operation.mode,
                    }
                ],
            }
        ]
    )
    output = tmp_path / "b.tar.gz"

    write_archive(output, manifest, source_root=source_root)
    verify_archive(output, manifest)

    with tarfile.open(output, "r:gz") as tar:
        names = [member.name for member in tar.getmembers()]
    assert names.count("apps/app/.config/app/config.toml") == 1


def test_app_entry_and_manifest_models() -> None:
    app = AppEntry(id="git", name="Git", files=[])
    manifest = _manifest([app.model_dump()])
    assert manifest.format_version == FORMAT_VERSION
    assert manifest.apps[0].id == "git"
