# deploy-conflicts Specification

## Purpose

Permitir que el despliegue muestre los conflictos entre el destino y el clon y que la persona decida, línea a línea dentro de cada bloque, qué se despliega y qué se conserva, sin sobrescribir cambios locales en silencio.

## Requirements

### Requirement: Detección de conflictos de contenido

El despliegue DEBE (`MUST`) tratar como conflicto todo archivo regular existente cuyo contenido difiera del clon tras normalizar CRLF, y todo enlace simbólico cuyo destino difiera del enlace del clon. Los archivos nuevos, idénticos, no soportados, los cambios solo de modo y los cambios de tipo entre enlace y archivo regular NO DEBEN abrir la revisión de conflictos.

#### Scenario: Archivo regular con contenido distinto

- **WHEN** el destino existe, es un archivo regular y su contenido difiere del clon
- **THEN** el archivo se presenta como conflicto y requiere una decisión antes de aplicar el despliegue.

#### Scenario: Diferencia solo de CRLF

- **GIVEN** un destino y un clon con el mismo contenido pero finales de línea CRLF y LF
- **WHEN** se planifica el despliegue
- **THEN** el archivo se considera idéntico y no se presenta como conflicto.

#### Scenario: Enlace con destino distinto

- **GIVEN** un destino y un clon que son enlaces simbólicos con destinos distintos
- **WHEN** se planifica el despliegue
- **THEN** el enlace se presenta como conflicto de archivo entero.

#### Scenario: Archivo nuevo

- **GIVEN** un archivo del clon que no existe en el destino
- **WHEN** se planifica el despliegue
- **THEN** se crea sin pedir una decisión de conflicto.

#### Scenario: Archivo idéntico

- **GIVEN** un destino y un clon con el mismo contenido
- **WHEN** se planifica el despliegue
- **THEN** el archivo se omite sin pedir una decisión de conflicto.

#### Scenario: Cambio solo de modo

- **GIVEN** un destino y un clon con el mismo contenido pero modos distintos
- **WHEN** se planifica el despliegue
- **THEN** el archivo no se presenta como conflicto y se mantiene el comportamiento existente.

#### Scenario: Cambio de tipo entre enlace y archivo regular

- **GIVEN** un destino que es un enlace simbólico y un clon que es un archivo regular, o al contrario
- **WHEN** se planifica el despliegue
- **THEN** el archivo no se presenta como conflicto y se mantiene el comportamiento de sobrescritura existente.

### Requirement: Selección de apps en el despliegue interactivo

El despliegue interactivo DEBE (`MUST`) integrar la selección de apps en la misma interfaz que la revisión de conflictos. Por defecto NO DEBE (`MUST NOT`) seleccionar ninguna app, DEBE (`MUST`) permitir seleccionar todas o ninguna y DEBE (`MUST`) omitir la pantalla de selección cuando se proporciona `--apps`.

#### Scenario: Selección inicial vacía

- **GIVEN** un despliegue interactivo con apps instaladas en el manifiesto
- **WHEN** se abre la pantalla de selección
- **THEN** ninguna app aparece seleccionada.

#### Scenario: Seleccionar todas o ninguna

- **WHEN** la persona elige seleccionar todas o seleccionar ninguna en la pantalla de apps
- **THEN** todas las apps quedan seleccionadas o ninguna queda seleccionada, respectivamente.

#### Scenario: Selección explícita con --apps

- **GIVEN** un despliegue interactivo con `--apps zsh,git`
- **WHEN** se inicia el flujo
- **THEN** la pantalla de selección de apps no se muestra y se usan `zsh` y `git`.

### Requirement: Tabla de conflictos

La tabla de conflictos DEBE (`MUST`) listar solo los archivos en conflicto y mostrar los conteos de archivos nuevos, idénticos y no soportados. Cada fila DEBE (`MUST`) ofrecer revisar, aplicar u omitir el archivo. En un conflicto de texto, la fila DEBE (`MUST`) indicar cuántos bloques y cuántas líneas cambiadas contiene.

#### Scenario: Tabla con solo conflictos y conteos

- **GIVEN** un despliegue con tres conflictos, cuatro archivos nuevos y diez idénticos
- **WHEN** se muestra la tabla
- **THEN** la tabla lista solo los tres conflictos y muestra los conteos de nuevos, idénticos y no soportados.

