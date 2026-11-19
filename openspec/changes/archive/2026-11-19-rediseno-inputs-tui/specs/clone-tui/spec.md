# Spec Delta

## ADDED Requirements

### Requirement: Filtro de C1 como campo

C1 DEBE (`MUST`) presentar el filtro `/` como un campo de una sola fila con etiqueta «Filtro», foco visible y cursor de edición. Al pulsar `/`, el foco DEBE pasar al campo. Escribir DEBE editar el valor y `Enter` DEBE aplicar el filtro y devolver el foco a la tabla; `Esc` DEBE limpiar el filtro y cerrar el campo. `F1` y `?` DEBEN abrir la ayuda sin modificar el valor.

#### Scenario: Abrir el filtro

- **GIVEN** C1 con el foco en la tabla
- **WHEN** se pulsa `/`
- **THEN** aparece una fila «Filtro» con el foco, el marcador y el cursor

#### Scenario: Aplicar el filtro

- **GIVEN** el campo de filtro con texto
- **WHEN** se pulsa `Enter`
- **THEN** la tabla muestra solo las apps coincidentes y el foco vuelve a la tabla

#### Scenario: Cerrar el filtro

- **GIVEN** el campo de filtro activo con texto
- **WHEN** se pulsa `Esc`
- **THEN** el filtro se limpia, la tabla vuelve a mostrar todas las apps y el foco vuelve a la tabla

#### Scenario: Filtrar con cursor en medio

- **GIVEN** el campo de filtro con texto y el cursor en una posición intermedia
- **WHEN** se escribe o se pulsa `Retroceso`
- **THEN** el cambio ocurre en la posición del cursor y la tabla sigue respondiendo al filtro resultante

#### Scenario: Ayuda desde el filtro

- **GIVEN** el campo de filtro con el foco y un valor escrito
- **WHEN** se pulsa `F1` o `?`
- **THEN** se abre la ayuda sin insertar caracteres en el filtro

### Requirement: Campos de C2

C2 DEBE (`MUST`) presentar la ruta manual y el nombre del archivo como campos compartidos de una sola fila, con etiquetas «Destino» y «Nombre». La ruta vacía DEBE mostrar un placeholder atenuado y las rutas largas DEBEN desplazarse con `…` manteniendo visible el cursor. `Enter` en la ruta DEBE pasar al nombre y `Enter` en el nombre DEBE validar y continuar. Los errores DEBEN ir a la zona de mensajes sin mover las filas de los campos.

#### Scenario: Ruta manual con foco

- **GIVEN** C2 con «Indicar carpeta» elegida
- **WHEN** la ruta manual recibe el foco
- **THEN** la fila «Destino» muestra `▌`, fondo resaltado, etiqueta en negrita y el cursor en el valor

#### Scenario: Nombre con foco

- **GIVEN** C2 con una carpeta elegida
- **WHEN** el campo del nombre recibe el foco
- **THEN** la fila «Nombre» muestra el foco y el nombre sugerido con su cursor

#### Scenario: Ruta manual vacía

- **GIVEN** C2 con «Indicar carpeta» elegida y la ruta vacía
- **WHEN** se renderiza el campo
- **THEN** aparece un placeholder atenuado y, si tiene el foco, el cursor al inicio

#### Scenario: Ruta manual larga

- **GIVEN** una ruta manual más ancha que el campo y el cursor al final
- **WHEN** se renderiza con foco
- **THEN** el inicio se abrevia con `…`, el final queda visible y el cursor sigue visible

#### Scenario: Editar en medio del nombre

- **GIVEN** el campo del nombre con foco y el cursor entre dos caracteres
- **WHEN** se escribe o se pulsa `Retroceso`
- **THEN** el cambio ocurre en la posición del cursor y la ruta completa se actualiza

#### Scenario: Error de ruta en C2

- **GIVEN** una ruta manual o un nombre inválido
- **WHEN** se escribe en C2
- **THEN** la zona de mensajes muestra un `ERROR` y las filas de los campos conservan su posición

#### Scenario: Continuar desde los campos

- **GIVEN** el campo de ruta manual con un valor
- **WHEN** se pulsa `Enter`
- **THEN** el foco pasa al nombre; un `Enter` posterior valida el destino y continúa el flujo
