# Spec Delta

## ADDED Requirements

### Requirement: Filtro de D0 como campo

D0 DEBE (`MUST`) presentar el filtro `/` como un campo de una sola fila con etiqueta «Filtro», foco visible y cursor de edición. Al pulsar `/`, el foco DEBE pasar al campo; `Enter` DEBE aplicar el filtro y devolver el foco a la tabla y `Esc` DEBE limpiar el filtro y volver a la tabla. `F1` y `?` DEBEN abrir la ayuda sin insertar caracteres. La fila `Otro archivo...` DEBE seguir disponible aunque el filtro no deje clones visibles.

#### Scenario: Abrir el filtro de D0

- **GIVEN** D0 con el foco en la tabla
- **WHEN** se pulsa `/`
- **THEN** aparece la fila «Filtro» con el foco, el marcador y el cursor

#### Scenario: Aplicar el filtro

- **GIVEN** el campo de filtro con texto
- **WHEN** se pulsa `Enter`
- **THEN** la tabla muestra solo los clones coincidentes y el foco vuelve a la tabla

#### Scenario: Cerrar el filtro

- **GIVEN** el campo de filtro activo con texto
- **WHEN** se pulsa `Esc`
- **THEN** el filtro se limpia, la lista completa vuelve a mostrarse y el foco vuelve a la tabla

#### Scenario: Filtro sin coincidencias

- **GIVEN** un filtro que no coincide con ningún clon
- **WHEN** la tabla queda vacía
- **THEN** la fila `Otro archivo...` sigue visible y utilizable

#### Scenario: Ayuda desde el filtro

- **GIVEN** el campo de filtro con el foco y un valor escrito
- **WHEN** se pulsa `F1` o `?`
- **THEN** se abre la ayuda sin modificar el filtro

### Requirement: Ruta manual de D0 como campo

Al elegir `Otro archivo...`, D0 DEBE (`MUST`) mostrar la ruta manual como un campo de una sola fila con etiqueta «Ruta», foco visible, cursor y placeholder atenuado cuando esté vacío. Las rutas largas DEBEN desplazarse con `…` manteniendo visible el cursor. `Enter` DEBE validar la ruta y `Esc` DEBE volver a la tabla. Una ruta inválida DEBE conservar el valor y mostrar un `ERROR` en la zona de mensajes sin mover las filas.

#### Scenario: Abrir la ruta manual

- **GIVEN** la tabla de D0 con foco
- **WHEN** se elige `Otro archivo...` y se pulsa `Enter`
- **THEN** aparece la fila «Ruta» con el foco, el marcador y el cursor

#### Scenario: Ruta manual vacía

- **GIVEN** la ruta manual abierta y vacía
- **WHEN** se renderiza
- **THEN** aparece un placeholder atenuado y el cursor al inicio del campo

#### Scenario: Ruta manual larga

- **GIVEN** una ruta más ancha que el campo y el cursor al final
- **WHEN** se renderiza con foco
- **THEN** el inicio se abrevia con `…`, el final queda visible y el cursor sigue visible

#### Scenario: Ruta manual válida

- **GIVEN** el campo de ruta con un `.tar.gz` con manifiesto válido
- **WHEN** se pulsa `Enter`
- **THEN** se valida el manifiesto y se continúa a D1 sin extraer ni verificar la integridad

#### Scenario: Ruta manual inválida

- **GIVEN** el campo de ruta con una ruta inexistente o no regular
- **WHEN** se pulsa `Enter`
- **THEN** D0 permanece abierta, conserva el valor escrito y muestra el `ERROR` correspondiente en la zona de mensajes

#### Scenario: Volver de la ruta a la tabla

- **GIVEN** el campo de ruta con el foco
- **WHEN** se pulsa `Esc`
- **THEN** el foco vuelve a la tabla sin cancelar el flujo
