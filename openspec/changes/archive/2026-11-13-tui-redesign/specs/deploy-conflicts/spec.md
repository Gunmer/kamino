# Spec Delta

## ADDED Requirements

### Requirement: Vista de archivo completo en dos columnas

D3 DEBE (`MUST`) mostrar el archivo completo en dos columnas alineadas, `Original` a la izquierda y `Clon` a la derecha, con el número de línea de cada lado. Las líneas iguales DEBEN ser contexto sin casilla. Solo se renderiza la ventana visible; el archivo puede tener 20 000 líneas. Las líneas largas se truncan por la derecha con `…`.

#### Scenario: Archivo completo

- **GIVEN** un conflicto de texto abierto en D3
- **WHEN** se muestra la vista
- **THEN** se ven las columnas `Original` y `Clon` con el archivo completo y números de línea en ambos lados

#### Scenario: Contexto igual

- **GIVEN** una línea que existe igual en el original y en el clon
- **WHEN** se muestra en D3
- **THEN** aparece como contexto en ambas columnas y no lleva casilla

#### Scenario: Archivo grande

- **GIVEN** un archivo de 20 000 líneas
- **WHEN** se renderiza D3
- **THEN** solo se renderiza la ventana visible y el scroll recorre el archivo completo

#### Scenario: Líneas largas

- **GIVEN** una línea más ancha que la columna
- **WHEN** se muestra en D3
- **THEN** se trunca por la derecha con `…` sin desbordar el panel

### Requirement: Decisión por línea en la vista completa

En D3 cada línea cambiada DEBE (`MUST`) tener exactamente una versión elegida: `●` en el lado elegido y `○` en el descartado, con el elegido en texto normal y el descartado en `Faint`; NO DEBE (`MUST NOT`) existir un estado sin decidir. El lado sin línea DEBE mostrar `— sin línea —` y elegirlo DEBE omitir la línea. Bajo cada título DEBE mostrarse `N líneas elegidas`.

#### Scenario: Estado inicial

- **WHEN** se abre D3 para un conflicto de texto
- **THEN** todas las líneas cambiadas tienen el clon como versión elegida y el original como descartado

#### Scenario: Alternar una línea

- **WHEN** la persona elige la otra versión de una línea cambiada
- **THEN** esa línea cambia de lado, el resto del bloque conserva sus decisiones y no queda ninguna fila sin decidir

#### Scenario: Inserción o eliminación

- **GIVEN** una fila donde una línea solo existe en el original o solo en el clon
- **WHEN** la persona elige el lado que no tiene línea
- **THEN** la línea se omite y el contador del lado elegido no la cuenta

#### Scenario: Contadores por lado

- **WHEN** se muestran las columnas `Original` y `Clon`
- **THEN** cada una muestra debajo del título el número de líneas cambiadas elegidas de ese lado

#### Scenario: Línea descartada

- **WHEN** una línea cambiada tiene una versión descartada
- **THEN** la descartada se muestra en `Faint` y la elegida en texto normal

## MODIFIED Requirements

### Requirement: Tabla de conflictos

La tabla de conflictos DEBE (`MUST`) listar solo los archivos en conflicto en columnas `ARCHIVO`, `BLOQUES`, `LÍNEAS` y `ACCIÓN`, con los números alineados a la derecha y el cero como `·`, y mostrar los conteos de archivos nuevos, clones perfectos, no soportados y avisos. Cada fila DEBE ofrecer revisar, aplicar el clon o conservar el original. En un conflicto de texto, la fila DEBE indicar cuántos bloques y cuántas líneas cambiadas contiene, y la decisión de revisión DEBE expresarse como “híbrido (N de M líneas)”. El cursor DEBE ser `▌`. Si hay más filas de las que caben, DEBE mostrarse el indicador de scroll. Si falta una decisión, la zona de mensajes DEBE mostrar un `AVISO`.

#### Scenario: Tabla con solo conflictos y conteos

- **GIVEN** un despliegue con tres conflictos, cuatro archivos nuevos, diez clones perfectos y un aviso
- **WHEN** se muestra la tabla
- **THEN** la tabla lista solo los tres conflictos en las columnas `ARCHIVO`, `BLOQUES`, `LÍNEAS` y `ACCIÓN` y muestra los conteos de nuevos, clones perfectos, no soportados y avisos

#### Scenario: Acciones por archivo

- **GIVEN** un archivo en conflicto en la tabla
- **WHEN** la persona selecciona una acción
- **THEN** puede elegir revisar el detalle, aplicar el clon completo o conservar el original

#### Scenario: Detalle de un conflicto de texto

- **WHEN** la tabla muestra un conflicto de texto
- **THEN** la fila indica el número de bloques y de líneas cambiadas, y la decisión de revisión se muestra como “híbrido (N de M líneas)”

#### Scenario: Cero como punto

- **GIVEN** un conflicto sin bloques o sin líneas cambiadas
- **WHEN** se muestra su fila
- **THEN** el cero se muestra como `·`

#### Scenario: Tabla desplazada

