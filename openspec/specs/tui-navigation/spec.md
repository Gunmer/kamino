# tui-navigation Specification

## Purpose

Definir la experiencia transversal de la interfaz interactiva: apertura, menú, navegación, ayuda, errores, terminal pequeña, salida y degradación visual, para que todos los flujos compartan el mismo comportamiento.

## Requirements

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
- **WHEN** se ejecuta `kamino clone` o `kamino deploy`
- **THEN** el flujo termina con el código del comando y no muestra el menú al finalizar

### Requirement: Menú principal

El menú DEBE (`MUST`) mostrar el wordmark KAMINO, un separador de ocho cápsulas de clonación con la última marcada como el original, las opciones Clonar, Desplegar, Actualizar y Abandonar Kamino, y la línea de última misión cuando exista. Los atajos propuestos `c`, `d`, `u` y `q` DEBEN respetarse; las flechas arriba y abajo DEBEN mover la selección y `Enter` DEBE aceptar la opción seleccionada; `q` y la opción de salida DEBEN cerrar la aplicación y Escape NO DEBE cerrarla.

#### Scenario: Menú completo

- **GIVEN** una terminal de al menos 30 filas y una sesión sin flujos ejecutados
- **WHEN** se abre el menú
- **THEN** se muestran cabecera, subtítulo, separador, opciones y no se muestra una última misión

#### Scenario: Navegación con flechas y Enter

- **GIVEN** el menú principal abierto
- **WHEN** se pulsan las flechas arriba o abajo y después `Enter`
- **THEN** la selección se mueve por las cuatro opciones y se ejecuta la opción seleccionada

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

### Requirement: Navegación por teclado y pie de atajos

Todas las pantallas DEBEN (`MUST`) poder recorrerse con el teclado y mostrar en el pie una leyenda de las teclas activas de esa pantalla. NO DEBEN (`MUST NOT`) mostrar barras de botones inferiores. Como norma, `Esc` DEBE retroceder cuando la pantalla lo permita, `Enter` DEBE continuar o aceptar la opción seleccionada y las flechas DEBEN mover la selección o el foco. Al cambiar de pantalla, el fondo DEBE repintarse completo y NO DEBEN quedar visibles controles de la pantalla anterior.

#### Scenario: Leyenda de teclas en lugar de botones

- **GIVEN** cualquier pantalla de un flujo interactivo
- **WHEN** se muestra la pantalla
- **THEN** el pie muestra las teclas que funcionan en ese contexto y no hay una botonera inferior

#### Scenario: Esc, Enter y flechas

- **GIVEN** una pantalla con varias acciones u opciones
- **WHEN** se pulsan las flechas, `Enter` o `Esc`
- **THEN** la selección o el foco se mueve, se acepta la opción seleccionada o se retrocede según corresponda

#### Scenario: Sin controles de la pantalla anterior

- **GIVEN** una pantalla abierta sobre otra que tenía controles
- **WHEN** la pantalla nueva ocupa el terminal
- **THEN** no quedan visibles controles ni textos de la pantalla anterior

### Requirement: Navegación y conservación de estado

La navegación DEBE (`MUST`) usar una pila de pantallas que conserve el estado de la pantalla anterior. Volver desde el destino de clonación DEBE conservar la selección de apps; volver desde conflictos a apps DEBE descartar las decisiones tras una confirmación breve. Al terminar un flujo iniciado desde el menú, la aplicación DEBE volver al menú y actualizar la última misión.

#### Scenario: Atrás conserva la selección de apps

- **GIVEN** el flujo de clonar con apps seleccionadas en C1 y el destino abierto en C2
- **WHEN** se vuelve de C2 a C1
- **THEN** las mismas apps siguen seleccionadas

#### Scenario: Atrás descarta decisiones de conflicto

