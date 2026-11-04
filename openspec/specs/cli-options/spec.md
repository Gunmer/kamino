# cli-options Specification

## Purpose

Permitir que las personas utilicen y descubran las opciones de Kamino mediante nombres cortos o largos, sin alterar la seguridad ni la compatibilidad de sus comandos.

## Requirements

### Requirement: Alias de las opciones de respaldo

El comando `backup` DEBE (`MUST`) aceptar `-a` como alias de `--apps`, `-o` de `--output`, `-n` de `--dry-run` y `-v` de `--verbose`, con los mismos valores y efectos que las formas largas correspondientes.

#### Scenario: Selección y ruta mediante alias

- **WHEN** se ejecuta `kamino backup -a vscode,git -o respaldo.tar.gz`
- **THEN** se seleccionan las mismas aplicaciones y ruta que con `--apps vscode,git --output respaldo.tar.gz`.

#### Scenario: Salida detallada del respaldo

- **WHEN** se ejecuta un respaldo con `-v` en lugar de `--verbose`
- **THEN** se presenta el mismo nivel de detalle y se conserva el resultado del comando.

### Requirement: Alias de las opciones de restauración

El comando `restore` DEBE (`MUST`) aceptar `-n` como alias de `--dry-run`, `-y` de `--yes` y `-v` de `--verbose`, con los mismos efectos que las formas largas correspondientes. El archivo de respaldo sigue siendo un argumento posicional.

#### Scenario: Confirmación y detalle mediante alias

- **WHEN** se ejecuta `kamino restore respaldo.tar.gz -y -v` con un respaldo válido
- **THEN** se utiliza el mismo archivo, la misma selección y el mismo nivel de detalle que con `--yes --verbose`.

### Requirement: Simulación sin escrituras

El alias `-n` DEBE (`MUST`) conservar el comportamiento de `--dry-run` en ambos comandos, sin crear ni modificar archivos de respaldo, destinos restaurados o instantáneas.

#### Scenario: Simulación de un respaldo

- **WHEN** se ejecuta `kamino backup -o respaldo.tar.gz -n`
- **THEN** se muestra el resultado de la simulación sin crear el archivo de salida, igual que con `--dry-run`.

#### Scenario: Simulación de una restauración no interactiva

- **GIVEN** un respaldo válido y una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino restore respaldo.tar.gz -n` sin `-y`
- **THEN** se permite la simulación sin modificar destinos ni crear instantáneas, igual que con `--dry-run`.

#### Scenario: Selección interactiva durante la simulación

- **GIVEN** una entrada con terminal interactiva
- **WHEN** se ejecuta `kamino restore respaldo.tar.gz -n` sin `-y`
- **THEN** se conserva la selección interactiva existente, igual que con `--dry-run`.

### Requirement: Confirmación de restauraciones no interactivas

Una restauración no interactiva DEBE (`MUST`) seguir exigiendo `-y` o `--yes`, salvo que se solicite una simulación. El alias no altera la validación ni las protecciones de restauración existentes.

#### Scenario: Restauración sin confirmación ni simulación

- **GIVEN** una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino restore respaldo.tar.gz` sin `-y`, `--yes`, `-n` ni `--dry-run`
- **THEN** el comando informa de que se requiere confirmación, termina con código 1 y no inicia la restauración.

#### Scenario: Restauración confirmada mediante el alias

- **GIVEN** una entrada sin terminal interactiva y un respaldo válido
- **WHEN** se ejecuta `kamino restore respaldo.tar.gz -y`
- **THEN** la restauración procede sin confirmación adicional, con los mismos controles que al usar `--yes`.

### Requirement: Ayuda y versión con alias

La CLI DEBE (`MUST`) aceptar `-V` como alias de `--version` en el nivel principal y `-h` como alias de `--help` en el nivel principal y en ambos comandos. Cada ayuda DEBE mostrar los alias y sus formas largas correspondientes.

#### Scenario: Consulta de la versión

- **WHEN** se ejecuta `kamino -V`
- **THEN** se muestra la misma versión que con `kamino --version` y se termina con código 0 sin iniciar un respaldo, restauración o menú.

#### Scenario: Consulta de ayuda en cada nivel

- **WHEN** se ejecuta `kamino -h`, `kamino backup -h` o `kamino restore -h`
- **THEN** se muestra la ayuda correspondiente, incluidos los nombres cortos y largos de sus opciones, y se termina con código 0 sin ejecutar el comando.

### Requirement: Compatibilidad de las opciones existentes

La CLI DEBE (`MUST`) mantener disponibles las opciones largas existentes y conservar sus resultados y códigos de salida. Las opciones automáticas de autocompletado conservan sus nombres y comportamiento actuales, sin recibir alias nuevos.

#### Scenario: Uso de las formas largas

- **WHEN** se utilizan `--apps`, `--output`, `--dry-run`, `--verbose`, `--yes`, `--version` o `--help` en los niveles donde ya existen
- **THEN** siguen funcionando con el mismo significado y resultado que antes del cambio.

#### Scenario: Resultado parcial con un alias

- **WHEN** un respaldo o una restauración invocados con alias omiten elementos
- **THEN** el comando termina con código 3, igual que con las formas largas equivalentes.

#### Scenario: Opciones de autocompletado

- **WHEN** se consulta la ayuda principal
- **THEN** `--install-completion` y `--show-completion` siguen disponibles sin alias cortos nuevos.
