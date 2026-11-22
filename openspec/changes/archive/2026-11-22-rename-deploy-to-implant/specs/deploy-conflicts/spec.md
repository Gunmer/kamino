# Spec Delta

## MODIFIED Requirements

### Requirement: Confirmación final

Cuando todos los conflictos tengan decisión, D4 DEBE (`MUST`) mostrar una tabla por app con columnas `NUEVOS`, `REEMPLAZOS`, `HÍBRIDOS`, `SIN IMPLANTAR` y `SIN CAMBIOS`, el cero como `·`, una fila de totales, el número de protegidos, la ruta del holocrón y el aviso de cerrar apps. Las acciones DEBEN ser horizontales, con Retirada preseleccionada, y NO DEBE (`MUST NOT`) existir un atajo de una tecla para implantar. Volver o cancelar NO DEBE escribir ni crear holocrón.

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
- **THEN** aparece un `AVISO` para cerrar las apps antes de implantar

#### Scenario: Foco inicial y acciones

- **GIVEN** la confirmación final abierta
- **WHEN** se muestra la pantalla
- **THEN** Abortar misión, Retirada e Implantar aparecen en una fila horizontal, Retirada está preseleccionada y las acciones se recorren con flechas y se aceptan con `Enter`

#### Scenario: Sin atajo de despliegue

- **GIVEN** la confirmación final abierta
- **WHEN** se pulsa una tecla que no sea de navegación o `Enter`
- **THEN** no se implanta nada

#### Scenario: Cancelar en la confirmación final

- **WHEN** la persona cancela en la confirmación final
- **THEN** no se escribe ningún destino y no se crea ningún holocrón

### Requirement: Modo y clasificación del resultado

Si el contenido final difiere del original, el archivo DEBE (`MUST`) recibir el modo del clon. Si la composición queda idéntica al original, NO DEBE (`MUST NOT`) escribirse ni cambiar permisos. El resultado DEBE (`MUST`) clasificar cada archivo como implantado, híbrido, omitido o sin cambios; los clones perfectos y las apps no instaladas NO DEBEN clasificarse como omitidos.

#### Scenario: Mezcla que cambia el contenido

- **GIVEN** una mezcla que difiere del original y del clon
- **WHEN** se aplica la implantación
- **THEN** el archivo se escribe con el modo del clon y se clasifica como híbrido.

#### Scenario: Mezcla idéntica al local

- **GIVEN** una composición que coincide con el contenido original
- **WHEN** se aplica la implantación
- **THEN** el archivo no se escribe, sus permisos no cambian y se clasifica como omitido.

#### Scenario: Aplicación completa del clon

- **GIVEN** un archivo decidido para aplicar el clon completo
- **WHEN** se aplica la implantación
- **THEN** el archivo se escribe con el contenido y el modo del clon y se clasifica como implantado.

#### Scenario: Revisión con todas las líneas en clon

- **GIVEN** un archivo revisado línea a línea con todas sus líneas decididas como clon
- **WHEN** se aplica la implantación
- **THEN** el archivo coincide con el clon y se clasifica como implantado.

#### Scenario: Clon perfecto en el resultado

- **GIVEN** un destino idéntico al clon
- **WHEN** termina la implantación
- **THEN** el archivo se clasifica como sin cambios y no se cuenta como omitido.

### Requirement: Resumen del resultado

El resultado DEBE (`MUST`) informar de los archivos implantados, híbridos, omitidos, sin cambios y de la ruta del holocrón creado. Un archivo híbrido DEBE (`MUST`) contar como éxito. La línea de salida DEBE mostrarse solo si el código no es 0 y DEBE incluir la orden para deshacer con `kamino implant <ruta>` cuando exista holocrón.

#### Scenario: Resumen con mezclas

- **GIVEN** una implantación que aplica dos archivos completos, hibrida dos y omite uno
- **WHEN** termina la implantación
- **THEN** el resumen informa de dos implantados, dos híbridos y uno omitido.

#### Scenario: Ruta del snapshot

- **GIVEN** una implantación que modificó al menos un archivo existente
- **WHEN** termina la implantación
- **THEN** el resumen muestra la ruta del holocrón creado y la orden para deshacer.

### Requirement: Holocrón de seguridad

Antes de escribir, la implantación DEBE (`MUST`) crear un holocrón con los destinos existentes que cambiarán y DEBE informar de su ruta y del número de archivos protegidos. Si ningún destino existe todavía, NO DEBE crear un holocrón vacío y DEBE informar de que no hay originales que proteger.

#### Scenario: Holocrón con archivos existentes

- **GIVEN** una implantación que reemplazará archivos existentes
- **WHEN** se aplica
- **THEN** el holocrón contiene esos archivos y el resumen muestra su ruta y el número de protegidos

#### Scenario: Solo archivos nuevos

- **GIVEN** una implantación cuyos destinos que cambiarán no existen
- **WHEN** se aplica
- **THEN** no se crea un holocrón vacío y se informa de que no había originales que proteger

#### Scenario: Deshacer desde el holocrón

- **GIVEN** un holocrón creado por una implantación
- **WHEN** se consulta el resumen
- **THEN** se muestra la orden `kamino implant <ruta>` para deshacer

### Requirement: Cancelación y aplicación parcial

La aplicación DEBE (`MUST`) poder cancelarse hasta crear el holocrón. Si falla una escritura después, el resultado DEBE informar de una implantación incompleta con los archivos aplicados, el archivo fallido y el holocrón disponible para deshacer, y NO DEBE presentarse como una implantación completada.

#### Scenario: Cancelar antes del holocrón

- **GIVEN** una aplicación que aún no creó el holocrón
- **WHEN** se cancela
- **THEN** no se escribe ningún destino y no se crea holocrón

#### Scenario: Fallo parcial

- **GIVEN** una implantación que escribe algunos archivos y falla en otro
- **WHEN** termina
- **THEN** se muestra la variante “Implantación incompleta” con los aplicados, el fallido y el holocrón

#### Scenario: Limpieza del staging

- **GIVEN** una preparación o aplicación cancelada
- **WHEN** termina la cancelación
- **THEN** el directorio temporal de implantación se elimina
