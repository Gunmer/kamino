# Spec Delta

## ADDED Requirements

### Requirement: Marco único y zonas

La TUI DEBE (`MUST`) usar un único marco que posea la cabecera, el cuerpo, la zona de mensajes, el pie, los separadores y la guardia de tamaño. Ninguna pantalla DEBE (`MUST NOT`) leer el tamaño del terminal; el marco DEBE entregar a cada pantalla el ancho y el alto del body. La cabecera DEBE mostrar `kamino · <Flujo> [· <Pantalla>]` a la izquierda y los pasos a la derecha solo cuando la pantalla tenga pasos.

#### Scenario: Marco único

- **WHEN** se muestra cualquier pantalla
- **THEN** la cabecera, los separadores, los mensajes y el pie son responsabilidad del marco, no de la pantalla

#### Scenario: Sin lectura del terminal

- **WHEN** se redimensiona la terminal
- **THEN** el marco actualiza el tamaño del body y la pantalla recibe el nuevo ancho y alto sin leer el terminal

#### Scenario: Cabecera con pasos

- **WHEN** la pantalla pertenece a un flujo con pasos
- **THEN** la cabecera muestra el flujo, la pantalla y `Paso N de M` con los puntos correspondientes

#### Scenario: Cabecera sin pasos

- **WHEN** la pantalla no pertenece a un flujo con pasos
- **THEN** la cabecera no muestra `Paso N de M`

### Requirement: Ancho y alto del body

El body DEBE (`MUST`) medir `termW - 4` por debajo de 100 columnas, 96 columnas entre 100 y 139, y 120 columnas desde 140. DEBE centrarse horizontalmente y alinearse arriba. Con 24 o 25 filas NO DEBE (`MUST NOT`) haber fila de aire entre la cabecera y el body; desde 26 filas SÍ DEBE haberla. La zona de mensajes y el pie NO DEBEN desbordar el alto. Por debajo de 80 columnas o 24 filas DEBE mostrarse el aviso de terminal pequeña.

#### Scenario: Ancho compacto

- **GIVEN** una terminal de 80 columnas
- **WHEN** se muestra una pantalla
- **THEN** el body mide 76 columnas y se centra con 2 columnas de gutter a cada lado

#### Scenario: Ancho medio

- **GIVEN** una terminal de 100 columnas
- **WHEN** se muestra una pantalla
- **THEN** el body mide 96 columnas

#### Scenario: Ancho amplio

- **GIVEN** una terminal de 140 columnas
- **WHEN** se muestra una pantalla
- **THEN** el body mide 120 columnas

#### Scenario: Alto compacto

- **GIVEN** una terminal de 24 filas
- **WHEN** se muestra una pantalla
- **THEN** no hay fila de aire entre la cabecera y el body

#### Scenario: Alto medio

- **GIVEN** una terminal de 26 filas
- **WHEN** se muestra una pantalla
- **THEN** hay una fila de aire entre la cabecera y el body

### Requirement: Zona de mensajes

La TUI DEBE (`MUST`) mostrar entre el body y el separador del pie hasta tres mensajes, uno por categoría: `INFO` en verde ANSI, `AVISO` en amarillo ANSI y `ERROR` en rojo ANSI. Cada mensaje DEBE incluir un icono y la palabra de su categoría además del color. La zona NO DEBE (`MUST NOT`) reservarse cuando está vacía y el body DEBE reducirse por cada línea mostrada. Los errores que terminan el flujo DEBEN seguir usando la pantalla de error común.

#### Scenario: Mensaje informativo

- **GIVEN** una pantalla que acaba de copiar una ruta
- **WHEN** se muestra la zona de mensajes
- **THEN** aparece una línea `INFO` verde con icono y palabra

#### Scenario: Aviso

- **GIVEN** una pantalla con un aviso de secretos o de apps abiertas
- **WHEN** se muestra la zona de mensajes
- **THEN** aparece una línea `AVISO` amarilla con icono y palabra

#### Scenario: Error recuperable

- **GIVEN** una validación de campo recuperable
- **WHEN** se muestra la zona de mensajes
- **THEN** aparece una línea `ERROR` roja con icono y palabra y la pantalla no se cierra

#### Scenario: Error bloqueante

- **GIVEN** un fallo que termina el flujo
- **WHEN** se muestra el error
- **THEN** se abre la pantalla de error común con las cuatro partes y la zona de mensajes no la sustituye

