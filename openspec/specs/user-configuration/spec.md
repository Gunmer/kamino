# user-configuration Specification

## Purpose

Permitir que Kamino lea y guarde las rutas de catálogo, clones y holocrones y el límite de holocrones en `~/.config/kamino/config.toml`, con una pantalla de configuración, validación estricta y valores por defecto bajo `vault/`, compartiendo `catalog.dir` y `implant.snapshots_limit` con la CLI de texto.

## Requirements

### Requirement: Archivo de configuración opcional de la TUI

La TUI DEBE (`MUST`) leer, si existe, `~/.config/kamino/config.toml` al arrancar. La ausencia del archivo NO DEBE (`MUST NOT`) ser un error y DEBE (`MUST`) equivaler a usar los valores por defecto. La TUI DEBE (`MUST`) cargar las definiciones de usuario desde `catalog.dir`.

#### Scenario: TUI sin archivo de configuración

- **GIVEN** un HOME sin `~/.config/kamino/config.toml`
- **WHEN** se abre la TUI
- **THEN** sus flujos usan los valores por defecto documentados

#### Scenario: TUI con archivo de configuración

- **GIVEN** un `~/.config/kamino/config.toml` válido con rutas distintas de las integradas
- **WHEN** se abre la TUI y se inicia un flujo
- **THEN** el flujo usa las rutas del archivo

#### Scenario: Catálogo de usuario configurado

- **GIVEN** un `config.toml` válido con `catalog.dir = "~/kamino-apps"` y una definición en ese directorio
- **WHEN** la TUI carga el catálogo
- **THEN** incluye la definición y la ofrece en los flujos como cualquier app del catálogo

#### Scenario: El archivo no se crea al arrancar

- **GIVEN** un HOME sin `~/.config/kamino/config.toml`
- **WHEN** se abre la TUI o se entra en la pantalla de configuración
- **THEN** el archivo no se crea

### Requirement: La vía no interactiva lee el catálogo y el límite de holocrones

La vía no interactiva de la CLI NO DEBE (`MUST NOT`) crear, modificar ni guardar `config.toml`. `clone`, `implant` y `apps` DEBEN (`MUST`) leer `catalog.dir`; `clone` y `implant` DEBEN (`MUST`) usar los valores integrados para `clone.output_dir` y `implant.snapshots_dir`, y `implant` DEBE (`MUST`) leer `implant.snapshots_limit`. Si el archivo falta o es inválido, DEBEN (`MUST`) usar el catálogo integrado y los valores por defecto sin bloquear; `clone` y `implant` callan y `apps` avisa.

#### Scenario: CLI con configuración distinta

- **GIVEN** un `config.toml` válido con `clone.output_dir`, `implant.snapshots_dir` y `catalog.dir` distintos de los integrados
- **WHEN** se ejecuta `kamino clone` o `kamino implant` sin terminal interactiva
- **THEN** el comando usa los valores integrados para `clone.output_dir` y `implant.snapshots_dir`
- **AND** no usa esos valores del archivo
- **AND** sí lee `catalog.dir` para el catálogo de usuario.

#### Scenario: Catálogo de usuario configurado

- **GIVEN** un `config.toml` válido con `catalog.dir = "~/kamino-apps"` y una definición `miapp.toml`
- **WHEN** se ejecuta `kamino clone --apps miapp` sin terminal interactiva
- **THEN** la app se detecta y se incluye en el clon
- **AND** `kamino implant` la trata como una app conocida del catálogo.

#### Scenario: Catálogo por defecto sin clave

- **GIVEN** un `config.toml` válido sin `catalog.dir`
- **WHEN** se ejecuta `kamino clone` sin terminal interactiva
- **THEN** se usa `~/.config/kamino/apps` como catálogo de usuario.

#### Scenario: CLI con límite de holocrones configurado

- **GIVEN** un `config.toml` válido con `implant.snapshots_limit = 2` y tres holocrones en el directorio integrado
- **WHEN** se ejecuta `kamino implant` sin terminal interactiva y crea un holocrón
- **THEN** el comando borra los más antiguos hasta dejar dos

#### Scenario: CLI con configuración inválida

