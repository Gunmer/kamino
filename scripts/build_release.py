"""Build and smoke-test the standalone macOS ARM64 executable."""

from __future__ import annotations

import argparse
import os
import platform
import subprocess
import tempfile
import tomllib
from pathlib import Path


def check_binary(binary: Path, version: str) -> None:
    with tempfile.TemporaryDirectory(prefix="kamino-smoke-") as directory:
        home = Path(directory)
        env = {**os.environ, "HOME": str(home)}
        subprocess.run([str(binary), "--help"], env=env, check=True, capture_output=True)
        result = subprocess.run(
            [str(binary), "--version"], env=env, check=True, capture_output=True, text=True
        )
        if result.stdout.strip() != f"kamino {version}":
            raise RuntimeError("La versión del binario no coincide con la release")

        # User definitions load after embedded ones, before detection or clone planning.
        apps = home / ".config" / "kamino" / "apps"
        apps.mkdir(parents=True)
        invalid_definition = apps / "smoke.toml"
        invalid_definition.write_text("invalid = [", encoding="utf-8")
        before = set(home.rglob("*"))
        result = subprocess.run(
            [str(binary), "clone", "--dry-run"],
            env=env,
            stdin=subprocess.DEVNULL,
            capture_output=True,
            text=True,
        )
        if result.returncode != 1 or "TOML inválido en 'smoke.toml'" not in result.stderr:
            raise RuntimeError(f"Falló la comprobación del catálogo embebido: {result.stderr}")
        if set(home.rglob("*")) != before or (
            invalid_definition.read_text(encoding="utf-8") != "invalid = ["
        ):
            raise RuntimeError("La comprobación del binario escribió en el HOME temporal")


def build(*, update_lock: bool = False) -> None:
    if platform.system() != "Darwin" or platform.machine() != "arm64":
        raise RuntimeError("El binario de release debe construirse en macOS ARM64")
    if update_lock:
        subprocess.run(["uv", "lock"], check=True)
        subprocess.run(["git", "add", "uv.lock"], check=True)

    metadata = tomllib.loads(Path("pyproject.toml").read_text(encoding="utf-8"))
    version = metadata["project"]["version"]
    if os.environ.get("NEW_VERSION", version) != version:
        raise RuntimeError("La versión del proyecto no coincide con la calculada")
    source = Path("src/kamino").resolve()
    subprocess.run(
        [
            "uv",
            "run",
            "--no-sync",
            "pyinstaller",
            "--noconfirm",
            "--clean",
            "--onefile",
            "--target-architecture",
            "arm64",
            "--name",
            "kamino-arm64",
            "--specpath",
            "build",
            "--add-data",
            f"{source / 'catalog/apps'}:kamino/catalog/apps",
            "--add-data",
            f"{source / 'tui/styles.tcss'}:kamino/tui",
            "--collect-data",
            "textual",
            str(source / "__main__.py"),
        ],
        check=True,
    )
    check_binary(Path("dist/kamino-arm64").resolve(), version)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Construye y comprueba el binario ARM64.")
    parser.add_argument("--update-lock", action="store_true", help="Actualiza y prepara uv.lock.")
    build(update_lock=parser.parse_args().update_lock)
