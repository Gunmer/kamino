# Especificación: verificación de publicaciones

## Purpose

Garantizar que las comprobaciones de publicación sean fiables y que una nueva versión conserve la integridad del historial y de sus notas.

## ADDED Requirements

### Requirement: Pruebas de publicación aisladas

La verificación automatizada de publicaciones DEBE (`MUST`) utilizar un repositorio de prueba independiente del historial y de la versión reales del proyecto.

#### Scenario: El historial real ya contiene la versión simulada

- **GIVEN** el historial real del proyecto contiene la versión que la prueba va a publicar
- **WHEN** se ejecuta la verificación de publicación
- **THEN** la prueba genera y verifica su propia entrada para esa versión, sin reutilizar la entrada real ni modificar el repositorio real.

### Requirement: Estado inicial coherente de las pruebas

El repositorio de prueba DEBE (`MUST`) partir de versiones iniciales coherentes y de un historial anterior a la publicación simulada.

#### Scenario: Preparación de una publicación simulada

- **WHEN** se prepara el repositorio de prueba para una nueva publicación
- **THEN** la etiqueta, los metadatos del proyecto, la versión del código y la del proyecto en el archivo de dependencias coinciden con la versión inicial.
- **AND** el historial previo contiene esa versión inicial y no contiene la nueva versión que se va a generar.

### Requirement: Integridad de la publicación verificada

Una nueva publicación DEBE (`MUST`) registrar sus cambios una sola vez, conservar el historial previo y mantener sincronizada la versión del proyecto.

#### Scenario: Publicación con cambio incompatible

- **GIVEN** un historial previo y un cambio publicable que incluye una descripción de incompatibilidad
- **WHEN** la prueba genera una nueva publicación
- **THEN** la nueva sección es única y contiene el cambio y su descripción de incompatibilidad, mientras que el historial anterior se conserva.
- **AND** la etiqueta, los metadatos, la versión del código y la versión del proyecto en el archivo de dependencias coinciden con la nueva versión.
- **AND** las notas de publicación corresponden únicamente a la nueva versión.

### Requirement: Repetición sin cambios publicables

Repetir la verificación sin cambios publicables nuevos DEBE (`MUST`) preservar la publicación ya generada.

#### Scenario: Repetición de la publicación

- **GIVEN** una publicación ya generada y ningún cambio publicable nuevo
- **WHEN** se repite la verificación
- **THEN** no se crea otro commit ni otra etiqueta y el historial permanece sin modificaciones.
