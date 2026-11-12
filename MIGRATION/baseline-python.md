# Línea base de Python

Medido: 2026-11-12 en macOS 26.6.2, Apple Silicon (`arm64`), Python 3.12.7.

## Método

`hyperfine` no está disponible sin red (`MIGRATION/BLOCKED.md`). Se usó
`scripts/bench_startup.py` (mediana de 5 ejecuciones, primera descartada) y
`scripts/bench_tui.py` como arnés PTY de 80×24 para el primer frame. Los
comandos se ejecutaron con `KAMINO_PLAIN=1`; la TUI se abrió sin
`KAMINO_PLAIN`. No se publicó ni se modificó nada.

## CLI con el binario PyInstaller (`dist/kamino-arm64`, 18 MB, 0.7.0)

| Comando | Mediana |
|---|---:|
| `--version` | 5964.27 ms |
| `--help` | 5966.94 ms |
| `clone -n -a git` | 5967.18 ms |

## CLI con el intérprete del proyecto (`.venv/bin/python -m kamino`)

| Comando | Mediana |
|---|---:|
| `--version` | 240.56 ms |
| `--help` | 268.40 ms |
| `clone -n -a git` | 252.80 ms |

## TUI (primer frame, PTY 80×24)

| Binario | Mediana |
|---|---:|
| `dist/kamino-arm64` | 6045.31 ms |
| `.venv/bin/python -m kamino` | 236.10 ms |

## Observaciones

- El `onefile` de PyInstaller paga la extracción en cada arranque; el objetivo
  de 30 ms de CLI y 150 ms de TUI exige el binario Go.
- La primera ejecución tras cuarentena/escaneo se excluye de la mediana.
- La medición con `hyperfine` queda pendiente para cuando haya red; este
  arnés conserva comandos, entorno y tiempos para repetir la comparación.
