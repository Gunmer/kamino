# cli-options Specification

## Purpose

Permitir que las personas utilicen y descubran las opciones de Kamino mediante nombres cortos o largos, sin alterar la seguridad ni la compatibilidad de sus comandos.

## Requirements

### Requirement: Alias de las opciones de clonación

El comando `clone` DEBE (`MUST`) aceptar `-a` como alias de `--apps`, `-o` de `--output`, `-n` de `--dry-run`, `-v` de `--verbose` y `-f` de `--force`, con los mismos valores y efectos que las formas largas correspondientes.

#### Scenario: Selección y ruta mediante alias

- **WHEN** se ejecuta `kamino clone -a vscode,git -o clon.tar.gz`
- **THEN** se seleccionan las mismas aplicaciones y ruta que con `--apps vscode,git --output clon.tar.gz`.

#### Scenario: Salida detallada del clon

- **WHEN** se ejecuta un clon con `-v` en lugar de `--verbose`
- **THEN** se presenta el mismo nivel de detalle y se conserva el resultado del comando.

#### Scenario: Reemplazo forzado mediante alias

- **GIVEN** un clon existente en la ruta de destino
- **WHEN** se ejecuta `kamino clone -f` con la misma ruta
- **THEN** el clon se reemplaza igual que con `--force`.

### Requirement: Alias de las opciones de despliegue

El comando `deploy` DEBE (`MUST`) aceptar `-a` como alias de `--apps`, `-n` como alias de `--dry-run`, `-y` de `--yes` y `-v` de `--verbose`, con los mismos efectos que las formas largas correspondientes. El archivo de clon sigue siendo un argumento posicional.

#### Scenario: Confirmación y detalle mediante alias

- **WHEN** se ejecuta `kamino deploy clon.tar.gz -y -v` con un clon válido
- **THEN** se utiliza el mismo archivo, la misma selección y el mismo nivel de detalle que con `--yes --verbose`.

#### Scenario: Selección de apps mediante alias

- **WHEN** se ejecuta `kamino deploy clon.tar.gz -a zsh,git`
- **THEN** se seleccionan las mismas aplicaciones que con `--apps zsh,git` y no se abre la pantalla de selección.

### Requirement: Selección de apps en el despliegue

El comando `deploy` DEBE (`MUST`) aceptar `--apps` con una lista de identificadores separados por comas. Solo las apps instaladas y presentes en el manifiesto DEBEN desplegarse. Las apps del manifiesto no instaladas DEBEN informarse y NO DEBEN contar como omitidas ni alterar el código. Los identificadores pedidos con `--apps` que no se puedan cumplir DEBEN avisarse y contarse como omitidos, con código 3. La opción DEBE omitir la pantalla interactiva de selección de apps.

#### Scenario: Despliegue de apps concretas

- **WHEN** se ejecuta `kamino deploy clon.tar.gz --apps zsh,git`
- **THEN** solo se planifican y despliegan `zsh` y `git`, y no se abre la pantalla de selección de apps.

#### Scenario: App solicitada no instalada

- **WHEN** se ejecuta `kamino deploy clon.tar.gz --apps zsh,no-existe`
- **THEN** se avisa de `no-existe`, se despliega `zsh` y el comando termina con código 3 por el elemento omitido.

#### Scenario: App del manifiesto no instalada

- **GIVEN** un clon que contiene una app que no está instalada en este Mac y no fue pedida con `--apps`
- **WHEN** se despliega
- **THEN** la app se informa como “sin base en este Mac”, no se despliega, no cuenta como omitida y no altera el código de salida.

### Requirement: Simulación sin escrituras

El alias `-n` DEBE (`MUST`) conservar el comportamiento de `--dry-run` en ambos comandos, sin crear ni modificar archivos de clon, destinos desplegados o holocrones de seguridad. En `deploy`, la simulación NO DEBE abrir la interfaz interactiva; DEBE volcar los conflictos a la salida estándar y usar `--apps` o, en su defecto, todas las apps instaladas. En `clone` con terminal interactiva, la simulación DEBE terminar en C3 sin acción de clonar y sin escribir.

#### Scenario: Simulación de una clonación

- **WHEN** se ejecuta `kamino clone -o clon.tar.gz -n`
- **THEN** se muestra el resultado de la simulación sin crear el archivo de salida, igual que con `--dry-run`; con terminal interactiva, la revisión se muestra en C3.

#### Scenario: Simulación de un despliegue no interactivo

