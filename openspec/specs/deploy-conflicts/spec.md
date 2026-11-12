# deploy-conflicts Specification

## Purpose

Permitir que el despliegue muestre los conflictos entre el destino y el clon y que la persona decida, línea a línea dentro de cada bloque, qué se despliega y qué se conserva, sin sobrescribir cambios locales en silencio.

## Requirements

### Requirement: Detección de conflictos de contenido

El despliegue DEBE (`MUST`) tratar como conflicto todo archivo regular existente cuyo contenido difiera del clon tras normalizar CRLF, y todo enlace simbólico cuyo destino difiera del enlace del clon. Los archivos nuevos, los clones perfectos (contenido idéntico), los no soportados, los cambios solo de modo y los cambios de tipo entre enlace y archivo regular NO DEBEN abrir la revisión de conflictos. Un clon perfecto NO DEBE escribirse ni contar como omitido.

#### Scenario: Archivo regular con contenido distinto

- **WHEN** el destino existe, es un archivo regular y su contenido difiere del clon
- **THEN** el archivo se presenta como conflicto y requiere una decisión antes de aplicar el despliegue.

#### Scenario: Diferencia solo de CRLF

- **GIVEN** un destino y un clon con el mismo contenido pero finales de línea CRLF y LF
- **WHEN** se planifica el despliegue
- **THEN** el archivo se considera clon perfecto y no se presenta como conflicto.

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
- **THEN** el archivo se cuenta como clon perfecto, no se escribe y no altera el código de salida.

#### Scenario: Cambio solo de modo

- **GIVEN** un destino y un clon con el mismo contenido pero modos distintos
- **WHEN** se planifica el despliegue
- **THEN** el archivo no se presenta como conflicto y se mantiene el comportamiento existente.

#### Scenario: Cambio de tipo entre enlace y archivo regular

- **GIVEN** un destino que es un enlace simbólico y un clon que es un archivo regular, o al contrario
- **WHEN** se planifica el despliegue
- **THEN** el archivo no se presenta como conflicto y se mantiene el comportamiento de sobrescritura existente.

### Requirement: Selección de apps en el despliegue interactivo

El despliegue interactivo DEBE (`MUST`) integrar la selección de apps en la misma interfaz que la revisión de conflictos. Por defecto NO DEBE (`MUST NOT`) seleccionar ninguna app, DEBE (`MUST`) permitir seleccionar todas o ninguna y DEBE (`MUST`) omitir la pantalla de selección cuando se proporciona `--apps`. Solo DEBE listar como seleccionables las apps del manifiesto instaladas, con su número de archivos; las no instaladas DEBEN mostrarse en una lista informativa aparte y NO DEBEN contar como omitidas.

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

#### Scenario: Apps no instaladas solo se informan

- **GIVEN** un manifiesto con apps que no están instaladas en este Mac
- **WHEN** se muestra la selección
- **THEN** aparecen en una lista informativa “sin base en este Mac”, no son seleccionables y no cuentan como omitidas.

### Requirement: Tabla de conflictos

La tabla de conflictos DEBE (`MUST`) listar solo los archivos en conflicto y mostrar los conteos de archivos nuevos, clones perfectos, no soportados y avisos. Cada fila DEBE (`MUST`) ofrecer revisar, aplicar el clon o conservar el original. En un conflicto de texto, la fila DEBE (`MUST`) indicar cuántos bloques y cuántas líneas cambiadas contiene, y la decisión de revisión DEBE expresarse como “híbrido (N de M líneas)”.

#### Scenario: Tabla con solo conflictos y conteos

- **GIVEN** un despliegue con tres conflictos, cuatro archivos nuevos, diez clones perfectos y un aviso
- **WHEN** se muestra la tabla
- **THEN** la tabla lista solo los tres conflictos y muestra los conteos de nuevos, clones perfectos, no soportados y avisos.

#### Scenario: Acciones por archivo

- **GIVEN** un archivo en conflicto en la tabla
- **WHEN** la persona selecciona una acción
- **THEN** puede elegir revisar el detalle, aplicar el clon completo o conservar el original.

#### Scenario: Detalle de un conflicto de texto

- **WHEN** la tabla muestra un conflicto de texto
- **THEN** la fila indica el número de bloques y de líneas cambiadas, y la decisión de revisión se muestra como “híbrido (N de M líneas)”.

### Requirement: Decisión por línea dentro del bloque

