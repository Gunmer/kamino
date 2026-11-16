# AGENTS.md

## Configuración y verificación

- El proyecto es Go (Go 1.25 o superior).
- Compila con `go build ./...` y ejecuta la CLI con `go run ./cmd/kamino --help`.
- Antes de terminar, ejecuta las mismas comprobaciones que CI, en este orden:
  `gofmt -l .` (vacío), `go vet ./...`, `golangci-lint run ./...` (si está instalado) y
  `go test ./... -race`.
- Ejecuta una prueba concreta con `go test ./internal/deploy -run TestName`.

## Arquitectura

- El módulo es `github.com/Gunmer/kamino`; el binario es `cmd/kamino`, el comando de release es
  `cmd/release` y los paquetes internos viven en `internal/`.
- `internal/cli` solo traduce argumentos, salida y códigos. `internal/app` contiene los flujos
  compartidos; `internal/clone`, `internal/deploy` y `internal/update` son casos de uso;
  `internal/texts`, `internal/ui` y `internal/tui` presentan.
- Las definiciones integradas están en `internal/catalog/apps/*.toml` y se cargan con `go:embed`.
  Se combinan con los TOML del usuario en `~/.config/kamino/apps/`; las definiciones del usuario
  reemplazan las integradas con el mismo id. `internal/catalog/apps/` es la única copia del
  catálogo.
- `internal/archive` es responsable del manifiesto versionado y del formato del archivo;
  `internal/plan` planifica la clonación y `internal/deploy` clasifica y aplica el despliegue.
- `internal/paths.SafeHome` usa `openat`/`O_NOFOLLOW`; no se debe degradar a comprobaciones léxicas.
- La TUI usa Bubble Tea, Lip Gloss y `bubbles` (`key`, `help`). `internal/tui/theme` centraliza
  tokens, símbolos y estilos ANSI; `internal/tui/frame.go` posee cabecera, body, mensajes, pie y
  guardia de tamaño; `internal/tui/screen.go` define el contrato `Screen` y `internal/tui/app.go`
  la pila y el estado compartido. `docs/diseno-tui.md` es la referencia visual.
- Los goldens viven en `internal/tui/testdata/golden/` y se regeneran con
  `go test ./internal/tui -run Golden -update`; el test de contrato comprueba altura, ancho y SGR
  ANSI. `View` no lee entorno, reloj ni disco.

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
- La versión vive en `internal/version/version.go`; no la edites a mano fuera de `cmd/release`. La
  construcción de release inyecta `Version` y `SelfContained` con `-ldflags`.

## Releases

- Los commits integrados en `main` siguen Conventional Commits. `cmd/release` calcula la
  siguiente versión, actualiza `CHANGELOG.md` e `internal/version/version.go`, construye y
  comprueba `dist/kamino-arm64`, crea el commit y el tag de release, y publica con `gh`.
- No edites a mano el changelog de una versión ya publicada ni publiques releases fuera de
  `cmd/release`; la recuperación de un tag existente se hace con `--tag`.