#### Scenario: Acciones por archivo

- **GIVEN** un archivo en conflicto en la tabla
- **WHEN** la persona selecciona una acción
- **THEN** puede elegir revisar el detalle, aplicar el clon completo u omitir el archivo.

#### Scenario: Detalle de un conflicto de texto

- **WHEN** la tabla muestra un conflicto de texto
- **THEN** la fila indica el número de bloques y de líneas cambiadas, y la decisión de revisión cuenta las líneas del clon.

### Requirement: Decisión por línea dentro del bloque

En la vista de detalle, cada bloque DEBE (`MUST`) mostrar el contexto una sola vez y cada línea cambiada DEBE (`MUST`) ser una fila con la versión local y la del clon. Cada fila DEBE (`MUST`) alternar entre local y clon. El contexto y la cabecera NO DEBEN (`MUST NOT`) ser decidibles. NO DEBE (`MUST NOT`) existir un estado sin decidir: lo mostrado DEBE (`MUST`) ser la decisión. La vista NO DEBE (`MUST NOT`) mostrar una previsualización completa del archivo.

#### Scenario: Estado inicial del bloque

- **WHEN** se abre el detalle de un archivo en conflicto
- **THEN** todas las líneas cambiadas de sus bloques muestran la versión del clon como decisión inicial.

#### Scenario: Alternar un bloque

- **WHEN** la persona aplica una acción de bloque al clon o al local
- **THEN** todas las líneas cambiadas del bloque actual toman ese lado y los demás bloques conservan sus decisiones.

#### Scenario: Alternar una línea

- **WHEN** la persona alterna una línea cambiada
- **THEN** esa línea pasa de la versión del clon a la local, o viceversa, y el resto del bloque conserva sus decisiones.

#### Scenario: Filas de inserción y eliminación

- **GIVEN** un bloque con una línea que solo existe en el local o solo en el clon
- **WHEN** la persona alterna esa fila
- **THEN** la línea se conserva o se omite según el lado elegido.

#### Scenario: Sin previsualización completa

- **WHEN** la persona revisa un archivo en el detalle
- **THEN** la interfaz muestra el diff por bloques con el contexto una sola vez y no una vista previa del archivo final.

### Requirement: Acciones globales y por bloque en el detalle

La vista de detalle DEBE (`MUST`) ofrecer acciones para aplicar todas las líneas del archivo o solo las del bloque actual al clon o al local. Las acciones por bloque NO DEBEN (`MUST NOT`) modificar las decisiones de otros bloques.

#### Scenario: Todo el archivo al clon

- **WHEN** la persona elige la acción global de clon
- **THEN** todas las líneas cambiadas del archivo quedan en clon.

#### Scenario: Todo el archivo al local

- **WHEN** la persona elige la acción global de local
- **THEN** todas las líneas cambiadas del archivo quedan en local.

#### Scenario: Bloque actual al clon

- **WHEN** la persona elige la acción de bloque para el clon
- **THEN** todas las líneas cambiadas del bloque actual quedan en clon y los demás bloques conservan sus decisiones.

#### Scenario: Bloque actual al local

- **WHEN** la persona elige la acción de bloque para el local
- **THEN** todas las líneas cambiadas del bloque actual quedan en local y los demás bloques conservan sus decisiones.

### Requirement: Decisión obligatoria y acciones globales

El despliegue NO DEBE (`MUST NOT`) aplicarse mientras algún conflicto carezca de decisión. La tabla DEBE (`MUST`) ofrecer acciones globales para aplicar u omitir todos los conflictos, y cada archivo DEBE (`MUST`) poder cambiarse individualmente antes de la confirmación final.

#### Scenario: Confirmación bloqueada

- **GIVEN** una tabla con un conflicto sin decisión
- **WHEN** la persona intenta confirmar el despliegue
- **THEN** la confirmación no está disponible y se indica qué archivo falta por decidir.

#### Scenario: Aplicar todo

- **WHEN** la persona elige aplicar todo en la tabla
- **THEN** todos los conflictos quedan decididos para aplicar el clon y se pueden cambiar individualmente antes de confirmar.

#### Scenario: Omitir todo

- **WHEN** la persona elige omitir todo en la tabla
- **THEN** todos los conflictos quedan decididos para conservar el local y se pueden cambiar individualmente antes de confirmar.