- **GIVEN** más conflictos de los que caben en el body
- **WHEN** se mueve el cursor fuera de la ventana
- **THEN** la tabla se desplaza y muestra el indicador de filas arriba y abajo

#### Scenario: Decisión pendiente

- **GIVEN** una tabla con un conflicto sin decisión
- **WHEN** se intenta confirmar
- **THEN** la zona de mensajes muestra un `AVISO` con el archivo pendiente y no se avanza

### Requirement: Acciones globales y por bloque en el detalle

La vista de detalle DEBE (`MUST`) ofrecer acciones para aplicar todas las líneas del archivo o solo las del bloque actual al clon o al original. Las acciones por bloque NO DEBEN (`MUST NOT`) modificar las decisiones de otros bloques. Las teclas `←→` o `Espacio` DEBEN elegir la versión de la fila; `c` y `o` DEBEN actuar sobre el bloque actual, `C` y `O` sobre todo el archivo, y `Tab`/`Mayús+Tab` DEBEN recorrer los bloques.

#### Scenario: Todo el archivo al clon

- **WHEN** la persona elige la acción global de clon
- **THEN** todas las líneas cambiadas del archivo quedan en clon

#### Scenario: Todo el archivo al local

- **WHEN** la persona elige la acción global de original
- **THEN** todas las líneas cambiadas del archivo quedan en original

#### Scenario: Bloque actual al clon

- **WHEN** la persona elige la acción de bloque para el clon
- **THEN** todas las líneas cambiadas del bloque actual quedan en clon y los demás bloques conservan sus decisiones

#### Scenario: Bloque actual al local

- **WHEN** la persona elige la acción de bloque para el original
- **THEN** todas las líneas cambiadas del bloque actual quedan en original y los demás bloques conservan sus decisiones

#### Scenario: Navegación entre bloques

- **GIVEN** un archivo con varios bloques
- **WHEN** se pulsa `Tab` o `Mayús+Tab`
- **THEN** el foco pasa al bloque siguiente o anterior sin cambiar decisiones

#### Scenario: Elegir versión con flechas o Espacio

- **GIVEN** una línea cambiada con foco
- **WHEN** se pulsa `←`, `→` o `Espacio`
- **THEN** la versión elegida cambia al lado correspondiente sin mover el foco de la fila

### Requirement: Confirmación final

Cuando todos los conflictos tengan decisión, D4 DEBE (`MUST`) mostrar una tabla por app con columnas `NUEVOS`, `REEMPLAZOS`, `HÍBRIDOS`, `SIN DESPLEGAR` y `SIN CAMBIOS`, el cero como `·` y una fila de totales; debajo, el número de protegidos, la ruta del holocrón y el aviso de cerrar apps en la zona de mensajes. Los protegidos DEBEN ser los reemplazos más los híbridos que escribirán un destino existente. Las acciones DEBEN ser horizontales, con Retirada preseleccionada, y NO DEBE (`MUST NOT`) existir un atajo de una tecla para aplicar. Volver o cancelar NO DEBE escribir ni crear holocrón.

#### Scenario: Resumen antes de aplicar

- **GIVEN** todos los conflictos decididos
- **WHEN** se muestra la confirmación final
- **THEN** la tabla muestra una fila por app, las cinco columnas, el cero como `·`, una fila de totales, el número de protegidos y la ruta del holocrón

#### Scenario: Protegidos

- **GIVEN** una tabla con reemplazos e híbridos que escribirán destinos existentes
- **WHEN** se muestra la confirmación final
- **THEN** el número de protegidos es la suma de esos reemplazos e híbridos

#### Scenario: Ruta del holocrón

- **GIVEN** una confirmación final con al menos un destino existente
- **WHEN** se muestra la pantalla
- **THEN** se muestra la ruta del holocrón que se creará

#### Scenario: Aviso de apps abiertas

- **GIVEN** una confirmación final abierta
- **WHEN** se muestra la zona de mensajes
- **THEN** aparece un `AVISO` para cerrar las apps antes de desplegar

#### Scenario: Foco inicial y acciones

- **GIVEN** la confirmación final abierta
- **WHEN** se muestra la pantalla
- **THEN** Abortar misión, Retirada y Desplegar aparecen en una fila horizontal, Retirada está preseleccionada y las acciones se recorren con flechas y se aceptan con `Enter`

#### Scenario: Sin atajo de despliegue

- **GIVEN** la confirmación final abierta
- **WHEN** se pulsa una tecla que no sea de navegación o `Enter`
- **THEN** no se despliega nada

#### Scenario: Cancelar en la confirmación final

- **WHEN** la persona cancela en la confirmación final
- **THEN** no se escribe ningún destino y no se crea ningún holocrón

## REMOVED Requirements

### Requirement: Decisión por línea dentro del bloque

**Reason**: La vista de un bloque a la vez y la prohibición de previsualización completa se sustituyen por la vista de archivo completo en dos columnas.

**Migration**: Usar la vista de archivo completo y las acciones por línea y por bloque; las decisiones y la composición del archivo no cambian.
