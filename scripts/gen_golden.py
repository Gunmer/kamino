#!/usr/bin/env python3
"""Genera fixtures doradas desde la implementación Python.

Los bytes del tar.gz se normalizan (mtime, uid/gid y cabecera gzip) para que dos
ejecuciones produzcan exactamente los mismos archivos. El script no toca el HOME
real ni /Applications: usa HOMEs sintéticos y un catálogo inyectado.
"""

from __future__ import annotations

import gzip
import hashlib
import io
import json
import os
import subprocess
import sys
import tarfile
import tempfile
from datetime import UTC, datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "src"
OUTPUT = ROOT / "testdata" / "golden"
PYTHON = os.environ.get("KAMINO_PYTHON", str(ROOT / ".venv" / "bin" / "python"))
sys.path.insert(0, str(SRC))

from kamino import texts  # noqa: E402
from kamino.archive import FORMAT_VERSION, AppEntry, FileEntry, Manifest, read_manifest  # noqa: E402
from kamino.catalog.models import AppDefinition  # noqa: E402
from kamino.clone import plan_clone, write_clone  # noqa: E402
from kamino.conflicts import RowDecision, build_hunks, compose  # noqa: E402
from kamino.plan import build_plan  # noqa: E402


def canonical_archive(source: Path, destination: Path) -> None:
    """Reescribe el tar.gz sin metadatos variables."""
    with tarfile.open(source, "r:gz") as tar:
        members = []
        for member in tar:
            data = tar.extractfile(member).read() if member.isfile() else b""
            members.append((member, data))
    buffer = io.BytesIO()
    with tarfile.open(fileobj=buffer, mode="w") as tar:
        for member, data in members:
            info = tarfile.TarInfo(member.name)
            info.type = member.type
            info.mode = member.mode
            info.size = len(data) if member.isfile() else 0
            info.linkname = member.linkname
            info.uid = 0
            info.gid = 0
            info.uname = ""
            info.gname = ""
            info.mtime = 0
            tar.addfile(info, io.BytesIO(data) if member.isfile() else None)
    with open(destination, "wb") as raw:
        with gzip.GzipFile(fileobj=raw, mode="wb", mtime=0) as compressed:
            compressed.write(buffer.getvalue())


def synthetic_home() -> tuple[tempfile.TemporaryDirectory, Path]:
    temporary = tempfile.TemporaryDirectory(prefix="kamino-golden-")
    home = Path(temporary.name)
    (home / ".config" / "git").mkdir(parents=True)
    (home / ".gitconfig").write_text("[user]\n\tname = Golden\n", encoding="utf-8")
    (home / ".config" / "git" / "ignore").write_text("ignore\n", encoding="utf-8")
    for path in home.rglob("*"):
        os.utime(path, (0, 0), follow_symlinks=False)
    return temporary, home


def make_clone(home: Path, output: Path) -> None:
    catalog = {
        "git": AppDefinition(id="git", name="Git", command="git", paths=[".gitconfig"]),
        "bat": AppDefinition(id="bat", name="bat", command="bat", optional_paths=[".config/bat"]),
    }
    metadata = {
        "created_at": "2026-11-12T00:00:00Z",
        "macos_version": "15.0",
        "arch": "arm64",
        "hostname": "golden-host",
    }
    planned = plan_clone(
        home=home,
        selected=["git", "bat"],
        output=output,
        catalog=catalog,
        path=os.environ.get("PATH"),
        metadata=metadata,
        now=datetime(2026, 11, 12, tzinfo=UTC),
    )
    write_clone(planned, home=home)


def dump_manifest(source: Path) -> None:
    manifest = read_manifest(source)
    (OUTPUT / "manifest.json").write_text(
        manifest.model_dump_json(indent=2) + "\n", encoding="utf-8"
    )


