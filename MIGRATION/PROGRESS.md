# Progreso de la migración a Go

Actualizado: 2026-11-12

## Fase actual

- **Fase:** 4 completada
- **Tarea actual:** verificación con herramientas reales completada; revisión y archivo pendientes
- **Rama:** `migrate/go`

## Estado por fase

| Fase | Estado | Tag |
|---|---|---|
| 1. Entorno, layout y fixtures | completada | `migration/phase-1` |
| 2. Tests por capas | completada | `migration/phase-2` |
| 3. Implementación por capas y TUI | completada | `migration/phase-3` |
| 4. Release, rendimiento y corte | completada | `migration/phase-4` |

## Registro

| Fecha | Tarea | Resultado | Comandos |
|---|---|---|---|
| 2026-11-12 | 1.1-1.10 | Layout Go, catálogo, baseline, golden y CI | `make fmt lint test`, `scripts/gen_golden.py` |
| 2026-11-12 | 2.1-2.7 | Pruebas por capas, matriz cruzada y cobertura de 99 requisitos | `go test ./... -race`, `go test ./internal/integration -run Cross` |
| 2026-11-12 | 3.1-3.4 | `archive`, `paths`, `catalog`, `detect`, `plan`, `ghostty`, `snapshot`, `clone`, `conflicts` y `deploy` | `go test ./internal/...` |
| 2026-11-12 | 4.1-4.5 | CLI, TUI, update y presupuesto de arranque | `go test ./internal/cli ./internal/tui ./internal/update ./internal/performance` |
| 2026-11-12 | 5.1-5.5 | `progress`, `SafeHome`, `archive` PAX y `guard.sh` | `scripts/guard.sh`, `go vet ./...` |
| 2026-11-12 | 6.1-6.5 | `catalog`, `detect`, `plan` y `ghostty` con TOML estricto y fnmatch compatible | `scripts/check_catalog_sync.sh` |
| 2026-11-12 | 7.1-7.3 | `snapshot` y `clone` con manifiesto, `--force` y cancelación | `go test ./internal/snapshot ./internal/clone` |
| 2026-11-12 | 8.1-8.3 | `SequenceMatcher`, composición, validación y `deploy` | `go test ./internal/conflicts ./internal/deploy` |
| 2026-11-12 | 9.1-9.4 | `texts`, `ui`, CLI de texto y códigos 0/1/2/3 | `go test ./internal/cli` |
| 2026-11-12 | 10.1-10.3 | `update` con GitHub, SemVer, SHA-256 y reemplazo atómico | `go test ./internal/update` |
| 2026-11-12 | 11.1-11.5 | TUI de menú, C1-C4, D0-D5, U1, configuración y guardia 80x24 | `go test ./internal/tui -race` |
| 2026-11-12 | 12.1-12.8 | GoReleaser, smoke, benchmark, documentación, corte y informe | `make release-snapshot`, `scripts/guard.sh` |
| 2026-11-12 | Verificación pendiente | Herramientas instaladas; 1 hallazgo de `ineffassign` corregido; formato, lint, release y benchmarks en verde | `golangci-lint run ./...`, `gofumpt -l .`, `goreleaser release --snapshot --clean`, `hyperfine` |

## Verificación final

| Comando | Resultado |
|---|---|
| `gofmt -l .` / `gofumpt -l .` | sin archivos pendientes |
| `go vet ./...` | sin errores |
| `golangci-lint run ./...` | 0 issues (se corrigió un `ineffassign` en `internal/update`) |
| `go test ./... -race` | todo en verde (la matriz cruzada se ejecuta con `KAMINO_PYTHON`) |
| `go test ./internal/integration -run Cross -v` | 3 pruebas en verde |
| `scripts/check_catalog_sync.sh` | catálogo sincronizado |
| `scripts/guard.sh` | cambios protegidos documentados en `DECISIONS.md` (D005) |
| `goreleaser release --snapshot --clean` | snapshot correcta; binario `kamino migration/phase-4-SNAPSHOT-755e93c` |
| `hyperfine` (CLI) | `--version` 6.39 ms, `--help` 5.72 ms, `clone -n` 8.72 ms |
| `scripts/bench_tui.py` | primer frame 25.97 ms |
