"""Test release policy, history rendering, and publication without touching real remotes."""

from __future__ import annotations

import importlib.util
import json
import shutil
import subprocess
import sys
import tomllib
from pathlib import Path
from types import SimpleNamespace

import pytest

ROOT = Path(__file__).resolve().parents[1]


def load_script(name):
    spec = importlib.util.spec_from_file_location(name, ROOT / "scripts" / f"{name}.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


release = load_script("release")
build_release = load_script("build_release")


@pytest.fixture
def repository(tmp_path, monkeypatch):
    home = tmp_path / "home"
    home.mkdir()
    monkeypatch.setenv("HOME", str(home))
    monkeypatch.setenv("GIT_CONFIG_GLOBAL", str(home / ".gitconfig"))
    monkeypatch.setenv("GIT_CONFIG_NOSYSTEM", "1")
    monkeypatch.delenv("GITHUB_OUTPUT", raising=False)
    monkeypatch.delenv("GH_TOKEN", raising=False)
    monkeypatch.delenv("GITHUB_TOKEN", raising=False)
    monkeypatch.setenv("GITHUB_ACTIONS", "false")
    path = tmp_path / "repository"
    path.mkdir()
    monkeypatch.chdir(path)
    (path / "uv.lock").write_text(
        'version = 1\nrevision = 3\nrequires-python = ">=3.11"\n\n'
        '[[package]]\nname = "kamino-cli"\nversion = "0.1.0"\nsource = { virtual = "." }\n',
        encoding="utf-8",
    )
    (path / "CHANGELOG.md").write_text(
        "# Historial de cambios\n\n"
        "<!-- version list -->\n\n"
        "## v0.1.0 (2026-10-31)\n\n"
        "- Versión inicial\n",
        encoding="utf-8",
    )
    (path / "src/kamino").mkdir(parents=True)
    (path / "src/kamino/__init__.py").write_text(
        '"""kamino: clonación y despliegue de configuraciones de aplicaciones en macOS."""\n\n'
        '__version__ = "0.1.0"\n',
        encoding="utf-8",
    )
    (path / "scripts").mkdir()
    (path / "scripts/build_release.py").write_text("# Test fixture\n", encoding="utf-8")
    shutil.copytree(ROOT / ".github/release-templates", path / ".github/release-templates")
    config = (ROOT / "pyproject.toml").read_text(encoding="utf-8")
    config = (
        '[project]\nname = "kamino-cli"\nversion = "0.1.0"\nrequires-python = ">=3.11"\n\n'
        "[tool.semantic_release]" + config.split("[tool.semantic_release]", 1)[1]
    )
    config = config.replace(
        'build_command = "uv run --no-sync python scripts/build_release.py --update-lock"',
        'build_command = "uv lock --offline && git add uv.lock"',
    )
    (path / "pyproject.toml").write_text(config, encoding="utf-8")
    git("init", "-b", "main")
    git("config", "user.name", "Release Test")
    git("config", "user.email", "release@example.invalid")
    git("remote", "add", "origin", "https://github.com/example/kamino.git")
    git("add", ".")
    git("commit", "-m", "chore: initial test repository")
    git("tag", "v0.1.0")
    return path


def git(*args):
    return subprocess.run(["git", *args], check=True, capture_output=True, text=True).stdout.strip()


def semantic_release(*args, check=True):
    result = subprocess.run(
        [sys.executable, "-m", "semantic_release", *args],
        capture_output=True,
        text=True,
    )
    if check:
        assert result.returncode == 0, result.stdout + result.stderr
    return result


def test_repository_base_is_coherent_and_independent_of_real_release_data(
    tmp_path, monkeypatch, request
):
    real_root = tmp_path / "real-project"
    real_root.mkdir()
    shutil.copytree(ROOT / ".github/release-templates", real_root / ".github/release-templates")
    shutil.copy(ROOT / "pyproject.toml", real_root / "pyproject.toml")
    (real_root / "CHANGELOG.md").write_text(
        "# Historial de cambios\n\n"
        "<!-- version list -->\n\n"
        "## v0.2.0 (2026-11-02)\n\n"
        "- Entrada real\n\n"
        "## v0.1.0 (2026-10-31)\n\n"
        "- Versión inicial\n",
        encoding="utf-8",
    )
    (real_root / "src/kamino").mkdir(parents=True)
    (real_root / "src/kamino/__init__.py").write_text('__version__ = "0.2.0"\n', encoding="utf-8")
    monkeypatch.setattr(sys.modules[__name__], "ROOT", real_root)
    request.getfixturevalue("repository")

    changelog = Path("CHANGELOG.md").read_text(encoding="utf-8")
    metadata = tomllib.loads(Path("pyproject.toml").read_text(encoding="utf-8"))
    lock = tomllib.loads(Path("uv.lock").read_text(encoding="utf-8"))
    source = Path("src/kamino/__init__.py").read_text(encoding="utf-8")

    assert git("tag", "--list") == "v0.1.0"
    assert "## v0.1.0 (" in changelog
    assert "## v0.2.0 (" not in changelog
    assert metadata["project"]["version"] == "0.1.0"
    assert '__version__ = "0.1.0"' in source
    assert [p["version"] for p in lock["package"] if p["name"] == "kamino-cli"] == ["0.1.0"]
    assert "## v0.2.0 (" in (real_root / "CHANGELOG.md").read_text(encoding="utf-8")
    assert '__version__ = "0.2.0"' in (real_root / "src/kamino/__init__.py").read_text(
        encoding="utf-8"
    )


@pytest.mark.parametrize(
    ("message", "expected"),
    [
        ("fix: corrige el despliegue", "0.1.1"),
        ("feat: añade una aplicación", "0.2.0"),
        ("feat!: cambia el formato", "0.2.0"),
        ("fix: cambia el formato\n\nBREAKING CHANGE: requiere migración", "0.2.0"),
        ("perf!: cambia el almacenamiento", "0.2.0"),
        ("chore!: cambia la configuración", "0.2.0"),
        ("ci: ajusta el workflow", "0.1.0"),
        ("docs: mejora la guía", "0.1.0"),
        ("chore: limpia archivos", "0.1.0"),
        ("perf: reduce lecturas", "0.1.0"),
        ("refactor: reorganiza módulos", "0.1.0"),
        ("mensaje sin convención", "0.1.0"),
    ],
)
def test_version_policy(repository, message, expected):
    git("commit", "--allow-empty", "-m", message)
    before = git("status", "--porcelain")
    result = semantic_release("version", "--print")
    assert result.stdout.strip() == expected
    assert git("status", "--porcelain") == before


@pytest.mark.parametrize("squash", [False, True])
def test_integrated_feature_is_detected(repository, squash):
    if squash:
        git("commit", "--allow-empty", "-m", "feat: añade una aplicación (#4)")
    else:
        git("switch", "-c", "feature/catalog")
        git("commit", "--allow-empty", "-m", "feat: añade una aplicación")
        git("switch", "main")
        git("merge", "--no-ff", "feature/catalog", "-m", "Merge pull request #4")
    assert semantic_release("version", "--print").stdout.strip() == "0.2.0"


def test_no_release_does_not_write_or_tag(repository):
    git("commit", "--allow-empty", "-m", "ci: ajusta el workflow")
    files = {
        name: Path(name).read_bytes() for name in ("pyproject.toml", "uv.lock", "CHANGELOG.md")
    }
    head = git("rev-parse", "HEAD")
    semantic_release("version", "--no-push", "--no-vcs-release")
    assert git("rev-parse", "HEAD") == head
    assert git("tag", "--list") == "v0.1.0"
    assert all(Path(name).read_bytes() == content for name, content in files.items())
    assert git("status", "--porcelain") == ""


def test_stable_breaking_change_increments_major(repository):
    for name in ("pyproject.toml", "uv.lock", "src/kamino/__init__.py"):
        path = Path(name)
        path.write_text(
            path.read_text(encoding="utf-8").replace("0.1.0", "1.0.0"), encoding="utf-8"
        )
    git("add", ".")
    git("commit", "-m", "chore(release): 1.0.0 [skip ci]")
    git("tag", "v1.0.0")
    git("commit", "--allow-empty", "-m", "fix!: cambia el formato")
    assert semantic_release("version", "--print").stdout.strip() == "2.0.0"


def test_release_keeps_versions_lock_history_and_notes_in_sync(repository):
    previous = Path("CHANGELOG.md").read_text(encoding="utf-8")
    original_lock = tomllib.loads(Path("uv.lock").read_text(encoding="utf-8"))
    git(
        "commit",
        "--allow-empty",
        "-m",
        "feat!: cambia el formato\n\nBREAKING CHANGE: requiere migración",
    )
    semantic_release("version", "--no-push", "--no-vcs-release")
    release.validate_release("v0.2.0")
    assert git("status", "--porcelain") == ""
    changelog = Path("CHANGELOG.md").read_text(encoding="utf-8")
    assert "### Funcionalidades" in changelog
    assert "**Cambio incompatible.** requiere migración" in changelog
    assert "**Cambio incompatible.** requiere migración" in release.release_notes(
        changelog, "v0.2.0"
    )
    assert previous.split("<!-- version list -->")[1] in changelog
    assert changelog.count("## v0.2.0 (") == 1
    assert "v0.1.0" not in release.release_notes(changelog, "v0.2.0")
    lock = tomllib.loads(Path("uv.lock").read_text(encoding="utf-8"))
    assert [p for p in lock["package"] if p["name"] != "kamino-cli"] == [
        p for p in original_lock["package"] if p["name"] != "kamino-cli"
    ]
    tags = git("tag", "--list")
    head = git("rev-parse", "HEAD")
    semantic_release("version", "--no-push", "--no-vcs-release")
    semantic_release("changelog")
    assert git("rev-parse", "HEAD") == head
    assert git("tag", "--list") == tags
    assert Path("CHANGELOG.md").read_text(encoding="utf-8") == changelog
    assert git("status", "--porcelain") == ""


def test_failed_build_never_commits_or_tags(repository):
    config = (
        Path("pyproject.toml")
        .read_text(encoding="utf-8")
        .replace(
            'build_command = "uv lock --offline && git add uv.lock"',
            'build_command = "exit 1"',
        )
    )
    Path("pyproject.toml").write_text(config, encoding="utf-8")
    git("add", "pyproject.toml")
    git("commit", "-m", "fix: prueba una construcción fallida")
    head = git("rev-parse", "HEAD")
    result = semantic_release("version", "--no-push", "--no-vcs-release", check=False)
    assert result.returncode != 0
    assert git("rev-parse", "HEAD") == head
    assert git("tag", "--list") == "v0.1.0"


@pytest.mark.parametrize("tag", ["--help", "v01.2.3", "v1.2", "v1.2.3\n", "v1.2.3;whoami"])
def test_invalid_tag_is_rejected(tag):
    with pytest.raises(ValueError):
        release.validate_tag(tag)


def test_duplicate_changelog_section_is_rejected():
    with pytest.raises(ValueError, match="exactamente una"):
        release.release_notes("## v0.2.0 (2026-11-02)\n\n## v0.2.0 (2026-11-02)\n", "v0.2.0")


def test_obsolete_ci_does_not_version_or_publish(monkeypatch):
    monkeypatch.setattr(release, "remote_main", lambda: "b" * 40)
    monkeypatch.setattr(
        release, "run", lambda *args, **kwargs: pytest.fail(f"Unexpected call: {args}")
    )
    assert release.prepare_release("a" * 40) is None


def test_changed_main_prevents_push(monkeypatch):
    sha = "a" * 40
    tips = iter([sha, "b" * 40])
    calls = []

    def fake_run(*args, **kwargs):
        calls.append(args)
        if args[:2] == ("git", "rev-parse"):
            return sha
        if args[:2] == ("git", "branch"):
            return "main"
        if "--print-tag" in args:
            return "v0.2.0"
        return ""

    monkeypatch.setattr(release, "remote_main", lambda: next(tips))
    monkeypatch.setattr(release, "run", fake_run)
    monkeypatch.setattr(release, "validate_release", lambda tag: None)
    with pytest.raises(RuntimeError, match="main cambió"):
        release.prepare_release(sha)
    assert not any(call[:2] == ("git", "push") for call in calls)


def test_push_is_atomic_and_not_forced(monkeypatch):
    sha = "a" * 40
    calls = []

    def fake_run(*args, **kwargs):
        calls.append(args)
        if args[:2] == ("git", "rev-parse"):
            return sha
        if args[:2] == ("git", "branch"):
            return "main"
        if "--print-tag" in args:
            return "v0.2.0"
        return ""

    monkeypatch.setattr(release, "remote_main", lambda: sha)
    monkeypatch.setattr(release, "run", fake_run)
    monkeypatch.setattr(release, "validate_release", lambda tag: None)
    assert release.prepare_release(sha) == "v0.2.0"
    assert calls[-1] == (
        "git",
        "push",
        "--atomic",
        "origin",
        "HEAD:refs/heads/main",
        "refs/tags/v0.2.0:refs/tags/v0.2.0",
    )


@pytest.mark.parametrize("existing", [None, {"draft": True, "assets": []}])
def test_publish_only_leaves_draft_after_failed_upload(monkeypatch, existing):
    calls = []

    def fake_run(*args, **kwargs):
        calls.append(args)
        if args[:2] == ("git", "show"):
            return "## v0.2.0 (2026-11-02)\n\n### Funcionalidades\n\n- Una función\n"
        if args[:3] == ("gh", "release", "upload"):
            raise subprocess.CalledProcessError(1, args)
        return ""

    monkeypatch.setattr(release, "run", fake_run)
    monkeypatch.setattr(release, "github_release", lambda *args: existing)
    with pytest.raises(subprocess.CalledProcessError):
        release.publish_release("example/kamino", "v0.2.0")
    assert not any("--draft=false" in call for call in calls)


def test_published_release_with_asset_is_not_overwritten(monkeypatch):
    calls = []

    def fake_run(*args, **kwargs):
        calls.append(args)
        return "## v0.2.0 (2026-11-02)\n\n- Una función\n"

    monkeypatch.setattr(release, "run", fake_run)
    monkeypatch.setattr(
        release,
        "github_release",
        lambda *args: {"draft": False, "assets": [{"name": "kamino-arm64", "state": "uploaded"}]},
    )
    release.publish_release("example/kamino", "v0.2.0")
    assert all(call[0] == "git" for call in calls)


def test_recovery_removes_only_empty_failed_upload_and_publishes_after_upload(monkeypatch):
    calls = []

    def fake_run(*args, **kwargs):
        calls.append(args)
        return "## v0.2.0 (2026-11-02)\n\n- Una función\n"

    monkeypatch.setattr(release, "run", fake_run)
    monkeypatch.setattr(
        release,
        "github_release",
        lambda *args: {
            "draft": True,
            "assets": [{"id": 7, "name": "kamino-arm64", "state": "starter", "size": 0}],
        },
    )
    release.publish_release("example/kamino", "v0.2.0")
    deletion = ("gh", "api", "--method", "DELETE", "repos/example/kamino/releases/assets/7")
    upload = ("gh", "release", "upload", "v0.2.0", "dist/kamino-arm64", "--repo", "example/kamino")
    assert calls.index(deletion) < calls.index(upload)
    assert calls[-1] == (
        "gh",
        "release",
        "edit",
        "v0.2.0",
        "--draft=false",
        "--repo",
        "example/kamino",
    )


def test_unsafe_incomplete_asset_is_not_deleted_or_published(monkeypatch):
    calls = []
    monkeypatch.setattr(
        release, "run", lambda *args, **kwargs: calls.append(args) or "## v0.2.0 (2026-11-02)\n"
    )
    monkeypatch.setattr(
        release,
        "github_release",
        lambda *args: {
            "draft": True,
            "assets": [{"id": 7, "name": "kamino-arm64", "state": "starter", "size": 42}],
        },
    )
    with pytest.raises(RuntimeError, match="revisión manual"):
        release.publish_release("example/kamino", "v0.2.0")
    assert not any("DELETE" in call or "--draft=false" in call for call in calls)


def test_recovery_does_not_version_or_regenerate_changelog(monkeypatch):
    calls = []
    monkeypatch.setenv("GH_REPO", "example/kamino")
    monkeypatch.setattr(release, "run", lambda *args, **kwargs: calls.append(args) or "")
    monkeypatch.setattr(release, "validate_release", lambda tag: None)
    monkeypatch.setattr(release, "publish_release", lambda *args: None)
    release.main(tag="v0.2.0")
    assert ("git", "merge-base", "--is-ancestor", "v0.2.0", "origin/main") in calls
    assert not any("semantic-release" in call for call in calls)
    assert not any(call[:2] == ("git", "push") for call in calls)


@pytest.mark.parametrize("status", [404, 403, 500])
def test_github_errors_are_not_all_treated_as_missing(monkeypatch, status):
    result = SimpleNamespace(returncode=1, stderr=f"gh: API error (HTTP {status})", stdout="")
    monkeypatch.setattr(release.subprocess, "run", lambda *args, **kwargs: result)
    if status == 404:
        assert release.github_release("example/kamino", "v0.2.0") is None
    else:
        with pytest.raises(RuntimeError):
            release.github_release("example/kamino", "v0.2.0")


def test_github_release_response(monkeypatch):
    result = SimpleNamespace(returncode=0, stdout=json.dumps({"draft": True, "assets": []}))
    monkeypatch.setattr(release.subprocess, "run", lambda *args, **kwargs: result)
    assert release.github_release("example/kamino", "v0.2.0")["draft"] is True


def test_smoke_test_uses_temporary_home_and_stops_before_detection(monkeypatch, tmp_path):
    calls = []

    def fake_run(args, **kwargs):
        home = Path(kwargs["env"]["HOME"])
        assert home != Path.home()
        calls.append(args)
        if "--version" in args:
            return SimpleNamespace(stdout="kamino 0.2.0\n")
        if "clone" in args:
            assert (home / ".config/kamino/apps/smoke.toml").read_text() == "invalid = ["
            return SimpleNamespace(returncode=1, stderr="TOML inválido en 'smoke.toml'")
        return SimpleNamespace()

    monkeypatch.setattr(build_release.subprocess, "run", fake_run)
    build_release.check_binary(tmp_path / "kamino-arm64", "0.2.0")
    assert len(calls) == 3


def test_wrong_binary_version_is_rejected(monkeypatch, tmp_path):
    monkeypatch.setattr(
        build_release.subprocess,
        "run",
        lambda *args, **kwargs: SimpleNamespace(stdout="kamino 0.1.0\n"),
    )
    with pytest.raises(RuntimeError, match="versión del binario"):
        build_release.check_binary(tmp_path / "kamino-arm64", "0.2.0")


def test_build_rejects_wrong_architecture(monkeypatch):
    monkeypatch.setattr(build_release.platform, "system", lambda: "Linux")
    with pytest.raises(RuntimeError, match="macOS ARM64"):
        build_release.build()


def test_build_includes_absolute_catalog_path_and_only_refreshes_project_lock(
    monkeypatch, tmp_path
):
    monkeypatch.chdir(tmp_path)
    (tmp_path / "pyproject.toml").write_text('[project]\nversion = "0.2.0"\n', encoding="utf-8")
    monkeypatch.setattr(build_release.platform, "system", lambda: "Darwin")
    monkeypatch.setattr(build_release.platform, "machine", lambda: "arm64")
    monkeypatch.delenv("NEW_VERSION", raising=False)
    calls = []
    monkeypatch.setattr(build_release.subprocess, "run", lambda args, **kwargs: calls.append(args))
    monkeypatch.setattr(build_release, "check_binary", lambda *args: None)
    build_release.build(update_lock=True)
    assert calls[:2] == [["uv", "lock"], ["git", "add", "uv.lock"]]
    command = calls[2]
    assert command[command.index("--add-data") + 1] == (
        f"{tmp_path / 'src/kamino/catalog/apps'}:kamino/catalog/apps"
    )
    assert command[command.index("--target-architecture") + 1] == "arm64"
    assert command[-1] == str(tmp_path / "src/kamino/__main__.py")


def test_missing_embedded_catalog_fails_binary_check(monkeypatch, tmp_path):
    def fake_run(args, **kwargs):
        if "--version" in args:
            return SimpleNamespace(stdout="kamino 0.2.0\n")
        return SimpleNamespace(returncode=1, stderr="no se pudo leer el catálogo embebido")

    monkeypatch.setattr(build_release.subprocess, "run", fake_run)
    with pytest.raises(RuntimeError, match="catálogo embebido"):
        build_release.check_binary(tmp_path / "kamino-arm64", "0.2.0")
