# Spec Delta

## RENAMED Requirements

- FROM: `### Requirement: Decisión por bloque de cambio`
- TO: `### Requirement: Decisión por línea dentro del bloque`

- FROM: `### Requirement: Composición del archivo por bloques`
- TO: `### Requirement: Composición del archivo por líneas`

## MODIFIED Requirements

### Requirement: Tabla de conflictos

La tabla de conflictos DEBE (`MUST`) listar solo los archivos en conflicto y mostrar los conteos de archivos nuevos, idénticos y no soportados. Cada fila DEBE (`MUST`) ofrecer revisar, aplicar u omitir el archivo. En un conflicto de texto, la fila DEBE (`MUST`) indicar cuántos bloques y cuántas líneas cambiadas contiene.

#### Scenario: Tabla con solo conflictos y conteos

- **GIVEN** una restauración con tres conflictos, cuatro archivos nuevos y diez idénticos
- **WHEN** se muestra la tabla
- **THEN** la tabla lista solo los tres conflictos y muestra los conteos de nuevos, idénticos y no soportados.

#### Scenario: Acciones por archivo

- **GIVEN** un archivo en conflicto en la tabla
- **WHEN** la persona selecciona una acción
- **THEN** puede elegir revisar el detalle, aplicar el respaldo completo u omitir el archivo.

#### Scenario: Detalle de un conflicto de texto

- **WHEN** la tabla muestra un conflicto de texto
- **THEN** la fila indica el número de bloques y de líneas cambiadas, y la decisión de revisión cuenta las líneas del respaldo.

### Requirement: Decisión por línea dentro del bloque

En la vista de detalle, cada bloque DEBE (`MUST`) mostrar el contexto una sola vez y cada línea cambiada DEBE (`MUST`) ser una fila con la versión local y la del respaldo. Cada fila DEBE (`MUST`) alternar entre local y respaldo. El contexto y la cabecera NO DEBEN (`MUST NOT`) ser decidibles. NO DEBE (`MUST NOT`) existir un estado sin decidir: lo mostrado DEBE (`MUST`) ser la decisión. La vista NO DEBE (`MUST NOT`) mostrar una previsualización completa del archivo.

#### Scenario: Estado inicial del bloque

- **WHEN** se abre el detalle de un archivo en conflicto
- **THEN** todas las líneas cambiadas de sus bloques muestran la versión del respaldo como decisión inicial.

#### Scenario: Alternar un bloque

- **WHEN** la persona aplica una acción de bloque al respaldo o al local
- **THEN** todas las líneas cambiadas del bloque actual toman ese lado y los demás bloques conservan sus decisiones.

#### Scenario: Alternar una línea

- **WHEN** la persona alterna una línea cambiada
- **THEN** esa línea pasa de la versión del respaldo a la local, o viceversa, y el resto del bloque conserva sus decisiones.

#### Scenario: Filas de inserción y eliminación

- **GIVEN** un bloque con una línea que solo existe en el local o solo en el respaldo
- **WHEN** la persona alterna esa fila
- **THEN** la línea se conserva o se omite según el lado elegido.

#### Scenario: Sin previsualización completa

- **WHEN** la persona revisa un archivo en el detalle
- **THEN** la interfaz muestra el diff por bloques con el contexto una sola vez y no una vista previa del archivo final.

### Requirement: Composición del archivo por líneas

El archivo resultante DEBE (`MUST`) componerse con las líneas elegidas del respaldo y las conservadas del local, permitiendo mezclar líneas dentro de un mismo bloque. DEBE (`MUST`) conservar el estilo de final de línea local y normalizar CRLF solo para comparar.

#### Scenario: Mezcla de bloques

- **GIVEN** un archivo con dos bloques en conflicto
- **WHEN** la persona elige líneas del respaldo en el primero y conserva líneas locales en el segundo
- **THEN** el archivo resultante contiene las líneas elegidas de cada bloque en el orden original.

#### Scenario: Mezcla de líneas dentro de un mismo bloque

- **GIVEN** un bloque con dos líneas cambiadas
- **WHEN** la persona elige el respaldo para una y conserva el local para la otra
- **THEN** el archivo resultante contiene una línea del respaldo y la otra del local, en el orden del bloque.

#### Scenario: Todos los bloques del respaldo

- **GIVEN** un archivo con todas sus líneas decididas como respaldo
- **WHEN** se compone el resultado
- **THEN** el archivo resultante coincide con el respaldo.

