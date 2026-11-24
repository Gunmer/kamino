# app-authoring Specification

## Purpose

Permitir que cada usuario cree, valide e inspeccione definiciones TOML de aplicaciones propias sin editar el código de Kamino.

## Requirements

### Requirement: Creación de definiciones con `apps new`

El comando `kamino apps new <id>` DEBE (`MUST`) crear una definición válida en el catálogo de usuario a partir de flags obligatorios de detección y rutas. NO DEBE (`MUST NOT`) escribir si la definición no supera la validación, si el id es inválido o si otro archivo ya declara ese id. `--print` DEBE (`MUST`) escribir el TOML a stdout sin tocar el disco y `--force` DEBE (`MUST`) permitir solo la sobrescritura de `<id>.toml`.

#### Scenario: Crear una app válida

- **GIVEN** un HOME con un `catalog.dir` efectivo que no contiene `miapp.toml`
- **WHEN** se ejecuta `kamino apps new miapp --command miapp --path .config/miapp/config.toml`
- **THEN** se crea `miapp.toml` con id, nombre, comando, ruta y exclusiones estándar
- **AND** `kamino apps validate` acepta la definición.

#### Scenario: Faltan detección o rutas

- **WHEN** se ejecuta `apps new miapp` sin `--command` ni `--bundle-id`, o sin `--path` ni `--optional-path`
- **THEN** el comando termina con código 2, explica qué falta y no escribe ningún archivo.

#### Scenario: Id duplicado en otro archivo

- **GIVEN** `otra.toml` declara `id = "miapp"`
- **WHEN** se ejecuta `apps new miapp` con flags válidos
- **THEN** el comando termina con código 1, nombra `otra.toml` y no escribe nada.

#### Scenario: Sobrescritura controlada

- **GIVEN** un `miapp.toml` existente
- **WHEN** se ejecuta `apps new miapp` con flags válidos sin `--force`
- **THEN** termina con código 1 y no modifica el archivo
- **AND** con `--force` lo reemplaza de forma atómica.

#### Scenario: Plantilla a stdout

- **WHEN** se ejecuta `apps new miapp` con flags válidos y `--print`
- **THEN** el TOML se escribe en la salida estándar y no se crea ni modifica ningún archivo.

#### Scenario: Override de una app integrada

- **GIVEN** un id que coincide con una app integrada
- **WHEN** se crea la definición
- **THEN** se escribe el archivo y se avisa de que reemplaza la definición integrada.

### Requirement: Validación e inspección del catálogo

`kamino apps validate [ruta]` DEBE (`MUST`) validar un archivo, un directorio o el catálogo de usuario efectivo con las mismas reglas que la carga, informar por archivo y terminar con código 1 si hay errores. `kamino apps list` DEBE (`MUST`) mostrar el catálogo fusionado con origen y estado, sin fallar por definiciones inválidas.

#### Scenario: Validar el catálogo efectivo

- **GIVEN** una definición válida y un TOML inválido en el catálogo de usuario
- **WHEN** se ejecuta `kamino apps validate`
- **THEN** informa OK para la primera y ERROR para el segundo, nombra el archivo y termina con código 1.

#### Scenario: Validar un archivo correcto

- **WHEN** se ejecuta `kamino apps validate ruta/al/archivo.toml`
- **THEN** termina con código 0 y no escribe nada.

#### Scenario: Directorio con id duplicado

- **GIVEN** dos archivos del mismo directorio declaran el mismo id
- **WHEN** se ejecuta `kamino apps validate <directorio>`
- **THEN** informa el duplicado, nombra ambos archivos y termina con código 1.

#### Scenario: Listar el catálogo fusionado

- **GIVEN** apps integradas y una definición de usuario
- **WHEN** se ejecuta `kamino apps list`
- **THEN** muestra cada app con su origen integrado o usuario
- **AND** la definición de usuario sustituye a la integrada con el mismo id.

#### Scenario: Listar con archivos inválidos

- **GIVEN** un TOML inválido en el catálogo de usuario
- **WHEN** se ejecuta `kamino apps list`
- **THEN** muestra el error del archivo junto a las demás entradas y el comando termina con código 0.

#### Scenario: Ruta efectiva visible

