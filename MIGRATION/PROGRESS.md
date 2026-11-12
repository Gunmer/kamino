# Progreso de la migración a Go

Actualizado: 2026-11-12

## Fase actual

- **Fase:** 1 completada; Fase 2 en curso
- **Tarea actual:** 2.1 Leer specs y escribir phase2-notes
- **Rama:** `migrate/go`

## Estado por fase

| Fase | Estado | Tag |
|---|---|---|
| 1. Entorno, layout y fixtures | completada | `migration/phase-1` |
| 2. Tests por capas con stubs | en curso | `migration/phase-2` |
| 3. Implementación por capas | en curso | `migration/phase-3` |
| 4. Release, rendimiento y corte | pendiente | `migration/phase-4` |

## Registro

| Fecha | Tarea | Resultado | Comandos |
|---|---|---|---|
| 2026-11-12 | 1.1 | Rama `migrate/go` creada desde `main`; árbol limpio tras confirmar el cambio OpenSpec | `git status --short --branch`, `git log --oneline -1` |
| 2026-11-12 | 1.2 | Cuatro documentos de migración creados | `test -f MIGRATION/*.md` |
| 2026-11-12 | 1.3 | Go 1.27.2 disponible; herramientas externas bloqueadas por red | `go version`, `MIGRATION/ENV.md` |
| 2026-11-12 | 1.4 | `go.mod`, `cmd/kamino` y layout `internal/` | `go build ./...`, `go vet ./...` |
| 2026-11-12 | 1.5 | Makefile y `.golangci.yml`; `make fmt lint test` en verde | `make fmt lint test` |
| 2026-11-12 | 1.6 | Catálogo copiado, `go:embed`, prueba de paridad y script | `go test ./internal/catalog`, `scripts/check_catalog_sync.sh` |
| 2026-11-12 | 1.7 | Baseline Python medido con arnés propio | `scripts/bench_startup.py`, `scripts/bench_tui.py` |
| 2026-11-12 | 1.8 | Golden fixtures deterministas y códigos 0/1/2/3 | `scripts/gen_golden.py` dos veces |
| 2026-11-12 | 1.9 | CI Go macOS/Ubuntu y job de compatibilidad | `.github/workflows/ci.yml` |
| 2026-11-12 | 1.10 | Notas de fase y tag local | `git tag --list 'migration/phase-*'` |
