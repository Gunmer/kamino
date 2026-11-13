# Proposal

## Why

La TUI actual funciona, pero su presentación está concentrada en un `Model` que
mezcla estado, navegación y renderizado ANSI propio. Cada pantalla inventa su
layout, sus colores y su leyenda; no hay un marco común, un tema centralizado ni
golden files. El rediseño aprobado en `docs/Diseño tui.md` necesita una base
común para que las pantallas sean consistentes, verificables y mantenibles.

## What Changes

- Migrar la TUI a Bubble Tea + Lip Gloss + `bubbles` (`key`, `help`).
- Añadir `theme` (tokens y estilos ANSI) y `Frame` (header, body, mensajes,
  footer, separadores, guardia de tamaño y `tea.WindowSizeMsg`).
- Introducir el contrato `Screen` y la pila de pantallas en `App`; renombrar el
  enum actual a `ScreenID`.
- Añadir la zona de mensajes `INFO`/`AVISO`/`ERROR` sobre el footer.
- Rediseñar C1 (tabla), C2 (tres opciones de carpeta + nombre + diálogo),
  C3 (revisión, progreso y resultado en la misma tabla; C4 absorbido),
  D2 (tabla), D3 (archivo completo en dos columnas) y D4 (tabla por app).
- Derivar menú, D0, D1, D5, U1, Configuración, Ayuda, Error y aviso de terminal
  pequeña desde el `Frame` y el tema.
- Mover a `tea.Cmd` las operaciones que hoy bloquean la UI (recuento,
  planificación, escritura, despliegue y actualización).
- Añadir golden files a 80x24, 100x26, 140x40 y 200x50, más `NO_COLOR` y ASCII.
- Actualizar `docs/Diseño técnico.md`, `docs/Diseño funcional.md` y `AGENTS.md`.
- **BREAKING** (interno): el runner de terminal crudo y el `Model` monolítico se
  sustituyen por Bubble Tea y la pila de `Screen`. No cambia la interfaz pública
  de la CLI.
- **BREAKING** (TUI): C2 retira `F2`/`F3`/`F4`; las tres opciones de carpeta los
  sustituyen. `KAMINO_NO_LOGO` pasa a forzar también los símbolos ASCII.

## Capabilities

### New Capabilities

Ninguna.

### Modified Capabilities

- `tui-navigation`: marco, contrato de pantalla, pila, zona de mensajes, footer y
  ayuda desde `Keys()`, menú y degradación por altura.
- `tui-ansi-theme`: paleta ANSI estricta, jerarquía semántica, estilos de
  mensajes, `NO_COLOR` y símbolos ASCII.
- `clone-tui`: C1 como tabla, C2 con tres opciones de carpeta y nombre, C3
  absorbe C4 y muestra revisión, progreso y resultado; mensajes.
- `deploy-conflicts`: D2 como tabla, D3 con archivo completo, D4 por app y
  protegidos; mensajes.

## Impact

- `internal/tui`: se reestructura en `App`, `Frame`, `Screen`, `theme`,
  `layout`, pantallas y goldens. Se retiran el lector de teclas, el modo raw y
  el render ANSI propio.
- `internal/texts`: nuevos textos, etiquetas de mensajes, cabeceras de tablas y
  descripciones de `key.Binding`.
- `go.mod`/`go.sum`: Bubble Tea, Lip Gloss y `bubbles`.
- `internal/clone`, `internal/deploy`, `internal/update`, `internal/conflicts`:
  sin cambios de lógica; solo se invocan desde `tea.Cmd`.
- `docs/Diseño técnico.md`, `docs/Diseño funcional.md`, `AGENTS.md` y
  `docs/Diseño tui.md`.
- Pruebas: goldens, contrato de límites/ANSI, modelo de teclas y pila, benchmark
  de D3 con 20 000 líneas. HOME temporal; sin red.