- **GIVEN** un `config.toml` con TOML inválido o claves desconocidas
- **WHEN** se ejecuta `kamino clone`, `kamino implant` o `kamino update` con salida de texto
- **THEN** el comando funciona con el catálogo integrado y los valores por defecto
- **AND** no informa del archivo.

#### Scenario: Comandos apps con configuración inválida

- **GIVEN** un `config.toml` con TOML inválido o claves desconocidas
- **WHEN** se ejecuta `kamino apps list` o `kamino apps validate`
- **THEN** el comando usa el catálogo integrado, avisa por la salida de error y muestra la ruta efectiva del catálogo.

#### Scenario: CLI no crea el archivo

- **GIVEN** un HOME sin `~/.config/kamino/config.toml`
- **WHEN** termina cualquier comando de la CLI con salida de texto
- **THEN** el archivo sigue sin existir

#### Scenario: CLI no modifica el archivo

- **GIVEN** un `config.toml` existente
- **WHEN** termina cualquier comando de la CLI con salida de texto
- **THEN** el archivo conserva sus bytes

### Requirement: Creación del archivo solo al guardar

La TUI NO DEBE (`MUST NOT`) crear `config.toml` al arrancar ni al abrir la pantalla de configuración. DEBE (`MUST`) crearlo con el primer guardado desde esa pantalla. Al guardar, DEBE (`MUST`) escribir siempre las cuatro claves admitidas.

#### Scenario: Primer guardado sin archivo

- **GIVEN** un HOME sin `~/.config/kamino/config.toml` y la pantalla de configuración abierta
- **WHEN** se guardan los valores
- **THEN** se crea el archivo con `catalog.dir`, `clone.output_dir`, `implant.snapshots_dir` y `implant.snapshots_limit`

#### Scenario: Guardado con archivo existente

- **GIVEN** un `config.toml` existente
- **WHEN** se guardan los valores desde la pantalla
- **THEN** las cuatro claves quedan escritas con los valores mostrados

#### Scenario: Cancelar sin guardar

- **GIVEN** la pantalla de configuración abierta con cambios sin guardar
- **WHEN** se cancela o se vuelve al menú
- **THEN** no se crea ni se modifica el archivo

### Requirement: Esquema estricto de configuración

El archivo DEBE (`MUST`) ser TOML y admitir solo `catalog.dir`, `clone.output_dir`, `implant.snapshots_dir` y `implant.snapshots_limit`. Las tres rutas DEBEN (`MUST`) ser cadenas y `implant.snapshots_limit` DEBE (`MUST`) ser un entero mayor o igual que 0. DEBE (`MUST`) rechazar TOML inválido, secciones o claves desconocidas, tipos incorrectos, valores vacíos, rutas relativas, rutas con `~usuario` y rutas existentes que no son directorios, con un mensaje que nombre la sección o la clave.

#### Scenario: TOML inválido

- **GIVEN** un archivo con sintaxis TOML incorrecta
- **WHEN** la TUI intenta cargarlo
- **THEN** muestra el error y abre la pantalla de configuración para corregirlo

#### Scenario: Sección desconocida

- **GIVEN** un archivo con una sección `[ui]`
- **WHEN** la TUI intenta cargarlo
- **THEN** rechaza la sección y muestra el error correspondiente

#### Scenario: Clave desconocida

- **GIVEN** un archivo con `clone.force`, `implant.yes` o una clave no documentada
- **WHEN** la TUI intenta cargarlo
- **THEN** rechaza la clave y muestra su nombre

#### Scenario: Límite no entero

- **GIVEN** un archivo con `implant.snapshots_limit = "5"` o `implant.snapshots_limit = 2.5`
- **WHEN** la TUI intenta cargarlo
- **THEN** rechaza el valor, muestra que debe ser un entero y abre la pantalla con el error

#### Scenario: Límite negativo

- **GIVEN** un archivo con `implant.snapshots_limit = -1`
- **WHEN** la TUI intenta cargarlo
- **THEN** rechaza el valor y usa los valores por defecto

#### Scenario: Tipo incorrecto

