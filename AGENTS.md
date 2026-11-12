# AGENTS.md

## Configuración y verificación

- El proyecto activo es Go (Go 1.25 o superior). La implementación Python vive en
  `legacy/python/` como referencia histórica y no se usa para construir el binario.
- Compila con `go build ./...` y ejecuta la CLI con `go run ./cmd/kamino --help`.
- Antes de terminar, ejecuta las mismas comprobaciones que CI, en este orden:
  `gofmt -l .` (vacío), `go vet ./...`, `golangci-lint run ./...` (si está instalado) y
  `go test ./... -race`.
- Ejecuta una prueba concreta con `go test ./internal/deploy -run TestName`.
- Para la matriz de compatibilidad con el oráculo Python: `go test ./internal/integration -run Cross`.

## Arquitectura

- El módulo es `github.com/Gunmer/kamino`; el binario es `cmd/kamino` y los paquetes internos
  viven en `internal/`. La distribución se llama `kamino-cli`, pero el ejecutable es `kamino`.
- `internal/cli` solo traduce argumentos, salida y códigos. `internal/app` contiene los flujos
  compartidos; `internal/clone`, `internal/deploy` y `internal/update` son casos de uso;
  `internal/texts`, `internal/ui` y `internal/tui` presentan.
- Las definiciones integradas están en `internal/catalog/apps/*.toml` y se cargan con `go:embed`.
  Se combinan con los TOML del usuario en `~/.config/kamino/apps/`; las definiciones del usuario
  reemplazan las integradas con el mismo id. `scripts/check_catalog_sync.sh` exige que la copia
  embebida coincida byte a byte con `src/kamino/catalog/apps/` de la referencia Python.
- `internal/archive` es responsable del manifiesto versionado y del formato del archivo;
  `internal/plan` planifica la clonación y `internal/deploy` clasifica y aplica el despliegue.
- `internal/paths.SafeHome` usa `openat`/`O_NOFOLLOW`; no se debe degradar a comprobaciones léxicas.
- La TUI usa terminal crudo y ANSI propio; no depende de Textual ni de un framework externo.

## Invariantes de seguridad y compatibilidad

- Las pruebas deben usar directorios HOME temporales e inyectar `Home`, `Catalog`,
  `ApplicationDirs` y `SnapshotsDir` cuando corresponda. Las pruebas de clon/despliegue nunca deben
  leer ni modificar el HOME real del desarrollador ni `/Applications`.
- Conserva el orden del despliegue: validar el manifiesto, verificar el archivo completo antes de
  escribir en cualquier destino, crear una instantánea de los destinos que cambiarán y, por último,
  aplicar reemplazos atómicos. `--dry-run` no debe escribir nada.
- Las rutas del archivo son rutas POSIX relativas al HOME. Rechaza rutas absolutas y `..`; solo se
  admiten archivos regulares y enlaces simbólicos relativos. Si modificas el manifiesto, considera
  la compatibilidad del formato y añade pruebas del archivo y del ciclo de serialización/despliegue.
- El despliegue no interactivo requiere `--yes`, salvo con `--dry-run`. El código de salida 1 de la
  CLI indica error/cancelación; el 3 indica éxito parcial con elementos omitidos.
- Escribe en español todo el contenido Markdown (`.md`) del repositorio y todo el texto visible
  para el usuario en la CLI. Escribe en inglés todo el código, incluidos nombres, identificadores y
  comentarios.
- Si cambias la versión manualmente, mantén sincronizados `internal/version/version.go` y el tag de
  release; la construcción de release inyecta `Version` y `SelfContained` con `-ldflags`.

## Migración

- El cambio OpenSpec `migrate-to-go` conserva los tags `migration/phase-*`.
- `scripts/guard.sh` falla si cambian `testdata/golden/` o los `*_test.go` de la Fase 2 sin una
  entrada nueva en `MIGRATION/DECISIONS.md`.