- **GIVEN** un clon válido y una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino deploy clon.tar.gz -n` sin `-y`
- **THEN** se permite la simulación sin modificar destinos ni crear holocrones de seguridad, igual que con `--dry-run`.

#### Scenario: Selección interactiva durante la simulación

- **GIVEN** una entrada con terminal interactiva
- **WHEN** se ejecuta `kamino deploy clon.tar.gz -n` sin `-y`
- **THEN** la simulación no abre la interfaz interactiva, usa `--apps` o todas las apps instaladas y vuelca los conflictos a la salida estándar.

#### Scenario: Simulación con selección explícita

- **GIVEN** un clon válido y una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino deploy clon.tar.gz -n --apps zsh`
- **THEN** se muestran los conflictos de `zsh` en la salida estándar, sin abrir la interfaz interactiva y sin modificar destinos ni crear holocrones de seguridad.

#### Scenario: Simulación con todas las apps

- **GIVEN** un clon válido y una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino deploy clon.tar.gz -n` sin `--apps`
- **THEN** se simulan todas las apps instaladas, se muestran sus conflictos en la salida estándar y no se modifica ningún destino ni se crea ningún holocrón de seguridad.

### Requirement: Confirmación de despliegues no interactivos

Un despliegue no interactivo DEBE (`MUST`) seguir exigiendo `-y` o `--yes`, salvo que se solicite una simulación. El alias no altera la validación ni las protecciones de despliegue existentes. Con `--yes`, los conflictos DEBEN resolverse aplicando el clon, sin abrir la revisión interactiva ni el menú.

#### Scenario: Despliegue sin confirmación ni simulación

- **GIVEN** una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino deploy clon.tar.gz` sin `-y`, `--yes`, `-n` ni `--dry-run`
- **THEN** el comando informa de que se requiere confirmación, termina con código 1 y no inicia el despliegue.

#### Scenario: Despliegue confirmado mediante el alias

- **GIVEN** una entrada sin terminal interactiva y un clon válido
- **WHEN** se ejecuta `kamino deploy clon.tar.gz -y`
- **THEN** el despliegue procede sin confirmación adicional, sin abrir la revisión interactiva y aplicando los conflictos con el clon, igual que con `--yes`.

#### Scenario: Conflictos con --yes