En la vista de detalle, cada bloque DEBE (`MUST`) mostrar el contexto una sola vez y cada línea cambiada DEBE (`MUST`) ser una fila con la versión del original y la del clon. Cada fila DEBE (`MUST`) alternar entre original y clon. El contexto y la cabecera NO DEBEN (`MUST NOT`) ser decidibles. NO DEBE (`MUST NOT`) existir un estado sin decidir: lo mostrado DEBE (`MUST`) ser la decisión. La vista NO DEBE (`MUST NOT`) mostrar una previsualización completa del archivo.

#### Scenario: Estado inicial del bloque

- **WHEN** se abre el detalle de un archivo en conflicto
- **THEN** todas las líneas cambiadas de sus bloques muestran la versión del clon como decisión inicial.

#### Scenario: Alternar un bloque

- **WHEN** la persona aplica una acción de bloque al clon o al original
- **THEN** todas las líneas cambiadas del bloque actual toman ese lado y los demás bloques conservan sus decisiones.

#### Scenario: Alternar una línea

- **WHEN** la persona alterna una línea cambiada
- **THEN** esa línea pasa de la versión del clon a la del original, o viceversa, y el resto del bloque conserva sus decisiones.

#### Scenario: Filas de inserción y eliminación

- **GIVEN** un bloque con una línea que solo existe en el original o solo en el clon
- **WHEN** la persona alterna esa fila
- **THEN** la línea se conserva o se omite según el lado elegido.

#### Scenario: Sin previsualización completa

- **WHEN** la persona revisa un archivo en el detalle
- **THEN** la interfaz muestra el diff por bloques con el contexto una sola vez y no una vista previa del archivo final.

### Requirement: Acciones globales y por bloque en el detalle

La vista de detalle DEBE (`MUST`) ofrecer acciones para aplicar todas las líneas del archivo o solo las del bloque actual al clon o al original. Las acciones por bloque NO DEBEN (`MUST NOT`) modificar las decisiones de otros bloques. Las teclas `c` y `o` DEBEN actuar sobre el bloque actual, `C` y `O` sobre todo el archivo, y `Tab`/`Mayús+Tab` DEBEN recorrer los bloques.

#### Scenario: Todo el archivo al clon

- **WHEN** la persona elige la acción global de clon
- **THEN** todas las líneas cambiadas del archivo quedan en clon.

#### Scenario: Todo el archivo al local

- **WHEN** la persona elige la acción global de original
- **THEN** todas las líneas cambiadas del archivo quedan en original.

#### Scenario: Bloque actual al clon

- **WHEN** la persona elige la acción de bloque para el clon
- **THEN** todas las líneas cambiadas del bloque actual quedan en clon y los demás bloques conservan sus decisiones.

#### Scenario: Bloque actual al local

- **WHEN** la persona elige la acción de bloque para el original
- **THEN** todas las líneas cambiadas del bloque actual quedan en original y los demás bloques conservan sus decisiones.

#### Scenario: Navegación entre bloques

- **GIVEN** un archivo con varios bloques
- **WHEN** se pulsa `Tab` o `Mayús+Tab`
- **THEN** el foco pasa al bloque siguiente o anterior sin cambiar decisiones.

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

Cuando todos los conflictos tengan decisión, el sistema DEBE (`MUST`) mostrar un resumen de lo que se aplicará, hibridará, omitirá y quedará sin cambios, junto con el número de archivos protegidos y el directorio del holocrón, y DEBE (`MUST`) exigir una confirmación explícita antes de escribir. Las acciones DEBEN navegarse con flechas y `Enter`, con Retirada preseleccionada, y NO DEBE existir un atajo de una tecla para aplicar. Volver o cancelar en esta pantalla NO DEBE (`MUST NOT`) escribir ni crear holocrón.

#### Scenario: Resumen antes de aplicar

- **GIVEN** todos los conflictos decididos
- **WHEN** se muestra la confirmación final
- **THEN** se informan los conteos de desplegados (nuevos y reemplazos), híbridos, sin desplegar y sin cambios, el número de archivos protegidos y el directorio del holocrón.

#### Scenario: Cancelar en la confirmación final

- **WHEN** la persona cancela en la confirmación final
- **THEN** no se escribe ningún destino y no se crea ningún holocrón.

#### Scenario: Foco inicial y acciones