### Requirement: Confirmación final

Cuando todos los conflictos tengan decisión, el sistema DEBE (`MUST`) mostrar un resumen de lo que se aplicará, mezclará y omitirá, junto con el directorio del snapshot, y DEBE (`MUST`) exigir una confirmación explícita antes de escribir. Volver o cancelar en esta pantalla NO DEBE (`MUST NOT`) escribir ni crear snapshot.

#### Scenario: Resumen antes de aplicar

- **GIVEN** todos los conflictos decididos
- **WHEN** se muestra la confirmación final
- **THEN** se informa de los conteos de desplegados, mezclados y omitidos y del directorio del snapshot.

#### Scenario: Cancelar en la confirmación final

- **WHEN** la persona cancela en la confirmación final
- **THEN** no se escribe ningún destino y no se crea ningún snapshot.

### Requirement: Composición del archivo por líneas

El archivo resultante DEBE (`MUST`) componerse con las líneas elegidas del clon y las conservadas del local, permitiendo mezclar líneas dentro de un mismo bloque. DEBE (`MUST`) conservar el estilo de final de línea local y normalizar CRLF solo para comparar.

#### Scenario: Mezcla de bloques

- **GIVEN** un archivo con dos bloques en conflicto
- **WHEN** la persona elige líneas del clon en el primero y conserva líneas locales en el segundo
- **THEN** el archivo resultante contiene las líneas elegidas de cada bloque en el orden original.

#### Scenario: Mezcla de líneas dentro de un mismo bloque

- **GIVEN** un bloque con dos líneas cambiadas
- **WHEN** la persona elige el clon para una y conserva el local para la otra
- **THEN** el archivo resultante contiene una línea del clon y la otra del local, en el orden del bloque.

#### Scenario: Todos los bloques del clon

- **GIVEN** un archivo con todas sus líneas decididas como clon
- **WHEN** se compone el resultado
- **THEN** el archivo resultante coincide con el clon.

#### Scenario: Todos los bloques locales

- **GIVEN** un archivo con todas sus líneas decididas como local
- **WHEN** se compone el resultado
- **THEN** el archivo resultante coincide con el local.

#### Scenario: Estilo de final de línea local

- **GIVEN** un archivo local con finales CRLF y un clon con finales LF
- **WHEN** se toma una línea del clon
- **THEN** la línea añadida usa el estilo de final de línea local.

#### Scenario: Salto de línea final en una mezcla

- **GIVEN** una mezcla en la que una línea sin salto de línea final queda antes de otra línea elegida
- **WHEN** se compone el resultado
- **THEN** la línea intermedia recibe el EOL dominante local y solo la última línea puede conservar la ausencia de salto final.

### Requirement: Modo y clasificación del resultado

Si el contenido final difiere del local, el archivo DEBE (`MUST`) recibir el modo del clon. Si la composición queda idéntica al local, NO DEBE (`MUST NOT`) escribirse ni cambiar permisos. El resultado DEBE (`MUST`) clasificar cada archivo como despliegado, mezclado u omitido.

#### Scenario: Mezcla que cambia el contenido

- **GIVEN** una mezcla que difiere del local y del clon
- **WHEN** se aplica el despliegue
- **THEN** el archivo se escribe con el modo del clon y se clasifica como mezclado.

#### Scenario: Mezcla idéntica al local

- **GIVEN** una composición que coincide con el contenido local
- **WHEN** se aplica el despliegue
- **THEN** el archivo no se escribe, sus permisos no cambian y se clasifica como omitido.

#### Scenario: Aplicación completa del clon

- **GIVEN** un archivo decidido para aplicar el clon completo
- **WHEN** se aplica el despliegue
- **THEN** el archivo se escribe con el modo del clon y se clasifica como despliegado.

#### Scenario: Revisión con todas las líneas en clon

- **GIVEN** un archivo revisado línea a línea con todas sus líneas decididas como clon
- **WHEN** se aplica el despliegue
- **THEN** el archivo se escribe con el modo del clon y se clasifica como despliegado.

### Requirement: Validación del resultado

