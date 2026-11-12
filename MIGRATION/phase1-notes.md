# Fase 1 — Entorno, layout y fixtures

## Contrato resumido

- Rama `migrate/go` desde `main`; Python permanece intacto hasta la Fase 4.
- Módulo `github.com/Gunmer/kamino`, binario `cmd/kamino`, paquetes internos con
  dependencias en una sola dirección.
- `format_version = 1`, manifiesto con el orden y los `null` de Pydantic, rutas
  POSIX relativas al HOME, códigos 0/1/2/3 y `dry_run` sin escrituras.
- El núcleo no imprime ni pregunta: `app` devuelve resultados tipados; `cli`,
  `ui`, `texts` y `tui` presentan.
- `SafeHome` usa `openat`/`O_NOFOLLOW`; no se degrada a comprobaciones léxicas.
- El catálogo embebido se copia byte a byte desde `src/kamino/catalog/apps/` y
  `scripts/check_catalog_sync.sh` falla si divergen.

## Entregado

- `go.mod`, `Makefile`, `.golangci.yml`, layout `cmd/` + `internal/`.
- Copia del catálogo con `go:embed`, prueba de paridad y script de sincronización.
- `scripts/bench_startup.py`, `scripts/bench_tui.py` y
  `MIGRATION/baseline-python.md`.
- `scripts/gen_golden.py` y `testdata/golden/` con manifiesto, tar canónico,
  plan, conflictos, textos y códigos 0/1/2/3; dos ejecuciones dan los mismos
  bytes normalizados.
- CI de Go para macOS y Ubuntu, más un job de compatibilidad con Python/uv.
- `MIGRATION/ENV.md`, `DECISIONS.md`, `BLOCKED.md` y `PROGRESS.md`.

## Verificación

```sh
go build ./...
go vet ./...
go test ./...
scripts/check_catalog_sync.sh
.venv/bin/python scripts/gen_golden.py
```

`golangci-lint`, `gofumpt`, `goreleaser` y `hyperfine` no se pudieron instalar
por falta de red; se usan sus equivalentes y queda registrado en
`MIGRATION/BLOCKED.md`.