- **GIVEN** la confirmación final abierta
- **WHEN** se muestra la pantalla
- **THEN** Retirada está preseleccionada y las acciones Abortar misión, Retirada y Desplegar se recorren con flechas y se aceptan con `Enter`.

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

Si el contenido final difiere del original, el archivo DEBE (`MUST`) recibir el modo del clon. Si la composición queda idéntica al original, NO DEBE (`MUST NOT`) escribirse ni cambiar permisos. El resultado DEBE (`MUST`) clasificar cada archivo como desplegado, híbrido, omitido o sin cambios; los clones perfectos y las apps no instaladas NO DEBEN clasificarse como omitidos.

#### Scenario: Mezcla que cambia el contenido

- **GIVEN** una mezcla que difiere del original y del clon
- **WHEN** se aplica el despliegue
- **THEN** el archivo se escribe con el modo del clon y se clasifica como híbrido.

#### Scenario: Mezcla idéntica al local

- **GIVEN** una composición que coincide con el contenido original
- **WHEN** se aplica el despliegue
- **THEN** el archivo no se escribe, sus permisos no cambian y se clasifica como omitido.

#### Scenario: Aplicación completa del clon

- **GIVEN** un archivo decidido para aplicar el clon completo
- **WHEN** se aplica el despliegue
- **THEN** el archivo se escribe con el modo del clon y se clasifica como desplegado.

#### Scenario: Revisión con todas las líneas en clon

- **GIVEN** un archivo revisado línea a línea con todas sus líneas decididas como clon
- **WHEN** se aplica el despliegue
- **THEN** el archivo se escribe con el modo del clon y se clasifica como desplegado.

#### Scenario: Clon perfecto en el resultado

- **GIVEN** un destino idéntico al clon
- **WHEN** termina el despliegue
- **THEN** el archivo se clasifica como sin cambios y no se cuenta como omitido.

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

Los archivos binarios, los archivos de texto de más de 5 MB y los enlaces con enlace DEBEN resolverse como archivo entero: aplicar el clon o conservar el original. NO DEBEN mostrar un diff de líneas. La fila DEBE (`MUST`) indicar el motivo y los datos disponibles del destino y del clon.

#### Scenario: Archivo binario

- **GIVEN** un conflicto entre un archivo binario original y uno del clon
- **WHEN** se muestra la tabla
- **THEN** el archivo se resuelve con aplicar el clon o conservar el original y no ofrece revisión por líneas.

#### Scenario: Archivo de texto grande

- **GIVEN** un conflicto entre archivos de texto de más de 5 MB
- **WHEN** se muestra la tabla
- **THEN** el archivo se resuelve con aplicar el clon o conservar el original y no ofrece revisión por líneas.

#### Scenario: Enlace con enlace

- **GIVEN** un conflicto entre dos enlaces simbólicos con destinos distintos
- **WHEN** se muestra la tabla
- **THEN** la fila muestra el destino del original y el del clon y se resuelve con aplicar el clon o conservar el original.

### Requirement: Seguridad y cancelación

El despliegue DEBE (`MUST`) verificar el archivo antes de extraerlo, extraer solo en un directorio temporal, crear un holocrón antes de escribir y abortar toda la operación si un destino cambia mientras la revisión está abierta. Cancelar antes de crear el holocrón NO DEBE (`MUST NOT`) escribir destinos ni crear holocrón. Una vez creado el holocrón, la cancelación NO DEBE interrumpir la aplicación.

#### Scenario: Destino cambiado durante la revisión

- **GIVEN** un destino que cambia después de planificar y antes de aplicar
- **WHEN** se intenta aplicar el despliegue
- **THEN** se aborta toda la operación, no se escribe ningún destino y no se crea ningún holocrón.

#### Scenario: Cancelar durante la revisión

- **WHEN** la persona cancela en cualquier pantalla de la revisión antes del holocrón
- **THEN** no se escribe ningún destino y no se crea ningún holocrón.

#### Scenario: Snapshot del estado anterior

- **GIVEN** un despliegue que va a modificar un archivo existente
- **WHEN** se aplica el despliegue
- **THEN** el holocrón contiene el contenido y el modo anteriores del archivo.

#### Scenario: Archivo de clon corrupto

- **GIVEN** un archivo de clon cuya verificación de integridad falla
- **WHEN** se intenta desplegar
- **THEN** se aborta antes de extraer o escribir y no se crea ningún holocrón.

#### Scenario: Cancelar durante la aplicación

