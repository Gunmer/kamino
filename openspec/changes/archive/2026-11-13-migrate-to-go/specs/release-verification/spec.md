# Spec Delta

## MODIFIED Requirements

### Requirement: Estado inicial coherente de las pruebas

El repositorio de prueba DEBE (`MUST`) partir de versiones iniciales coherentes y de un historial anterior a la publicación simulada.

#### Scenario: Preparación de una publicación simulada

- **WHEN** se prepara el repositorio de prueba para una nueva publicación
- **THEN** la etiqueta, la versión inyectada en el binario, la versión declarada en el código y el manifiesto de release coinciden con la versión inicial.
- **AND** el historial previo contiene esa versión inicial y no contiene la nueva versión que se va a generar.

### Requirement: Integridad de la publicación verificada

Una nueva publicación DEBE (`MUST`) registrar sus cambios una sola vez, conservar el historial previo y mantener sincronizada la versión del proyecto.

#### Scenario: Publicación con cambio incompatible

- **GIVEN** un historial previo y un cambio publicable que incluye una descripción de incompatibilidad
- **WHEN** la prueba genera una nueva publicación
- **THEN** la nueva sección es única y contiene el cambio y su descripción de incompatibilidad, mientras que el historial anterior se conserva.
- **AND** la etiqueta, la versión inyectada en el binario y el manifiesto de release coinciden con la nueva versión.
- **AND** las notas de publicación corresponden únicamente a la nueva versión.

## ADDED Requirements

### Requirement: Verificación del binario de release

La verificación de publicación DEBE (`MUST`) construir el binario Go con la versión de la release, comprobar que el asset se llama `kamino-arm64`, que `--version` devuelve `kamino <versión>` y que `--help` funciona sin Python. La verificación de migración NO DEBE (`MUST NOT`) publicar, crear tags remotos ni subir assets.

#### Scenario: Construcción de snapshot

- **GIVEN** un árbol de trabajo limpio y una versión de release
- **WHEN** se ejecuta `goreleaser release --snapshot --clean`
- **THEN** se genera el asset `kamino-arm64`, `--version` muestra la versión esperada y `--help` termina con código 0

#### Scenario: Sin publicación durante la migración

- **GIVEN** una verificación de release en la rama `migrate/go`
- **WHEN** termina la construcción de snapshot
- **THEN** no se publica ninguna release, no se crea ningún tag remoto y no se sube ningún asset