- **GIVEN** un clon con un conflicto y una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino deploy clon.tar.gz --yes`
- **THEN** el conflicto se resuelve aplicando el clon, sin abrir la interfaz interactiva y sin pedir decisiones.

### Requirement: Resumen de despliegue con mezclas

El resumen del despliegue DEBE (`MUST`) distinguir los archivos desplegados, híbridos, omitidos y sin cambios. Los clones perfectos y las apps no instaladas NO DEBEN contar como omitidos. El código 3 DEBE reservarse para los conflictos conservados en el original, los no soportados y los `--apps` no cumplidos; un archivo híbrido NO DEBE contar como omitido.

#### Scenario: Despliegue con archivos mezclados

- **GIVEN** un despliegue que aplica dos archivos completos e hibrida otros dos
- **WHEN** termina el despliegue
- **THEN** el resumen informa de dos desplegados y dos híbridos, y el comando termina con código 0.

#### Scenario: Despliegue con archivos omitidos

- **GIVEN** un despliegue que deja al menos un archivo sin desplegar por decisión de conservar el original, por no ser compatible o por un `--apps` no cumplido
- **WHEN** termina el despliegue
- **THEN** el resumen lo cuenta como omitido y el comando termina con código 3.

#### Scenario: Despliegue con clones perfectos

- **GIVEN** un despliegue cuyos únicos archivos no escritos son clones perfectos
- **WHEN** termina el despliegue
- **THEN** se informan como clones perfectos, no como omitidos, y el comando termina con código 0.

#### Scenario: Apps no instaladas en el resumen

- **GIVEN** un clon con apps que no están instaladas en este Mac y no fueron pedidas con `--apps`
- **WHEN** termina el despliegue
- **THEN** se informan como “sin base en este Mac” y no alteran el código de salida.

### Requirement: Ayuda y versión con alias

La CLI DEBE (`MUST`) aceptar `-V` como alias de `--version` en el nivel principal y `-h` como alias de `--help` en el nivel principal y en ambos comandos. Cada ayuda DEBE mostrar los alias y sus formas largas correspondientes, incluido `-f` para `--force`.

#### Scenario: Consulta de la versión

- **WHEN** se ejecuta `kamino -V`
- **THEN** se muestra la misma versión que con `kamino --version` y se termina con código 0 sin iniciar un clon, despliegue o menú.

#### Scenario: Consulta de ayuda en cada nivel

- **WHEN** se ejecuta `kamino -h`, `kamino clone -h` o `kamino deploy -h`
- **THEN** se muestra la ayuda correspondiente, incluidos los nombres cortos y largos de sus opciones, y se termina con código 0 sin ejecutar el comando.

#### Scenario: Ayuda de la opción de fuerza

- **WHEN** se ejecuta `kamino clone -h`
- **THEN** la ayuda muestra `--force` y `-f` con la explicación del reemplazo del clon.

### Requirement: Compatibilidad de las opciones existentes

La CLI DEBE (`MUST`) mantener disponibles las opciones largas existentes y conservar sus resultados y códigos de salida. Las opciones automáticas de autocompletado conservan sus nombres y comportamiento actuales, sin recibir alias nuevos.

#### Scenario: Uso de las formas largas

- **WHEN** se utilizan `--apps`, `--output`, `--dry-run`, `--verbose`, `--yes`, `--version` o `--help` en los niveles donde ya existen
- **THEN** siguen funcionando con el mismo significado y resultado que antes del cambio.

#### Scenario: Resultado parcial con un alias

- **WHEN** un clon o un despliegue invocados con alias omiten elementos
- **THEN** el comando termina con código 3, igual que con las formas largas equivalentes.

#### Scenario: Opciones de autocompletado

- **WHEN** se consulta la ayuda principal
- **THEN** `--install-completion` y `--show-completion` siguen disponibles sin alias cortos nuevos.

### Requirement: Nombre por defecto y reemplazo del clon

En la vía no interactiva, `clone` DEBE (`MUST`) usar `clone-yyyyMMdd.tar.gz` con la fecha local en `~/.config/kamino/vault/`, creando ese directorio si no existe. Si el clon existe y no se indicó `--force`, DEBE terminar con código 1 y un error literal que indique cómo reemplazarlo. Con `--force`, DEBE reemplazarlo de forma atómica. Con `--dry-run`, NO DEBE crear ni reemplazar nada, incluido el directorio por defecto.

#### Scenario: Nombre por defecto sin TUI

- **GIVEN** un HOME sin `~/.config/kamino/vault/` ni un clon para la fecha local
- **WHEN** se ejecuta `kamino clone` sin `--output`
- **THEN** se crea `~/.config/kamino/vault/` y el archivo `~/.config/kamino/vault/clone-yyyyMMdd.tar.gz` con la fecha local

#### Scenario: Clon existente sin fuerza

- **GIVEN** un clon existente en el nombre por defecto
- **WHEN** se ejecuta `kamino clone` sin `--force`
- **THEN** el comando termina con código 1, no sobrescribe y explica que use `--force`

#### Scenario: Reemplazo con fuerza

- **GIVEN** un clon existente y un clon nuevo válido
- **WHEN** se ejecuta `kamino clone --force`
- **THEN** el clon anterior se reemplaza de forma atómica y no queda un temporal

#### Scenario: Simulación no escribe

- **GIVEN** un clon existente en el destino por defecto y `--force`
- **WHEN** se ejecuta `kamino clone --dry-run --force`
- **THEN** no se crea, reemplaza ni modifica ningún archivo

#### Scenario: Simulación no crea el directorio por defecto

- **GIVEN** un HOME sin `~/.config/kamino/vault/`
- **WHEN** se ejecuta `kamino clone --dry-run`
- **THEN** el directorio no se crea y el comando no escribe nada

### Requirement: Apertura de la TUI y omisión de pantallas

Con terminal interactiva y sin `KAMINO_PLAIN`, `clone` DEBE (`MUST`) abrir la TUI; `--apps` y `--output` DEBEN omitir C1 y C2 respectivamente, y un valor inválido DEBE abrir la pantalla correspondiente precargada con el error. `deploy` DEBE conservar la vía no interactiva con `--yes` y `--dry-run`; `--apps` DEBE omitir D1.

#### Scenario: Clonación con TUI y argumentos

- **GIVEN** una terminal interactiva y `kamino clone --apps zsh --output clon.tar.gz`
- **WHEN** se inicia el flujo
- **THEN** C1 y C2 se omiten y se muestra C3 con el plan antes de escribir

#### Scenario: Argumento inválido abre su pantalla

- **GIVEN** una terminal interactiva y un `--apps` o `--output` inválido
- **WHEN** se inicia el flujo
- **THEN** la pantalla correspondiente se abre precargada con el error

#### Scenario: Despliegue no interactivo con TUI disponible

- **GIVEN** una terminal interactiva
- **WHEN** se ejecuta `kamino deploy clon.tar.gz --yes` o `--dry-run`
- **THEN** no se abre la TUI y se usa la salida de texto

#### Scenario: --apps omite la selección de despliegue

- **GIVEN** un despliegue con `--apps` y terminal interactiva
- **WHEN** el valor es válido
- **THEN** D1 se omite y el flujo continúa con los conflictos
