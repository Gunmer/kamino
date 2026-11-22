# Spec Delta

## ADDED Requirements

### Requirement: Terminología visible de implantación

La TUI DEBE (`MUST`) usar «Implantar» como nombre visible de la acción que aplica un clon y «implantación» como sustantivo del flujo. Las pantallas, botones, mensajes, resúmenes y la orden de deshacer DEBEN (`MUST`) usar esa terminología; `deploy` NO DEBE (`MUST NOT`) aparecer como etiqueta principal de la TUI.

#### Scenario: Menú con Implantar

- **GIVEN** el menú principal abierto
- **WHEN** se muestran las opciones
- **THEN** la segunda opción se llama «Implantar» y no «Desplegar».

#### Scenario: Títulos y acciones del flujo

- **GIVEN** el flujo de implantación abierto
- **WHEN** se muestran sus pantallas
- **THEN** los títulos usan «Implantar un clon» y «Confirmar la implantación», y la acción final se llama «Implantar».

#### Scenario: Resumen y orden de deshacer

- **GIVEN** una implantación terminada con holocrón
- **WHEN** se muestra el resumen
- **THEN** se usan «Implantación completada», «Implantados» y «Sin implantar», y la orden para deshacer es `kamino implant <ruta>`.

## MODIFIED Requirements

### Requirement: Aplicación única y apertura con terminal

La interfaz interactiva DEBE (`MUST`) ser una sola aplicación con menú y flujos. DEBE abrirse solo cuando la entrada y la salida estándar son terminales y no se solicitó la vía no interactiva; sin terminal, o con `KAMINO_PLAIN`, los comandos DEBEN usar la salida de texto. Un comando directo NO DEBE volver al menú; un flujo iniciado desde el menú SÍ DEBE volver a él.

#### Scenario: Menú con terminal interactiva

- **GIVEN** una entrada y una salida estándar conectadas a una terminal
- **WHEN** se ejecuta `kamino` sin argumentos
- **THEN** se abre la aplicación interactiva en el menú principal

#### Scenario: Comando sin terminal

- **GIVEN** una entrada o una salida estándar que no es una terminal
- **WHEN** se ejecuta `kamino` sin argumentos
- **THEN** se muestra la ayuda de la CLI y no se abre la aplicación interactiva

#### Scenario: KAMINO_PLAIN fuerza la vía no interactiva

- **GIVEN** una terminal interactiva y `KAMINO_PLAIN` definido
- **WHEN** se ejecuta cualquier comando, incluido `kamino` sin argumentos
- **THEN** se usa la salida de texto y no se abre la aplicación interactiva

#### Scenario: Comando directo no vuelve al menú

- **GIVEN** una terminal interactiva
- **WHEN** se ejecuta `kamino clone`, `kamino implant` o el alias `kamino deploy`
- **THEN** el flujo termina con el código del comando y no muestra el menú al finalizar

### Requirement: Menú principal

El menú DEBE (`MUST`) mostrar el wordmark KAMINO en el cuerpo, las ocho cápsulas de clonación en el lado derecho de la cabecera con la última marcada como el original, las opciones Clonar, Implantar, Actualizar, Configuración y Abandonar Kamino, en ese orden, y la línea de última misión cuando exista. Las opciones NO DEBEN (`MUST NOT`) tener atajos de una tecla. Las flechas arriba y abajo DEBEN mover la selección y `Enter` DEBE aceptar la opción seleccionada; la opción de salida DEBE cerrar la aplicación y Escape NO DEBE cerrarla.

#### Scenario: Menú completo

- **GIVEN** una terminal de al menos 30 filas y una sesión sin flujos ejecutados
- **WHEN** se abre el menú
- **THEN** se muestran la cabecera con las ocho cápsulas, el wordmark, el subtítulo, las opciones Clonar, Implantar, Actualizar, Configuración y Abandonar, y no se muestra una última misión

#### Scenario: Cápsulas en la cabecera

- **WHEN** se abre el menú
- **THEN** las ocho cápsulas aparecen en el lado derecho de la cabecera y el wordmark en el cuerpo

#### Scenario: Navegación con flechas y Enter

- **GIVEN** el menú principal abierto
- **WHEN** se pulsan las flechas arriba o abajo y después `Enter`
- **THEN** la selección se mueve por las cinco opciones y se ejecuta la opción seleccionada

#### Scenario: Abrir Configuración

- **GIVEN** el menú principal abierto
- **WHEN** se selecciona Configuración con las flechas y se pulsa `Enter`
- **THEN** se abre la pantalla de configuración con las tres rutas y sus valores actuales

#### Scenario: Opciones sin atajos de una tecla

- **GIVEN** el menú principal abierto
- **WHEN** se pulsan `c`, `i`, `u` o `q`
- **THEN** no se ejecuta ninguna opción y la aplicación permanece en el menú

#### Scenario: Salir con la opción de salida

- **GIVEN** el menú principal abierto
- **WHEN** se selecciona Abandonar Kamino con las flechas y se pulsa `Enter`
- **THEN** la aplicación se cierra con el código del último flujo ejecutado, o 0 si no hubo ninguno

#### Scenario: Escape no sale

