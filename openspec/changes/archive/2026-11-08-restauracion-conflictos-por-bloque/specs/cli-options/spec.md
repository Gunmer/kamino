# Spec Delta

## ADDED Requirements

### Requirement: Selección de apps en la restauración

El comando `restore` DEBE (`MUST`) aceptar `--apps` con una lista de identificadores separados por comas. Solo las apps instaladas y presentes en el manifiesto DEBEN restaurarse; las desconocidas o no instaladas DEBEN avisarse y contarse como omitidas. La opción DEBE omitir la pantalla interactiva de selección de apps.

#### Scenario: Restauración de apps concretas

- **WHEN** se ejecuta `kamino restore respaldo.tar.gz --apps zsh,git`
- **THEN** solo se planifican y restauran `zsh` y `git`, y no se abre la pantalla de selección de apps.

#### Scenario: App solicitada no instalada

- **WHEN** se ejecuta `kamino restore respaldo.tar.gz --apps zsh,no-existe`
- **THEN** se avisa de `no-existe`, se restaura `zsh` y el comando termina con código 3 por el elemento omitido.

### Requirement: Resumen de restauración con mezclas

El resumen de la restauración DEBE (`MUST`) distinguir los archivos restaurados, mezclados y omitidos. Un archivo mezclado NO DEBE contar como omitido y el código de salida 3 DEBE reservarse para las restauraciones con archivos omitidos.

#### Scenario: Restauración con archivos mezclados

- **GIVEN** una restauración que aplica dos archivos completos y mezcla otros dos
- **WHEN** termina la restauración
- **THEN** el resumen informa de dos restaurados y dos mezclados, y el comando termina con código 0.

#### Scenario: Restauración con archivos omitidos

- **GIVEN** una restauración que deja al menos un archivo sin restaurar por decisión o por no ser compatible
- **WHEN** termina la restauración
- **THEN** el resumen lo cuenta como omitido y el comando termina con código 3.

## MODIFIED Requirements

### Requirement: Alias de las opciones de restauración

El comando `restore` DEBE (`MUST`) aceptar `-a` como alias de `--apps`, `-n` como alias de `--dry-run`, `-y` de `--yes` y `-v` de `--verbose`, con los mismos efectos que las formas largas correspondientes. El archivo de respaldo sigue siendo un argumento posicional.

#### Scenario: Confirmación y detalle mediante alias

- **WHEN** se ejecuta `kamino restore respaldo.tar.gz -y -v` con un respaldo válido
- **THEN** se utiliza el mismo archivo, la misma selección y el mismo nivel de detalle que con `--yes --verbose`.

#### Scenario: Selección de apps mediante alias

- **WHEN** se ejecuta `kamino restore respaldo.tar.gz -a zsh,git`
- **THEN** se seleccionan las mismas aplicaciones que con `--apps zsh,git` y no se abre la pantalla de selección.

### Requirement: Simulación sin escrituras

El alias `-n` DEBE (`MUST`) conservar el comportamiento de `--dry-run` en ambos comandos, sin crear ni modificar archivos de respaldo, destinos restaurados o instantáneas. En `restore`, la simulación NO DEBE abrir la interfaz interactiva; DEBE volcar los conflictos a la salida estándar y usar `--apps` o, en su defecto, todas las apps instaladas.

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
- **THEN** la simulación no abre la interfaz interactiva, usa `--apps` o todas las apps instaladas y vuelca los conflictos a la salida estándar.

#### Scenario: Simulación con selección explícita

- **GIVEN** un respaldo válido y una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino restore respaldo.tar.gz -n --apps zsh`
- **THEN** se muestran los conflictos de `zsh` en la salida estándar, sin abrir la interfaz interactiva y sin modificar destinos ni crear instantáneas.

#### Scenario: Simulación con todas las apps

- **GIVEN** un respaldo válido y una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino restore respaldo.tar.gz -n` sin `--apps`
- **THEN** se simulan todas las apps instaladas, se muestran sus conflictos en la salida estándar y no se modifica ningún destino ni se crea ninguna instantánea.

### Requirement: Confirmación de restauraciones no interactivas

Una restauración no interactiva DEBE (`MUST`) seguir exigiendo `-y` o `--yes`, salvo que se solicite una simulación. El alias no altera la validación ni las protecciones de restauración existentes. Con `--yes`, los conflictos DEBEN resolverse aplicando el respaldo, sin abrir la revisión interactiva.

#### Scenario: Restauración sin confirmación ni simulación

- **GIVEN** una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino restore respaldo.tar.gz` sin `-y`, `--yes`, `-n` ni `--dry-run`
- **THEN** el comando informa de que se requiere confirmación, termina con código 1 y no inicia la restauración.

#### Scenario: Restauración confirmada mediante el alias

- **GIVEN** una entrada sin terminal interactiva y un respaldo válido
- **WHEN** se ejecuta `kamino restore respaldo.tar.gz -y`
- **THEN** la restauración procede sin confirmación adicional, sin abrir la revisión interactiva y aplicando los conflictos con el respaldo, igual que con `--yes`.

#### Scenario: Conflictos con --yes

- **GIVEN** un respaldo con un conflicto y una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino restore respaldo.tar.gz --yes`
- **THEN** el conflicto se resuelve aplicando el respaldo, sin abrir la interfaz interactiva y sin pedir decisiones.
