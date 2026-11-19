# Spec Delta

## ADDED Requirements

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
