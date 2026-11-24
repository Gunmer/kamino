# Spec Delta

## ADDED Requirements

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
