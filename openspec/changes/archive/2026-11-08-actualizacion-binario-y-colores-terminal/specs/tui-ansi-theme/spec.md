# Spec Delta

## Purpose

Hacer que la TUI de restauración respete la paleta ANSI del terminal en el que se ejecuta, sin imponer colores truecolor propios.

## ADDED Requirements

### Requirement: Paleta ANSI del terminal

La TUI de restauración DEBE (`MUST`) usar un tema ANSI de Textual para que el fondo, el texto y los colores de énfasis provengan de la paleta del terminal. NO DEBE (`MUST NOT`) imponer colores truecolor propios para esos elementos.

#### Scenario: Tema ANSI activo

- **WHEN** se abre la TUI de restauración
- **THEN** el tema activo es `ansi-dark` y los colores ANSI nativos están habilitados

#### Scenario: Terminal con paleta personalizada

- **WHEN** la terminal usa una paleta ANSI personalizada
- **THEN** el fondo, los bordes, los botones y los estados de la TUI usan los colores de esa paleta

### Requirement: Tema sobrescribible

El tema `ansi-dark` DEBE (`MUST`) ser el valor por defecto, pero la variable de entorno `TEXTUAL_THEME` DEBE (`MUST`) poder sobrescribirlo con cualquier tema válido de Textual.

#### Scenario: Selección explícita de otro tema

- **WHEN** se define `TEXTUAL_THEME=ansi-light` antes de abrir la TUI
- **THEN** la TUI usa el tema `ansi-light` en lugar de `ansi-dark`