#### Scenario: Zona vacía

- **GIVEN** una pantalla sin mensajes
- **WHEN** se muestra
- **THEN** la zona no ocupa filas y el body llega hasta el separador del pie

## MODIFIED Requirements

### Requirement: Navegación por teclado y pie de atajos

Todas las pantallas DEBEN (`MUST`) poder recorrerse con el teclado y mostrar en el pie una leyenda de las teclas activas de esa pantalla. NO DEBEN (`MUST NOT`) mostrar barras de botones inferiores. Como norma, `Esc` DEBE retroceder cuando la pantalla lo permita, `Enter` DEBE continuar o aceptar la opción seleccionada y las flechas DEBEN mover la selección o el foco. Al cambiar de pantalla, el fondo DEBE repintarse completo y NO DEBEN quedar visibles controles de la pantalla anterior. El pie y la ayuda contextual DEBEN generarse de la misma lista de atajos activos de la pantalla; NO DEBEN mantenerse leyendas duplicadas.

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

#### Scenario: Pie y ayuda desde los atajos activos

- **GIVEN** una pantalla con un filtro activo
- **WHEN** se muestran el pie y la ayuda
- **THEN** ambos reflejan los atajos activos de esa pantalla y no una lista escrita a mano

### Requirement: Menú principal

El menú DEBE (`MUST`) mostrar el wordmark KAMINO en el cuerpo, las ocho cápsulas de clonación en el lado derecho de la cabecera con la última marcada como el original, las opciones Clonar, Desplegar, Actualizar, Configuración y Abandonar Kamino, en ese orden, y la línea de última misión cuando exista. Las opciones NO DEBEN (`MUST NOT`) tener atajos de una tecla. Las flechas arriba y abajo DEBEN mover la selección y `Enter` DEBE aceptar la opción seleccionada; la opción de salida DEBE cerrar la aplicación y Escape NO DEBE cerrarla.

#### Scenario: Menú completo

- **GIVEN** una terminal de al menos 30 filas y una sesión sin flujos ejecutados
- **WHEN** se abre el menú
- **THEN** se muestran la cabecera con las ocho cápsulas, el wordmark, el subtítulo, las opciones Clonar, Desplegar, Actualizar, Configuración y Abandonar, y no se muestra una última misión

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
- **WHEN** se pulsan `c`, `d`, `u` o `q`
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

### Requirement: Frases, logo y degradación por altura

La frase de rotación DEBE (`MUST`) elegirse al arrancar y permanecer fija durante la sesión. La cabecera del marco DEBE estar presente con 24 filas o más y el cuerpo del menú DEBE degradarse por altura: con 30 o más filas muestra wordmark, subtítulo, frase, opciones y última misión; con 26 a 29, wordmark y opciones; con 24 o 25, wordmark y opciones sin fila de aire; por debajo de 24, aviso. Las ocho cápsulas DEBEN permanecer en la cabecera. Sin codificación UTF-8 o con `KAMINO_NO_LOGO`, NO DEBE (`MUST NOT`) mostrarse el wordmark y DEBEN usarse símbolos ASCII.

#### Scenario: Frase fija durante la sesión

- **GIVEN** una sesión iniciada con una frase de rotación
- **WHEN** se navega entre pantallas y se vuelve al menú
- **THEN** la misma frase sigue visible

#### Scenario: Degradación a 30 filas

- **GIVEN** una terminal de 80x30
- **WHEN** se abre el menú
- **THEN** se muestran wordmark, subtítulo, frase, opciones y última misión, con las cápsulas en la cabecera

#### Scenario: Degradación a 26 filas

- **GIVEN** una terminal de 80x26
- **WHEN** se abre el menú
- **THEN** se muestran wordmark y opciones, con las cápsulas en la cabecera y sin subtítulo, frase ni última misión

#### Scenario: Degradación a 24 filas

- **GIVEN** una terminal de 80x24
- **WHEN** se abre el menú
- **THEN** se muestran wordmark y opciones, con las cápsulas en la cabecera y sin subtítulo, frase, última misión ni fila de aire

#### Scenario: Sin UTF-8 o con KAMINO_NO_LOGO

- **GIVEN** una terminal sin codificación UTF-8 o `KAMINO_NO_LOGO` definido
- **WHEN** se abre el menú
- **THEN** no se muestra el wordmark, se usan símbolos ASCII y el resto del menú sigue siendo legible
