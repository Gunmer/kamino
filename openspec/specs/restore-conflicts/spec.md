# restore-conflicts Specification

## Purpose

Permitir que la restauración muestre los conflictos entre el destino y el respaldo y que la persona decida, bloque a bloque, qué se restaura y qué se conserva, sin sobrescribir cambios locales en silencio.

## Requirements

### Requirement: Detección de conflictos de contenido

La restauración DEBE (`MUST`) tratar como conflicto todo archivo regular existente cuyo contenido difiera del respaldo tras normalizar CRLF, y todo enlace simbólico cuyo destino difiera del enlace del respaldo. Los archivos nuevos, idénticos, no soportados, los cambios solo de modo y los cambios de tipo entre enlace y archivo regular NO DEBEN abrir la revisión de conflictos.

#### Scenario: Archivo regular con contenido distinto

- **WHEN** el destino existe, es un archivo regular y su contenido difiere del respaldo
- **THEN** el archivo se presenta como conflicto y requiere una decisión antes de aplicar la restauración.

#### Scenario: Diferencia solo de CRLF

- **GIVEN** un destino y un respaldo con el mismo contenido pero finales de línea CRLF y LF
- **WHEN** se planifica la restauración
- **THEN** el archivo se considera idéntico y no se presenta como conflicto.

#### Scenario: Enlace con destino distinto

- **GIVEN** un destino y un respaldo que son enlaces simbólicos con destinos distintos
- **WHEN** se planifica la restauración
- **THEN** el enlace se presenta como conflicto de archivo entero.

#### Scenario: Archivo nuevo

- **GIVEN** un archivo del respaldo que no existe en el destino
- **WHEN** se planifica la restauración
- **THEN** se crea sin pedir una decisión de conflicto.

#### Scenario: Archivo idéntico

- **GIVEN** un destino y un respaldo con el mismo contenido
- **WHEN** se planifica la restauración
- **THEN** el archivo se omite sin pedir una decisión de conflicto.

#### Scenario: Cambio solo de modo

- **GIVEN** un destino y un respaldo con el mismo contenido pero modos distintos
- **WHEN** se planifica la restauración
- **THEN** el archivo no se presenta como conflicto y se mantiene el comportamiento existente.

#### Scenario: Cambio de tipo entre enlace y archivo regular

- **GIVEN** un destino que es un enlace simbólico y un respaldo que es un archivo regular, o al contrario
- **WHEN** se planifica la restauración
- **THEN** el archivo no se presenta como conflicto y se mantiene el comportamiento de sobrescritura existente.

### Requirement: Selección de apps en la restauración interactiva

La restauración interactiva DEBE (`MUST`) integrar la selección de apps en la misma interfaz que la revisión de conflictos. Por defecto NO DEBE (`MUST NOT`) seleccionar ninguna app, DEBE (`MUST`) permitir seleccionar todas o ninguna y DEBE (`MUST`) omitir la pantalla de selección cuando se proporciona `--apps`.

#### Scenario: Selección inicial vacía

- **GIVEN** una restauración interactiva con apps instaladas en el manifiesto
- **WHEN** se abre la pantalla de selección
- **THEN** ninguna app aparece seleccionada.

#### Scenario: Seleccionar todas o ninguna

- **WHEN** la persona elige seleccionar todas o seleccionar ninguna en la pantalla de apps
- **THEN** todas las apps quedan seleccionadas o ninguna queda seleccionada, respectivamente.

#### Scenario: Selección explícita con --apps

- **GIVEN** una restauración interactiva con `--apps zsh,git`
- **WHEN** se inicia el flujo
- **THEN** la pantalla de selección de apps no se muestra y se usan `zsh` y `git`.

### Requirement: Tabla de conflictos

La tabla de conflictos DEBE (`MUST`) listar solo los archivos en conflicto y mostrar los conteos de archivos nuevos, idénticos y no soportados. Cada fila DEBE (`MUST`) ofrecer revisar, aplicar u omitir el archivo.

#### Scenario: Tabla con solo conflictos y conteos

- **GIVEN** una restauración con tres conflictos, cuatro archivos nuevos y diez idénticos
- **WHEN** se muestra la tabla
- **THEN** la tabla lista solo los tres conflictos y muestra los conteos de nuevos, idénticos y no soportados.

#### Scenario: Acciones por archivo

