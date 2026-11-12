"""Build and smoke-test the standalone macOS ARM64 Go executable."""

from __future__ import annotations

import argparse
import os
import platform
import re
import subprocess
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
BINARY = ROOT / "dist" / "kamino-arm64"


def project_version() -> str:
    pyproject = ROOT / "pyproject.toml"
    if pyproject.exists():
        import tomllib

        metadata = tomllib.loads(pyproject.read_text(encoding="utf-8"))
        return metadata["project"]["version"]
    source = ROOT / "internal" / "version" / "version.go"
    match = re.search(r'var Version = "([^"]+)"', source.read_text(encoding="utf-8"))
    if match is None:
        raise RuntimeError("No se pudo resolver la versión del proyecto")
    return match.group(1)


def check_binary(binary: Path, version: str) -> None:
    with tempfile.TemporaryDirectory(prefix="kamino-smoke-") as directory:
        home = Path(directory)
        env = {**os.environ, "HOME": str(home), "KAMINO_PLAIN": "1"}
        subprocess.run([str(binary), "--help"], env=env, check=True, capture_output=True)
        result = subprocess.run(
            [str(binary), "--version"], env=env, check=True, capture_output=True, text=True
        )
        if result.stdout.strip() != f"kamino {version}":
            raise RuntimeError("La versión del binario no coincide con la release")

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
        if result.returncode != 1 or "TOML inválido" not in result.stderr:
            raise RuntimeError(f"Falló la comprobación del catálogo embebido: {result.stderr}")
        if set(home.rglob("*")) != before or (
            invalid_definition.read_text(encoding="utf-8") != "invalid = ["
        ):
            raise RuntimeError("La comprobación del binario escribió en el HOME temporal")


def build(*, update_lock: bool = False) -> None:
    if platform.system() != "Darwin" or platform.machine() != "arm64":
        raise RuntimeError("El binario de release debe construirse en macOS ARM64")
    if update_lock:
        raise RuntimeError("uv.lock ya no participa en la construcción del binario Go")

    version = project_version()
    if os.environ.get("NEW_VERSION", version) != version:
        raise RuntimeError("La versión del proyecto no coincide con la calculada")
    BINARY.parent.mkdir(parents=True, exist_ok=True)
    environment = {**os.environ, "CGO_ENABLED": "0"}
    subprocess.run(
        [
            "go",
            "build",
            "-trimpath",
            "-ldflags",
            "-s -w "
            f"-X github.com/Gunmer/kamino/internal/version.Version={version} "
            "-X github.com/Gunmer/kamino/internal/buildinfo.SelfContained=true",
            "-o",
            str(BINARY),
            "./cmd/kamino",
        ],
        cwd=ROOT,
        env=environment,
        check=True,
    )
    check_binary(BINARY, version)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Construye y comprueba el binario ARM64.")
    parser.add_argument("--update-lock", action="store_true", help="Obsoleto: el binario es Go.")
    build(update_lock=parser.parse_args().update_lock)
