# Design

## Context

- El catálogo de usuario reemplaza por id a la definición integrada, entera y sin fusionar (`LoadCatalog`).
- `RenderTemplate` no puede reproducir una definición integrada: no conoce `[[install]]`, `command_aliases`, `install_marker`, etc. y no hay serializador `AppDefinition -> TOML`.
- Los TOML integrados están embebidos (`go:embed apps/*.toml`), así que el texto original, con comentarios, está disponible.
- `ExecuteAppsNew` + `CreateDefinition` ya validan, detectan duplicados, escriben de forma atómica y avisan del reemplazo. `appAuthoringScreen` ya implementa borrador, editor, validación y confirmación.

## Goals / Non-Goals

**Goals:**
- Partir de la definición integrada completa, recetas incluidas.
- Reutilizar validación, escritura atómica y avisos existentes.

**Non-Goals:**
- Override parcial o fusión con la definición integrada (decisión: copia completa).
- Serializar `AppDefinition` a TOML.
- Avisar de deriva cuando el catálogo integrado cambia respecto de la copia (posible mejora futura).
- Cambiar el formato de definición o la carga del catálogo.

## Decisions

1. **Semilla = TOML integrado en bruto.** Nueva función en `internal/catalog` que devuelve los bytes embebidos de un id integrado. Conserva comentarios y recetas sin serializador. Alternativa descartada: serializar la `AppDefinition` (código nuevo y pierde comentarios).
2. **Caso de uso `ExecuteAppsCustomize`** junto a `ExecuteAppsNew`, reutilizando `CreateDefinition(directory, content, force)`; `--print` devuelve el contenido sin escribir. Id inexistente: error de uso (código 2).
3. **TUI:** entrada «Personalizar app» en Configuración, selector de apps integradas y `newAppAuthoringScreen` con el borrador sembrado, para no duplicar el flujo. Diferencia: el borrador ya es válido, así que no se exige haberlo editado.
4. **CLI:** `apps customize <id>` (no un flag de `apps new`), porque el contrato y los códigos de salida de `apps new` exigen flags de detección y rutas que aquí no aplican.

## Risks / Trade-offs

- **Deriva:** la copia no recibe cambios futuros del catálogo integrado (p. ej. nuevas recetas). Se documenta en `docs/guia/catalogo.md`.
- **Nombre de archivo vs id integrado:** el acceso por id debe resolver por el `id` de la definición, no asumir `apps/<id>.toml`; se verifica al implementar.
- **Aviso «reemplaza la definición integrada»** aparece siempre tras personalizar; es esperado.
- **Suposiciones (no confirmadas):** nombre `customize` y que la TUI sea una opción nueva de Configuración, no un submodo de «Añadir app».
