# Rendimiento después de la migración

Medido: 2026-11-12 en el mismo equipo que la línea base (macOS 26.6.2, Apple
Silicon `arm64`). Binario: `dist/kamino_darwin_arm64_v8.0/kamino` construido con
`goreleaser release --snapshot --clean` (`CGO_ENABLED=0`, `-trimpath`, `-s -w` y
versión inyectada).

## Método

Se repiten los mismos comandos de `MIGRATION/baseline-python.md` con
`hyperfine 2.0.0` (3 calentamientos, 20 ejecuciones, `KAMINO_PLAIN=1`) y con
`scripts/bench_tui.py` (PTY 80×24, mediana de 10 muestras, primera descartada,
sin `KAMINO_PLAIN`).

## CLI

| Comando | Python PyInstaller | Python `.venv` | Go (mediana) | Presupuesto |
|---|---:|---:|---:|---:|
| `--version` | 5964.27 ms | 240.56 ms | 6.39 ms | < 30 ms |
| `--help` | 5966.94 ms | 268.40 ms | 5.72 ms | < 30 ms |
| `clone -n -a git` | 5967.18 ms | 252.80 ms | 8.72 ms | — |

## TUI (primer frame)

| Binario | Mediana |
|---|---:|
| Python PyInstaller | 6045.31 ms |
| Python `.venv` | 236.10 ms |
| Go | 25.97 ms |

## Conclusión

Los dos presupuestos se cumplen con margen: CLI por debajo de 30 ms y TUI por
debajo de 150 ms. No fue necesario perfilar ni iterar. La medición pendiente con
`hyperfine` ya está ejecutada con el binario de la snapshot de GoReleaser.