- **GIVEN** un archivo en conflicto en la tabla
- **WHEN** la persona selecciona una acción
- **THEN** puede elegir revisar el detalle, aplicar el respaldo completo u omitir el archivo.

### Requirement: Decisión por bloque de cambio

En la vista de detalle, cada bloque de cambio DEBE (`MUST`) empezar con la versión del respaldo y poder alternarse con la versión local. NO DEBE (`MUST NOT`) existir un estado sin decidir: el valor mostrado DEBE (`MUST`) ser la decisión. La vista NO DEBE (`MUST NOT`) mostrar una previsualización completa del archivo.

#### Scenario: Estado inicial del bloque

- **WHEN** se abre el detalle de un archivo en conflicto
- **THEN** todos sus bloques muestran la versión del respaldo como decisión inicial.

#### Scenario: Alternar un bloque

- **WHEN** la persona alterna un bloque
- **THEN** el bloque pasa de la versión del respaldo a la local, o viceversa, y el valor mostrado refleja la decisión.

#### Scenario: Sin previsualización completa

- **WHEN** la persona revisa un archivo en el detalle
- **THEN** la interfaz muestra el diff por bloques y no una vista previa del archivo final.

### Requirement: Decisión obligatoria y acciones globales

La restauración NO DEBE (`MUST NOT`) aplicarse mientras algún conflicto carezca de decisión. La tabla DEBE (`MUST`) ofrecer acciones globales para aplicar u omitir todos los conflictos, y cada archivo DEBE (`MUST`) poder cambiarse individualmente antes de la confirmación final.

#### Scenario: Confirmación bloqueada

- **GIVEN** una tabla con un conflicto sin decisión
- **WHEN** la persona intenta confirmar la restauración
- **THEN** la confirmación no está disponible y se indica qué archivo falta por decidir.

#### Scenario: Aplicar todo

- **WHEN** la persona elige aplicar todo en la tabla
- **THEN** todos los conflictos quedan decididos para aplicar el respaldo y se pueden cambiar individualmente antes de confirmar.

#### Scenario: Omitir todo

- **WHEN** la persona elige omitir todo en la tabla
- **THEN** todos los conflictos quedan decididos para conservar el local y se pueden cambiar individualmente antes de confirmar.

### Requirement: Confirmación final

Cuando todos los conflictos tengan decisión, el sistema DEBE (`MUST`) mostrar un resumen de lo que se aplicará, mezclará y omitirá, junto con el directorio del snapshot, y DEBE (`MUST`) exigir una confirmación explícita antes de escribir. Volver o cancelar en esta pantalla NO DEBE (`MUST NOT`) escribir ni crear snapshot.

#### Scenario: Resumen antes de aplicar

- **GIVEN** todos los conflictos decididos
- **WHEN** se muestra la confirmación final
- **THEN** se informa de los conteos de restaurados, mezclados y omitidos y del directorio del snapshot.

#### Scenario: Cancelar en la confirmación final

- **WHEN** la persona cancela en la confirmación final
- **THEN** no se escribe ningún destino y no se crea ningún snapshot.

### Requirement: Composición del archivo por bloques

El archivo resultante DEBE (`MUST`) componerse con los bloques elegidos del respaldo y los conservados del local, sin mezclar líneas dentro de un mismo bloque. DEBE (`MUST`) conservar el estilo de final de línea local y normalizar CRLF solo para comparar.

#### Scenario: Mezcla de bloques

- **GIVEN** un archivo con dos bloques en conflicto
- **WHEN** la persona toma uno del respaldo y conserva el otro local
- **THEN** el archivo resultante contiene el primer bloque del respaldo y el segundo del local.

#### Scenario: Todos los bloques del respaldo

- **GIVEN** un archivo con todos sus bloques decididos como respaldo
- **WHEN** se compone el resultado
- **THEN** el archivo resultante coincide con el respaldo.

#### Scenario: Todos los bloques locales

- **GIVEN** un archivo con todos sus bloques decididos como local
- **WHEN** se compone el resultado
- **THEN** el archivo resultante coincide con el local.

#### Scenario: Estilo de final de línea local

- **GIVEN** un archivo local con finales CRLF y un respaldo con finales LF
- **WHEN** se toma un bloque del respaldo
- **THEN** las líneas añadidas usan el estilo de final de línea local.

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

### Requirement: Validación del resultado

