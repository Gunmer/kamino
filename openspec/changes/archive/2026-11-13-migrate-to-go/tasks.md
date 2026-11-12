# Tasks

## 1. Fase 1 — Entorno, layout y fixtures

- [x] 1.1 Crear la rama `migrate/go` desde `main` y comprobar que el árbol está limpio; verificar con `git status --short --branch` y `git log --oneline -1`.
- [x] 1.2 Crear `MIGRATION/PROGRESS.md`, `MIGRATION/DECISIONS.md`, `MIGRATION/BLOCKED.md` y `MIGRATION/ENV.md` con el formato acordado; verificar que los cuatro archivos existen y que `PROGRESS.md` registra la fase y la tarea actual.
- [x] 1.3 Verificar e instalar Go, `golangci-lint`, `gofumpt`, `goreleaser` y `hyperfine`; anotar versiones en `MIGRATION/ENV.md`; verificar con `go version`, `golangci-lint version`, `gofumpt -version`, `goreleaser --version` y `hyperfine --version`.
- [x] 1.4 Crear `go.mod` con el módulo `github.com/Gunmer/kamino`, `cmd/kamino/main.go` y el layout de `internal/`; verificar con `go build ./...` y `go vet ./...`.
- [x] 1.5 Crear `Makefile` con `build`, `test`, `lint`, `fmt`, `bench-startup`, `golden`, `guard` y `sync-catalog`, y `.golangci.yml`; verificar con `make fmt lint test`.
- [x] 1.6 Copiar `src/kamino/catalog/apps/*.toml` a `internal/catalog/apps/`, añadir `go:embed`, una prueba de paridad y `scripts/check_catalog_sync.sh`; verificar con `go test ./internal/catalog` y `scripts/check_catalog_sync.sh`.
- [x] 1.7 Medir la línea base Python con `hyperfine` para `--version`, `--help`, `clone -n -a git` con `KAMINO_PLAIN=1` y `scripts/bench_tui.py` como arnés PTY para la TUI; guardar `MIGRATION/baseline-python.md`; verificar que contiene comandos, entorno y tiempos.
- [x] 1.8 Crear `scripts/gen_golden.py` y `testdata/golden/README.md`; generar manifiestos, tar.gz, conflictos, ayuda, resúmenes y códigos 0/1/2/3 desde Python con HOMEs sintéticos; verificar que dos ejecuciones producen los mismos bytes normalizados.
- [x] 1.9 Añadir CI de Go para macOS y Ubuntu con `go vet`, `golangci-lint` y `go test ./... -race`, más un job de compatibilidad con Python/uv; verificar localmente los mismos comandos y revisar que el workflow no toca el release existente.
- [x] 1.10 Escribir `MIGRATION/phase1-notes.md` con el contrato resumido, actualizar `MIGRATION/PROGRESS.md`, hacer commits pequeños y crear el tag `migration/phase-1`; verificar con `git tag --list 'migration/phase-*'` y `git status --short`.

## 2. Fase 2a — Tests de archive, paths, catálogo, detección, plan y Ghostty

- [x] 2.1 Leer los specs de `clone-safety`, `global-app-config` y `ghostty-config-references` y `AGENTS.md`; escribir `MIGRATION/phase2-notes.md`; verificar que resume los requisitos cubiertos.
- [x] 2.2 Crear stubs de `internal/archive` y tests de lectura/escritura, `format_version = 1`, checksums, rutas absolutas y `..`, tar malicioso, manifiesto duplicado, manifiesto de más de 4 MiB y tope de miembros; verificar con `go test ./internal/archive` (deben fallar por implementación, no por compilación).
- [x] 2.3 Crear stubs de `internal/paths` y tests de `SafeHome`: rutas relativas, enlaces intermedios y finales, enlaces finales relativos seguros, sustitución de directorio y escrituras atómicas; verificar con `go test ./internal/paths`.
- [x] 2.4 Crear stubs de `internal/catalog` y `internal/detect` y tests de TOML, rutas opcionales, patrones, `env_paths`, `xdg_paths`, `install_marker`, prioridad del catálogo del usuario, detección sin ejecutar binarios y `Info.plist`; verificar con `go test ./internal/catalog ./internal/detect`.
- [x] 2.5 Crear stubs de `internal/plan` y `internal/ghostty` y tests de exclusiones, deduplicación, rutas omitidas, avisos, `config-file`, temas, ciclos, límites, referencias externas y credenciales; verificar con `go test ./internal/plan ./internal/ghostty`.
- [x] 2.6 Añadir a `MIGRATION/spec-coverage.md` los requisitos de `clone-safety`, `global-app-config` y `ghostty-config-references` con sus tests; verificar que no queda ningún requisito sin fila.
- [x] 2.7 Actualizar `MIGRATION/PROGRESS.md` con resultados y comandos ejecutados y hacer commit; verificar con `git status --short`.

