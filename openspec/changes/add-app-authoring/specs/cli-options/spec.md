# Spec Delta

## ADDED Requirements

### Requirement: Descubrimiento de los comandos de aplicaciones

La ayuda principal DEBE (`MUST`) listar `apps` con sus subcomandos `list`, `validate` y `new`. El autocompletado DEBE (`MUST`) incluir `apps` para bash, zsh y fish. Cada subcomando DEBE (`MUST`) aceptar `-h` y `--help`, mostrar sus flags y terminar con código 0.

#### Scenario: Ayuda principal con apps

- **WHEN** se ejecuta `kamino -h` o `kamino --help`
- **THEN** la ayuda muestra el comando `apps` con los subcomandos `list`, `validate` y `new`.

#### Scenario: Autocompletado con apps

- **WHEN** se consulta `kamino --show-completion` para bash, zsh o fish
- **THEN** el script incluye `apps` entre los comandos disponibles.

#### Scenario: Ayuda de un subcomando

- **WHEN** se ejecuta `kamino apps new -h`
- **THEN** se muestran sus flags, incluidos `--command`, `--bundle-id`, `--path`, `--optional-path`, `--name`, `--force` y `--print`
- **AND** el comando termina con código 0 sin crear ni modificar archivos.
