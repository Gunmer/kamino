# Proposal

## Why

Kamino ya carga definiciones TOML de usuario desde `~/.config/kamino/apps/`, pero no ofrece forma de crearlas ni comprobarlas: hay que copiar el esquema del repositorio, adivinarlo y descubrir los errores al iniciar un flujo. Además, `catalog.dir` se guarda en Configuración pero la TUI lo ignora y la CLI usa siempre el directorio integrado, así que una app creada en la TUI puede no existir para `clone` o `deploy`. Un solo TOML inválido bloquea todo el catálogo.

## What Changes

- Nuevo núcleo de autoría compartido: plantilla comentada, validación por archivo, comprobación de duplicados y escritura atómica.
- Nuevo grupo `kamino apps`:
  - `apps new <id>` con flags obligatorios; no escribe nada si la definición no es válida. `--print` escribe a stdout y `--force` solo permite sobrescribir `<id>.toml`.
  - `apps validate [ruta]` valida un archivo, un directorio o el catálogo de usuario, con código 1 si algo falla.
  - `apps list` muestra el catálogo fusionado con origen y errores por archivo, sin fallar por definiciones inválidas.
- Nueva pantalla TUI «Añadir app», accesible desde Configuración: borrador `.toml.tmp`, editor (`$VISUAL`/`$EDITOR`/`vi`), validación al volver, confirmación de override o sobrescritura, rename atómico y recarga del catálogo.
- `catalog.dir` pasa a ser la ubicación única del catálogo de usuario: TUI, `clone`, `deploy` y `apps` lo leen. Un `config.toml` ausente o inválido cae al directorio integrado sin bloquear; solo los comandos `apps` avisan.
- Plantilla con la lista estándar de `exclude` desde una constante única; `apps validate` avisa si una definición no declara exclusiones.
- **BREAKING** (comportamiento): un `catalog.dir` configurado que hoy se ignora empieza a aplicarse. Si contiene TOMLs inválidos, `clone` o `deploy` fallarán hasta corregirlos.
- Documentación actualizada: README, `docs/diseno-tecnico.md`, ayuda principal y scripts de autocompletado.

## Capabilities

### New Capabilities
- `app-authoring`: creación, plantilla, validación e inspección de definiciones de usuario, tanto por CLI como por la TUI.

### Modified Capabilities
- `user-configuration`: `catalog.dir` deja de ser exclusivo de la TUI; la CLI lo lee; fallback ante configuración inválida, recarga tras guardar y acción «Añadir app» dentro de Configuración.
- `cli-options`: el comando `apps` aparece en la ayuda principal y en el autocompletado; sus subcomandos aceptan `-h`.

## Impact

- Código: `internal/catalog` (parseo, validación, plantilla e inspección), `internal/app` (resolución de `catalog.dir`, flujos de `apps` y creación atómica), `internal/cli` (subcomando), `internal/tui` (Configuración, pantalla de autoría, editor inyectable y recarga), `internal/texts` (Configuración y pantalla), `internal/config` (sin cambios de esquema).
- Documentación: `README.md` y `docs/diseno-tecnico.md`.
- Pruebas: unitarias de catálogo, app y CLI; goldens de Configuración y de la pantalla nueva; ciclo crear, validar y clonar.
- Sin dependencias nuevas: `tea.ExecProcess` ya está disponible en bubbletea v1.3.10.