- **GIVEN** un despliegue que ya creó el holocrón
- **WHEN** se intenta cancelar
- **THEN** la aplicación continúa y la pantalla indica que ya no se puede cancelar

### Requirement: Resumen del resultado

El resultado DEBE (`MUST`) informar de los archivos desplegados, híbridos, omitidos, sin cambios y de la ruta del holocrón creado. Un archivo híbrido DEBE (`MUST`) contar como éxito. La línea de salida DEBE mostrarse solo si el código no es 0 y DEBE incluir la orden para deshacer con `kamino deploy <ruta>` cuando exista holocrón.

#### Scenario: Resumen con mezclas

- **GIVEN** un despliegue que aplica dos archivos completos, hibrida dos y omite uno
- **WHEN** termina el despliegue
- **THEN** el resumen informa de dos desplegados, dos híbridos y uno omitido.

#### Scenario: Ruta del snapshot

- **GIVEN** un despliegue que modificó al menos un archivo existente
- **WHEN** termina el despliegue
- **THEN** el resumen muestra la ruta del holocrón creado y la orden para deshacer.

### Requirement: Validación del manifiesto en D0

D0 DEBE (`MUST`) validar solo el manifiesto del clon, sin extraer ni seguir enlaces, y DEBE bloquear un archivo que no sea un `tar.gz` reconocible, esté dañado, tenga más de un `manifest.json`, supere el tope de tamaño o declare un formato más nuevo. DEBE avisar, sin bloquear, si la arquitectura o la versión de macOS del manifiesto difieren de las actuales. Un manifiesto válido NO implica que la integridad esté verificada.

#### Scenario: Archivo no reconocido o dañado

- **GIVEN** un archivo que no es un `tar.gz` válido o está corrupto
- **WHEN** se selecciona en D0
- **THEN** se muestra un error literal y no se extrae ni se escribe nada

#### Scenario: Formato más nuevo

- **GIVEN** un manifiesto con una versión de formato mayor que la soportada
- **WHEN** se selecciona en D0
- **THEN** se bloquea el flujo y se informa de la versión no soportada

#### Scenario: Manifiesto duplicado

- **GIVEN** un archivo con más de un miembro `manifest.json`
- **WHEN** se selecciona en D0
- **THEN** se bloquea por manifiesto ambiguo

#### Scenario: Manifiesto demasiado grande

- **GIVEN** un manifiesto que supera el tope de tamaño
- **WHEN** se selecciona en D0
- **THEN** se bloquea sin leerlo completo

#### Scenario: Arquitectura o macOS distintos

- **GIVEN** un manifiesto con otra arquitectura u otra versión de macOS
- **WHEN** se selecciona en D0
- **THEN** se avisa de la diferencia y se permite continuar

#### Scenario: Manifiesto válido sin integridad

- **GIVEN** un manifiesto válido con checksums alterados
- **WHEN** se selecciona en D0
- **THEN** se permite continuar hasta D1, donde la verificación completa detecta el fallo antes de extraer

### Requirement: Categorías del resultado y código 3

El código 3 DEBE (`MUST`) reservarse para los conflictos conservados en el original, los archivos no soportados y los `--apps` pedidos que no se puedan cumplir. Los clones perfectos, las apps del manifiesto no instaladas y los archivos híbridos NO DEBEN producir código 3.

#### Scenario: Solo clones perfectos

- **GIVEN** un despliegue sin escrituras y con clones perfectos
- **WHEN** termina
- **THEN** el código es 0 y los archivos se informan como sin cambios

#### Scenario: Apps no instaladas no alteran el código

- **GIVEN** un clon con apps no instaladas que no fueron pedidas con `--apps`
- **WHEN** termina el despliegue
- **THEN** se informan como “sin base en este Mac” y el código es 0

#### Scenario: --apps no cumplido

- **GIVEN** un `--apps` que incluye una app no instalada o ausente del manifiesto
- **WHEN** termina el despliegue
- **THEN** el elemento se cuenta como omitido y el código es 3

#### Scenario: Híbrido no es omisión

- **GIVEN** un despliegue que solo produce archivos híbridos
- **WHEN** termina
- **THEN** el código es 0

### Requirement: Holocrón de seguridad

Antes de escribir, el despliegue DEBE (`MUST`) crear un holocrón con los destinos existentes que cambiarán y DEBE informar de su ruta y del número de archivos protegidos. Si ningún destino existe todavía, NO DEBE crear un holocrón vacío y DEBE informar de que no hay originales que proteger.