- **WHEN** se ejecuta `kamino apps list` o `kamino apps validate` sin ruta
- **THEN** la salida muestra el directorio efectivo del catálogo de usuario.

### Requirement: Avisos de autoría

La validación DEBE (`MUST`) avisar, sin fallar, cuando una definición no declara `exclude`, cuando su `id` no coincide con el nombre del archivo o cuando reemplaza una app integrada. Los avisos NO DEBEN (`MUST NOT`) cambiar el código de una validación correcta.

#### Scenario: Sin exclusiones

- **GIVEN** una definición válida sin `exclude`
- **WHEN** se ejecuta `apps validate`
- **THEN** muestra un AVISO sobre credenciales, cachés e historiales y termina con código 0.

#### Scenario: Id distinto del archivo

- **GIVEN** `miapp.toml` con `id = "otra"`
- **WHEN** se ejecuta `apps validate`
- **THEN** muestra un AVISO con ambos nombres y termina con código 0.

#### Scenario: Override integrado

- **GIVEN** una definición de usuario con el id de una app integrada
- **WHEN** se ejecuta `apps validate`
- **THEN** muestra un AVISO de que reemplaza la definición integrada y termina con código 0.

### Requirement: Plantilla de definición

La plantilla DEBE (`MUST`) ser TOML válido, incluir comentarios en español, declarar la detección y al menos una ruta, y traer activa la lista estándar de exclusiones de credenciales, cachés e historiales. Los campos avanzados DEBEN (`MUST`) aparecer comentados como ejemplo.

#### Scenario: Contenido de la plantilla

- **WHEN** se genera una definición con flags de detección y rutas
- **THEN** contiene `id`, `name`, `command` o `bundle_id`, las rutas, la lista estándar de `exclude` y ejemplos comentados de `optional_paths`, `optional_patterns`, `env_paths` y `xdg_paths`.

#### Scenario: Valores con caracteres especiales

- **GIVEN** un nombre o una ruta con comillas o barras invertidas
- **WHEN** se genera la plantilla
- **THEN** el TOML resultante se parsea y supera la validación antes de escribirse.

### Requirement: Pantalla «Añadir app»

La TUI DEBE (`MUST`) ofrecer «Añadir app» en la pantalla de Configuración. La pantalla DEBE (`MUST`) crear un borrador oculto en el catálogo efectivo, abrir el editor del usuario y validar al volver. Un borrador sin editar NO DEBE (`MUST NOT`) guardarse. Los errores DEBEN (`MUST`) mostrarse sin cerrar la pantalla y permitir reeditar o descartar.

#### Scenario: Abrir desde Configuración

- **GIVEN** la pantalla de Configuración abierta
- **WHEN** se enfoca «Añadir app» y se pulsa `Enter`
- **THEN** se abre la pantalla de autoría con el catálogo efectivo.

#### Scenario: Cancelar antes de editar

- **GIVEN** la pantalla de Configuración abierta
- **WHEN** se elige «Añadir app» y se cancela antes de abrir el editor
- **THEN** no se crea ningún archivo
- **AND** se vuelve a Configuración.

#### Scenario: Borrador sin editar

- **GIVEN** el editor se abre y se cierra sin cambiar el borrador
- **WHEN** la pantalla valida el contenido
- **THEN** muestra los errores y no crea `<id>.toml`.

#### Scenario: Edición válida

- **GIVEN** un borrador completado con una definición válida
- **WHEN** se vuelve del editor
- **THEN** se valida, se renombra de forma atómica a `<id>.toml` y se recarga el catálogo.

#### Scenario: TOML inválido al volver

- **GIVEN** un borrador con TOML inválido o campos obligatorios vacíos
- **WHEN** se vuelve del editor
- **THEN** se muestran los errores, no se escribe nada y se ofrece reeditar o descartar.

#### Scenario: Editor no disponible

- **GIVEN** ningún editor disponible
- **WHEN** se intenta editar
- **THEN** se muestra la ruta del borrador y las instrucciones sin perderlo.

#### Scenario: Confirmación de reemplazo

- **GIVEN** un borrador válido cuyo id ya existe o cuyo archivo destino ya existe
- **WHEN** se intenta guardar
- **THEN** se pide confirmación antes de escribir y, si se rechaza, no se modifica ningún archivo.