Antes de aplicar una mezcla, el sistema DEBE (`MUST`) validar TOML, JSON, INI y plist con la biblioteca estándar. Si ambos originales eran válidos y la mezcla no, DEBE (`MUST`) bloquear y volver al detalle. Si algún original ya era inválido, DEBE (`MUST`) avisar y permitir aplicar. YAML y shell NO DEBEN validarse.

#### Scenario: Mezcla válida

- **GIVEN** un archivo TOML, JSON, INI o plist cuyos originales y mezcla son válidos
- **WHEN** se valida la mezcla
- **THEN** la validación pasa y el despliegue puede continuar.

#### Scenario: Mezcla inválida con originales válidos

- **GIVEN** un archivo cuyos originales parsean pero cuya mezcla no parsea
- **WHEN** se valida la mezcla
- **THEN** se muestra el error, se bloquea la aplicación y se vuelve al detalle del archivo.

#### Scenario: Original inválido

- **GIVEN** un archivo cuyo original local o cuyo clon ya era inválido
- **WHEN** se valida la mezcla
- **THEN** se muestra un aviso y se permite aplicar la mezcla.

#### Scenario: Formato sin validación

- **GIVEN** un archivo YAML o shell
- **WHEN** se valida la mezcla
- **THEN** no se aplica validación automática y se permite continuar.

### Requirement: Conflictos sin diff de líneas

Los archivos binarios, los archivos de texto de más de 5 MB y los enlaces con enlace DEBEN resolverse como archivo entero: aplicar el clon u omitir. NO DEBEN mostrar un diff de líneas. La fila DEBE (`MUST`) indicar el motivo y los datos disponibles del destino y del clon.

#### Scenario: Archivo binario

- **GIVEN** un conflicto entre un archivo binario local y uno del clon
- **WHEN** se muestra la tabla
- **THEN** el archivo se resuelve con aplicar u omitir y no ofrece revisión por líneas.

#### Scenario: Archivo de texto grande

- **GIVEN** un conflicto entre archivos de texto de más de 5 MB
- **WHEN** se muestra la tabla
- **THEN** el archivo se resuelve con aplicar u omitir y no ofrece revisión por líneas.

#### Scenario: Enlace con enlace

- **GIVEN** un conflicto entre dos enlaces simbólicos con destinos distintos
- **WHEN** se muestra la tabla
- **THEN** la fila muestra el destino local y el del clon y se resuelve con aplicar u omitir.

### Requirement: Seguridad y cancelación

El despliegue DEBE (`MUST`) verificar el archivo antes de extraerlo, extraer solo en un directorio temporal, crear un snapshot antes de escribir y abortar toda la operación si un destino cambia mientras la revisión está abierta. Cancelar en cualquier pantalla NO DEBE (`MUST NOT`) escribir ni crear snapshot.

#### Scenario: Destino cambiado durante la revisión

- **GIVEN** un destino que cambia después de planificar y antes de aplicar
- **WHEN** se intenta aplicar el despliegue
- **THEN** se aborta toda la operación, no se escribe ningún destino y no se crea ningún snapshot.

#### Scenario: Cancelar durante la revisión

- **WHEN** la persona cancela en cualquier pantalla de la revisión
- **THEN** no se escribe ningún destino y no se crea ningún snapshot.

#### Scenario: Snapshot del estado anterior

- **GIVEN** un despliegue que va a modificar un archivo existente
- **WHEN** se aplica el despliegue
- **THEN** el snapshot contiene el contenido y el modo anteriores del archivo.

#### Scenario: Archivo de clon corrupto

- **GIVEN** un archivo de clon cuya verificación de integridad falla
- **WHEN** se intenta desplegar
- **THEN** se aborta antes de extraer o escribir y no se crea ningún snapshot.

### Requirement: Resumen del resultado

El resultado DEBE (`MUST`) informar de los archivos desplegados, mezclados y omitidos y de la ruta del snapshot creado. Un archivo mezclado DEBE (`MUST`) contar como éxito.

#### Scenario: Resumen con mezclas

- **GIVEN** un despliegue que aplica dos archivos completos, mezcla dos y omite uno
- **WHEN** termina el despliegue
- **THEN** el resumen informa de dos desplegados, dos mezclados y uno omitido.

#### Scenario: Ruta del snapshot

- **GIVEN** un despliegue que modificó al menos un archivo
- **WHEN** termina el despliegue
- **THEN** el resumen muestra la ruta del snapshot creado.
