# La TUI

Una sola app de terminal construida con Bubble Tea, Lip Gloss y `bubbles` reúne menú, clonación,
implantación y actualización. El menú muestra el wordmark KAMINO en el cuerpo, ocho cápsulas en la
cabecera con la última marcada como el original, una frase fija por sesión y las opciones Clonar,
Implantar, Actualizar, Configuración y Abandonar Kamino. Las flechas mueven la selección y `Enter`
acepta la opción seleccionada; no hay atajos de una tecla. Escape nunca cierra. La línea de última
misión aparece tras el primer flujo de la sesión y no se persiste.

Todas las pantallas se manejan con teclado y muestran al pie una leyenda de las teclas activas, sin
botoneras inferiores: `Esc` retrocede, `Enter` continúa o acepta y las flechas mueven el foco. C1 es
una tabla `SEL | APP | ARCHIVOS` con contador y casillas, y el filtro `/` es un campo de una fila;
C2 ofrece tres opciones de carpeta y los campos «Destino» y «Nombre»; D0 filtra con `/` y abre la
ruta manual «Ruta» con `o`; C3 usa una tabla única para revisión, progreso y resultado, de modo que
C4 es su estado final. D2 es una tabla de conflictos, D3 muestra el archivo completo en dos columnas
y D4 resume por app con los protegidos y el holocrón.

Todos los campos de texto comparten una sola fila con marcador `▌`, etiqueta alineada y valor. El
foco añade fondo ANSI, etiqueta en negrita y un cursor de edición en vídeo inverso; el resto de
campos queda atenuado. `←`/`→` mueven el cursor dentro del valor, `Retroceso` borra en esa posición
y las rutas largas se desplazan con `…` sin partir caracteres UTF-8 ni de doble ancho. Un valor
vacío muestra un placeholder atenuado. Con `NO_COLOR` el foco se distingue por `▌`, el cursor
inverso y la palabra «editando»; sin UTF-8 se usan `>` y `...`. Los errores de validación van a la
zona de mensajes y marcan el campo sin añadir ni mover filas.

Cada flujo muestra el paso actual y el total visible. Los argumentos reducen el total cuando omiten
pantallas. La ayuda contextual se abre con `?` y, en campos de texto, también con `F1`, sin insertar
caracteres en el valor. Todo error usa cuatro partes —qué pasó, estado de seguridad, qué hacer y
acciones— con el detalle plegado (`t`) y copiable (`c`). Por debajo de 80×24 aparece un aviso que
conserva el estado y bloquea la salida; al redimensionar de vuelta se recupera la pantalla. La zona
de mensajes `INFO`/`AVISO`/`ERROR` vive sobre el pie y no reserva filas cuando está vacía.

`Ctrl+C` cancela antes del punto de no retorno: en clonar, antes de renombrar el temporal; en
implantar, antes de crear el holocrón; en actualizar, antes de `os.replace`. Después de ese punto
se ignora con aviso. La barra de progreso avanza por archivos y `Ctrl+C` elimina el temporal.

La TUI usa la paleta ANSI del terminal: el tema por defecto es `ansi-dark` y `KAMINO_THEME`
admite `ansi-light`; cualquier otro valor cae a `ansi-dark`. Con `NO_COLOR` se muestra en
monocromo y cada estado conserva icono y palabra. El wordmark se oculta sin UTF-8 o con
`KAMINO_NO_LOGO`. `KAMINO_PLAIN` fuerza la salida de texto para scripting.

## Glosario visible

| Nombre visible | Significado |
| --- | --- |
| original | configuración actual del Mac; antes “local” |
| holocrón de seguridad | copia previa a implantar; antes “snapshot” |
| híbrido | archivo compuesto con líneas del original y del clon |
| clon perfecto | archivo idéntico al clon; no se escribe ni cuenta como omitido |
| sin base en este Mac | app del manifiesto que no está instalada aquí |
| perturbación en la Fuerza | el destino cambió durante la revisión |
| clon con mutaciones | la integridad del clon no coincide con el manifiesto |
| registro de los kaminoanos | manifiesto `manifest.json` del clon |

Los identificadores internos (`LocalFile`, `RowDecision.LOCAL`, `snapshot`, `SNAPSHOT_APP_ID`)
no cambian: el formato del archivo y los holocrones antiguos siguen siendo compatibles.
