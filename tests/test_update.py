"""Pruebas de `update.py`: consulta, descarga y reemplazo sin red."""

from __future__ import annotations

import hashlib
import io
import json
import urllib.error
from pathlib import Path

import pytest

from kamino import update
from kamino.errors import UpdateError

API_URL = update.API_URL
ASSET_URL = "https://example.test/kamino-arm64"
NEW_BINARY = b"nuevo binario"
_UNSET = object()

_ORIGINAL_ENSURE_PLATFORM = update._ensure_supported_platform


@pytest.fixture(autouse=True)
def _allow_platform(monkeypatch) -> None:
    monkeypatch.setattr(update, "_ensure_supported_platform", lambda: None)


def _payload(
    *,
    tag: str = "v0.5.0",
    body: bytes = NEW_BINARY,
    asset_name: str = update.ASSET_NAME,
    asset_url: str = ASSET_URL,
    size: object = _UNSET,
    digest: object = _UNSET,
) -> dict:
    return {
        "tag_name": tag,
        "assets": [
            {
                "name": asset_name,
                "browser_download_url": asset_url,
                "size": len(body) if size is _UNSET else size,
                "digest": (
                    "sha256:" + hashlib.sha256(body).hexdigest() if digest is _UNSET else digest
                ),
            }
        ],
    }


def _install_fake_http(
    monkeypatch,
    *,
    payload: dict,
    asset_body: bytes = NEW_BINARY,
    requested: list[str] | None = None,
) -> None:
    api_body = json.dumps(payload).encode()

    def fake_open(request, *, timeout=update.TIMEOUT_SECONDS):
        url = request.full_url
        if requested is not None:
            requested.append(url)
        if url == API_URL:
            return io.BytesIO(api_body)
        if url == ASSET_URL:
            return io.BytesIO(asset_body)
        raise AssertionError(f"URL inesperada: {url}")

    monkeypatch.setattr(update, "_open_url", fake_open)


def _release(
    *,
    body: bytes = NEW_BINARY,
    size: int | None = None,
    digest: str | None = None,
) -> update.ReleaseInfo:
    return update.ReleaseInfo(
        version="0.5.0",
        asset_url=ASSET_URL,
        asset_size=len(body) if size is None else size,
        asset_digest=("sha256:" + hashlib.sha256(body).hexdigest() if digest is None else digest),
    )


def test_parse_version_accepts_prefix_and_spaces() -> None:
    assert update.parse_version("0.5.0") == (0, 5, 0)
    assert update.parse_version("v0.5.0") == (0, 5, 0)
    assert update.parse_version(" 1.2.3 ") == (1, 2, 3)


@pytest.mark.parametrize("value", ["", "v1.2", "1.2.3.4", "v1.2.x", "1.2.3-beta"])
def test_parse_version_rejects_invalid_formats(value: str) -> None:
    with pytest.raises(UpdateError, match="versión no válida"):
        update.parse_version(value)


def test_version_comparison_is_numeric() -> None:
    assert update.parse_version("0.10.0") > update.parse_version("0.9.0")


def test_normalize_version_removes_prefix() -> None:
    assert update.normalize_version("v0.5.0") == "0.5.0"


def test_latest_release_reads_the_expected_asset(monkeypatch) -> None:
    _install_fake_http(monkeypatch, payload=_payload())

    release = update.latest_release()

    assert release.version == "0.5.0"
    assert release.asset_url == ASSET_URL
    assert release.asset_size == len(NEW_BINARY)
    assert release.asset_digest == "sha256:" + hashlib.sha256(NEW_BINARY).hexdigest()


def test_latest_release_rejects_missing_asset(monkeypatch) -> None:
    _install_fake_http(monkeypatch, payload=_payload(asset_name="otro-binario"))

    with pytest.raises(UpdateError, match="no incluye el asset"):
        update.latest_release()


def test_latest_release_rejects_missing_digest(monkeypatch) -> None:
    _install_fake_http(monkeypatch, payload=_payload(digest=None))

    with pytest.raises(UpdateError, match="SHA-256 válido"):
        update.latest_release()


def test_latest_release_rejects_invalid_json(monkeypatch) -> None:
    monkeypatch.setattr(update, "_open_url", lambda request, **kwargs: io.BytesIO(b"no json"))

    with pytest.raises(UpdateError, match="JSON válido"):
        update.latest_release()


