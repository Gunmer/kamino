# user-configuration Specification

## Purpose

Permitir que la TUI de Kamino lea y guarde las rutas de catálogo, clones y holocrones en `~/.config/kamino/config.toml`, con una pantalla de configuración, validación estricta y valores por defecto bajo `vault/`, sin que la CLI dependa de ese archivo.

## Requirements

### Requirement: Archivo de configuración opcional de la TUI

La TUI DEBE (`MUST`) leer, si existe, `~/.config/kamino/config.toml` al arrancar. La ausencia del archivo NO DEBE (`MUST NOT`) ser un error y DEBE (`MUST`) equivaler a usar los valores por defecto.

#### Scenario: TUI sin archivo de configuración

- **GIVEN** un HOME sin `~/.config/kamino/config.toml`
- **WHEN** se abre la TUI
- **THEN** sus flujos usan los valores por defecto documentados

#### Scenario: TUI con archivo de configuración

- **GIVEN** un `~/.config/kamino/config.toml` válido con rutas distintas de las integradas
- **WHEN** se abre la TUI y se inicia un flujo
- **THEN** el flujo usa las rutas del archivo

#### Scenario: El archivo no se crea al arrancar

- **GIVEN** un HOME sin `~/.config/kamino/config.toml`
- **WHEN** se abre la TUI o se entra en la pantalla de configuración
- **THEN** el archivo no se crea

### Requirement: La vía no interactiva no usa la configuración

La vía no interactiva de la CLI NO DEBE (`MUST NOT`) leer, crear, modificar ni guardar `config.toml`, ni aplicar sus valores. `clone`, `deploy` y `update` con salida de texto DEBEN (`MUST`) usar los valores integrados. Un archivo inválido NO DEBE (`MUST NOT`) alterar esa vía.

#### Scenario: CLI con configuración distinta

- **GIVEN** un `config.toml` válido con `clone.output_dir`, `deploy.snapshots_dir` y `catalog.dir` distintos de los integrados
- **WHEN** se ejecuta `kamino clone`, `kamino deploy` o `kamino update` sin terminal interactiva
- **THEN** el comando usa los valores integrados y no los del archivo

#### Scenario: CLI con configuración inválida

- **GIVEN** un `config.toml` con TOML inválido o claves desconocidas
- **WHEN** se ejecuta un comando de la CLI con salida de texto
- **THEN** el comando funciona con los valores integrados y no informa del archivo

#### Scenario: CLI no crea el archivo

- **GIVEN** un HOME sin `~/.config/kamino/config.toml`
- **WHEN** termina cualquier comando de la CLI con salida de texto
- **THEN** el archivo sigue sin existir

#### Scenario: CLI no modifica el archivo

- **GIVEN** un `config.toml` existente
- **WHEN** termina cualquier comando de la CLI con salida de texto
- **THEN** el archivo conserva sus bytes

### Requirement: Creación del archivo solo al guardar

La TUI NO DEBE (`MUST NOT`) crear `config.toml` al arrancar ni al abrir la pantalla de configuración. DEBE (`MUST`) crearlo con el primer guardado desde esa pantalla. Al guardar, DEBE (`MUST`) escribir siempre las tres claves admitidas.

#### Scenario: Primer guardado sin archivo

- **GIVEN** un HOME sin `~/.config/kamino/config.toml` y la pantalla de configuración abierta
- **WHEN** se guardan los valores
- **THEN** se crea el archivo con `catalog.dir`, `clone.output_dir` y `deploy.snapshots_dir`

#### Scenario: Guardado con archivo existente

- **GIVEN** un `config.toml` existente
- **WHEN** se guardan los valores desde la pantalla
- **THEN** las tres claves quedan escritas con los valores mostrados

#### Scenario: Cancelar sin guardar

- **GIVEN** la pantalla de configuración abierta con cambios sin guardar
- **WHEN** se cancela o se vuelve al menú
- **THEN** no se crea ni se modifica el archivo

### Requirement: Esquema estricto de configuración

El archivo DEBE (`MUST`) ser TOML y admitir solo `catalog.dir`, `clone.output_dir` y `deploy.snapshots_dir`. DEBE (`MUST`) rechazar TOML inválido, secciones o claves desconocidas, tipos incorrectos, valores vacíos, rutas relativas, rutas con `~usuario` y rutas existentes que no son directorios, con un mensaje que nombre la sección o la clave.

#### Scenario: TOML inválido

- **GIVEN** un archivo con sintaxis TOML incorrecta
- **WHEN** la TUI intenta cargarlo
- **THEN** muestra el error y abre la pantalla de configuración para corregirlo

#### Scenario: Sección desconocida

- **GIVEN** un archivo con una sección `[ui]`
- **WHEN** la TUI intenta cargarlo
- **THEN** rechaza la sección y muestra el error correspondiente

#### Scenario: Clave desconocida

- **GIVEN** un archivo con `clone.force`, `deploy.yes` o una clave no documentada
- **WHEN** la TUI intenta cargarlo
- **THEN** rechaza la clave y muestra su nombre

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