## 3. Fase 2b — Tests de clon, deploy, snapshot y conflictos

- [x] 3.1 Crear stubs de `internal/clone` y tests de selección, exclusiones, deduplicación, categorías, código 3, `--force` con revalidación, `dry_run` sin escrituras, cancelación y limpieza de temporales; verificar con `go test ./internal/clone`.
- [x] 3.2 Crear stubs de `internal/snapshot` y `internal/deploy` y tests de orden verificar -> extraer -> decidir -> holocrón -> aplicar, categorías, código 3, holocrón solo con destinos existentes, `PartialApplyError`, cancelación y `dry_run`; verificar con `go test ./internal/snapshot ./internal/deploy`.
- [x] 3.3 Crear stubs de `internal/conflicts` y tests de todo-clon, todo-original, mezcla dentro de un bloque, CRLF, salto final y validación TOML/JSON/INI/plist con claves duplicadas; verificar con `go test ./internal/conflicts`.
- [x] 3.4 Añadir tests de compatibilidad cruzada: clon y holocrón de Python desplegados por Go, y clon y holocrón de Go desplegados por Python, usando HOMEs temporales y `KAMINO_PLAIN=1`; verificar con `go test ./internal/integration -run Cross`.
- [x] 3.5 Actualizar `MIGRATION/PROGRESS.md` con resultados y comandos ejecutados y hacer commit; verificar con `git status --short`.

## 4. Fase 2c — Tests de CLI, TUI, update y rendimiento

- [x] 4.1 Crear stubs de `internal/cli` y tests de opciones, alias, ayuda, códigos de salida, `KAMINO_PLAIN`, TTY/no TTY, `--yes`, `--dry-run` y nombre por defecto `clone-yyyyMMdd.tar.gz`; verificar con `go test ./internal/cli`.
- [x] 4.2 Crear stubs de `internal/tui` y tests de 80x24, 79x23, redimensionado, navegación, `NO_COLOR`, ayuda, error de cuatro partes y `Ctrl+C` antes/después del punto de no retorno; verificar con `go test ./internal/tui`.
- [x] 4.3 Crear stubs de `internal/update` y tests de consulta, comparación SemVer, digest, reemplazo atómico, symlink, plataforma y servidor HTTP de prueba con `httptest`, sin red real; verificar con `go test ./internal/update`.
- [x] 4.4 Añadir benchmarks o tests de presupuesto para `--version` y `--help`; verificar con `go test ./internal/performance -run Startup`.
- [x] 4.5 Completar `MIGRATION/spec-coverage.md` para los 99 requisitos, escribir `MIGRATION/review-request-phase2.md`, actualizar `MIGRATION/PROGRESS.md`, hacer commit y crear el tag `migration/phase-2`; verificar con `go vet ./...`, `go test ./...` (falla solo por funcionalidad no implementada) y `git tag --list 'migration/phase-*'`.

## 5. Fase 3a — paths, archive y progress

- [x] 5.1 Implementar `internal/progress` con `context.Context`, callbacks de progreso y cancelación cooperativa; verificar con `go test ./internal/progress`.
- [x] 5.2 Implementar `internal/paths` con `SafeHome` basado en `openat`/`O_NOFOLLOW`, rutas relativas, `walk` y escrituras atómicas; verificar con `go test ./internal/paths`.
- [x] 5.3 Implementar `internal/archive` con lectura/escritura PAX, gzip determinista, manifiesto, checksums, verificación, extracción segura y límites; verificar con `go test ./internal/archive`.
- [x] 5.4 Crear `scripts/guard.sh` para fallar si cambian `testdata/golden/` o los `*_test.go` de la Fase 2 sin entrada en `MIGRATION/DECISIONS.md`; verificar con `scripts/guard.sh` y una prueba controlada de fallo.
- [x] 5.5 Escribir `MIGRATION/phase3-notes.md` y `MIGRATION/review-request-3a.md`, actualizar `MIGRATION/PROGRESS.md` y hacer commit; verificar con `go vet ./...`, `golangci-lint run ./...`, `gofumpt -l .` vacío y `scripts/guard.sh`.

