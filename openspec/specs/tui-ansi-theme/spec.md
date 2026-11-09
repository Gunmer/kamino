# tui-ansi-theme Specification

## Purpose

Hacer que la TUI de despliegue respete la paleta ANSI del terminal en el que se ejecuta, sin imponer colores truecolor propios.

## Requirements

### Requirement: Paleta ANSI del terminal

La TUI DEBE (`MUST`) usar un tema ANSI de Textual para que el fondo, el texto y los colores de énfasis provengan de la paleta del terminal. NO DEBE (`MUST NOT`) imponer colores truecolor propios para esos elementos, incluidos el wordmark, las cápsulas, los iconos y los estados. Los colores semánticos DEBEN ser verde para éxito, amarillo para aviso, rojo para error y gris para secundario; el original DEBE ir en amarillo y además con el glifo `◉`.

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

El tema `ansi-dark` DEBE (`MUST`) ser el valor por defecto, pero la variable de entorno `TEXTUAL_THEME` DEBE (`MUST`) poder sobrescribirlo con cualquier tema válido de Textual. Elegir un tema no ANSI DEBE considerarse una decisión explícita de la persona usuaria y no un incumplimiento del valor por defecto.

#### Scenario: Selección explícita de otro tema

- **WHEN** se define `TEXTUAL_THEME=ansi-light` antes de abrir la TUI
- **THEN** la TUI usa el tema `ansi-light` en lugar de `ansi-dark`

#### Scenario: Tema no ANSI explícito

- **GIVEN** `TEXTUAL_THEME` con un tema no ANSI
- **WHEN** se abre la TUI
- **THEN** se respeta la elección y la interfaz sigue siendo legible con icono y palabra en cada estado

### Requirement: NO_COLOR y monocromo

Con `NO_COLOR` definido, la TUI DEBE (`MUST`) mostrarse en monocromo y todos los estados DEBEN conservar su icono y su palabra. La ausencia de color NO DEBE ocultar información ni impedir la navegación.

#### Scenario: NO_COLOR activo

- **GIVEN** `NO_COLOR` definido antes de abrir la TUI
- **WHEN** se abre la interfaz
- **THEN** se muestra en monocromo y todos los estados siguen siendo distinguibles por icono y palabra

#### Scenario: Navegación sin color

- **GIVEN** la TUI en monocromo
- **WHEN** se revisan conflictos o se confirma un despliegue
- **THEN** las decisiones, los avisos y los errores siguen siendo legibles sin depender del color
