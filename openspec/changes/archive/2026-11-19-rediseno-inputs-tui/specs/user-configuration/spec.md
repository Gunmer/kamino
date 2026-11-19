# Spec Delta

## ADDED Requirements

### Requirement: Campos de la pantalla de configuración

La pantalla de Configuración DEBE (`MUST`) mostrar `catalog.dir`, `clone.output_dir`, `deploy.snapshots_dir` y `deploy.snapshots_limit` con el campo compartido de una sola fila, etiquetas alineadas y foco visible. El foco DEBE empezar en Catálogo y `Tab`/`Mayús+Tab` y las flechas verticales DEBEN moverlo. El límite DEBE admitir solo dígitos. `Ctrl+S` DEBE guardar y `Esc` DEBE volver sin escribir. La pantalla DEBE caber en 80x24.

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