### Requirement: Integración con la detección

La creación DEBE (`MUST`) comprobar si la app está instalada con la misma detección que el catálogo y avisar si no lo está, sin impedir la creación. La app creada DEBE (`MUST`) participar en la selección y el despliegue como cualquier app del catálogo.

#### Scenario: App no detectada

- **GIVEN** el comando o bundle de la nueva definición no está instalado
- **WHEN** se crea la definición
- **THEN** se avisa de que existirá pero no aparecerá como instalada y la creación se completa.

#### Scenario: App detectada

- **GIVEN** el comando o bundle de la nueva definición está instalado
- **WHEN** se crea la definición
- **THEN** no se muestra el aviso y la app aparece en la selección de clonación.

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

### Requirement: Personalización con `apps customize`

El comando `kamino apps customize <id>` DEBE (`MUST`) escribir en el catálogo de usuario `<id>.toml` con el contenido íntegro de la definición integrada de ese id. NO DEBE (`MUST NOT`) escribir si el id no es integrado, si otro archivo ya declara ese id o si `<id>.toml` existe sin `--force`. `--print` DEBE (`MUST`) escribir el TOML a stdout sin tocar el disco.

#### Scenario: Personalizar una app integrada

- **GIVEN** un catálogo de usuario sin `codex.toml` y `codex` integrada con recetas `[[install]]`
- **WHEN** se ejecuta `kamino apps customize codex`
- **THEN** se crea `codex.toml` con el mismo contenido que la definición integrada, recetas incluidas
- **AND** `kamino apps validate` lo acepta y avisa de que reemplaza la definición integrada.

#### Scenario: Id no integrado

- **WHEN** se ejecuta `kamino apps customize noexiste`
- **THEN** el comando termina con código 2, indica que no hay una app integrada con ese id y no escribe nada.

#### Scenario: Archivo ya existente

- **GIVEN** un `codex.toml` en el catálogo de usuario
- **WHEN** se ejecuta `kamino apps customize codex` sin `--force`
- **THEN** termina con código 1 y no modifica el archivo
- **AND** con `--force` lo reemplaza de forma atómica por la definición integrada.

#### Scenario: Id declarado en otro archivo

- **GIVEN** `otra.toml` declara `id = "codex"`
- **WHEN** se ejecuta `kamino apps customize codex`
- **THEN** termina con código 1, nombra `otra.toml` y no escribe nada.

#### Scenario: Copia a stdout

- **WHEN** se ejecuta `kamino apps customize codex --print`
- **THEN** el TOML integrado se escribe en la salida estándar y no se crea ni modifica ningún archivo.

### Requirement: Pantalla «Personalizar app»

La TUI DEBE (`MUST`) ofrecer «Personalizar app» en la pantalla de Configuración. La pantalla DEBE (`MUST`) listar las apps integradas, crear un borrador oculto con el TOML integrado de la elegida y seguir el mismo flujo de edición, validación, confirmación de reemplazo y descarte que «Añadir app». Un borrador sin editar PUEDE (`MAY`) guardarse porque ya es válido.

#### Scenario: Elegir una app integrada

- **GIVEN** la pantalla de Configuración abierta
- **WHEN** se enfoca «Personalizar app», se pulsa `Enter` y se elige una app
- **THEN** se abre la pantalla de autoría con el borrador sembrado con la definición integrada de esa app.

#### Scenario: Cancelar antes de editar

- **GIVEN** el selector de apps abierto
- **WHEN** se cancela con `Esc`
- **THEN** no se crea ningún archivo y se vuelve a Configuración.

#### Scenario: Edición válida

- **GIVEN** un borrador sembrado y modificado con una definición válida
- **WHEN** se vuelve del editor y se confirma el reemplazo
- **THEN** se renombra de forma atómica a `<id>.toml` y se recarga el catálogo.

#### Scenario: Id modificado en el borrador

- **GIVEN** un borrador sembrado cuyo `id` se cambia a otro valor válido
- **WHEN** se vuelve del editor
- **THEN** se valida y se guarda como `<nuevo-id>.toml` con las mismas reglas que «Añadir app».
