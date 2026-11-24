# Proposal

## Why

Un usuario que quiere ajustar una app del catálogo integrado (rutas, exclusiones, recetas de instalación) hoy solo puede crear una definición con el mismo id mediante `apps new --force` o «Añadir app». Ambas parten de una plantilla que solo conoce id, nombre, detección y rutas: se pierden las recetas `[[install]]` y todo lo demás hay que reescribirlo a mano.

## What Changes

- Nuevo comando `kamino apps customize <id>`: copia la definición integrada de `<id>` al catálogo de usuario como `<id>.toml`, sin cambios, para que el usuario la edite. Admite `--print` y `--force` con la misma semántica que `apps new`.
- Nueva opción «Personalizar app» en la pantalla de Configuración de la TUI: elige una app integrada y reutiliza la pantalla de autoría con el borrador sembrado con el TOML integrado.
- El reemplazo sigue siendo total por id (sin fusión): la copia del usuario sustituye a la integrada, igual que hoy.

## Capabilities

### New Capabilities

### Modified Capabilities
- `app-authoring`: se añaden el comando `apps customize` y la pantalla «Personalizar app».

## Impact

- `internal/catalog`: acceso al TOML integrado en bruto por id.
- `internal/app` (`apps.go`): nuevo caso de uso junto a `ExecuteAppsNew`.
- `internal/cli`: subcomando `apps customize` y ayuda.
- `internal/tui`: selector de app integrada, entrada en Configuración y reutilización de `appAuthoringScreen`; goldens nuevos.
- `internal/texts`, `docs/guia/catalogo.md` y `docs/tui/`: textos y documentación.
- Sin cambios en el formato de definición ni en la carga del catálogo.