- **GIVEN** el flujo de desplegar con decisiones tomadas en D2
- **WHEN** se vuelve de D2 a D1 y se confirma la retirada
- **THEN** las decisiones se descartan antes de volver a preparar el despliegue

#### Scenario: Volver al menú tras un flujo

- **GIVEN** un flujo iniciado desde el menú
- **WHEN** el flujo termina, se cancela o falla
- **THEN** se vuelve al menú y la última misión refleja el resultado

### Requirement: Numeración de pasos adaptable

Cada flujo DEBE (`MUST`) mostrar el paso actual y el total de pasos visibles. Clonar DEBE tener tres pasos (apps, destino y revisión) y Desplegar cuatro (clon, apps, conflictos y confirmación); el detalle por líneas DEBE ser una subpantalla. Los totales DEBEN bajar cuando un argumento omita una pantalla, pero DEBEN incluirla si el valor es inválido y se abre precargada con el error.

#### Scenario: Tres pasos al clonar

- **GIVEN** el flujo de clonar sin argumentos que omitan pantallas
- **WHEN** se muestra C1
- **THEN** se indica el paso 1 de 3 y el flujo continúa con destino y revisión

#### Scenario: Cuatro pasos al desplegar

- **GIVEN** el flujo de desplegar iniciado desde el menú
- **WHEN** se muestra D0
- **THEN** se indica el paso 1 de 4 y el flujo continúa con apps, conflictos y confirmación

#### Scenario: Totales reducidos por argumentos

- **GIVEN** un despliegue con `--apps` y un archivo posicional
- **WHEN** se inicia el flujo
- **THEN** el total de pasos baja porque D0 y D1 se omiten

#### Scenario: Valor inválido conserva la pantalla

- **GIVEN** un `--output` de clonación inválido
- **WHEN** se inicia el flujo con terminal interactiva
- **THEN** C2 se abre precargada con el error y el total de pasos la incluye

### Requirement: Ayuda contextual

La ayuda DEBE (`MUST`) abrirse con `?` y, en campos de texto, también con `F1`. DEBE mostrar los atajos de la pantalla actual, los atajos globales, el flujo con el paso actual resaltado y, en D2 y D3, el aviso de que los diffs pueden contener secretos.

#### Scenario: Ayuda de la pantalla actual

- **GIVEN** la tabla de conflictos D2 abierta
- **WHEN** se pulsa `?`
- **THEN** la ayuda muestra los atajos de D2, los globales, el flujo con D2 resaltado y el aviso de secretos

#### Scenario: Ayuda desde un campo de texto

- **GIVEN** el campo de destino de C2 con el foco
- **WHEN** se pulsa `F1`
- **THEN** se abre la ayuda sin insertar caracteres en el campo

### Requirement: Error común

Todo error mostrado en la interfaz DEBE (`MUST`) tener cuatro partes: qué pasó, el estado de seguridad, qué hacer y acciones contextuales más Volver al menú y Salir. El detalle técnico DEBE estar plegado, abrirse con `t`, copiarse con `c` y NO DEBE incluir contenido de archivos. En un comando directo NO DEBE ofrecerse Volver al menú.

#### Scenario: Error antes de escribir

- **GIVEN** un fallo de validación antes de cualquier escritura
- **WHEN** se muestra el error
- **THEN** se indica que no se escribió nada, se ofrece la acción contextual y se permite volver o salir

#### Scenario: Detalle plegado y copiable

- **GIVEN** una pantalla de error con detalle técnico
- **WHEN** se pulsa `t` y después `c`
- **THEN** el detalle se muestra o se copia sin incluir el contenido de archivos

#### Scenario: Error en comando directo

- **GIVEN** un error en un flujo iniciado con un comando directo
- **WHEN** se muestra la pantalla de error
- **THEN** no se ofrece Volver al menú y solo se ofrecen las acciones contextuales y Salir

### Requirement: Terminal pequeña y redimensionado

