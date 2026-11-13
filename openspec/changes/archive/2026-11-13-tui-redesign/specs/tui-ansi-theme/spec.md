# Spec Delta

## ADDED Requirements

### Requirement: Colores ANSI estrictos

La TUI DEBE (`MUST`) usar solo colores ANSI. NO DEBE (`MUST NOT`) usar hex, RGB ni `AdaptiveColor`. Todos los estilos DEBEN venir de un tema único; ninguna pantalla DEBE definir colores propios.

#### Scenario: Sin truecolor

- **WHEN** se inspecciona cualquier render de la TUI
- **THEN** no contiene secuencias de color truecolor ni valores hex o RGB

#### Scenario: Tema único

- **GIVEN** una pantalla nueva
- **WHEN** se renderiza
- **THEN** sus colores provienen del tema y no de estilos locales

### Requirement: Jerarquía visual ANSI

La jerarquía DEBE (`MUST`) ser: primario texto por defecto + negrita; secundario texto por defecto; terciario `Faint` o ANSI 8; acento ANSI 6; éxito e info ANSI 2; aviso ANSI 3; error ANSI 1. La selección DEBE usar `▌` + negrita y ningún estado DEBE depender solo del color.

#### Scenario: Jerarquía de texto

- **WHEN** se muestran textos primarios, secundarios y terciarios
- **THEN** se distinguen por peso o estilo además de por color

#### Scenario: Selección sin color

- **GIVEN** una fila seleccionada
- **WHEN** se muestra en monocromo
- **THEN** conserva el marcador `▌` y la negrita

### Requirement: Mensajes semánticos

La zona de mensajes DEBE (`MUST`) usar ANSI 2 para `INFO`, ANSI 3 para `AVISO` y ANSI 1 para `ERROR`. Cada mensaje DEBE incluir un icono y la etiqueta de su categoría; el color NO DEBE (`MUST NOT`) ser la única señal.

#### Scenario: INFO verde

- **GIVEN** un mensaje informativo
- **WHEN** se muestra
- **THEN** usa ANSI 2 e incluye el icono y la etiqueta `INFO`

#### Scenario: AVISO amarillo

- **GIVEN** un aviso
- **WHEN** se muestra
- **THEN** usa ANSI 3 e incluye el icono y la etiqueta `AVISO`

#### Scenario: ERROR rojo

- **GIVEN** un error recuperable
- **WHEN** se muestra
- **THEN** usa ANSI 1 e incluye el icono y la etiqueta `ERROR`

#### Scenario: Mensaje sin color

- **GIVEN** `NO_COLOR` definido
- **WHEN** se muestra un mensaje
- **THEN** conserva el icono y la etiqueta de la categoría

### Requirement: Equivalentes ASCII

Los símbolos `◉ ● ○ ◀ ▶ ▌ ▸ ▾ ─ · ⚠ ℹ ✕` DEBEN (`MUST`) tener equivalentes ASCII cuando no haya UTF-8 o `KAMINO_NO_LOGO` esté definido. `KAMINO_NO_LOGO` además DEBE ocultar el wordmark.

#### Scenario: Terminal sin UTF-8

- **GIVEN** una terminal sin codificación UTF-8
- **WHEN** se muestra cualquier pantalla
- **THEN** los símbolos se sustituyen por sus equivalentes ASCII

#### Scenario: KAMINO_NO_LOGO

- **GIVEN** `KAMINO_NO_LOGO` definido
- **WHEN** se muestra el menú
- **THEN** no se muestra el wordmark y los símbolos usan su equivalente ASCII

## MODIFIED Requirements

### Requirement: Tema sobrescribible

El tema `ansi-dark` DEBE (`MUST`) ser el valor por defecto. `TEXTUAL_THEME` DEBE (`MUST`) aceptar `ansi-dark` y `ansi-light`. Cualquier otro valor DEBE (`MUST`) caer a `ansi-dark` y la interfaz DEBE seguir siendo legible con icono y palabra en cada estado. `ansi-light` DEBE evitar `Faint` y usar ANSI 8 para el terciario.

#### Scenario: Selección explícita de otro tema

- **WHEN** se define `TEXTUAL_THEME=ansi-light` antes de abrir la TUI
- **THEN** la TUI usa el tema `ansi-light` en lugar de `ansi-dark`

#### Scenario: Tema no ANSI explícito

- **GIVEN** `TEXTUAL_THEME` con un valor distinto de `ansi-dark` o `ansi-light`
- **WHEN** se abre la TUI
- **THEN** la TUI usa `ansi-dark` y la interfaz sigue siendo legible con icono y palabra en cada estado

#### Scenario: Tema claro sin Faint

- **GIVEN** `TEXTUAL_THEME=ansi-light`
- **WHEN** se muestra un texto terciario
- **THEN** se usa ANSI 8 en lugar de `Faint`

### Requirement: NO_COLOR y monocromo

Con `NO_COLOR` definido, la TUI DEBE (`MUST`) mostrarse en monocromo y todos los estados DEBEN conservar su icono y su palabra. La ausencia de color NO DEBE ocultar información ni impedir la navegación. `NO_COLOR` DEBE desactivar los colores; puede conservar negrita y `Faint` como atributos. Los mensajes DEBEN conservar icono y etiqueta.

#### Scenario: NO_COLOR activo

- **GIVEN** `NO_COLOR` definido antes de abrir la TUI
- **WHEN** se abre la interfaz
- **THEN** se muestra en monocromo y todos los estados siguen siendo distinguibles por icono y palabra

#### Scenario: Navegación sin color

- **GIVEN** la TUI en monocromo
- **WHEN** se revisan conflictos o se confirma un despliegue
- **THEN** las decisiones, los avisos y los errores siguen siendo legibles sin depender del color

#### Scenario: Atributos sin color

- **GIVEN** `NO_COLOR` definido
- **WHEN** se muestra una fila seleccionada o un texto terciario
- **THEN** no hay color, pero se conservan la negrita, el marcador `▌` o `Faint` cuando aportan jerarquía