- **GIVEN** un valor que no es una cadena, como `clone.output_dir = 42`
- **WHEN** la TUI intenta cargarlo
- **THEN** muestra el tipo esperado y no aplica el valor

#### Scenario: Valor vacío, ruta relativa o `~usuario`

- **GIVEN** `clone.output_dir = ""`, `clone.output_dir = "Downloads"` o `clone.output_dir = "~otro/vault"`
- **WHEN** la TUI intenta cargarlo
- **THEN** rechaza el valor y muestra que la ruta debe ser absoluta o empezar por `~` o `~/`

#### Scenario: Ruta existente que no es directorio

- **GIVEN** `catalog.dir` apunta a un archivo existente
- **WHEN** la TUI intenta cargarlo
- **THEN** lo rechaza, muestra que debe ser un directorio y abre la pantalla de configuración con el error

### Requirement: Valores por defecto

Los valores por defecto DEBEN (`MUST`) ser `~/.config/kamino/apps` para `catalog.dir`, `~/.config/kamino/vault/` para `clone.output_dir`, `~/.config/kamino/vault/snapshot/` para `implant.snapshots_dir` y `5` para `implant.snapshots_limit`. Una clave ausente DEBE (`MUST`) usar su valor por defecto.

#### Scenario: Valores por defecto del catálogo

- **GIVEN** un archivo sin `catalog.dir`
- **WHEN** la TUI carga el catálogo
- **THEN** usa `~/.config/kamino/apps`

#### Scenario: Valores por defecto de la salida

- **GIVEN** un archivo sin `clone.output_dir`
- **WHEN** se abre C2 en la TUI
- **THEN** se propone `~/.config/kamino/vault/clone-yyyyMMdd.tar.gz`

#### Scenario: Valores por defecto de los holocrones

- **GIVEN** un archivo sin `implant.snapshots_dir`
- **WHEN** un despliegue de la TUI necesita crear un holocrón
- **THEN** se escribe en `~/.config/kamino/vault/snapshot/`

### Requirement: Creación de directorios al usarlos

Los directorios de salida y de holocrones DEBEN (`MUST`) crearse cuando se usan, no al guardar la configuración ni al abrir la pantalla de configuración. Un `catalog.dir` inexistente DEBE (`MUST`) equivaler a un catálogo de usuario vacío al cargar y NO DEBE (`MUST NOT`) crearse automáticamente. La creación explícita de una definición con `apps new` o la pantalla «Añadir app» SÍ DEBE (`MUST`) crear `catalog.dir` si falta.

#### Scenario: Directorio de salida ausente al clonar

- **GIVEN** `~/.config/kamino/vault/` o el `clone.output_dir` configurado no existe
- **WHEN** se inicia una clonación
- **THEN** el directorio se crea antes de escribir el clon

#### Scenario: Directorio de holocrones ausente al desplegar

- **GIVEN** `~/.config/kamino/vault/snapshot/` o el `implant.snapshots_dir` configurado no existe
- **WHEN** un despliegue necesita crear un holocrón
- **THEN** el directorio se crea y el holocrón se escribe en él

#### Scenario: Guardar no crea directorios configurados

- **GIVEN** la pantalla de configuración con rutas que no existen
- **WHEN** se guardan los valores
- **THEN** solo se escribe el archivo y no se crean los directorios de catálogo, salida ni holocrones
- **AND** sí se crea `~/.config/kamino` si falta para poder escribir el archivo

#### Scenario: Catálogo ausente

- **GIVEN** `catalog.dir` apunta a un directorio que no existe
- **WHEN** la TUI carga el catálogo
- **THEN** usa solo las definiciones integradas y no crea el directorio

#### Scenario: Creación explícita del catálogo

- **GIVEN** `catalog.dir` apunta a un directorio que no existe
- **WHEN** se crea una definición con `apps new` o desde la pantalla «Añadir app»
- **THEN** el directorio se crea y el archivo se escribe en él.

### Requirement: Precedencia de rutas en la TUI

En la TUI, una ruta escrita manualmente en C2 DEBE (`MUST`) prevalecer sobre `clone.output_dir` y sobre el valor por defecto. Sin ruta manual, `clone.output_dir` DEBE (`MUST`) prevalecer sobre `~/.config/kamino/vault/`. `catalog.dir` y `implant.snapshots_dir` DEBEN (`MUST`) usar el archivo si existe la clave y, si no, su valor por defecto.