Por debajo de 80 columnas o 24 filas, la interfaz DEBE (`MUST`) mostrar una pantalla de aviso que se actualice al redimensionar sin perder el estado. Una operación en curso DEBE continuar en segundo plano y `q` DEBE quedar deshabilitado hasta que termine.

#### Scenario: Terminal demasiado pequeña

- **GIVEN** una terminal de 79 columnas o 23 filas
- **WHEN** se abre o se redimensiona la interfaz
- **THEN** se muestra la pantalla de aviso y no se pierde el estado del flujo

#### Scenario: Operación en curso y terminal pequeña

- **GIVEN** una clonación o un despliegue en curso
- **WHEN** la terminal baja de 80x24
- **THEN** la operación continúa y `q` no cierra la aplicación hasta que termine

#### Scenario: Redimensionar de vuelta

- **GIVEN** la pantalla de aviso visible
- **WHEN** la terminal vuelve a 80x24 o más
- **THEN** se recupera la pantalla y el estado anteriores

### Requirement: Ctrl+C, salida y códigos

`Ctrl+C` DEBE (`MUST`) cancelar sin efectos antes de aplicar y DEBE ignorarse, con aviso en pantalla, mientras se escribe en destinos o se renombra un clon. Al cerrar desde el menú, el código DEBE ser el del último flujo ejecutado, o 0 si no hubo ninguno. Un comando directo DEBE salir con el código de la CLI.

#### Scenario: Ctrl+C antes de aplicar

- **GIVEN** una revisión de clonación o despliegue sin escrituras iniciadas
- **WHEN** se pulsa `Ctrl+C`
- **THEN** la operación se cancela y no se escribe nada

#### Scenario: Ctrl+C durante una escritura

- **GIVEN** una escritura de destinos o un renombrado en curso
- **WHEN** se pulsa `Ctrl+C`
- **THEN** la operación continúa y la pantalla indica que no se puede cancelar en ese punto

#### Scenario: Código al salir del menú

- **GIVEN** un flujo que terminó con código 3 y el menú abierto
- **WHEN** se sale con `q`
- **THEN** la aplicación termina con código 3

### Requirement: Frases, logo y degradación por altura

La frase de rotación DEBE (`MUST`) elegirse al arrancar y permanecer fija durante la sesión. La cabecera DEBE degradarse por altura: con 30 filas o más muestra cabecera, subtítulo, separador, menú y última misión; con 26 a 29, cabecera, separador y menú; con 24 o 25, cabecera y menú; por debajo de 24, aviso. Sin codificación UTF-8 o con `KAMINO_NO_LOGO`, NO DEBE mostrarse el wordmark.

#### Scenario: Frase fija durante la sesión

- **GIVEN** una sesión iniciada con una frase de rotación
- **WHEN** se navega entre pantallas y se vuelve al menú
- **THEN** la misma frase sigue visible

#### Scenario: Degradación a 24 filas

- **GIVEN** una terminal de 80x24
- **WHEN** se abre el menú
- **THEN** se muestran cabecera y menú, sin subtítulo, separador ni última misión

#### Scenario: Sin UTF-8 o con KAMINO_NO_LOGO

- **GIVEN** una terminal sin codificación UTF-8 o `KAMINO_NO_LOGO` definido
- **WHEN** se abre el menú
- **THEN** no se muestra el wordmark y el resto del menú sigue siendo legible

### Requirement: Última misión

La línea de última misión DEBE (`MUST`) aparecer solo después de un flujo ejecutado en la sesión actual y DEBE resumir su resultado. NO DEBE persistirse entre sesiones.

#### Scenario: Última misión tras clonar

- **GIVEN** una clonación completada en la sesión
- **WHEN** se vuelve al menú
- **THEN** la línea de última misión resume la clonación y su resultado

#### Scenario: Sesión nueva

- **GIVEN** una sesión recién iniciada
- **WHEN** se abre el menú
- **THEN** no se muestra la línea de última misión