#### Scenario: Holocrón con archivos existentes

- **GIVEN** un despliegue que reemplazará archivos existentes
- **WHEN** se aplica
- **THEN** el holocrón contiene esos archivos y el resumen muestra su ruta y el número de protegidos

#### Scenario: Solo archivos nuevos

- **GIVEN** un despliegue cuyos destinos que cambiarán no existen
- **WHEN** se aplica
- **THEN** no se crea un holocrón vacío y se informa de que no había originales que proteger

#### Scenario: Deshacer desde el holocrón

- **GIVEN** un holocrón creado por un despliegue
- **WHEN** se consulta el resumen
- **THEN** se muestra la orden `kamino deploy <ruta>` para deshacer

### Requirement: Cancelación y aplicación parcial

La aplicación DEBE (`MUST`) poder cancelarse hasta crear el holocrón. Si falla una escritura después, el resultado DEBE informar de un despliegue incompleto con los archivos aplicados, el archivo fallido y el holocrón disponible para deshacer, y NO DEBE presentarse como un despliegue completado.

#### Scenario: Cancelar antes del holocrón

- **GIVEN** una aplicación que aún no creó el holocrón
- **WHEN** se cancela
- **THEN** no se escribe ningún destino y no se crea holocrón

#### Scenario: Fallo parcial

- **GIVEN** un despliegue que escribe algunos archivos y falla en otro
- **WHEN** termina
- **THEN** se muestra la variante “Despliegue incompleto” con los aplicados, el fallido y el holocrón

#### Scenario: Limpieza del staging

- **GIVEN** una preparación o aplicación cancelada
- **WHEN** termina la cancelación
- **THEN** el directorio temporal de despliegue se elimina

### Requirement: Fuentes y orden de los clones en D0

D0 DEBE (`MUST`) listar los archivos `.tar.gz` hijos directos de `clone.output_dir` y del directorio actual, sin recorrer subdirectorios ni incluir holocrones. DEBE (`MUST`) deduplicar por ruta resuelta, mostrar el origen de cada fila y ordenar por fecha de modificación descendente.

#### Scenario: Ambos orígenes con clones

- **GIVEN** una carpeta de clones configurada y un directorio actual con archivos `.tar.gz`
- **WHEN** se abre D0
- **THEN** la pantalla muestra los clones de ambas rutas en una sola lista plana, ordenados del más reciente al más antiguo y con el origen visible en cada fila

#### Scenario: Mismo clon en ambas rutas

- **GIVEN** el mismo archivo `.tar.gz` presente en la carpeta configurada y en el directorio actual
- **WHEN** se abre D0
- **THEN** el clon aparece una sola vez y se conserva el origen de la carpeta configurada

#### Scenario: Carpeta configurada y directorio actual coinciden

- **GIVEN** que `clone.output_dir` es el mismo directorio desde el que se lanzó la TUI
- **WHEN** se abre D0
- **THEN** cada clon aparece una sola vez y la pantalla no duplica la lista

#### Scenario: Solo hijos directos y clones

- **GIVEN** un `.tar.gz` en un subdirectorio y un holocrón en el subdirectorio `snapshot/`
- **WHEN** se abre D0
- **THEN** ninguno de los dos aparece en la lista porque D0 solo lista clones directos de las dos rutas

#### Scenario: Ruta ausente o sin clones

- **GIVEN** una de las dos rutas no existe o no contiene `.tar.gz`
- **WHEN** se abre D0
- **THEN** se informa de que esa ruta no tiene clones y se siguen mostrando los clones de la otra ruta sin tratarlo como error

#### Scenario: Ruta ilegible

- **GIVEN** una de las dos rutas no se puede leer
- **WHEN** se abre D0
- **THEN** se muestra el motivo y la otra ruta sigue siendo utilizable

### Requirement: Navegación y selección por teclado en D0

D0 DEBE (`MUST`) abrir con el foco en la tabla. Las flechas DEBEN (`MUST`) mover la selección, `Enter` DEBE (`MUST`) validar el manifiesto del clon seleccionado y continuar, y `/` DEBE (`MUST`) filtrar por nombre u origen sin abandonar la pantalla. La fila resaltada DEBE (`MUST`) mostrar la ruta completa, la fecha y el tamaño.

#### Scenario: Foco inicial en la tabla