def test_latest_release_reports_rate_limit(monkeypatch) -> None:
    def limited(request, **kwargs):
        raise urllib.error.HTTPError(API_URL, 403, "rate limited", None, None)

    monkeypatch.setattr(update, "_open_url", limited)

    with pytest.raises(UpdateError, match="limitado"):
        update.latest_release()


def test_latest_release_rejects_invalid_tag(monkeypatch) -> None:
    _install_fake_http(monkeypatch, payload=_payload(tag="main"))

    with pytest.raises(UpdateError, match="versión válida"):
        update.latest_release()


def test_latest_release_rejects_insecure_asset_url(monkeypatch) -> None:
    _install_fake_http(
        monkeypatch,
        payload=_payload(asset_url="http://example.test/kamino-arm64"),
    )

    with pytest.raises(UpdateError, match="URL de descarga"):
        update.latest_release()


def test_download_asset_writes_verified_bytes(monkeypatch, tmp_path: Path) -> None:
    monkeypatch.setattr(update, "_open_url", lambda request, **kwargs: io.BytesIO(NEW_BINARY))
    destination = tmp_path / "temporal"

    update._download_asset(_release(), destination)

    assert destination.read_bytes() == NEW_BINARY


def test_download_asset_rejects_wrong_size(monkeypatch, tmp_path: Path) -> None:
    monkeypatch.setattr(update, "_open_url", lambda request, **kwargs: io.BytesIO(NEW_BINARY))
    destination = tmp_path / "temporal"

    with pytest.raises(UpdateError, match="incompleta"):
        update._download_asset(_release(size=len(NEW_BINARY) + 1), destination)


def test_download_asset_rejects_wrong_digest(monkeypatch, tmp_path: Path) -> None:
    monkeypatch.setattr(update, "_open_url", lambda request, **kwargs: io.BytesIO(NEW_BINARY))
    destination = tmp_path / "temporal"

    with pytest.raises(UpdateError, match="SHA-256"):
        update._download_asset(_release(digest="sha256:" + "0" * 64), destination)


def test_installed_binary_path_requires_frozen(monkeypatch) -> None:
    monkeypatch.delattr(update.sys, "frozen", raising=False)

    with pytest.raises(UpdateError, match="binario autocontenido"):
        update.installed_binary_path()


def test_installed_binary_path_resolves_symlink(monkeypatch, tmp_path: Path) -> None:
    real = tmp_path / "real"
    real.write_bytes(b"binario")
    link = tmp_path / "kamino"
    link.symlink_to(real)
    monkeypatch.setattr(update.sys, "frozen", True, raising=False)
    monkeypatch.setattr(update.sys, "executable", str(link))

    assert update.installed_binary_path() == real.resolve()


@pytest.mark.parametrize(("system", "machine"), [("Linux", "arm64"), ("Darwin", "x86_64")])
def test_platform_check_rejects_unsupported_systems(monkeypatch, system: str, machine: str) -> None:
    monkeypatch.setattr(update.platform, "system", lambda: system)
    monkeypatch.setattr(update.platform, "machine", lambda: machine)

    with pytest.raises(UpdateError, match="macOS"):
        _ORIGINAL_ENSURE_PLATFORM()


def test_platform_check_accepts_macos_arm64(monkeypatch) -> None:
    monkeypatch.setattr(update.platform, "system", lambda: "Darwin")
    monkeypatch.setattr(update.platform, "machine", lambda: "arm64")

    _ORIGINAL_ENSURE_PLATFORM()


def test_replace_binary_replaces_and_preserves_mode(tmp_path: Path) -> None:
    target = tmp_path / "kamino"
    target.write_bytes(b"viejo")
    target.chmod(0o755)
    temp = tmp_path / "temporal"
    temp.write_bytes(b"nuevo")

    update._replace_binary(temp, target)

    assert target.read_bytes() == b"nuevo"
    assert target.stat().st_mode & 0o777 == 0o755
    assert not temp.exists()