#### Scenario: Ruta manual en C2

- **GIVEN** `clone.output_dir` configurado y una ruta escrita manualmente en C2
- **WHEN** se continúa el flujo
- **THEN** se usa la ruta escrita manualmente

#### Scenario: Sin ruta manual

- **GIVEN** `clone.output_dir` configurado y C2 sin cambios
- **WHEN** se continúa el flujo
- **THEN** se usa `clone.output_dir`

#### Scenario: Sin clave en el archivo

- **GIVEN** un archivo sin `clone.output_dir`, `catalog.dir` o `implant.snapshots_dir`
- **WHEN** la TUI usa cada valor
- **THEN** aplica el valor por defecto correspondiente

### Requirement: Pantalla de configuración

La TUI DEBE (`MUST`) ofrecer una pantalla de Configuración accesible desde el menú principal después de Actualizar, sin atajo de teclado. La pantalla DEBE (`MUST`) mostrar y permitir editar `catalog.dir`, `clone.output_dir`, `implant.snapshots_dir` y `implant.snapshots_limit` con sus valores efectivos. El campo del límite DEBE (`MUST`) admitir solo dígitos. `Ctrl+S` DEBE guardar y `Esc` DEBE volver sin guardar.

#### Scenario: Acceso desde el menú

- **GIVEN** el menú principal abierto
- **WHEN** se selecciona Configuración y se pulsa `Enter`
- **THEN** se abre la pantalla con las cuatro opciones

#### Scenario: Guardar con Ctrl+S

- **GIVEN** la pantalla de configuración abierta desde el menú con valores válidos
- **WHEN** se pulsa `Ctrl+S`
- **THEN** se guardan los valores y se vuelve al menú

#### Scenario: Valores efectivos

- **GIVEN** un archivo sin algunas claves
- **WHEN** se abre la pantalla de configuración
- **THEN** los campos sin clave muestran el valor por defecto correspondiente

#### Scenario: Error de validación al guardar

- **GIVEN** un campo con una ruta relativa, vacía o con un archivo existente que no es un directorio
- **WHEN** se intenta guardar
- **THEN** se muestra el error y no se escribe el archivo

#### Scenario: Cancelar la pantalla

- **GIVEN** la pantalla de configuración abierta desde el menú con cambios
- **WHEN** se pulsa `Esc`
- **THEN** se vuelve al menú sin modificar el archivo

### Requirement: Guardado atómico y aplicación inmediata

Al guardar, la TUI DEBE (`MUST`) escribir las cuatro claves de forma atómica, crear `~/.config/kamino` y el archivo si faltan, y aplicar los nuevos valores de inmediato a los flujos de la sesión actual. Si la escritura falla, el archivo anterior DEBE (`MUST`) quedar intacto.

#### Scenario: Guardado correcto

- **GIVEN** la pantalla de configuración abierta desde el menú con valores válidos
- **WHEN** se guarda
- **THEN** el archivo se reemplaza de forma atómica y se vuelve al menú con los nuevos valores activos

#### Scenario: Aplicación inmediata

- **GIVEN** una configuración guardada con un `implant.snapshots_dir` nuevo
- **WHEN** se inicia un despliegue en la misma sesión de la TUI
- **THEN** el holocrón se escribe en el directorio nuevo sin reiniciar la aplicación

#### Scenario: Fallo al guardar

- **GIVEN** un archivo existente y un fallo al escribir el temporal
- **WHEN** se intenta guardar
- **THEN** el archivo anterior conserva sus bytes y la pantalla informa del error

### Requirement: Carga inválida al arrancar la TUI

Si el archivo existe pero es inválido, la TUI DEBE (`MUST`) abrir la pantalla de configuración con el error. DEBE (`MUST`) mostrar los valores que se hayan podido leer y usar el valor por defecto en los campos no legibles. El archivo NO DEBE (`MUST NOT`) sobrescribirse hasta que se guarde.

#### Scenario: TOML inválido

