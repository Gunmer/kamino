# Spec Delta

## ADDED Requirements

### Requirement: El clon nunca guarda binarios de aplicaciones

La clonación NO DEBE (`MUST NOT`) incluir ejecutables ni bundles de aplicación (`.app`) en el clon, aunque una definición del catálogo, incluida una de usuario, los declare en `paths`, `optional_paths` u `optional_patterns`. Los elementos rechazados DEBEN (`MUST`) informarse como no clonados.

#### Scenario: Definición de usuario que apunta a un binario

- **GIVEN** una definición de usuario cuyo `paths` apunta a un ejecutable bajo `.local/bin`
- **WHEN** se planifica y escribe el clon
- **THEN** el ejecutable no se incluye en el archivo y se informa que no se clonó.

#### Scenario: Bundle de aplicación

- **GIVEN** una ruta declarada que resuelve a un directorio `.app`
- **WHEN** se planifica el clon
- **THEN** el bundle no se incluye y se informa que no se clonó.

#### Scenario: Configuración junto a un binario

- **GIVEN** una app con un archivo de ajustes permitido y un binario en el mismo directorio declarado
- **WHEN** se clona la app
- **THEN** se incluye el archivo de ajustes y se excluye el binario.
