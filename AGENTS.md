# AGENTS.md

Reglas mínimas para trabajar en este repositorio. El detalle vive en `docs/` (índice en
`docs/README.md`): léelo solo cuando la tarea toque esa área.

## Verificación

Proyecto Go 1.25+, módulo `github.com/Gunmer/kamino`. Antes de terminar, en este orden:
`gofmt -l .` (vacío), `go vet ./...`, `golangci-lint run ./...` (si está instalado) y
`go test ./... -race`. Una prueba concreta: `go test ./internal/implant -run TestName`.
CLI: `go run ./cmd/kamino --help`.

## Mapa

- `cmd/kamino` (binario) y `cmd/release` (release); el código vive en `internal/`.
- `internal/cli` solo traduce argumentos, salida y códigos; `internal/app` tiene los flujos
  compartidos; `clone`, `implant` y `update` son casos de uso; `texts`, `ui` y `tui` presentan.
- El catálogo integrado está en `internal/catalog/apps/*.toml` (única copia, `go:embed`); los TOML
  del usuario en `~/.config/kamino/apps/` reemplazan por id.
- Capas, formatos y flujos: `docs/tecnico/`. TUI (`internal/tui`): `docs/tui/`. Comportamiento
  de producto: `docs/producto/funcional.md`.
- Goldens de la TUI: `go test ./internal/tui -run Golden -update`. `View` no lee entorno,
  reloj ni disco.

## Invariantes (no negociables)

- Las pruebas usan HOME temporal e inyectan `Home`, `Catalog`, `ApplicationDirs` y
  `SnapshotsDir`; nunca tocan el HOME real ni `/Applications`.
- Orden del despliegue: validar manifiesto, verificar el archivo completo antes de escribir,
  instantánea de los destinos y, al final, reemplazos atómicos. `--dry-run` no escribe nada.
- Rutas del archivo: POSIX relativas al HOME; rechaza absolutas y `..`; solo archivos regulares y
  enlaces simbólicos relativos. Si cambias el manifiesto, cuida la compatibilidad y prueba el
  ciclo de serialización/despliegue.
- `internal/paths.SafeHome` usa `openat`/`O_NOFOLLOW`; no lo degrades a comprobaciones léxicas.
- Despliegue no interactivo requiere `--yes` (salvo `--dry-run`). Salida 1 = error/cancelación;
  3 = éxito parcial con omitidos.
- Markdown y texto visible en la CLI, en español; código, identificadores y comentarios, en inglés.
- Si cambias un comando, opción o mensaje, actualiza ayuda y docs.

## Releases

- Commits en `main` con Conventional Commits. Solo `cmd/release` calcula la versión, edita
  `CHANGELOG.md` e `internal/version/version.go`, construye, etiqueta y publica con `gh`.
- No edites a mano la versión ni el changelog publicado; recupera un tag con `--tag`.