- **GIVEN** un archivo con sintaxis TOML incorrecta
- **WHEN** se abre la TUI
- **THEN** se abre la pantalla de configuración con el error y los valores por defecto en los campos

#### Scenario: Valor parcialmente legible

- **GIVEN** un archivo TOML válido con `catalog.dir` correcto y `clone.output_dir = "Downloads"`
- **WHEN** se abre la TUI
- **THEN** el campo de catálogo muestra el valor leído, el de salida muestra el valor relativo leído y la pantalla explica que debe ser absoluto o empezar por `~`

#### Scenario: No sobrescribir al abrir

- **GIVEN** un archivo inválido y la pantalla de configuración abierta con el error
- **WHEN** se permanece en la pantalla o se cancela sin guardar
- **THEN** el archivo conserva sus bytes

### Requirement: Recuperación tras corregir la configuración

Tras guardar desde la pantalla abierta por un archivo inválido, la TUI DEBE (`MUST`) continuar el flujo directo solicitado o volver al menú. Si se cancela sin guardar, DEBE (`MUST`) salir con código 1 y NO DEBE (`MUST NOT`) escribir el archivo.

#### Scenario: Continuar un flujo directo tras corregir

- **GIVEN** `kamino clone` o `kamino implant` abrió la TUI con una configuración inválida
- **WHEN** se corrige y se guarda desde la pantalla de configuración
- **THEN** la TUI continúa con la primera pantalla del flujo solicitado

#### Scenario: Volver al menú tras corregir

- **GIVEN** `kamino` abrió el menú con una configuración inválida
- **WHEN** se corrige y se guarda desde la pantalla de configuración
- **THEN** la TUI vuelve al menú principal

#### Scenario: Cancelar la corrección

- **GIVEN** la pantalla de configuración abierta por un archivo inválido
- **WHEN** se pulsa `Esc` sin guardar
- **THEN** la TUI sale con código 1 y el archivo conserva sus bytes

### Requirement: Sin automatización de confirmaciones destructivas

El archivo NO DEBE (`MUST NOT`) admitir claves que automaticen `--force`, `--yes`, `--dry-run` ni decisiones de conflicto. Esas decisiones DEBEN (`MUST`) seguir tomándose en cada operación.

#### Scenario: Intento de fijar force o yes

- **GIVEN** una configuración con `clone.force`, `implant.yes` o una política de conflictos
- **WHEN** la TUI intenta cargarla
- **THEN** la rechaza por contener claves desconocidas y no la aplica

#### Scenario: Despliegue no interactivo sin confirmación

- **GIVEN** un despliegue no interactivo sin `--yes` y sin `--dry-run`
- **WHEN** se intenta desplegar
- **THEN** el comando sigue exigiendo `--yes` y termina con código 1, aunque exista configuración

### Requirement: Invariantes de seguridad conservados

La configuración NO DEBE (`MUST NOT`) alterar el confinamiento al HOME, la verificación de integridad, la creación previa del holocrón, los reemplazos atómicos ni los códigos de salida existentes. El guardado de la configuración NO DEBE (`MUST NOT`) modificar clones, holocrones ni configuraciones de aplicaciones.

#### Scenario: Confinamiento al HOME

- **GIVEN** un `catalog.dir` fuera del HOME y una definición que intenta leer una ruta de una aplicación fuera del HOME
- **WHEN** se planifica el clon
- **THEN** la ruta externa de la aplicación se rechaza y no se lee ningún archivo de configuración de una aplicación fuera del HOME

#### Scenario: Clon alterado

- **GIVEN** una configuración válida y un clon con un checksum incorrecto
- **WHEN** se despliega
- **THEN** el despliegue se rechaza antes del holocrón y de modificar destinos

#### Scenario: Guardado sin efectos laterales

- **GIVEN** un clon existente, un holocrón existente y configuraciones de apps existentes
- **WHEN** se guarda la configuración de la TUI
- **THEN** solo cambia `config.toml`

### Requirement: Retención de holocrones