## 6. Fase 3b — catalog, detect, plan y ghostty

- [x] 6.1 Implementar `internal/catalog` con modelos estrictos, `go:embed`, carga del catálogo del usuario y errores; verificar con `go test ./internal/catalog` y `scripts/check_catalog_sync.sh`.
- [x] 6.2 Implementar `internal/detect` con `Info.plist`, `which` inyectable, `search_paths`, alias y `install_marker`; verificar con `go test ./internal/detect`.
- [x] 6.3 Implementar `internal/plan` con expansión de rutas, patrones, `env_paths`, `xdg_paths`, exclusiones, `fnmatch` compatible con Python y conteo; verificar con `go test ./internal/plan`.
- [x] 6.4 Implementar `internal/ghostty` con directivas, referencias, temas, ciclos, límites y avisos de portabilidad; verificar con `go test ./internal/ghostty`.
- [x] 6.5 Actualizar `MIGRATION/PROGRESS.md` con resultados y comandos ejecutados y hacer commit; verificar con `go vet ./...`, `golangci-lint run ./...`, `gofumpt -l .` vacío y `scripts/guard.sh`.

## 7. Fase 3c — snapshot y clone

- [x] 7.1 Implementar `internal/snapshot` con creación, conteo, nombres con sufijo y app sintética `snapshot`; verificar con `go test ./internal/snapshot`.
- [x] 7.2 Implementar `internal/clone` con `plan_clone`, `write_clone`, manifiesto, nombre por defecto, `--force`, revalidación, progreso y cancelación; verificar con `go test ./internal/clone`.
- [x] 7.3 Actualizar `MIGRATION/PROGRESS.md` con resultados y comandos ejecutados y hacer commit; verificar con `go vet ./...`, `golangci-lint run ./...`, `gofumpt -l .` vacío y `scripts/guard.sh`.

## 8. Fase 3d — conflicts y deploy

- [x] 8.1 Implementar `internal/conflicts` con `SequenceMatcher` (`autojunk` configurable), `get_grouped_opcodes`, `compose`, `validate_merge` y `splitlines` de Python; verificar con `go test ./internal/conflicts`.
- [x] 8.2 Implementar `internal/deploy` con sesión, plan, clasificación, `preview`, `apply`, holocrón, revalidación, `PartialApplyError`, cancelación y `dry_run`; verificar con `go test ./internal/deploy`.
- [x] 8.3 Escribir `MIGRATION/review-request-3d.md`, actualizar `MIGRATION/PROGRESS.md` con resultados y comandos ejecutados y hacer commit; verificar con `go vet ./...`, `golangci-lint run ./...`, `gofumpt -l .` vacío y `scripts/guard.sh`.

## 9. Fase 3e — texts, ui y CLI de texto

- [x] 9.1 Implementar `internal/texts` con todos los textos visibles, glosario, ayuda y plantillas, sin literales de usuario fuera de ese paquete; verificar con `go test ./internal/texts`.
- [x] 9.2 Implementar `internal/ui` con avisos, resúmenes de clonación/despliegue, diffs de `--dry-run` y actualización; verificar con `go test ./internal/ui`.
- [x] 9.3 Implementar `internal/cli` con Cobra, comandos, alias, `--version`/`-V`, `-h`, códigos 0/1/2/3, `--install-completion`/`--show-completion` y la vía de texto; verificar con `go test ./internal/cli`.
- [x] 9.4 Actualizar `MIGRATION/PROGRESS.md` con resultados y comandos ejecutados y hacer commit; verificar con `go vet ./...`, `golangci-lint run ./...`, `gofumpt -l .` vacío y `scripts/guard.sh`.

## 10. Fase 3f — update

