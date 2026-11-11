# Spec Delta

## MODIFIED Requirements

### Requirement: Destino del clon

C2 DEBE (`MUST`) proponer `clone-yyyyMMdd.tar.gz` con la fecha local en `~/.config/kamino/vault/` o, si la TUI tiene `clone.output_dir` configurado, en ese directorio; DEBE resolver la ruta a absoluta y validar en vivo. `Enter` DEBE continuar, `Esc` DEBE volver y `F2`, `F3` y `F4` DEBEN usar el directorio actual, Descargas y Escritorio. Si el clon existe, DEBE mostrar un diálogo con ruta, fecha y tamaño, con las opciones Volver, Otro nombre y Reemplazar clon navegables con flechas y `Enter`, y con Volver preseleccionada; Otro nombre DEBE proponer el primer sufijo libre. El destino NO DEBE crearse hasta que se inicie la clonación y NO DEBE limitarse al HOME. Si el directorio por defecto o configurado no existe, C2 DEBE aceptar la ruta propuesta y la clonación DEBE crear el directorio al empezar, nunca al abrir la pantalla.

#### Scenario: Nombre por defecto

- **GIVEN** una fecha local, `~/.config/kamino/vault/` existente o ausente y la TUI sin `clone.output_dir` configurado
- **WHEN** se abre C2
- **THEN** se propone `~/.config/kamino/vault/clone-yyyyMMdd.tar.gz` con la fecha local y la ruta absoluta resuelta

#### Scenario: Directorio configurado en la TUI

- **GIVEN** `clone.output_dir = "~/Descargas"` en la configuración de la TUI
- **WHEN** se abre C2
- **THEN** se propone `~/Descargas/clone-yyyyMMdd.tar.gz` con la fecha local y la ruta absoluta resuelta

#### Scenario: Directorio por defecto ausente

- **GIVEN** `~/.config/kamino/vault/` no existe y la TUI no tiene `clone.output_dir` configurado
- **WHEN** se abre C2 y se continúa a la revisión
- **THEN** C2 no muestra un error por el directorio ausente y la clonación crea el directorio al empezar a escribir

#### Scenario: Directorio configurado ausente

- **GIVEN** `clone.output_dir` apunta a un directorio que no existe
- **WHEN** se abre C2 y se continúa a la revisión
- **THEN** C2 no muestra un error por el directorio ausente y la clonación crea el directorio configurado al empezar a escribir

#### Scenario: Validación en vivo

- **GIVEN** un nombre o una ruta escrita manualmente en C2 que no es válida
- **WHEN** se escribe en C2
- **THEN** se muestra el error sin crear el archivo ni el directorio

#### Scenario: Atajos de directorio

- **WHEN** se elige el directorio actual, Descargas o Escritorio
- **THEN** la ruta del destino se actualiza a la ubicación correspondiente

#### Scenario: Clon existente

- **GIVEN** un clon con el mismo nombre en el destino
- **WHEN** se intenta continuar
- **THEN** se muestra el diálogo con ruta, fecha, tamaño, Volver preseleccionada y las tres acciones

#### Scenario: Otro nombre propone sufijo

- **GIVEN** el diálogo de clon existente abierto
- **WHEN** se elige Otro nombre
- **THEN** se propone el primer sufijo libre para el mismo nombre base

#### Scenario: Reemplazo forzado

- **GIVEN** un clon iniciado con `--force` y un destino existente
- **WHEN** se llega a C2
- **THEN** no se muestra el diálogo y se avisa de que el clon existente se reemplazará al final

#### Scenario: --output omite el destino

- **GIVEN** un clon iniciado con `--output`
- **WHEN** el valor es válido
- **THEN** C2 se omite y se usa esa ruta

#### Scenario: --output inválido

- **GIVEN** un clon iniciado con `--output` y un valor inválido
- **WHEN** se inicia el flujo con terminal interactiva
- **THEN** C2 se abre precargada con el error