def test_replace_binary_failure_leaves_target(monkeypatch, tmp_path: Path) -> None:
    target = tmp_path / "kamino"
    target.write_bytes(b"viejo")
    temp = tmp_path / "temporal"
    temp.write_bytes(b"nuevo")

    def fail(*args, **kwargs):
        raise OSError("sin permisos")

    monkeypatch.setattr(update.os, "replace", fail)

    with pytest.raises(UpdateError, match="no se pudo reemplazar"):
        update._replace_binary(temp, target)

    assert target.read_bytes() == b"viejo"


def test_replace_binary_keeps_symlink(tmp_path: Path) -> None:
    real = tmp_path / "real"
    real.write_bytes(b"viejo")
    link = tmp_path / "kamino"
    link.symlink_to(real)
    temp = tmp_path / "temporal"
    temp.write_bytes(b"nuevo")

    update._replace_binary(temp, link.resolve())

    assert link.is_symlink()
    assert link.read_bytes() == b"nuevo"
    assert real.read_bytes() == b"nuevo"


def test_run_update_installs_newer_release(monkeypatch, tmp_path: Path) -> None:
    target = tmp_path / "kamino"
    target.write_bytes(b"viejo")
    target.chmod(0o755)
    requested: list[str] = []
    _install_fake_http(monkeypatch, payload=_payload(), requested=requested)

    result = update.run_update("0.4.0", target=target)

    assert result.updated is True
    assert result.current_version == "0.4.0"
    assert result.latest_version == "0.5.0"
    assert result.target == target.resolve()
    assert target.read_bytes() == NEW_BINARY
    assert target.stat().st_mode & 0o111
    assert requested == [API_URL, ASSET_URL]
    assert list(tmp_path.glob(".kamino-update-*")) == []


def test_run_update_does_nothing_when_already_current(monkeypatch, tmp_path: Path) -> None:
    target = tmp_path / "kamino"
    target.write_bytes(b"viejo")
    requested: list[str] = []
    _install_fake_http(monkeypatch, payload=_payload(tag="v0.4.0"), requested=requested)

    result = update.run_update("0.4.0", target=target)

    assert result.updated is False
    assert target.read_bytes() == b"viejo"
    assert requested == [API_URL]
    assert list(tmp_path.glob(".kamino-update-*")) == []


def test_run_update_does_not_downgrade(monkeypatch, tmp_path: Path) -> None:
    target = tmp_path / "kamino"
    target.write_bytes(b"viejo")
    requested: list[str] = []
    _install_fake_http(monkeypatch, payload=_payload(tag="v0.5.0"), requested=requested)

    result = update.run_update("0.6.0", target=target)

    assert result.updated is False
    assert result.current_version == "0.6.0"
    assert result.latest_version == "0.5.0"
    assert target.read_bytes() == b"viejo"
    assert requested == [API_URL]


def test_run_update_rejects_digest_mismatch_and_cleans_up(monkeypatch, tmp_path: Path) -> None:
    target = tmp_path / "kamino"
    target.write_bytes(b"viejo")
    _install_fake_http(monkeypatch, payload=_payload(digest="sha256:" + "0" * 64))

    with pytest.raises(UpdateError, match="SHA-256"):
        update.run_update("0.4.0", target=target)

    assert target.read_bytes() == b"viejo"
    assert list(tmp_path.glob(".kamino-update-*")) == []


def test_run_update_rejects_size_mismatch_and_cleans_up(monkeypatch, tmp_path: Path) -> None:
    target = tmp_path / "kamino"
    target.write_bytes(b"viejo")
    _install_fake_http(monkeypatch, payload=_payload(size=len(NEW_BINARY) + 1))

    with pytest.raises(UpdateError, match="incompleta"):
        update.run_update("0.4.0", target=target)

    assert target.read_bytes() == b"viejo"
    assert list(tmp_path.glob(".kamino-update-*")) == []


def test_run_update_replaces_symlink_target(monkeypatch, tmp_path: Path) -> None:
    real = tmp_path / "real"
    real.write_bytes(b"viejo")
    link = tmp_path / "kamino"
    link.symlink_to(real)
    _install_fake_http(monkeypatch, payload=_payload())

    result = update.run_update("0.4.0", target=link)

    assert result.updated is True
    assert link.is_symlink()
    assert link.read_bytes() == NEW_BINARY
    assert real.read_bytes() == NEW_BINARY