def dump_plan(home: Path) -> None:
    catalog = {
        "git": AppDefinition(id="git", name="Git", command="git", paths=[".gitconfig"]),
    }
    plan = build_plan(list(catalog.values()), home=home)
    payload = {
        "operations": [
            {"app_id": item.app_id, "path": item.path, "size": item.size, "sha256": item.sha256}
            for item in plan.operations
        ],
        "skipped": [{"path": item.path, "reason": item.reason} for item in plan.skipped],
        "notices": plan.notices,
    }
    (OUTPUT / "plan.json").write_text(
        json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )


def dump_conflicts() -> None:
    local = "".join(f"line {index}\n" for index in range(10))
    clone = local.replace("line 4\n", "LINE FOUR\n")
    hunks = build_hunks(local, clone)
    payload = {
        "hunks": [
            {
                "index": hunk.index,
                "local_start": hunk.local_start,
                "local_end": hunk.local_end,
                "clone_start": hunk.clone_start,
                "clone_end": hunk.clone_end,
                "rows": [
                    {"kind": row.kind.value, "local": row.local, "clone": row.clone}
                    for row in hunk.rows
                ],
            }
            for hunk in hunks
        ],
        "compose_clone": compose(local, clone, [RowDecision.CLONE] * sum(
            1 for hunk in hunks for row in hunk.rows if row.kind.value != "equal"
        )),
        "compose_local": compose(local, clone, [RowDecision.LOCAL] * sum(
            1 for hunk in hunks for row in hunk.rows if row.kind.value != "equal"
        )),
    }
    (OUTPUT / "conflicts.json").write_text(
        json.dumps(payload, indent=2, sort_keys=True, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )


def dump_texts() -> None:
    payload = {
        "help_c1": texts.help_texts(
            texts.SCREEN_C1,
            texts.HELP_C1,
            flow_steps=texts.CLONE_STEPS,
            current_step="apps",
        ),
        "help_d2": texts.help_texts(
            texts.SCREEN_D2,
            texts.HELP_D2,
            flow_steps=texts.DEPLOY_STEPS,
            current_step="conflictos",
            secrets=True,
        ),
        "summary_deploy": texts.SUMMARY_DEPLOY,
        "glossary": {
            "original": texts.ORIGINAL,
            "snapshot": texts.SNAPSHOT,
            "hybrid": texts.HYBRID,
            "perfect_clone": texts.PERFECT_CLONE,
            "not_installed": texts.NOT_INSTALLED,
        },
    }
    (OUTPUT / "texts.json").write_text(
        json.dumps(payload, indent=2, sort_keys=True, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )


def dump_codes(home: Path, clone_file: Path) -> None:
    cases = [
        ["--version"],
        ["does-not-exist"],
        ["clone", "-a", "git", "-n"],
        ["clone", "-a", "missing", "-n"],
        ["deploy", str(clone_file)],
        ["deploy", str(clone_file), "--dry-run"],
        ["deploy", str(clone_file), "--yes"],
    ]
    results = []
    for args in cases:
        environment = dict(os.environ)
        environment["HOME"] = str(home)
        environment["KAMINO_PLAIN"] = "1"
        completed = subprocess.run(
            [PYTHON, "-m", "kamino", *args],
            cwd=ROOT,
            env=environment,
            capture_output=True,
            text=True,
            check=False,
        )
        results.append({"args": args, "code": completed.returncode})
    (OUTPUT / "codes.json").write_text(
        json.dumps(results, indent=2) + "\n", encoding="utf-8"
    )


def checksums() -> None:
    lines = []
    for path in sorted(OUTPUT.iterdir()):
        if path.name == "SHA256SUMS":
            continue
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        lines.append(f"{digest}  {path.name}")
    (OUTPUT / "SHA256SUMS").write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    OUTPUT.mkdir(parents=True, exist_ok=True)
    temporary, home = synthetic_home()
    try:
        clone_file = Path(temporary.name) / "clone.tar.gz"
        make_clone(home, clone_file)
        canonical_archive(clone_file, OUTPUT / "clone.tar.gz")
        dump_manifest(clone_file)
        dump_plan(home)
        dump_conflicts()
        dump_texts()
        dump_codes(home, OUTPUT / "clone.tar.gz")
    finally:
        temporary.cleanup()
    checksums()
    print(f"fixtures doradas en {OUTPUT}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