Cada vez que un despliegue crea un holocrón DEBE (`MUST`) borrar los más antiguos hasta que queden como máximo `implant.snapshots_limit`, cuyo valor por defecto DEBE (`MUST`) ser 5; un `0` DEBE (`MUST`) conservar todos. Solo DEBEN (`MUST`) ser elegibles los archivos regulares del directorio de holocrones con nombre `<fecha ISO>.tar.gz` o `<fecha ISO>-N.tar.gz`; cualquier otro archivo, directorio o enlace del directorio NO DEBE (`MUST NOT`) tocarse. La poda DEBE (`MUST`) ocurrir después de escribir el holocrón nuevo, así que una escritura fallida NO DEBE (`MUST NOT`) borrar ninguno. Si el excedente no se puede borrar, el despliegue DEBE (`MUST`) abortarse antes de escribir en los destinos. `--dry-run` NO DEBE (`MUST NOT`) borrar holocrones. Dos despliegues que comparten el directorio de holocrones DEBEN (`MUST`) serializarse con un bloqueo exclusivo del directorio, tomado antes de crear el holocrón y liberado al terminar de escribir los destinos; la poda de uno NO DEBE (`MUST NOT`) borrar el holocrón de un despliegue en curso.

#### Scenario: Despliegue por encima del límite

- **GIVEN** cinco holocrones en el directorio y `implant.snapshots_limit = 5`
- **WHEN** un despliegue crea el sexto
- **THEN** se borra el más antiguo y quedan cinco

#### Scenario: Límite desactivado

- **GIVEN** `implant.snapshots_limit = 0` y diez holocrones
- **WHEN** un despliegue crea otro
- **THEN** no se borra ninguno

#### Scenario: Archivos ajenos en el directorio

- **GIVEN** un directorio de holocrones con `notes.txt`, un `.tar.gz.bak` y un enlace simbólico
- **WHEN** la poda se ejecuta
- **THEN** esos archivos permanecen intactos

#### Scenario: Holocrón nuevo no escrito

- **GIVEN** un despliegue cuyo holocrón nuevo no se puede escribir
- **WHEN** falla la escritura
- **THEN** no se borra ningún holocrón anterior

#### Scenario: Despliegues concurrentes en el mismo directorio

- **GIVEN** dos despliegues simultáneos que comparten el directorio de holocrones y `implant.snapshots_limit = 1`
- **WHEN** el segundo despliegue intenta podar mientras el primero todavía aplica
- **THEN** el segundo espera el bloqueo del directorio y el holocrón del primero sigue existiendo hasta que el primero termina

#### Scenario: Simulación sin borrados

- **GIVEN** más holocrones que el límite configurado
- **WHEN** se ejecuta el despliegue con `--dry-run`
- **THEN** no se borra ningún holocrón

### Requirement: Holocrones anteriores sin migración

Los holocrones existentes en `~/Library/Application Support/kamino/snapshots/` NO DEBEN (`MUST NOT`) moverse ni copiarse automáticamente. DEBEN (`MUST`) seguir pudiendo desplegarse indicando su ruta.

#### Scenario: Holocrón anterior

- **GIVEN** un holocrón en `~/Library/Application Support/kamino/snapshots/`
- **WHEN** se actualiza Kamino y se guarda la nueva configuración
- **THEN** el holocrón permanece en su ruta original y puede desplegarse con `kamino implant <ruta>`

### Requirement: Campos de la pantalla de configuración

La pantalla de Configuración DEBE (`MUST`) mostrar `catalog.dir`, `clone.output_dir`, `implant.snapshots_dir` y `implant.snapshots_limit` con el campo compartido de una sola fila, etiquetas alineadas y foco visible. El foco DEBE empezar en Catálogo y `Tab`/`Mayús+Tab` y las flechas verticales DEBEN moverlo. El límite DEBE admitir solo dígitos. `Ctrl+S` DEBE guardar y `Esc` DEBE volver sin escribir. La pantalla DEBE caber en 80x24.

#### Scenario: Foco inicial en Catálogo

- **GIVEN** la pantalla de Configuración abierta desde el menú
- **WHEN** se renderiza por primera vez
- **THEN** el campo «Catálogo» tiene el foco, el marcador y el cursor, y los otros tres aparecen sin foco

#### Scenario: Recorrer los campos con Tab