#### Scenario: Todos los bloques locales

- **GIVEN** un archivo con todas sus líneas decididas como local
- **WHEN** se compone el resultado
- **THEN** el archivo resultante coincide con el local.

#### Scenario: Estilo de final de línea local

- **GIVEN** un archivo local con finales CRLF y un respaldo con finales LF
- **WHEN** se toma una línea del respaldo
- **THEN** la línea añadida usa el estilo de final de línea local.

#### Scenario: Salto de línea final en una mezcla

- **GIVEN** una mezcla en la que una línea sin salto de línea final queda antes de otra línea elegida
- **WHEN** se compone el resultado
- **THEN** la línea intermedia recibe el EOL dominante local y solo la última línea puede conservar la ausencia de salto final.

### Requirement: Modo y clasificación del resultado

Si el contenido final difiere del local, el archivo DEBE (`MUST`) recibir el modo del respaldo. Si la composición queda idéntica al local, NO DEBE (`MUST NOT`) escribirse ni cambiar permisos. El resultado DEBE (`MUST`) clasificar cada archivo como restaurado, mezclado u omitido.

#### Scenario: Mezcla que cambia el contenido

- **GIVEN** una mezcla que difiere del local y del respaldo
- **WHEN** se aplica la restauración
- **THEN** el archivo se escribe con el modo del respaldo y se clasifica como mezclado.

#### Scenario: Mezcla idéntica al local

- **GIVEN** una composición que coincide con el contenido local
- **WHEN** se aplica la restauración
- **THEN** el archivo no se escribe, sus permisos no cambian y se clasifica como omitido.

#### Scenario: Aplicación completa del respaldo

- **GIVEN** un archivo decidido para aplicar el respaldo completo
- **WHEN** se aplica la restauración
- **THEN** el archivo se escribe con el modo del respaldo y se clasifica como restaurado.

#### Scenario: Revisión con todas las líneas en respaldo

- **GIVEN** un archivo revisado línea a línea con todas sus líneas decididas como respaldo
- **WHEN** se aplica la restauración
- **THEN** el archivo se escribe con el modo del respaldo y se clasifica como restaurado.

### Requirement: Conflictos sin diff de líneas

Los archivos binarios, los archivos de texto de más de 5 MB y los enlaces con enlace DEBEN resolverse como archivo entero: aplicar el respaldo u omitir. NO DEBEN mostrar un diff de líneas. La fila DEBE (`MUST`) indicar el motivo y los datos disponibles del destino y del respaldo.

#### Scenario: Archivo binario

- **GIVEN** un conflicto entre un archivo binario local y uno del respaldo
- **WHEN** se muestra la tabla
- **THEN** el archivo se resuelve con aplicar u omitir y no ofrece revisión por líneas.

#### Scenario: Archivo de texto grande

- **GIVEN** un conflicto entre archivos de texto de más de 5 MB
- **WHEN** se muestra la tabla
- **THEN** el archivo se resuelve con aplicar u omitir y no ofrece revisión por líneas.

#### Scenario: Enlace con enlace

- **GIVEN** un conflicto entre dos enlaces simbólicos con destinos distintos
- **WHEN** se muestra la tabla
- **THEN** la fila muestra el destino local y el del respaldo y se resuelve con aplicar u omitir.

## ADDED Requirements

### Requirement: Acciones globales y por bloque en el detalle

La vista de detalle DEBE (`MUST`) ofrecer acciones para aplicar todas las líneas del archivo o solo las del bloque actual al respaldo o al local. Las acciones por bloque NO DEBEN (`MUST NOT`) modificar las decisiones de otros bloques.

#### Scenario: Todo el archivo al respaldo

- **WHEN** la persona elige la acción global de respaldo
- **THEN** todas las líneas cambiadas del archivo quedan en respaldo.

#### Scenario: Todo el archivo al local

- **WHEN** la persona elige la acción global de local
- **THEN** todas las líneas cambiadas del archivo quedan en local.

#### Scenario: Bloque actual al respaldo

- **WHEN** la persona elige la acción de bloque para el respaldo
- **THEN** todas las líneas cambiadas del bloque actual quedan en respaldo y los demás bloques conservan sus decisiones.

#### Scenario: Bloque actual al local

- **WHEN** la persona elige la acción de bloque para el local
- **THEN** todas las líneas cambiadas del bloque actual quedan en local y los demás bloques conservan sus decisiones.
