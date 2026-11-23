# Teclas y estados transversales

## 5. Matriz de teclas

| Pantalla | Teclas |
| -------- | ------ |
| Global   | `?` ayuda; `Ctrl+C` cancelar o ignorar con aviso |
| Menú     | `↑↓` mover; `Enter` aceptar |
| C1       | `↑↓` mover; `Espacio` marcar; `a`/`n` visibles; `/` filtrar; `Enter` continuar; `Esc` volver o cerrar filtro |
| C2       | `↑↓` mover; `Enter` elegir/continuar; `Tab`/`Shift+Tab` campo; `Esc` volver; `F1` ayuda. Diálogo: `↑↓`/`Enter`/`Esc` |
| C3       | Revisión: `d`/`f`, `Enter` clonar, `Esc` volver, `Ctrl+C` cancelar. Escritura: `Ctrl+C` cancelar o ignorar. Final: `d` omitidos, `c` copiar ruta, `Enter`/`Esc` continuar |
| D0       | `↑↓` mover; `/` filtrar; `o` otro archivo; `Enter` continuar; `Esc` volver |
| D1       | `↑↓` mover; `Espacio` marcar; `a`/`n` todas/ninguna; `Enter` continuar; `Esc` volver |
| D2       | `↑↓` mover; `r` revisar; `c`/`o` clon/original; `C`/`O` todo; `Enter` confirmar; `Esc` volver |
| D3       | `↑↓` línea; `←→`/`Espacio` elegir; `c`/`o` bloque; `C`/`O` archivo; `Tab`/`Shift+Tab` bloque; `Esc` volver y validar |
| D4       | `←→` mover; `Enter` aceptar; `Esc` retroceder |
| D5       | `c` copiar holocrón; `Enter`/`Esc` continuar |
| U1       | `Enter` continuar; `Esc` cancelar/salir |
| Settings | `Tab`/`Shift+Tab` mover; `←→` cursor; `Ctrl+S` guardar; `Enter` abrir «Añadir app»; `Esc` volver; `F1`/`?` ayuda |
| A1       | `Enter`/`e` editar; `Esc` descartar. Confirmación: `Enter` guardar, `Esc` volver. Final: `Enter`/`Esc` continuar |
| Help     | `Enter`/`Esc`/`?` cerrar |
| Error    | `t` detalle; `c` copiar; `Enter` aceptar; `Esc` volver |

## 6. Estados transversales

### 6.1 Mensajes

La zona de mensajes vive entre el body y el separador del footer. No se reserva
cuando está vacía: si no hay mensajes, el body llega hasta el separador.

```text
kamino · <Flujo> · <Pantalla>                     Paso N de M   * * o
--------------------------------------------------------------------------------
<body>
INFO   ...
AVISO  ...
ERROR  ...
--------------------------------------------------------------------------------
<footer>
```

| Categoría | Color | Icono | Etiqueta | Uso |
| --------- | ----- | ----- | -------- | --- |
| Info | ANSI 2 | `i` (`ℹ`) | `INFO` | confirmaciones: ruta copiada, configuración guardada, clon completado |
| Aviso | ANSI 3 | `!` (`⚠`) | `AVISO` | secretos, apps abiertas, decisiones pendientes, cancelación |
| Error | ANSI 1 | `x` (`✕`) | `ERROR` | validación recuperable, copia fallida, checksum; los errores bloqueantes usan la pantalla de error común |

- Hasta tres líneas, una por categoría, en el orden `INFO`, `AVISO`, `ERROR`.
- El icono y la etiqueta llevan el color de la categoría; el texto va en color
  por defecto para no perder contraste. La categoría nunca depende solo del color.
- El `Frame` trunca cada línea con `...` y no permite que la zona supere las tres
  líneas. Si una pantalla devuelve varias del mismo tipo, las compacta en una
  sola línea separadas por ` · `.
- `NO_COLOR` y ASCII conservan icono, etiqueta y palabra.
- Los avisos persistentes de las pantallas viven aquí: secretos en C2/C3/D2/D3,
  apps abiertas en D1/D4, decisiones pendientes en C1/D2 y errores de campo en
  C2/Settings.
- La zona de mensajes no sustituye a la pantalla de error común: un fallo que
  termina el flujo sigue mostrando las cuatro partes.

Ejemplo con las tres categorías:

