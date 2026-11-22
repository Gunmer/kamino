# Spec Delta

## ADDED Requirements

### Requirement: Comando implant y alias deploy

`kamino implant` DEBE (`MUST`) aplicar un clon en este Mac con el mismo argumento posicional, las mismas opciones, validaciones, resultados y códigos de salida que `kamino deploy`. `kamino deploy` DEBE (`MUST`) seguir disponible como alias exacto y silencioso de `implant`.

#### Scenario: Implantación de un clon

- **GIVEN** un clon válido y una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino implant clon.tar.gz --yes`
- **THEN** se aplican las configuraciones igual que con `kamino deploy clon.tar.gz --yes`, sin abrir la interfaz interactiva y con el mismo código de salida.

#### Scenario: Compatibilidad de deploy

- **GIVEN** un script existente que invoca `kamino deploy clon.tar.gz -y -v`
- **WHEN** se ejecuta el script después del cambio
- **THEN** el comportamiento, la salida y el código son los mismos que antes y no se exige migración.

#### Scenario: Alias con resultado parcial

- **GIVEN** un clon que omite una app pedida con `--apps`
- **WHEN** se ejecuta `kamino implant clon.tar.gz --apps no-existe -y`
- **THEN** se informa de la app omitida y el comando termina con código 3, igual que con `deploy`.

#### Scenario: Simulación mediante implant

- **GIVEN** un clon válido y una entrada sin terminal interactiva
- **WHEN** se ejecuta `kamino implant clon.tar.gz --dry-run`
- **THEN** se muestran los conflictos sin escribir destinos ni crear holocrón, igual que con `deploy --dry-run`.

### Requirement: Descubrimiento de implant en ayuda y autocompletado

La ayuda principal DEBE (`MUST`) presentar `implant` como comando principal y mencionar `deploy` como alias de compatibilidad. La ayuda de `implant` DEBE (`MUST`) mostrar las mismas opciones que `deploy` y terminar con código 0. El autocompletado DEBE (`MUST`) incluir `implant` y `deploy` para bash, zsh y fish.

#### Scenario: Ayuda principal con implant

- **WHEN** se ejecuta `kamino -h` o `kamino --help`
- **THEN** la ayuda muestra `implant` con su descripción, `deploy` como alias y conserva `clone`, `update` y `apps`.

#### Scenario: Ayuda de implant

- **WHEN** se ejecuta `kamino implant -h`
- **THEN** se muestran el argumento `CLONE_FILE`, las opciones `--apps`, `--dry-run`, `--yes` y `--verbose`, y el alias `deploy`, y el comando termina con código 0 sin aplicar nada.

#### Scenario: Ayuda de deploy como alias

- **WHEN** se ejecuta `kamino deploy -h`
- **THEN** la ayuda indica que `deploy` es un alias de `implant` y muestra las mismas opciones.

#### Scenario: Autocompletado con ambos nombres

- **WHEN** se consulta `kamino --show-completion` para bash, zsh o fish
- **THEN** el script incluye `implant` y `deploy` entre los comandos disponibles.

### Requirement: Textos de ayuda en español

La ayuda de `clone`, `implant` y `deploy` DEBE (`MUST`) estar en español, incluida la línea final sobre cómo cerrar la ayuda, y los errores de comando desconocido DEBEN (`MUST`) estar en español.

#### Scenario: Cierre de la ayuda

- **WHEN** se ejecuta `kamino clone -h`, `kamino implant -h` o `kamino deploy -h`
- **THEN** la última opción de ayuda dice que muestra la ayuda y termina, sin texto en inglés.

#### Scenario: Comando desconocido

- **WHEN** se ejecuta `kamino comando-inexistente`
- **THEN** el error está en español, enumera los comandos disponibles e incluye `implant` y `deploy` como alias.
