# Propuesta

## Por qué

Kamino tarda demasiado en arrancar: el binario `onefile` de PyInstaller medido en este equipo tarda alrededor de 6,5 s en `--version`, mientras que el objetivo es un arranque de CLI inferior a 30 ms y de TUI inferior a 150 ms. La lógica de clonación, despliegue, conflictos, catálogo y actualización ya está cubierta por 99 requisitos y 327 escenarios, y el código Python es la referencia ejecutable. Migrar a un binario Go estático reduce el arranque y elimina la extracción de PyInstaller, pero debe hacerse por fases para no cambiar el comportamiento observable.

## What Changes

- **BREAKING**: la distribución pasa de Python/PyInstaller/`uv`/`pip` a un binario Go estático para macOS Apple Silicon. El asset sigue llamándose `kamino-arm64`; `scripts/install.sh` y `kamino update` conservan su contrato.
- **BREAKING**: `uv` y `pip` dejan de ser vías de instalación. Las instalaciones desde código fuente quedan para desarrollo y `kamino update` las rechaza igual que hoy rechaza una instalación no autocontenida.
- **BREAKING**: la TUI Go mantiene `ansi-dark` como tema por defecto y admite `ansi-light`; `TEXTUAL_THEME` con otros valores cae a `ansi-dark`, porque no se porta el catálogo de temas de Textual.
- Se añade un proyecto Go en la raíz (`go.mod`, `cmd/kamino`, `internal/...`, `Makefile`, `.golangci.yml`, `.goreleaser.yaml`) que convive con Python hasta la fase de corte.
- Se implementan las mismas capacidades por capas: `paths`, `archive`, `catalog`, `detect`, `plan`, `ghostty`, `snapshot`, `clone`, `conflicts`, `deploy`, `texts`, `progress`, `cli`, `update` y `tui`.
- Se añaden pruebas derivadas de los specs y de fixtures doradas generadas desde Python, incluida la compatibilidad cruzada de clones y holocrones.
- Se conservan el formato `format_version = 1`, el manifiesto, las rutas relativas al HOME, los códigos 0/1/2/3, el orden de seguridad y las garantías de `dry_run`.
- Se añaden presupuestos de arranque medibles y un benchmark antes/después.
- Se adaptan el workflow de release, el smoke test y la verificación de publicación al binario Go, sin publicar nada durante la migración.

## Capabilities

### New Capabilities

- `go-distribution`: binario Go estático, asset de release, presupuestos de arranque, instalación y actualización compatibles, y transición desde el binario Python legado.

### Modified Capabilities

- `self-update`: el binario autocontenido pasa a ser Go; las instalaciones desde código se rechazan; se elimina la vía `uv`/`pip`; un binario Python legado debe poder actualizar al asset Go.
- `release-verification`: la verificación de publicación se adapta al nuevo layout Go, al binario y al asset, conservando el aislamiento y la integridad del historial.
- `tui-ansi-theme`: se fija el contrato de `TEXTUAL_THEME` para la TUI Go: `ansi-dark` por defecto, `ansi-light` admitido y valores desconocidos con caída a `ansi-dark`.

## Impact

- Nuevo árbol Go: `cmd/kamino`, `internal/{cli,app,catalog,detect,paths,plan,ghostty,clone,deploy,archive,conflicts,snapshot,update,progress,config,defaults,texts,tui,ui,version,buildinfo}`.
- Nuevos `go.mod`, `Makefile`, `.golangci.yml`, `.goreleaser.yaml`, `scripts/gen_golden.py`, `scripts/guard.sh`, `scripts/check_catalog_sync.sh`, `testdata/golden/` y `MIGRATION/`.
- Python permanece en `src/`, `tests/` y `scripts/` hasta la Fase 4; después se mueve a `legacy/python/` con `git mv`, sin borrarse.
- Documentación: `README.md`, `docs/Diseño técnico.md`, `docs/Diseño funcional.md`, `AGENTS.md` y los specs de distribución, actualización y release.
- No cambian el formato del archivo, el manifiesto, el catálogo, la detección, la planificación, los conflictos, las instantáneas ni los flujos visibles de clonación y despliegue.
- El corte final es un commit `feat!: migrate to Go` con `BREAKING CHANGE:`; no se publica ni se fusiona a `main`.