```text
INFO   Ruta copiada al portapapeles.
AVISO  El clon no está cifrado y puede contener secretos.
ERROR  No se pudo copiar la ruta.
```

### 6.2 Scroll

- Listas y tablas muestran `^ N arriba   v M abajo` cuando hay contenido fuera
  de la ventana.
- El scroll mantiene el cursor visible y no monta filas fuera de la ventana.
- En D3 el scroll es por fila del archivo completo; en D4 la fila de totales
  queda fija.

### 6.3 Campos de texto

Todos los campos de C1, C2, D0 y Configuración usan el mismo componente de una
fila. La maqueta usa `>` para `▌` y `|` para la celda en vídeo inverso.

```text
  > Destino   ~/.config/kamino/vault/clone-20261113.tar.gz|
    Nombre    clone-20261113.tar.gz
```

- Marcador: `▌` y un espacio en el campo enfocado; dos espacios sin foco.
- Etiqueta: alineada a la izquierda en la misma columna para toda la pantalla;
  con foco en negrita, sin foco atenuada.
- Valor: con foco, acento sobre fondo ANSI 8 (`ansi-dark`) o ANSI 7
  (`ansi-light`); sin foco, atenuado; con error, color de error.
- Cursor: solo en el campo enfocado, en vídeo inverso sobre el carácter de
  inserción o sobre un espacio al final. No se usa el cursor real del terminal.
- Placeholder: se muestra atenuado cuando el valor está vacío; si el campo tiene
  foco, el cursor inverso va antes del placeholder.
- Scroll: si el valor no cabe, la ventana se calcula en cada render y se
  desplaza para mantener visible el cursor, con `…` en cada lado oculto. Nunca
  parte UTF-8 ni caracteres de doble ancho; se reserva el ancho real de `…`, que
  en ASCII vale `...`.
- Error: va a la zona de mensajes y marca la etiqueta y el valor del campo sin
  añadir ni mover filas. Al corregir el valor, el error desaparece.
- `NO_COLOR`: sin color ni fondo; el foco se distingue por `▌`, la etiqueta en
  negrita, el cursor inverso y «editando» junto a la etiqueta. El ancho de la
  columna reserva ese sufijo para no desalinear los valores.
- `F1` y `?` abren la ayuda sin insertar caracteres.

### 6.4 Filtros

- `/` activa el filtro en C1 y D0.
- El campo «Filtro» aparece bajo el título como una fila única con marcador `▌`,
  etiqueta alineada y valor; el cursor de edición se dibuja en vídeo inverso
  sobre el carácter de inserción y solo mientras el campo tiene foco.
- Escribir inserta en el cursor y `←`/`→` lo mueven; `Enter` aplica y devuelve el
  foco a la tabla y `Esc` limpia y cierra. El footer muestra `Enter aplicar`,
  `Esc cerrar` o `Esc volver`, `←→ cursor` y `F1/? ayuda`.
- `a`/`n` en C1 y D1 actúan solo sobre las filas visibles.
- Un filtro sin coincidencias no oculta la fila `Otro archivo...` de D0.

### 6.5 Diálogos

- Los diálogos son subestados de la pantalla, no pantallas de la pila. Así el
  header y el footer siguen siendo los de la pantalla madre.
- El diálogo se centra horizontalmente y usa `PadX`/`PadY`; no lleva borde
  exterior.
- El footer cambia a las teclas del diálogo; `Esc` vuelve a la pantalla madre.
- El diálogo de C2 es el único diálogo modal del rediseño.

### 6.6 NO_COLOR, ASCII y temas

- `NO_COLOR`: sin color ni fondo; se conservan negrita, faint y vídeo inverso. La
  selección sigue usando `▌` y los estados siguen usando icono y palabra. En un
  campo enfocado, el foco se distingue por `▌`, el cursor inverso y la palabra
  «editando» junto a la etiqueta; los demás campos siguen atenuados.
- Sin UTF-8 o con `KAMINO_NO_LOGO`: se usan los símbolos ASCII de la tabla de
  notación (`>` y `...` en los campos) y se conserva el cursor en vídeo inverso;
  `KAMINO_NO_LOGO` además oculta el wordmark.
- `KAMINO_THEME=ansi-dark` es el valor por defecto. `ansi-light` evita `Faint` y
  usa ANSI 8 para el terciario. Cualquier otro valor cae a `ansi-dark`.