Antes de aplicar una mezcla, el sistema DEBE (`MUST`) validar TOML, JSON, INI y plist con la biblioteca estándar. Si ambos originales eran válidos y la mezcla no, DEBE (`MUST`) bloquear y volver al detalle. Si algún original ya era inválido, DEBE (`MUST`) avisar y permitir aplicar. YAML y shell NO DEBEN validarse.

#### Scenario: Mezcla válida

- **GIVEN** un archivo TOML, JSON, INI o plist cuyos originales y mezcla son válidos
- **WHEN** se valida la mezcla
- **THEN** la validación pasa y la restauración puede continuar.

#### Scenario: Mezcla inválida con originales válidos

- **GIVEN** un archivo cuyos originales parsean pero cuya mezcla no parsea
- **WHEN** se valida la mezcla
- **THEN** se muestra el error, se bloquea la aplicación y se vuelve al detalle del archivo.

#### Scenario: Original inválido

- **GIVEN** un archivo cuyo original local o cuyo respaldo ya era inválido
- **WHEN** se valida la mezcla
- **THEN** se muestra un aviso y se permite aplicar la mezcla.

#### Scenario: Formato sin validación

- **GIVEN** un archivo YAML o shell
- **WHEN** se valida la mezcla
- **THEN** no se aplica validación automática y se permite continuar.

### Requirement: Conflictos sin diff de líneas

Los archivos binarios, los archivos de texto de más de 5 MB y los enlaces con enlace DEBEN resolverse como archivo entero: aplicar el respaldo u omitir. NO DEBEN mostrar un diff de líneas. La fila DEBE (`MUST`) indicar el motivo y los datos disponibles del destino y del respaldo.

#### Scenario: Archivo binario

- **GIVEN** un conflicto entre un archivo binario local y uno del respaldo
- **WHEN** se muestra la tabla
- **THEN** el archivo se resuelve con aplicar u omitir y no ofrece revisión por bloques.

#### Scenario: Archivo de texto grande

- **GIVEN** un conflicto entre archivos de texto de más de 5 MB
- **WHEN** se muestra la tabla
- **THEN** el archivo se resuelve con aplicar u omitir y no ofrece revisión por bloques.

#### Scenario: Enlace con enlace

- **GIVEN** un conflicto entre dos enlaces simbólicos con destinos distintos
- **WHEN** se muestra la tabla
- **THEN** la fila muestra el destino local y el del respaldo y se resuelve con aplicar u omitir.

### Requirement: Seguridad y cancelación

La restauración DEBE (`MUST`) verificar el archivo antes de extraerlo, extraer solo en un directorio temporal, crear un snapshot antes de escribir y abortar toda la operación si un destino cambia mientras la revisión está abierta. Cancelar en cualquier pantalla NO DEBE (`MUST NOT`) escribir ni crear snapshot.

#### Scenario: Destino cambiado durante la revisión

- **GIVEN** un destino que cambia después de planificar y antes de aplicar
- **WHEN** se intenta aplicar la restauración
- **THEN** se aborta toda la operación, no se escribe ningún destino y no se crea ningún snapshot.

#### Scenario: Cancelar durante la revisión

- **WHEN** la persona cancela en cualquier pantalla de la revisión
- **THEN** no se escribe ningún destino y no se crea ningún snapshot.

#### Scenario: Snapshot del estado anterior

- **GIVEN** una restauración que va a modificar un archivo existente
- **WHEN** se aplica la restauración
- **THEN** el snapshot contiene el contenido y el modo anteriores del archivo.

#### Scenario: Archivo de respaldo corrupto

- **GIVEN** un archivo de respaldo cuya verificación de integridad falla
- **WHEN** se intenta restaurar
- **THEN** se aborta antes de extraer o escribir y no se crea ningún snapshot.

### Requirement: Resumen del resultado

El resultado DEBE (`MUST`) informar de los archivos restaurados, mezclados y omitidos y de la ruta del snapshot creado. Un archivo mezclado DEBE (`MUST`) contar como éxito.

#### Scenario: Resumen con mezclas

- **GIVEN** una restauración que aplica dos archivos completos, mezcla dos y omite uno
- **WHEN** termina la restauración
- **THEN** el resumen informa de dos restaurados, dos mezclados y uno omitido.

#### Scenario: Ruta del snapshot

- **GIVEN** una restauración que modificó al menos un archivo
- **WHEN** termina la restauración
- **THEN** el resumen muestra la ruta del snapshot creado.
