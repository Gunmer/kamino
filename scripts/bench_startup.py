#!/usr/bin/env python3
"""Mide el arranque de la CLI con el mismo contrato que `hyperfine`.

`hyperfine` no está disponible sin red; este arnés registra la mediana de varias
ejecuciones y documenta el método en `MIGRATION/baseline-python.md`.
"""

from __future__ import annotations

import argparse
import os
import platform
import shlex
import statistics
import subprocess
import time


def measure(command: list[str], environment: dict[str, str], runs: int) -> float:
    times: list[float] = []
    for run in range(runs):
        start = time.perf_counter()
        subprocess.run(command, env=environment, capture_output=True, check=False)
        elapsed = time.perf_counter() - start
        if run > 0:
            times.append(elapsed)
    return statistics.median(times)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--runs", type=int, default=7)
    parser.add_argument("--binary", default="dist/kamino-arm64")
    args = parser.parse_args()

    home = os.environ.get("HOME", "")
    environment = dict(os.environ)
    environment["KAMINO_PLAIN"] = "1"
    cases = {
        "--version": [args.binary, "--version"],
        "--help": [args.binary, "--help"],
        "clone -n -a git": [args.binary, "clone", "-n", "-a", "git"],
    }
    print(f"platform={platform.platform()}")
    print(f"machine={platform.machine()}")
    print(f"python={platform.python_version()}")
    print(f"home={home}")
    print(f"binary={args.binary}")
    for label, command in cases.items():
        seconds = measure(command, environment, args.runs)
        print(f"{label}: median_ms={seconds * 1000:.2f}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
