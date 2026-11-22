# Spec Delta

## RENAMED Requirements

- FROM: `### Requirement: La vía no interactiva solo lee el límite de holocrones`
- TO: `### Requirement: La vía no interactiva lee el catálogo y el límite de holocrones`

## MODIFIED Requirements

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

La vía no interactiva de la CLI NO DEBE (`MUST NOT`) crear, modificar ni guardar `config.toml`. `clone`, `deploy` y `apps` DEBEN (`MUST`) leer `catalog.dir`; `clone` y `deploy` DEBEN (`MUST`) usar los valores integrados para `clone.output_dir` y `deploy.snapshots_dir`, y `deploy` DEBE (`MUST`) leer `deploy.snapshots_limit`. Si el archivo falta o es inválido, DEBEN (`MUST`) usar el catálogo integrado y los valores por defecto sin bloquear; `clone` y `deploy` callan y `apps` avisa.

#### Scenario: CLI con configuración distinta

- **GIVEN** un `config.toml` válido con `clone.output_dir`, `deploy.snapshots_dir` y `catalog.dir` distintos de los integrados
- **WHEN** se ejecuta `kamino clone` o `kamino deploy` sin terminal interactiva
- **THEN** el comando usa los valores integrados para `clone.output_dir` y `deploy.snapshots_dir`
- **AND** no usa esos valores del archivo
- **AND** sí lee `catalog.dir` para el catálogo de usuario.

#### Scenario: Catálogo de usuario configurado

- **GIVEN** un `config.toml` válido con `catalog.dir = "~/kamino-apps"` y una definición `miapp.toml`
- **WHEN** se ejecuta `kamino clone --apps miapp` sin terminal interactiva
- **THEN** la app se detecta y se incluye en el clon
- **AND** `kamino deploy` la trata como una app conocida del catálogo.

#### Scenario: Catálogo por defecto sin clave

- **GIVEN** un `config.toml` válido sin `catalog.dir`
- **WHEN** se ejecuta `kamino clone` sin terminal interactiva
- **THEN** se usa `~/.config/kamino/apps` como catálogo de usuario.

#### Scenario: CLI con límite de holocrones configurado

- **GIVEN** un `config.toml` válido con `deploy.snapshots_limit = 2` y tres holocrones en el directorio integrado
- **WHEN** se ejecuta `kamino deploy` sin terminal interactiva y crea un holocrón
- **THEN** el comando borra los más antiguos hasta dejar dos

#### Scenario: CLI con configuración inválida

- **GIVEN** un `config.toml` con TOML inválido o claves desconocidas
- **WHEN** se ejecuta `kamino clone`, `kamino deploy` o `kamino update` con salida de texto
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

### Requirement: Creación de directorios al usarlos

Los directorios de salida y de holocrones DEBEN (`MUST`) crearse cuando se usan, no al guardar la configuración ni al abrir la pantalla de configuración. Un `catalog.dir` inexistente DEBE (`MUST`) equivaler a un catálogo de usuario vacío al cargar y NO DEBE (`MUST NOT`) crearse automáticamente. La creación explícita de una definición con `apps new` o la pantalla «Añadir app» SÍ DEBE (`MUST`) crear `catalog.dir` si falta.

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

#### Scenario: Creación explícita del catálogo

- **GIVEN** `catalog.dir` apunta a un directorio que no existe
- **WHEN** se crea una definición con `apps new` o desde la pantalla «Añadir app»
- **THEN** el directorio se crea y el archivo se escribe en él.

## ADDED Requirements

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