- **GIVEN** D0 abierto con al menos un clon
- **WHEN** se pulsa una flecha sin haber pulsado `Tab`
- **THEN** la selección se mueve por la lista de clones

#### Scenario: Enter sobre un clon

- **GIVEN** un clon seleccionado en la tabla
- **WHEN** se pulsa `Enter`
- **THEN** se valida su manifiesto y se continúa a D1 sin extraer el archivo ni verificar su integridad

#### Scenario: Filtro por nombre u origen

- **GIVEN** una lista con clones de ambas rutas
- **WHEN** se pulsa `/` y se escribe parte de un nombre o de un origen
- **THEN** la lista se reduce a las filas coincidentes y `Esc` limpia el filtro y devuelve el foco a la tabla

#### Scenario: Filtro sin coincidencias

- **GIVEN** un filtro que no coincide con ningún clon
- **WHEN** la tabla queda vacía
- **THEN** la fila `Otro archivo...` sigue disponible para continuar con una ruta manual

#### Scenario: Preview del clon resaltado

- **GIVEN** una fila de clon resaltada
- **WHEN** la selección cambia
- **THEN** la pantalla muestra la ruta completa, la fecha de modificación y el tamaño de ese clon

### Requirement: Ruta manual como última opción en D0

La última fila de D0 DEBE (`MUST`) ser `Otro archivo...` y DEBE (`MUST`) revelar un campo de ruta al elegirla. `o` DEBE (`MUST`) saltar a esa fila. `Esc` DEBE (`MUST`) volver del campo de ruta a la tabla. Una ruta válida DEBE (`MUST`) continuar y una inválida DEBE (`MUST`) mostrar el error sin salir de D0.

#### Scenario: Elegir Otro archivo

- **GIVEN** la tabla de D0 con foco
- **WHEN** se selecciona la fila `Otro archivo...` y se pulsa `Enter`
- **THEN** aparece el campo de ruta y recibe el foco

#### Scenario: Atajo para la ruta manual

- **GIVEN** la tabla de D0 con o sin filtro activo
- **WHEN** se pulsa `o`
- **THEN** la selección salta a `Otro archivo...` y se puede abrir el campo de ruta con `Enter`

#### Scenario: Ruta manual válida

- **GIVEN** el campo de ruta abierto
- **WHEN** se escribe un `.tar.gz` con un manifiesto válido y se pulsa `Enter`
- **THEN** se valida el manifiesto y se continúa a D1 sin extraer el archivo ni verificar su integridad

#### Scenario: Ruta manual inválida

- **GIVEN** el campo de ruta abierto
- **WHEN** se escribe una ruta que no existe, no es un archivo o no tiene un manifiesto válido y se pulsa `Enter`
- **THEN** D0 permanece abierta, conserva el valor escrito y muestra el error correspondiente

#### Scenario: Volver de la ruta a la tabla

- **GIVEN** el campo de ruta abierto
- **WHEN** se pulsa `Esc`
- **THEN** el foco vuelve a la tabla sin cancelar el flujo

#### Scenario: Flujo directo con archivo inválido

- **GIVEN** una invocación de `kamino deploy` con un archivo inexistente o inválido
- **WHEN** se abre D0
- **THEN** el campo de ruta aparece precargado con esa ruta, muestra el error y la lista de clones sigue disponible

### Requirement: Conservación de la selección al volver a D0

Al volver de D1, D0 DEBE (`MUST`) preseleccionar el clon que se estaba desplegando. Si ese clon se eligió con una ruta manual que no pertenece a las fuentes listadas, D0 DEBE (`MUST`) mostrar el campo de ruta precargado en lugar de perder la selección.

#### Scenario: Volver desde D1 con un clon listado

- **GIVEN** un clon elegido desde la lista y la pantalla D1 abierta
- **WHEN** se vuelve a D0
- **THEN** la fila de ese clon queda preseleccionada y la tabla recibe el foco

#### Scenario: Volver desde D1 con una ruta manual

- **GIVEN** un clon elegido con una ruta manual que no está en la carpeta configurada ni en el directorio actual
- **WHEN** se vuelve desde D1 a D0
- **THEN** el campo de ruta aparece precargado con ese clon y recibe el foco

#### Scenario: Volver desde D1 tras un error

- **GIVEN** un error de D1 que devuelve a D0
- **WHEN** D0 vuelve a mostrarse
- **THEN** la selección del clon se conserva y el error queda visible en la pantalla
