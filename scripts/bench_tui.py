#!/usr/bin/env python3
"""Arnés PTY para medir el primer frame de la TUI de Python.

Uso: .venv/bin/python scripts/bench_tui.py [--runs N] [--command "..."]
"""

from __future__ import annotations

import argparse
import fcntl
import os
import pty
import statistics
import struct
import subprocess
import sys
import termios
import time


def measure(command: list[str], runs: int) -> float:
    times: list[float] = []
    for run in range(runs):
        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 24, 80, 0, 0))
        environment = dict(os.environ)
        environment.pop("KAMINO_PLAIN", None)
        environment["TERM"] = "xterm-256color"
        start = time.perf_counter()
        process = subprocess.Popen(
            command,
            stdin=slave,
            stdout=slave,
            stderr=slave,
            env=environment,
            close_fds=True,
        )
        os.close(slave)
        first_frame = None
        try:
            while first_frame is None:
                ready, _, _ = select_read(master, 15.0)
                if not ready:
                    break
                chunk = os.read(master, 65536)
                if chunk:
                    first_frame = time.perf_counter() - start
            if first_frame is None:
                raise RuntimeError("la TUI no mostró ningún frame en 15 s")
            if run > 0:
                times.append(first_frame)
        finally:
            process.terminate()
            try:
                process.wait(timeout=2)
            except subprocess.TimeoutExpired:
                process.kill()
            os.close(master)
    return statistics.median(times)


def select_read(fd: int, timeout: float) -> tuple[bool, bool, bool]:
    import select

    return select.select([fd], [], [], timeout)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--runs", type=int, default=7)
    parser.add_argument("--command", default=".venv/bin/python -m kamino")
    args = parser.parse_args()
    command = args.command.split()
    seconds = measure(command, args.runs)
    print(f"tui_first_frame_ms={seconds * 1000:.2f}")
    print(f"command={' '.join(command)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