- **GIVEN** el menú principal abierto
- **WHEN** se pulsa Escape
- **THEN** la aplicación permanece en el menú

#### Scenario: Actualizar desde el menú

- **GIVEN** el menú principal abierto
- **WHEN** se selecciona Actualizar con las flechas y se pulsa `Enter`
- **THEN** se ejecuta el mismo flujo que `kamino update` y se vuelve al menú al terminar

### Requirement: Navegación y conservación de estado

La navegación DEBE (`MUST`) usar una pila de pantallas que conserve el estado de la pantalla anterior. Volver desde el destino de clonación DEBE conservar la selección de apps; volver desde conflictos a apps DEBE descartar las decisiones tras una confirmación breve. Al terminar un flujo iniciado desde el menú, la aplicación DEBE volver al menú y actualizar la última misión.

#### Scenario: Atrás conserva la selección de apps

- **GIVEN** el flujo de clonar con apps seleccionadas en C1 y el destino abierto en C2
- **WHEN** se vuelve de C2 a C1
- **THEN** las mismas apps siguen seleccionadas

#### Scenario: Atrás descarta decisiones de conflicto

- **GIVEN** el flujo de implantar con decisiones tomadas en D2
- **WHEN** se vuelve de D2 a D1 y se confirma la retirada
- **THEN** las decisiones se descartan antes de volver a preparar la implantación

#### Scenario: Volver al menú tras un flujo

- **GIVEN** un flujo iniciado desde el menú
- **WHEN** el flujo termina, se cancela o falla
- **THEN** se vuelve al menú y la última misión refleja el resultado

### Requirement: Numeración de pasos adaptable

Cada flujo DEBE (`MUST`) mostrar el paso actual y el total de pasos visibles. Clonar DEBE tener tres pasos (apps, destino y revisión) e Implantar cuatro (clon, apps, conflictos y confirmación); el detalle por líneas DEBE ser una subpantalla. Los totales DEBEN bajar cuando un argumento omita una pantalla, pero DEBEN incluirla si el valor es inválido y se abre precargada con el error.

#### Scenario: Tres pasos al clonar

- **GIVEN** el flujo de clonar sin argumentos que omitan pantallas
- **WHEN** se muestra C1
- **THEN** se indica el paso 1 de 3 y el flujo continúa con destino y revisión

#### Scenario: Cuatro pasos al desplegar

- **GIVEN** el flujo de implantar iniciado desde el menú
- **WHEN** se muestra D0
- **THEN** se indica el paso 1 de 4 y el flujo continúa con apps, conflictos y confirmación

#### Scenario: Totales reducidos por argumentos

- **GIVEN** una implantación con `--apps` y un archivo posicional
- **WHEN** se inicia el flujo
- **THEN** el total de pasos baja porque D0 y D1 se omiten

#### Scenario: Valor inválido conserva la pantalla

- **GIVEN** un `--output` de clonación inválido
- **WHEN** se inicia el flujo con terminal interactiva
- **THEN** C2 se abre precargada con el error y el total de pasos la incluye

### Requirement: Terminal pequeña y redimensionado

Por debajo de 80 columnas o 24 filas, la interfaz DEBE (`MUST`) mostrar una pantalla de aviso que se actualice al redimensionar sin perder el estado. Una operación en curso DEBE continuar en segundo plano y la aplicación NO DEBE cerrarse hasta que termine.

#### Scenario: Terminal demasiado pequeña

- **GIVEN** una terminal de 79 columnas o 23 filas
- **WHEN** se abre o se redimensiona la interfaz
- **THEN** se muestra la pantalla de aviso y no se pierde el estado del flujo

#### Scenario: Operación en curso y terminal pequeña

- **GIVEN** una clonación o una implantación en curso
- **WHEN** la terminal baja de 80x24
- **THEN** la operación continúa y la aplicación no se cierra hasta que termine

#### Scenario: Redimensionar de vuelta

- **GIVEN** la pantalla de aviso visible
- **WHEN** la terminal vuelve a 80x24 o más
- **THEN** se recupera la pantalla y el estado anteriores

### Requirement: Ctrl+C, salida y códigos

`Ctrl+C` DEBE (`MUST`) cancelar sin efectos antes de aplicar y DEBE ignorarse, con aviso en pantalla, mientras se escribe en destinos o se renombra un clon. Al cerrar desde el menú, el código DEBE ser el del último flujo ejecutado, o 0 si no hubo ninguno. Un comando directo DEBE salir con el código de la CLI.

#### Scenario: Ctrl+C antes de aplicar

- **GIVEN** una revisión de clonación o implantación sin escrituras iniciadas
- **WHEN** se pulsa `Ctrl+C`
- **THEN** la operación se cancela y no se escribe nada

#### Scenario: Ctrl+C durante una escritura

- **GIVEN** una escritura de destinos o un renombrado en curso
- **WHEN** se pulsa `Ctrl+C`
- **THEN** la operación continúa y la pantalla indica que no se puede cancelar en ese punto

#### Scenario: Código al salir del menú

- **GIVEN** un flujo que terminó con código 3 y el menú abierto
- **WHEN** se elige Abandonar Kamino
- **THEN** la aplicación termina con código 3
