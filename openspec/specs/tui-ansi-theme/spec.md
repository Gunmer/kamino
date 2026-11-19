# tui-ansi-theme Specification

## Purpose

Hacer que la TUI de despliegue respete la paleta ANSI del terminal en el que se ejecuta, sin imponer colores truecolor propios.

## Requirements

### Requirement: Paleta ANSI del terminal

La TUI DEBE (`MUST`) usar un tema ANSI para que el fondo, el texto y los colores de énfasis provengan de la paleta del terminal. NO DEBE (`MUST NOT`) imponer colores truecolor propios para esos elementos, incluidos el wordmark, las cápsulas, los iconos y los estados. Los colores semánticos DEBEN ser verde para éxito, amarillo para aviso, rojo para error y gris para secundario; el original DEBE ir en amarillo y además con el glifo `◉`.

#### Scenario: Tema ANSI activo

- **WHEN** se abre la TUI
- **THEN** el tema activo es `ansi-dark` y los colores ANSI nativos están habilitados

#### Scenario: Terminal con paleta personalizada

- **WHEN** la terminal usa una paleta ANSI personalizada
- **THEN** el fondo, los bordes, los controles, el wordmark, las cápsulas y los estados de la TUI usan los colores de esa paleta

#### Scenario: Estado sin depender del color

- **GIVEN** un estado de éxito, aviso o error
- **WHEN** se muestra en la TUI
- **THEN** incluye un icono y una palabra además del color

### Requirement: Tema sobrescribible

El tema `ansi-dark` DEBE (`MUST`) ser el valor por defecto. `KAMINO_THEME` DEBE (`MUST`) aceptar `ansi-dark` y `ansi-light`. Cualquier otro valor DEBE (`MUST`) caer a `ansi-dark` y la interfaz DEBE seguir siendo legible con icono y palabra en cada estado. `ansi-light` DEBE evitar `Faint` y usar ANSI 8 para el terciario.

#### Scenario: Selección explícita de otro tema

- **WHEN** se define `KAMINO_THEME=ansi-light` antes de abrir la TUI
- **THEN** la TUI usa el tema `ansi-light` en lugar de `ansi-dark`

#### Scenario: Tema no ANSI explícito

- **GIVEN** `KAMINO_THEME` con un valor distinto de `ansi-dark` o `ansi-light`
- **WHEN** se abre la TUI
- **THEN** la TUI usa `ansi-dark` y la interfaz sigue siendo legible con icono y palabra en cada estado

#### Scenario: Tema claro sin Faint

- **GIVEN** `KAMINO_THEME=ansi-light`
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

### Requirement: Estilos de los campos de entrada

La TUI DEBE (`MUST`) obtener del tema los estilos de sus campos de texto. Con foco, el campo DEBE mostrar el marcador `▌`, un fondo resaltado de la paleta ANSI, la etiqueta en negrita y el valor en color de acento. Sin foco, NO DEBE mostrar marcador ni fondo y DEBE usar texto atenuado. Con error, la etiqueta y el valor DEBEN usar color de error. NO DEBE (`MUST NOT`) usar truecolor, hex ni RGB.

#### Scenario: Campo con foco

- **GIVEN** un campo de texto con el foco
- **WHEN** se renderiza su fila
- **THEN** aparece el marcador `▌`, el fondo resaltado, la etiqueta en negrita y el valor en color de acento

#### Scenario: Campo sin foco

- **GIVEN** un campo de texto sin el foco
- **WHEN** se renderiza su fila
- **THEN** no aparece el marcador ni el fondo resaltado, y la etiqueta y el valor se muestran atenuados

#### Scenario: Campo con error

- **GIVEN** un campo con un error de validación
- **WHEN** se renderiza su fila
- **THEN** la etiqueta y el valor usan el color de error del tema y el campo conserva su única fila

#### Scenario: Fondo según el tema

- **GIVEN** los temas `ansi-dark` y `ansi-light`
- **WHEN** se enfoca un campo en cada uno
- **THEN** el fondo resaltado sale de la paleta ANSI del tema y el valor sigue siendo legible

#### Scenario: Sin truecolor

- **WHEN** se inspecciona el render de un campo en cualquier estado
- **THEN** no contiene secuencias de color truecolor ni valores hex o RGB

### Requirement: Foco de campo sin color

Con `NO_COLOR` definido, el foco de un campo DEBE (`MUST`) distinguirse por forma y no por color: DEBE mostrar el marcador `▌`, un cursor en vídeo inverso y la palabra «editando» junto a la etiqueta. El fondo resaltado NO DEBE (`MUST NOT`) ser la única señal de foco y los demás campos DEBEN seguir atenuados.

#### Scenario: Foco en monocromo

- **GIVEN** `NO_COLOR` definido y un campo con el foco
- **WHEN** se renderiza su fila
- **THEN** aparecen `▌`, el cursor en vídeo inverso y la palabra «editando», sin secuencias de color

#### Scenario: Campo inactivo en monocromo

- **GIVEN** `NO_COLOR` definido y un campo sin el foco
- **WHEN** se renderiza su fila
- **THEN** no aparecen el marcador, el cursor ni la palabra «editando»

#### Scenario: Cursor sin color

- **GIVEN** `NO_COLOR` definido y un campo con el cursor sobre un carácter
- **WHEN** se renderiza el valor
- **THEN** ese carácter usa vídeo inverso y el resto del valor conserva el estilo atenuado

### Requirement: Equivalentes ASCII de los campos

Sin codificación UTF-8 o con `KAMINO_NO_LOGO` definido, el marcador `▌` DEBE (`MUST`) sustituirse por `>` o `|`, la elipsis `…` por `...` y el cursor en vídeo inverso DEBE conservarse. La etiqueta, la palabra «editando» y el valor DEBEN seguir siendo legibles.

#### Scenario: Campo enfocado en ASCII

- **GIVEN** una terminal sin UTF-8 o `KAMINO_NO_LOGO` definido
- **WHEN** se renderiza un campo con foco
- **THEN** el marcador es `>` o `|`, no se usa `…` y el cursor sigue distinguiéndose por vídeo inverso

#### Scenario: Ruta larga en ASCII

- **GIVEN** una ruta más ancha que el campo y una terminal sin UTF-8
- **WHEN** se renderiza el valor desplazado
- **THEN** la elipsis inicial es `...` y el cursor permanece visible