- [x] 10.1 Implementar `internal/update` con consulta a GitHub, SemVer, descarga por bloques, SHA-256, reemplazo atómico, symlinks, plataforma y `SelfContained` por ldflags; verificar con `go test ./internal/update`.
- [x] 10.2 Conectar el comando `update` a la vía de texto y comprobar que una construcción de desarrollo no descarga ni escribe; verificar con `go test ./internal/cli ./internal/update`.
- [x] 10.3 Actualizar `MIGRATION/PROGRESS.md` con resultados y comandos ejecutados y hacer commit; verificar con `go vet ./...`, `golangci-lint run ./...`, `gofumpt -l .` vacío y `scripts/guard.sh`.

## 11. Fase 3g — TUI

- [x] 11.1 Implementar 3g1: app raíz, pila de pantallas, menú, tema ANSI, `NO_COLOR`, guardia 80x24, ayuda contextual, error de cuatro partes y diálogos; verificar con `go test ./internal/tui -run 'Menu|Common|Theme|Size'`.
- [x] 11.2 Implementar 3g2: flujo C1-C4 con selección, destino, revisión, progreso, cancelación, resumen y copia de ruta; verificar con `go test ./internal/tui -run Clone`.
- [x] 11.3 Implementar 3g3: flujo D0-D5 con fuentes de clones, validación de manifiesto, selección, conflictos, detalle por líneas, confirmación, holocrón y resumen; verificar con `go test ./internal/tui -run Deploy`.
- [x] 11.4 Implementar 3g4: U1, pantalla de Configuración, recuperación de configuración inválida y cableado del gating TUI en `internal/cli`; verificar con `go test ./internal/tui ./internal/cli`.
- [x] 11.5 Ejecutar `go test ./... -race`, la matriz de compatibilidad cruzada, actualizar `MIGRATION/PROGRESS.md`, hacer commit y crear el tag `migration/phase-3`; verificar con `git tag --list 'migration/phase-*'`.

## 12. Fase 4 — Release, rendimiento, documentación y corte

- [x] 12.1 Leer los specs de distribución, actualización y release y `AGENTS.md`; escribir `MIGRATION/phase4-notes.md`; verificar que resume el contrato de la fase.
- [x] 12.2 Crear `.goreleaser.yaml` con asset `kamino-arm64`, `CGO_ENABLED=0`, `-trimpath`, `-s -w` y versión inyectada; verificar con `goreleaser release --snapshot --clean` y `./dist/kamino-arm64 --version`.
- [x] 12.3 Adaptar sin ejecutar el workflow de release, el smoke test, `scripts/install.sh` y la lógica de `update` para el binario Go; verificar con el smoke test local en un HOME temporal y `go test ./internal/update ./internal/cli`.
- [x] 12.4 Ejecutar `hyperfine` con los mismos comandos del baseline y guardar `MIGRATION/benchmark-after.md` con tabla antes/después; verificar que CLI < 30 ms y TUI < 150 ms o documentar el perfilado y la iteración.
- [x] 12.5 Actualizar `README.md`, `docs/Diseño técnico.md`, `docs/Diseño funcional.md`, `AGENTS.md` y los specs de distribución/actualización/release; verificar que los comandos documentados coinciden con `go test`, `golangci-lint`, `gofumpt` y el binario.
- [x] 12.6 Mover Python a `legacy/python/` con `git mv`, sin borrar nada, y dejar la raíz como proyecto Go; verificar con `git status --short`, `go build ./...` y que `legacy/python/src/kamino` existe.
- [x] 12.7 Preparar el commit final `feat!: migrate to Go` con footer `BREAKING CHANGE:` y crear el tag `migration/phase-4`; verificar con `git log -1 --format=%B`, `git tag --list 'migration/phase-*'` y que no se hizo push.
- [x] 12.8 Escribir `MIGRATION/REPORT.md` con estado por fase, tabla de tiempos, decisiones, bloqueos, cobertura de specs, riesgos residuales y pasos manuales; verificar con `go test ./... -race`, `scripts/guard.sh`, `goreleaser release --snapshot --clean`, la matriz de compatibilidad cruzada y que todas las secciones del informe existen.

## Workflow follow-up

- Revisar `MIGRATION/review-request-phase2.md`, `review-request-3a.md` y `review-request-3d.md` con otro modelo antes de archivar.
- Archivar el cambio `migrate-to-go` cuando la revisión y las puertas de calidad estén satisfechas.
- Pasos manuales pendientes: notarización, Homebrew tap y retirada de `legacy/python/`.
