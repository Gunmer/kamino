# Base de conocimiento de kamino

Índice de la documentación de diseño. Empieza aquí y abre solo el documento que necesites.

## Producto

| Documento | Léelo cuando |
| --- | --- |
| [producto/funcional.md](producto/funcional.md) | necesites saber qué hace kamino: comandos, opciones, conflictos, alcance y versiones |

## Técnico

| Documento | Léelo cuando |
| --- | --- |
| [tecnico/arquitectura.md](tecnico/arquitectura.md) | vayas a crear o mover paquetes, o a decidir en qué capa va algo |
| [tecnico/formatos.md](tecnico/formatos.md) | toques el TOML del catálogo, `manifest.json` o la estructura del archivo del clon |
| [tecnico/flujos.md](tecnico/flujos.md) | cambies el orden de clonar, implantar, actualizar o restaurar |
| [tecnico/pruebas-y-distribucion.md](tecnico/pruebas-y-distribucion.md) | escribas pruebas, goldens o toques el empaquetado y la release |
| [tecnico/plan-y-decisiones.md](tecnico/plan-y-decisiones.md) | quieras el historial de decisiones ya tomadas |

## TUI

| Documento | Léelo cuando |
| --- | --- |
| [tui/arquitectura.md](tui/arquitectura.md) | toques `internal/tui`: capas, contrato `Screen`, pila, tamaño, tema, goldens |
| [tui/notacion-e-inventario.md](tui/notacion-e-inventario.md) | necesites la lista de pantallas y cómo leer las maquetas |
| [tui/pantallas-clon.md](tui/pantallas-clon.md) | cambies las pantallas C1–C4 |
| [tui/pantallas-implantacion.md](tui/pantallas-implantacion.md) | cambies D0–D5 (clon, apps, conflictos, revisión por líneas, resumen) |
| [tui/pantallas-generales.md](tui/pantallas-generales.md) | cambies menú, actualización, ajustes, ayuda, errores o añadir app |
| [tui/teclas-y-estados.md](tui/teclas-y-estados.md) | cambies teclas, mensajes, scroll, campos, filtros, diálogos o `NO_COLOR` |
| [tui/plan.md](tui/plan.md) | revises el plan de implementación y los puntos pendientes de la TUI |

## Fuera de `docs/`

- `openspec/specs/`: comportamiento exigible por capacidad.
- `README.md` y `docs/guia/`: uso para personas usuarias.
- `AGENTS.md`: reglas mínimas para agentes.
