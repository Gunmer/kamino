# Spec Delta

## ADDED Requirements

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
