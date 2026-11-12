# Rendimiento después de la migración

Medido: 2026-11-12 en el mismo equipo que la línea base (macOS 26.6.2, Apple
Silicon `arm64`). Binario: `dist/kamino-arm64` de GoReleaser equivalente,
construido con `CGO_ENABLED=0`, `-trimpath`, `-s -w` y versión inyectada.

## Método

`hyperfine` no está disponible sin red; se repiten los mismos comandos de
`MIGRATION/baseline-python.md` con `scripts/bench_startup.py` (mediana de 7
ejecuciones, primera descartada) y `scripts/bench_tui.py` (PTY 80×24, mediana de
5 primeras muestras). `KAMINO_PLAIN=1` para la CLI.

## CLI

| Comando | Python PyInstaller | Python `.venv` | Go | Presupuesto |
|---|---:|---:|---:|---:|
| `--version` | 5964.27 ms | 240.56 ms | 5.68 ms | < 30 ms |
| `--help` | 5966.94 ms | 268.40 ms | 5.73 ms | < 30 ms |
| `clone -n -a git` | 5967.18 ms | 252.80 ms | 8.36 ms | — |

## TUI (primer frame)

| Binario | Mediana |
|---|---:|
| Python PyInstaller | 6045.31 ms |
| Python `.venv` | 236.10 ms |
| Go | 15.90 ms |

## Conclusión

Los dos presupuestos se cumplen con margen: CLI por debajo de 30 ms y TUI por
debajo de 150 ms. No fue necesario perfilar ni iterar. La medición con
`hyperfine` queda pendiente para cuando haya red.
