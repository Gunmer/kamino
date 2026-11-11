# Spec Delta

## MODIFIED Requirements

### Requirement: Menú principal

El menú DEBE (`MUST`) mostrar el wordmark KAMINO, un separador de ocho cápsulas de clonación con la última marcada como el original, las opciones Clonar, Desplegar, Actualizar, Configuración y Abandonar Kamino, en ese orden, y la línea de última misión cuando exista. Los atajos propuestos `c`, `d`, `u` y `q` DEBEN respetarse; Configuración NO DEBE tener atajo de teclado. Las flechas arriba y abajo DEBEN mover la selección y `Enter` DEBE aceptar la opción seleccionada; `q` y la opción de salida DEBEN cerrar la aplicación y Escape NO DEBE cerrarla.

#### Scenario: Menú completo

- **GIVEN** una terminal de al menos 30 filas y una sesión sin flujos ejecutados
- **WHEN** se abre el menú
- **THEN** se muestran cabecera, subtítulo, separador, las opciones Clonar, Desplegar, Actualizar, Configuración y Abandonar, y no se muestra una última misión

#### Scenario: Navegación con flechas y Enter

- **GIVEN** el menú principal abierto
- **WHEN** se pulsan las flechas arriba o abajo y después `Enter`
- **THEN** la selección se mueve por las cinco opciones y se ejecuta la opción seleccionada

#### Scenario: Abrir Configuración

- **GIVEN** el menú principal abierto
- **WHEN** se selecciona Configuración con las flechas y se pulsa `Enter`
- **THEN** se abre la pantalla de configuración con las tres rutas y sus valores actuales

#### Scenario: Configuración sin atajo

- **GIVEN** el menú principal abierto y el cursor fuera de Configuración
- **WHEN** se pulsa `s`
- **THEN** no se abre la pantalla de configuración

#### Scenario: Salir con el atajo

- **GIVEN** el menú principal abierto
- **WHEN** se pulsa `q`
- **THEN** la aplicación se cierra con el código del último flujo ejecutado, o 0 si no hubo ninguno

#### Scenario: Escape no sale

- **GIVEN** el menú principal abierto
- **WHEN** se pulsa Escape
- **THEN** la aplicación permanece en el menú

#### Scenario: Actualizar desde el menú

- **GIVEN** el menú principal abierto
- **WHEN** se elige Actualizar o se pulsa `u`
- **THEN** se ejecuta el mismo flujo que `kamino update` y se vuelve al menú al terminar
