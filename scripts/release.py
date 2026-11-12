"""Coordinate a checked local semantic release and recover GitHub publication."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path

SEMANTIC_RELEASE = (
    "uv",
    "run",
    "--project",
    "legacy/python",
    "--no-sync",
    "semantic-release",
    "--config",
    "release.toml",
)


def run(*args: str, capture: bool = False) -> str:
    result = subprocess.run(
        args, check=True, text=True, stdout=subprocess.PIPE if capture else None
    )
    return result.stdout.strip() if capture else ""


def validate_tag(tag: str) -> str:
    if not re.fullmatch(r"v(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)", tag):
        raise ValueError("El tag debe tener el formato vX.Y.Z")
    return tag[1:]


def release_notes(changelog: str, tag: str) -> str:
    validate_tag(tag)
    sections = list(re.finditer(r"^## (v\d+\.\d+\.\d+) \([^\n]+\)$", changelog, re.MULTILINE))
    matches = [index for index, section in enumerate(sections) if section.group(1) == tag]
    if len(matches) != 1:
        raise ValueError("El changelog debe contener exactamente una entrada para el tag")
    index = matches[0]
    end = sections[index + 1].start() if index + 1 < len(sections) else len(changelog)
    return changelog[sections[index].start() : end].strip() + "\n"


def validate_release(tag: str) -> None:
    version = validate_tag(tag)
    if run("git", "log", "-1", "--format=%s", tag, capture=True) != (
        f"chore(release): {version} [skip ci]"
    ):
        raise ValueError("El tag no identifica un commit de release de Kamino")
    source = run("git", "show", f"{tag}:internal/version/version.go", capture=True)
    if f'var Version = "{version}"' not in source:
        raise ValueError("El tag y la versión Go no coinciden")
    run("git", "cat-file", "-e", f"{tag}:scripts/build_release.py")
    release_notes(run("git", "show", f"{tag}:CHANGELOG.md", capture=True), tag)


def remote_main() -> str:
    return run("git", "ls-remote", "origin", "refs/heads/main", capture=True).split()[0]


def prepare_release(sha: str) -> str | None:
    if not re.fullmatch(r"[0-9a-f]{40}", sha):
        raise ValueError("El SHA de CI no es válido")
    if remote_main() != sha or run("git", "rev-parse", "HEAD", capture=True) != sha:
        print("Ejecución obsoleta: main cambió; no se publicará una nueva versión.")
        return None
    if run("git", "branch", "--show-current", capture=True) != "main":
        raise ValueError("La publicación automática requiere la rama main")

    tag = run(*SEMANTIC_RELEASE, "version", "--print-tag", capture=True)
    validate_tag(tag)
    if run("git", "tag", "--list", tag, capture=True):
        print("No hay cambios publicables desde el último tag; no se genera una versión.")
        return None
    run(*SEMANTIC_RELEASE, "version", "--no-push", "--no-vcs-release")
    validate_release(tag)
    if run("git", "rev-parse", "HEAD^", capture=True) != sha or remote_main() != sha:
        raise RuntimeError("main cambió durante la construcción; se cancela el push de la release")
    # Atomic, non-forced publication prevents a branch race from leaving an orphan tag.
    run(
        "git",
        "push",
        "--atomic",
        "origin",
        "HEAD:refs/heads/main",
        f"refs/tags/{tag}:refs/tags/{tag}",
    )
    return tag


def github_release(repository: str, tag: str) -> dict | None:
    result = subprocess.run(
        ["gh", "api", f"repos/{repository}/releases/tags/{tag}"],
        capture_output=True,
        text=True,
    )
    if result.returncode:
        if "HTTP 404" in result.stderr:
            return None
        raise RuntimeError(f"No se pudo consultar la release: {result.stderr}")
    return json.loads(result.stdout)


def publish_release(repository: str, tag: str) -> None:
    notes = release_notes(run("git", "show", f"{tag}:CHANGELOG.md", capture=True), tag)
    release = github_release(repository, tag)
    with tempfile.TemporaryDirectory(prefix="kamino-release-") as directory:
        notes_file = Path(directory) / "notes.md"
        notes_file.write_text(notes, encoding="utf-8")
        if release is None:
            run(
                "gh",
                "release",
                "create",
                tag,
                "--verify-tag",
                "--draft",
                "--title",
                tag,
                "--notes-file",
                str(notes_file),
                "--repo",
                repository,
            )
        elif release["draft"]:
            run("gh", "release", "edit", tag, "--notes-file", str(notes_file), "--repo", repository)

        assets = release["assets"] if release else []
        for asset in assets:
            if asset["name"] == "kamino-arm64" and asset["state"] != "uploaded":
                if asset["state"] != "starter" or asset["size"] != 0:
                    raise RuntimeError(
                        "El asset existente está incompleto y requiere revisión manual"
                    )
                # Failed uploads can leave an empty starter asset blocking subsequent uploads.
                run(
                    "gh",
                    "api",
                    "--method",
                    "DELETE",
                    f"repos/{repository}/releases/assets/{asset['id']}",
                )
        if not any(
            asset["name"] == "kamino-arm64" and asset["state"] == "uploaded" for asset in assets
        ):
            run("gh", "release", "upload", tag, "dist/kamino-arm64", "--repo", repository)
        if release is None or release["draft"]:
            run("gh", "release", "edit", tag, "--draft=false", "--repo", repository)
    print(f"Release {tag} publicada con kamino-arm64.")


def main(*, sha: str | None = None, tag: str | None = None) -> None:
    repository = os.environ.get("GH_REPO", "")
    if not re.fullmatch(r"[\w.-]+/[\w.-]+", repository):
        raise ValueError("GH_REPO debe identificar el repositorio de GitHub")
    if run("git", "status", "--porcelain", capture=True):
        raise RuntimeError("La publicación requiere un árbol de trabajo limpio")
    run("git", "fetch", "origin", "main", "--tags")
    if tag:
        validate_release(tag)
        run("git", "merge-base", "--is-ancestor", tag, "origin/main")
        run("git", "switch", "--detach", tag)
        # Never bump or regenerate history when recovering an existing tag.
        run(sys.executable, "scripts/build_release.py")
    elif sha:
        tag = prepare_release(sha)
    else:
        raise ValueError("Se requiere --sha o --tag")
    if tag:
        publish_release(repository, tag)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Publica o recupera una release de Kamino.")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--sha", help="Commit de main validado por CI.")
    group.add_argument("--tag", help="Tag existente cuya publicación se debe completar.")
    arguments = parser.parse_args()
    main(sha=arguments.sha, tag=arguments.tag)