def test_run_update_requires_existing_target(monkeypatch, tmp_path: Path) -> None:
    target = tmp_path / "no-existe"
    monkeypatch.setattr(update, "_open_url", lambda *args, **kwargs: pytest.fail("sin red"))

    with pytest.raises(UpdateError, match="no se encontró el binario"):
        update.run_update("0.4.0", target=target)


def test_install_script_uses_same_repository_and_asset() -> None:
    script = (Path(__file__).resolve().parents[1] / "scripts" / "install.sh").read_text(
        encoding="utf-8"
    )

    assert f'REPO="{update.REPOSITORY}"' in script
    assert f'ASSET="{update.ASSET_NAME}"' in script


# --- consulta, descarga y clasificación de errores ---


def test_query_update_reports_updated_up_to_date_and_newer(tmp_path: Path) -> None:
    target = tmp_path / "kamino"
    target.write_bytes(b"actual")
    newer = _release()

    updated = update.query_update("0.4.0", target=target, release=newer)
    current = update.query_update("0.5.0", target=target, release=newer)
    installed = update.query_update("0.6.0", target=target, release=newer)

    assert updated.should_update is True
    assert updated.outcome is update.UpdateOutcome.UPDATED
    assert current.should_update is False
    assert current.outcome is update.UpdateOutcome.UP_TO_DATE
    assert installed.should_update is False
    assert installed.outcome is update.UpdateOutcome.NEWER


def test_download_update_cancels_and_removes_temp(tmp_path: Path, monkeypatch) -> None:
    from kamino.errors import OperationCancelled

    monkeypatch.setattr(update, "_open_url", lambda request, **kwargs: io.BytesIO(NEW_BINARY))
    target = tmp_path / "kamino"
    target.write_bytes(b"actual")

    with pytest.raises(OperationCancelled):
        update.download_update(_release(), target, cancelled=lambda: True)

    assert target.read_bytes() == b"actual"
    assert not list(tmp_path.glob(".kamino-update-*"))


def test_download_update_reports_progress(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setattr(update, "_open_url", lambda request, **kwargs: io.BytesIO(NEW_BINARY))
    target = tmp_path / "kamino"
    target.write_bytes(b"actual")
    events = []

    temp = update.download_update(_release(), target, progress=events.append)

    assert temp.read_bytes() == NEW_BINARY
    assert events[-1].phase.value == "downloading"
    assert events[-1].current == len(NEW_BINARY)
    temp.unlink()


def test_update_error_kinds_are_stable(tmp_path: Path, monkeypatch) -> None:
    from kamino.errors import UpdateErrorKind

    monkeypatch.setattr(update.sys, "frozen", False, raising=False)
    with pytest.raises(UpdateError) as not_frozen:
        update.installed_binary_path()
    assert not_frozen.value.kind is UpdateErrorKind.NOT_SELF_CONTAINED

    monkeypatch.setattr(update.platform, "system", lambda: "Linux")
    monkeypatch.setattr(update.platform, "machine", lambda: "x86_64")
    with pytest.raises(UpdateError) as platform_error:
        _ORIGINAL_ENSURE_PLATFORM()
    assert platform_error.value.kind is UpdateErrorKind.PLATFORM

    target = tmp_path / "kamino"
    target.write_bytes(b"actual")
    monkeypatch.setattr(update, "_open_url", lambda request, **kwargs: io.BytesIO(NEW_BINARY))
    with pytest.raises(UpdateError) as digest_error:
        update.run_update("0.4.0", target=target, release=_release(digest="sha256:" + "0" * 64))
    assert digest_error.value.kind is UpdateErrorKind.INTEGRITY


def test_run_update_cancel_before_replace_keeps_binary(tmp_path: Path, monkeypatch) -> None:
    from kamino.errors import OperationCancelled

    target = tmp_path / "kamino"
    target.write_bytes(b"actual")
    monkeypatch.setattr(update, "_open_url", lambda request, **kwargs: io.BytesIO(NEW_BINARY))

    with pytest.raises(OperationCancelled):
        update.run_update("0.4.0", target=target, release=_release(), cancelled=lambda: True)

    assert target.read_bytes() == b"actual"
    assert not list(tmp_path.glob(".kamino-update-*"))