Los valores por defecto DEBEN (`MUST`) ser `~/.config/kamino/apps` para `catalog.dir`, `~/.config/kamino/vault/` para `clone.output_dir` y `~/.config/kamino/vault/snapshot/` para `deploy.snapshots_dir`. Una clave ausente DEBE (`MUST`) usar su valor por defecto.

#### Scenario: Valores por defecto del catálogo

- **GIVEN** un archivo sin `catalog.dir`
- **WHEN** la TUI carga el catálogo
- **THEN** usa `~/.config/kamino/apps`

#### Scenario: Valores por defecto de la salida

- **GIVEN** un archivo sin `clone.output_dir`
- **WHEN** se abre C2 en la TUI
- **THEN** se propone `~/.config/kamino/vault/clone-yyyyMMdd.tar.gz`

#### Scenario: Valores por defecto de los holocrones

- **GIVEN** un archivo sin `deploy.snapshots_dir`
- **WHEN** un despliegue de la TUI necesita crear un holocrón
- **THEN** se escribe en `~/.config/kamino/vault/snapshot/`

### Requirement: Creación de directorios al usarlos

Los directorios de salida y de holocrones DEBEN (`MUST`) crearse cuando se usan, no al guardar la configuración ni al abrir la pantalla de configuración. Un `catalog.dir` inexistente DEBE (`MUST`) equivaler a un catálogo de usuario vacío y NO DEBE (`MUST NOT`) crearse automáticamente.

#### Scenario: Directorio de salida ausente al clonar

- **GIVEN** `~/.config/kamino/vault/` o el `clone.output_dir` configurado no existe
- **WHEN** se inicia una clonación
- **THEN** el directorio se crea antes de escribir el clon

#### Scenario: Directorio de holocrones ausente al desplegar

- **GIVEN** `~/.config/kamino/vault/snapshot/` o el `deploy.snapshots_dir` configurado no existe
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

### Requirement: Precedencia de rutas en la TUI

En la TUI, una ruta escrita manualmente en C2 DEBE (`MUST`) prevalecer sobre `clone.output_dir` y sobre el valor por defecto. Sin ruta manual, `clone.output_dir` DEBE (`MUST`) prevalecer sobre `~/.config/kamino/vault/`. `catalog.dir` y `deploy.snapshots_dir` DEBEN (`MUST`) usar el archivo si existe la clave y, si no, su valor por defecto.

#### Scenario: Ruta manual en C2

- **GIVEN** `clone.output_dir` configurado y una ruta escrita manualmente en C2
- **WHEN** se continúa el flujo
- **THEN** se usa la ruta escrita manualmente

#### Scenario: Sin ruta manual

- **GIVEN** `clone.output_dir` configurado y C2 sin cambios
- **WHEN** se continúa el flujo
- **THEN** se usa `clone.output_dir`

#### Scenario: Sin clave en el archivo

- **GIVEN** un archivo sin `clone.output_dir`, `catalog.dir` o `deploy.snapshots_dir`
- **WHEN** la TUI usa cada valor
- **THEN** aplica el valor por defecto correspondiente

### Requirement: Pantalla de configuración

La TUI DEBE (`MUST`) ofrecer una pantalla de Configuración accesible desde el menú principal después de Actualizar, sin atajo de teclado. La pantalla DEBE (`MUST`) mostrar y permitir editar `catalog.dir`, `clone.output_dir` y `deploy.snapshots_dir` con sus valores efectivos. `Ctrl+S` DEBE guardar y `Esc` DEBE volver sin guardar.

#### Scenario: Acceso desde el menú

- **GIVEN** el menú principal abierto
- **WHEN** se selecciona Configuración y se pulsa `Enter`
- **THEN** se abre la pantalla con las tres rutas

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

Al guardar, la TUI DEBE (`MUST`) escribir las tres claves de forma atómica, crear `~/.config/kamino` y el archivo si faltan, y aplicar los nuevos valores de inmediato a los flujos de la sesión actual. Si la escritura falla, el archivo anterior DEBE (`MUST`) quedar intacto.

#### Scenario: Guardado correcto

- **GIVEN** la pantalla de configuración abierta desde el menú con valores válidos
- **WHEN** se guarda
- **THEN** el archivo se reemplaza de forma atómica y se vuelve al menú con los nuevos valores activos

#### Scenario: Aplicación inmediata

- **GIVEN** una configuración guardada con un `deploy.snapshots_dir` nuevo
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

- **GIVEN** `kamino clone` o `kamino deploy` abrió la TUI con una configuración inválida
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

- **GIVEN** una configuración con `clone.force`, `deploy.yes` o una política de conflictos
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

### Requirement: Holocrones anteriores sin migración

Los holocrones existentes en `~/Library/Application Support/kamino/snapshots/` NO DEBEN (`MUST NOT`) moverse ni copiarse automáticamente. DEBEN (`MUST`) seguir pudiendo desplegarse indicando su ruta.

#### Scenario: Holocrón anterior

- **GIVEN** un holocrón en `~/Library/Application Support/kamino/snapshots/`
- **WHEN** se actualiza Kamino y se guarda la nueva configuración
- **THEN** el holocrón permanece en su ruta original y puede desplegarse con `kamino deploy <ruta>`
