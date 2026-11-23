# Spec Delta

## ADDED Requirements

### Requirement: Validación de recetas de instalación

`kamino apps validate` y la carga del catálogo DEBEN (`MUST`) aplicar las mismas reglas a las recetas `[[install]]`: método permitido, URL `https` y `sha256` válido en `binary`, y `os`/`arch` reconocidos. Los errores DEBEN (`MUST`) informarse por archivo en español y terminar con código 1.

#### Scenario: Receta binary sin checksum

- **GIVEN** un archivo del catálogo de usuario con una receta `binary` sin `sha256`
- **WHEN** se ejecuta `kamino apps validate`
- **THEN** se informa el error de ese archivo y el comando termina con código 1.

#### Scenario: Catálogo con recetas válidas

- **GIVEN** definiciones con recetas `brew`, `installer` y `binary` correctas
- **WHEN** se ejecuta `kamino apps validate`
- **THEN** la validación termina con código 0.