- **GIVEN** la pantalla de Configuración con el foco en Catálogo
- **WHEN** se pulsa `Tab` o `Mayús+Tab`
- **THEN** el foco avanza o retrocede entre los cuatro campos y solo el campo enfocado muestra marcador, fondo y cursor

#### Scenario: Recorrer los campos con flechas

- **GIVEN** la pantalla de Configuración con un campo enfocado
- **WHEN** se pulsan las flechas arriba o abajo
- **THEN** el foco cambia al campo anterior o siguiente sin modificar los valores

#### Scenario: Ruta vacía

- **GIVEN** un campo de ruta sin valor
- **WHEN** se renderiza con o sin foco
- **THEN** aparece un placeholder atenuado y el cursor solo se muestra si el campo tiene el foco

#### Scenario: Ruta larga con scroll

- **GIVEN** una ruta más ancha que el campo y el cursor al final
- **WHEN** se renderiza con foco
- **THEN** el inicio se abrevia con `…`, el final queda visible y el cursor sigue visible

#### Scenario: Límite solo con dígitos

- **GIVEN** el campo del límite con foco
- **WHEN** se escribe una letra o un signo
- **THEN** el valor no cambia y solo se aceptan dígitos

#### Scenario: Error de validación en Configuración

- **GIVEN** un campo con una ruta relativa, vacía o no válida
- **WHEN** se intenta guardar con `Ctrl+S`
- **THEN** la zona de mensajes muestra un `ERROR`, el campo afectado se marca y las filas de los demás campos no se desplazan

#### Scenario: Guardar y volver

- **GIVEN** la pantalla de Configuración con valores válidos
- **WHEN** se pulsa `Ctrl+S`
- **THEN** se guardan los valores; con `Esc` se vuelve sin modificar el archivo

#### Scenario: Pantalla a 80x24

- **GIVEN** una terminal de 80x24
- **WHEN** se abre la pantalla de Configuración
- **THEN** los cuatro campos caben con una fila cada uno, el pie permanece visible y no se desborda el body

### Requirement: Recarga del catálogo tras guardar Configuración

Cuando `catalog.dir` cambia al guardar la pantalla de Configuración, la TUI DEBE (`MUST`) descartar el catálogo en memoria y cargarlo desde la nueva ruta en el siguiente flujo de la misma sesión, sin reiniciar la aplicación.

#### Scenario: Cambio de catálogo

- **GIVEN** la pantalla de Configuración con un `catalog.dir` nuevo que contiene una definición de usuario
- **WHEN** se guarda y se abre el flujo de clonación en la misma sesión
- **THEN** la definición aparece en la selección de apps.

#### Scenario: Catálogo nuevo inválido

- **GIVEN** la pantalla de Configuración con un `catalog.dir` nuevo que contiene un TOML inválido
- **WHEN** se guarda y se abre un flujo que carga el catálogo
- **THEN** la TUI muestra el error del catálogo y no modifica los archivos del catálogo.

### Requirement: Acción Añadir app en Configuración

La pantalla de Configuración DEBE (`MUST`) ofrecer una acción «Añadir app» después de los cuatro campos. La acción DEBE (`MUST`) alcanzarse con `Tab`, `Mayús+Tab` y las flechas verticales, y `Enter` DEBE (`MUST`) abrir la pantalla de autoría. Al volver, la TUI DEBE (`MUST`) regresar a Configuración conservando los valores.

#### Scenario: Abrir la autoría desde Configuración

- **GIVEN** la pantalla de Configuración abierta
- **WHEN** se enfoca «Añadir app» y se pulsa `Enter`
- **THEN** se abre la pantalla de autoría con el catálogo efectivo.

#### Scenario: Volver a Configuración

- **GIVEN** la pantalla de autoría abierta desde Configuración
- **WHEN** se cancela o se termina la definición
- **THEN** se vuelve a la pantalla de Configuración con el foco en «Añadir app».

#### Scenario: Guardar con la acción enfocada

- **GIVEN** la pantalla de Configuración con la acción «Añadir app» enfocada y valores válidos
- **WHEN** se pulsa `Ctrl+S`
- **THEN** se guardan los cuatro campos y la acción no modifica el archivo.
