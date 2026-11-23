# Diseño de pantallas de la TUI

> **Estado:** referencia visual aprobada.
> **Cambio OpenSpec:** `tui-redesign`.
> **Alcance:** Path A. La TUI pasa a Bubble Tea + Lip Gloss + `bubbles` (`key`, `help`).
> **Referencia:** este documento es la especificación visual. Las specs describen el
> comportamiento; este documento describe cómo se ve y cómo se compone cada pantalla.

Las maquetas usan ASCII para que se puedan revisar en cualquier editor. La columna
«Real» de la notación indica el glifo que renderiza la TUI. Los golden files usarán
los glifos reales.

## 1. Arquitectura de presentación

### 1.1 Capas

```text
+--------------------------------------------------------------+
| App (raíz de Bubble Tea)                                     |
|   stack []Screen        flujo compartido (clone/implant/...)  |
|                                                              |
|   +------------------------------------------------------+   |
|   | Frame                                                |   |
|   |   header · separador · body · mensajes ·             |   |
|   |   separador · footer                                |   |
|   |   tamaño del terminal y guardia de 80x24             |   |
|   +------------------------------------------------------+   |
|                                                              |
|   theme.Styles     layout helpers     keys/help              |
+--------------------------------------------------------------+
```

- `internal/tui/app.go`: raíz, pila, contexto de flujo y comandos.
- `internal/tui/frame.go`: header, footer, separadores, mensajes, tamaño y guardia.
- `internal/tui/screen.go`: interfaz `Screen`, `ScreenID` y mensajes de navegación.
- `internal/tui/theme/`: tokens, estilos ANSI, símbolos y renderer.
- `internal/tui/layout.go`: `Place`, truncado, tablas, columnas y `Fit`.
- `internal/tui/screens_*.go`: una pantalla por archivo, dentro del paquete `tui`.
- `internal/tui/testdata/golden/`: golden files.
- `internal/texts`: todo el texto visible, incluidos los `key.Binding`.

No se crea un paquete `screens` separado para evitar ciclos con `App`.

### 1.2 Contrato de pantalla

```go
type ScreenID string

type Screen interface {
	Init() tea.Cmd
	Update(tea.Msg) (Screen, tea.Cmd)
	View(w, h int) string
	Title() string
	Step() (cur, total int)
	Keys() []key.Binding
	Messages() []Message
	SetSize(w, h int)
}

type MessageKind int

const (
	MessageInfo MessageKind = iota
	MessageWarning
	MessageError
)

type Message struct {
	Kind MessageKind
	Text string
}
```

Decisiones:

- El enum actual `type Screen string` pasa a `ScreenID`. `Screen` queda para la
  interfaz.
- `SetSize` se añade a la interfaz propuesta porque `Update` necesita saber cuántas
  filas caben para hacer scroll. `View(w,h)` sigue siendo puro.
- `Messages` devuelve hasta un mensaje por categoría. El `Frame` los pinta como
  `INFO` (ANSI 2), `AVISO` (ANSI 3) y `ERROR` (ANSI 1), con icono y palabra. Los
  mensajes complementarios viven aquí; el título y el body mantienen el contenido
  principal. Los errores bloqueantes siguen abriendo la pantalla de error común.
- `View` no lee entorno, no usa `time.Now()` y no hace E/S.
- Cada pantalla posee su estado local (cursor, filtro, scroll, foco). El estado de
  dominio compartido vive en `App` y se inyecta en las pantallas.

### 1.3 Pila y navegación

```text
App.stack = [Menu, C1, C2]
                 ^        ^
                 |        +-- pantalla actual: C2
                 +----------- C1 conserva selección y cursor
```

- `App` es el único dueño de la pila.
- Las pantallas emiten `pushMsg`, `popMsg` o `replaceMsg` como `tea.Cmd`.
- `Esc` no es global: cada pantalla decide si vuelve, cierra un filtro o cancela.
- `App` intercepta `tea.WindowSizeMsg`, calcula el tamaño del body y llama a
  `SetSize` de la pantalla actual.
- Al terminar un flujo:
  - modo menú: se vuelve al menú y se actualiza la última misión;
  - modo directo: se sale con el código del flujo.

### 1.4 Tamaño y presupuesto de filas

```text
Ancho del body:
  termW < 100   -> termW - 2*Gutter        (Gutter = 2)
  100..139      -> 96
  140+          -> 120
```

Desde 100 columnas el body deja de crecer con la terminal. El bloque que
devuelve la pantalla se centra dentro del ancho del body; las líneas que ya
ocupan el ancho completo conservan su posición. Por debajo de 100 columnas el
contenido mantiene la alineación izquierda aprobada.

```text
Alto:
  termH < 24 o termW < 80 -> aviso de terminal pequeña
  24..25                  -> header 1 + sep 1 + body + mensajes + sep 1 + footer 1
  26..29                  -> lo anterior + 1 fila de aire
  30+                     -> igual que 26..29; el body extra lo usa el contenido
```

| Filas | Header | Aire | Body        | Mensajes | Footer | Separadores |
| ----- | ------ | ---- | ----------- | -------- | ------ | ----------- |
| 24–25 | 1      | 0    | `h - 4 - M` | 0–3      | 1      | 2           |
| 26–29 | 1      | 1    | `h - 5 - M` | 0–3      | 1      | 2           |
| 30+   | 1      | 1    | `h - 5 - M` | 0–3      | 1      | 2           |

`M` es el número de categorías con mensaje (0–3): `INFO`, `AVISO` y `ERROR`. La
zona no se reserva cuando está vacía. «30+ layout completo» significa que el body
tiene altura suficiente para las secciones opcionales de cada pantalla, no que el
`Frame` añada una zona distinta.

Dos columnas:

```text
bodyW = 96   -> 46 | 4 | 46
bodyW = 76   -> 36 | 4 | 36
bodyW = 120  -> 58 | 4 | 58
```

`ColGap` efectivo = **4**. Las reglas dicen `ColGap=3`, pero 46+3+46 = 95, no 96.
Se corrige a 4 para que el ejemplo aprobado `46 | 4 | 46` sume el ancho del body.
Si el ancho es impar, la columna sobrante va a la derecha.

Tokens: `Gutter=2`, `PadX=2`, `PadY=1`, `Gap=1`, `ColGap=4`. Ningún número suelto
en una pantalla.

### 1.5 Tema y símbolos

- Solo colores ANSI: `lipgloss.ANSIColor(n)`.
- Prohibidos hex, RGB y `AdaptiveColor`.
- Jerarquía:
  - primario: texto por defecto + negrita;
  - secundario: texto por defecto;
  - terciario: `Faint` o ANSI 8;
  - acento: ANSI 6;
  - éxito e info: ANSI 2;
  - aviso: ANSI 3;
  - error: ANSI 1.
- Todos los estilos viven en `theme.Styles`, incluidos `Info`, `Warning` y
  `Error` para la zona de mensajes. Ninguna pantalla crea
  `lipgloss.NewStyle()`.
- `NO_COLOR` desactiva los colores; conserva negrita y faint porque son atributos,
  no color. La información siempre lleva glifo o palabra.
- `KAMINO_THEME` selecciona el tema: `ansi-dark` por defecto,
  `ansi-light` aceptado. `ansi-light` evita `Faint` y usa ANSI 8 para el terciario.
- Sin UTF-8, o con `KAMINO_NO_LOGO`, se usan los símbolos ASCII. `KAMINO_NO_LOGO`
  además oculta el wordmark.
- La detección de UTF-8 se hace al arrancar (`LC_ALL`, `LC_CTYPE`, `LANG`) y se
  inyecta en el tema. `View` no lee entorno.

### 1.6 Teclas, footer y ayuda

- Cada pantalla define sus `key.Binding` con textos de `internal/texts`.
- El footer es `help.ShortHelpView(screen.Keys())`, en una sola fila.
- Si el footer no cabe, se recorta con `...`; `?` se conserva siempre.
- `?` abre la pantalla de ayuda. En campos de texto, `F1` también la abre sin
  insertar caracteres.
- La ayuda muestra los atajos de la pantalla, los globales, el flujo con el paso
  actual resaltado y el aviso de secretos en D2 y D3.
- No se escribe ninguna leyenda a mano en una pantalla.

### 1.7 Operaciones asíncronas

Path A incluye mover a `tea.Cmd` las operaciones que hoy bloquean la UI:

- recuento de archivos de C1;
- planificación y escritura del clon;
- preparación y planificación de la implantación;
- aplicación de la implantación;
- consulta y ejecución de la actualización.

Las funciones de dominio no cambian. Solo cambia quién las ejecuta y cómo se
reciben el progreso y la cancelación. Esto cumple lo que las specs ya prometen:
recuento en segundo plano, preparación cancelable, progreso y Ctrl+C.

### 1.8 Golden files

- Tamaños: `80x24`, `100x26`, `140x40`, `200x50`.
- Variantes: color + Unicode en los tamaños de cada caso; `NO_COLOR` y ASCII en
  `100x26` y, cuando el caso lo declara, en `80x24`. C1 con filtro, C2 con foco
  en ruta y nombre, D0 con filtro y ruta manual, y Configuración vacía tienen
  casos propios. Un golden de la zona de mensajes reúne `INFO`, `AVISO` y
  `ERROR`.
- La zona de mensajes se valida con las tres categorías a la vez; el test de
  contrato comprueba que está sobre el separador del footer y que no depende solo
  del color.
- Ruta: `internal/tui/testdata/golden/<pantalla>/<w>x<h>.golden`,
  `.../no-color/100x26.golden`, `.../ascii/100x26.golden`.
- Fixtures deterministas: `Home=/Users/test`, fecha fija, catálogo fijo, planes y
  conflictos fijos. `View` no toca el disco.
- Test de contrato: todas las pantallas, tamaños y modos; exactamente `h` líneas,
  ninguna línea supera `w`, sin `\r`, sin tabs, sin truecolor y solo SGR ANSI
  permitidos. En `NO_COLOR` rechaza colores y fondos ANSI.
- D3: benchmark con 20 000 líneas y comprobación de que solo se renderiza la
  ventana visible.
- Tests de modelo: teclas, foco, selección, pila, mensajes asíncronos, cancelación
  y códigos de salida. HOME temporal; sin red y sin `~/.config/kamino` real.

## 2. Notación de las maquetas

| ASCII       | Real       | Uso                                      |
| ----------- | ---------- | ---------------------------------------- |
| `>`         | `▌`        | cursor o foco                            |
| `*`         | `●` / `◉`  | elegido, cápsula original                |
| `o`         | `○`        | no elegido, cápsula vacía                |
| `[x]` `[ ]` | `[x]` `[ ]`| casilla marcada / vacía                  |
| `[-]`       | `[–]`      | casilla deshabilitada                    |
| `-`         | `─`        | separador                                |
| `...`       | `…`        | truncado                                 |
| `.`         | `·`        | cero                                     |
| `^` `v`     | `▲` `▼`    | scroll arriba / abajo                    |
| `<` `>`   | `◀` `▶`    | elegir versión original / clon           |
| `|`         | vídeo inverso | cursor de edición sobre la celda     |
| `i`         | `ℹ`        | mensaje informativo                      |
| `!`         | `⚠`        | aviso                                    |
| `x`         | `✕`        | error                                    |
| `+` `-`     | `▸` `▾`    | sección colapsada / expandida            |
| `~`         | `~`        | abreviatura del home                     |

Las maquetas muestran el body de 96 columnas. En una terminal de 100 columnas el
bloque se centra con 2 columnas de gutter a cada lado. En 80 columnas el body baja
a 76; en 140+ sube a 120.

## 3. Inventario de pantallas

| ID       | Header                         | Paso    | Mensajes | Spec             |
| -------- | ------------------------------ | ------- | -------- | ---------------- |
| Menu     | `kamino · Menú`                | no      | no     | tui-navigation   |
| C1       | `kamino · Clonar · Apps`       | 1 de 3  | sí     | clone-tui        |
| C2       | `kamino · Clonar · Destino`    | 2 de 3  | sí     | clone-tui        |
| C3       | `kamino · Clonar · Revisión` / `Resultado` | 3 de 3 | sí | clone-tui |
| D0       | `kamino · Implantar · Clon`    | 1 de 4  | sí     | implant-conflicts |
| D1       | `kamino · Implantar · Apps`    | 2 de 4  | sí     | implant-conflicts |
| D2       | `kamino · Implantar · Conflictos` | 3 de 4 | sí    | implant-conflicts |
| D3       | `kamino · Implantar · Conflictos` | 3 de 4 | sí    | implant-conflicts |
| D4       | `kamino · Implantar · Confirmación` | 4 de 4 | sí  | implant-conflicts |
| D5       | `kamino · Implantar · Resumen` | no      | sí     | implant-conflicts |
| U1       | `kamino · Actualización`       | no      | sí     | self-update      |
| Settings | `kamino · Configuración`       | no      | sí     | user-configuration |
| A1       | `kamino · Configuración · Añadir app` | no | sí    | app-authoring    |
| Help     | `kamino · <Flujo> · Ayuda`     | sí      | no     | tui-navigation   |
| Error    | `kamino · Error`               | no      | no     | tui-navigation   |
| Small    | `kamino · Aviso`               | no      | sí     | tui-navigation   |

## 4. Pantallas

### 4.1 Menú

Tamaño de la maqueta: 100×30 (layout completo). A 26–29 se ocultan subtítulo,
frase y última misión; a 24–25 se oculta además el aire.

```text
kamino · Menú                                    o o o o o o o *
--------------------------------------------------------------------------------

   [wordmark KAMINO]
   v0.10.0

   Clona e implanta configuraciones de macOS en un solo archivo.
   La Fuerza estará contigo.

   > Clonar             Crea un clon de las apps instaladas.
     Implantar          Implanta un clon desde un archivo.
     Actualizar         Actualiza a la última versión publicada.
     Configuración      Edita las rutas y añade apps.
     Abandonar Kamino   Cierra kamino.

   Última misión: clonación completada · 20 archivos
   aviso: el clon no está cifrado y puede contener secretos.

--------------------------------------------------------------------------------
^v mover   Enter aceptar   ? ayuda
```

Notas:

- Las ocho cápsulas viven en el lado derecho del header; la última es la original
  (`*`). El wordmark sigue en el body.
- La versión de la compilación se muestra en dim justo debajo del wordmark, alineada
  con su borde izquierdo. Acompaña al cuerpo del menú: aparece desde 24 filas y sigue
  visible cuando `KAMINO_NO_LOGO` oculta el wordmark; por debajo de 24 filas el aviso de
  terminal pequeña reemplaza el cuerpo.
- No hay atajos de una tecla para las opciones. Escape no cierra.
- La frase se elige al arrancar y no cambia durante la sesión.
- `KAMINO_NO_LOGO` oculta el wordmark y fuerza los símbolos ASCII.

### 4.2 C1 · Apps del clon

```text
kamino · Clonar · Apps                                       Paso 1 de 3   * o o
--------------------------------------------------------------------------------

  Elige las apps que quieres clonar                    2 de 9 marcadas

  SEL  APP                     ARCHIVOS
  --------------------------------------------------------------------------
  > [x] herdr                       12
    [x] ghostty                      8
    [ ] git                          4
    [ ] bat                          3
    [-] glow                         0
    [ ] codex                        6

  ^ 0 arriba   v 3 abajo
  No detectadas: claude-code, yazi

--------------------------------------------------------------------------------
^v mover  Espacio marcar  a/n todas/ninguna  / filtrar  Enter continuar  Esc volver  ? ayuda
```

Notas:

- Solo aparecen apps instaladas. Las no detectadas van en una línea aparte y no
  son seleccionables.
- `glow` está instalada pero no tiene archivos: casilla `[-]` deshabilitada.
- El contador es de selección total, no de filas visibles.
- `Espacio` marca sin mover el cursor ni sacar el foco.
- `a`/`n` actúan solo sobre las filas visibles del filtro.
- Enter sin selección muestra un `AVISO` en la zona de mensajes y no avanza.
- Variantes: filtro activo, lista vacía (`No hay apps instaladas con archivos`) y
  lista desplazada con indicador. El filtro es una fila de campo, no texto suelto.

Variante: filtro con foco.

```text
  > Filtro    gh|
```

- El campo «Filtro» solo aparece mientras hay valor o foco. `Enter` aplica y
  devuelve el foco a la tabla; `Esc` lo limpia y lo cierra.

### 4.3 C2 · Destino del clon

```text
kamino · Clonar · Destino                                    Paso 2 de 3   * * o
--------------------------------------------------------------------------------

  Elige el destino del clon

  Carpeta de destino
  > * Vault por defecto        ~/.config/kamino/vault/
    o Carpeta actual           /Users/test/Projects/kamino
    o Indicar carpeta

    Nombre    clone-20261113.tar.gz

  Ruta: ~/.config/kamino/vault/clone-20261113.tar.gz

AVISO  El clon no está cifrado y puede contener secretos.
--------------------------------------------------------------------------------
^v mover  Enter elegir  Esc volver  ? ayuda
```

Variante: «Indicar carpeta» seleccionada. La opción abre el campo «Destino».

```text
  Carpeta de destino
    o Vault por defecto        ~/.config/kamino/vault/
    o Carpeta actual           /Users/test/Projects/kamino
    * Indicar carpeta

  > Destino   /Users/test/Descargas|
    Nombre    clone-20261113.tar.gz

  Ruta: /Users/test/Descargas/clone-20261113.tar.gz

--------------------------------------------------------------------------------
Enter fijar  Esc volver  ←→ cursor  Tab campo  F1/? ayuda
```

Variante: el clon ya existe.

```text
  El clon ya existe

  Ruta: ~/.config/kamino/vault/clone-20261113.tar.gz
  Fecha: 2026-11-12 10:30   Tamaño: 1.2 MB

    Sobrescribir
    Cambiar nombre
  > Volver
```

Notas:

- La primera opción muestra la ruta efectiva: `clone.output_dir` si está
  configurado, o `~/.config/kamino/vault/`.
- La segunda opción usa el directorio de trabajo actual (`pwd`), capturado al
  arrancar e inyectado; `View` no llama a `os.Getwd`.
- «Indicar carpeta» abre el campo «Destino». La carpeta puede no existir; se crea
  recién al empezar a escribir, nunca al abrir C2.
- El campo «Nombre» se rellena con `clone-20261113.tar.gz` y es editable. La
  sugerencia conserva el prefijo `clone-`, la fecha local y la extensión
  `.tar.gz`.
- Los dos campos comparten una fila: marcador `▌`, etiqueta alineada y valor. El
  foco añade fondo ANSI, etiqueta en negrita y el cursor en vídeo inverso; sin
  foco, el valor queda atenuado. Una ruta vacía muestra un placeholder atenuado.
- `Enter` en la ruta pasa al nombre; `Enter` en el nombre fija el destino. Si el
  archivo existe, abre el diálogo; si no, continúa a C3.
- El diálogo muestra ruta, fecha y tamaño. `Volver` está preseleccionada;
  `Sobrescribir` confirma el reemplazo y continúa a C3; `Cambiar nombre` enfoca
  el campo «Nombre» y propone el primer sufijo libre.
- `--force` omite el diálogo y muestra el aviso de reemplazo como `AVISO` en la
  zona de mensajes. `--output` omite C2; si el valor es inválido, C2 se abre con
  «Indicar carpeta» y los campos precargados, y un `ERROR` en la zona de
  mensajes.
- La validación en vivo muestra un `ERROR` en la zona de mensajes y marca el
  campo; no abre la pantalla de error común ni añade filas.
- Las tres opciones sustituyen a `F2`, `F3` y `F4`. Esos atajos se retiran de C2
  y de su footer; el cambio se documenta en `clone-tui` y en la leyenda.
- El footer cambia de contexto: en las opciones, `^v mover`/`Enter elegir`; en un
  campo, `Enter fijar` o `Enter continuar`, `Esc volver`, `←→ cursor`,
  `Tab campo` cuando hay dos y `F1/? ayuda`; en el diálogo,
  `^v mover`/`Enter aceptar`.
- Las rutas largas se desplazan dentro del valor con `…` al inicio y al final
  cuando corresponde, sin partir UTF-8 ni caracteres de doble ancho. El home se
  abrevia con `~` en el preview de la ruta completa.

### 4.4 C3 · Revisión, progreso y resultado

C3 absorbe la antigua C4. La pantalla es la misma tabla en todos los estados;
solo cambia la fila `Estado` y las acciones disponibles. El flujo de clonación
sigue teniendo tres pasos: apps, destino y revisión. El estado final no añade un
cuarto paso.

Estado de revisión:

```text
kamino · Clonar · Revisión                                   Paso 3 de 3   * * *
--------------------------------------------------------------------------------

  Clonación
  Revisa el plan antes de continuar.

  PLAN
  --------------------------------------------------------------------------
  Apps                                                                   2
  Archivos                                                              20
  Tamaño                                                             1.2 MB
  Ruta                              ~/.config/kamino/vault/clone-20261113.tar.gz
  Estado   Listo para clonar

  + Rutas omitidas (3)      + Archivos incluidos (20)

AVISO  El clon no está cifrado y puede contener secretos.
--------------------------------------------------------------------------------
d omitidos  f archivos  Enter clonar  Esc volver  Ctrl+C cancelar  ? ayuda
```

Estado de escritura:

```text
kamino · Clonar · Revisión                                   Paso 3 de 3   * * *
--------------------------------------------------------------------------------

  Clonación
  No cierres la terminal; la operación continúa.

  PLAN
  --------------------------------------------------------------------------
  Apps                                                                   2
  Archivos                                                              20
  Tamaño                                                             1.2 MB
  Ruta                              ~/.config/kamino/vault/clone-20261113.tar.gz
  Estado   Escribiendo · 12 de 20 archivos   [=========>          ]  60%

AVISO  Ctrl+C cancela antes del renombrado; el clon anterior queda intacto.
--------------------------------------------------------------------------------
Ctrl+C cancelar   ? ayuda
```

Estado final:

```text
kamino · Clonar · Resultado                                  Paso 3 de 3   * * *
--------------------------------------------------------------------------------

  Clonación
  El clon está listo.

  PLAN
  --------------------------------------------------------------------------
  Apps                                                                   2
  Archivos                                                              20
  Tamaño                                                             1.2 MB
  Ruta                              ~/.config/kamino/vault/clone-20261113.tar.gz
  Estado   Completado

  + Rutas omitidas (3)      c Copiar ruta

AVISO  Completado con 3 omitidos · código 3; pulsa d para verlos.
--------------------------------------------------------------------------------
d omitidos  c copiar ruta  Enter/Esc continuar  ? ayuda
```

Notas:

- La fila `Estado` es el único lugar donde vive la fase: `Planificando`,
  `Listo para clonar`, `Escribiendo` con barra de progreso, `Completado`,
  `Simulación completada` o `Cancelado antes de escribir`.
- El código 3 y los omitidos se muestran como `AVISO` en la zona de mensajes; la
  fila `Estado` no los duplica.
- El header cambia la última sección de `Revisión` a `Resultado` al terminar; el
  paso sigue siendo `Paso 3 de 3`.
- Acciones por estado:
  - revisión: `d`, `f`, `Enter` clonar, `Esc` volver, `Ctrl+C` cancelar;
  - escritura: solo `Ctrl+C` antes del renombrado; después se ignora con aviso;
  - final: `d` omitidos si hay, `c` copiar ruta si se creó el clon, `Enter`/`Esc`
    continuar.
- `f` solo está disponible en revisión. `d` está en revisión y en el estado
  final. `c` no aparece en simulación porque no se creó ningún archivo.
- La fila `Estado` se alinea a la izquierda y puede ocupar todo el ancho del
  valor; las filas numéricas siguen a la derecha.
- La barra de progreso se calcula con el ancho disponible y se actualiza por
  mensajes; a 80 columnas se acorta y el texto se trunca con `...`. `View` no usa
  `time.Now()`.
- `--dry-run` termina en `Resultado` con `Estado Simulación completada · no se
  escribió nada`, sin acción de copiar.
- Un error de planificación o escritura abre la pantalla de error común; la tabla
  queda en la pila para volver o reintentar.

### 4.5 C4 · absorbido por C3

La antigua C4 ya no es una pantalla separada. Su resumen es el estado final de
C3: la misma tabla, la misma ruta y una fila `Estado` que pasa de «Listo para
clonar» a la barra de progreso y al resultado. El flujo sigue teniendo tres
pasos; C3 es el tercero y su estado final no añade un paso. El enum `ScreenID`
no incluye `C4`; el estado final es `C3` con `phase = done`.

### 4.6 D0 · Elegir el clon

```text
kamino · Implantar · Clon                                    Paso 1 de 4   * o o o
--------------------------------------------------------------------------------

  Elige el clon que quieres implantar

  ARCHIVO                          ORIGEN            TAMAÑO
  --------------------------------------------------------------------------
  > clone-20261113.tar.gz          por defecto        1.2 MB
    clone-20261112.tar.gz          este directorio    980 KB
    clone-20261001.tar.gz          por defecto        1.1 MB

  Ruta: ~/.config/kamino/vault/clone-20261113.tar.gz
  Fecha: 2026-11-13 10:30   Tamaño: 1.2 MB

  o Otro archivo...

  ! No se pudo leer ~/Downloads; se muestran los clones del vault.

--------------------------------------------------------------------------------
^v mover  / filtrar  o otro archivo  Enter continuar  Esc volver  ? ayuda
```

Variante: filtro con foco.

```text
  > Filtro    clone|
```

Variante: ruta manual.

```text
  > Ruta    /Users/test/Descargas/clone-20261113.tar.gz|

--------------------------------------------------------------------------------
Enter abrir  Esc volver  ←→ cursor  F1/? ayuda
```

Notas:

- La tabla combina el vault y el directorio actual, ordena por fecha y deduplica
  por ruta resuelta.
- El preview de la fila enfocada muestra ruta, fecha y tamaño.
- `/` abre el campo «Filtro»: `Enter` lo aplica y devuelve el foco a la tabla;
  `Esc` lo limpia y vuelve. Mientras tiene foco, el footer muestra
  `Enter aplicar`, `Esc volver`, `←→ cursor` y `F1/? ayuda`.
- `o` abre el campo «Ruta» con placeholder atenuado; `Enter` valida el manifiesto
  y `Esc` vuelve a la tabla sin cancelar el flujo. Una ruta inválida conserva el
  valor, marca el campo y muestra un `ERROR` en la zona de mensajes sin mover las
  filas. Las rutas largas se desplazan con `…` manteniendo visible el cursor.
- Si una fuente no existe o no se puede leer, se informa sin ocultar la otra.
- Variantes: filtro activo, sin clones (`No hay clones disponibles`), ruta manual
  con error de manifiesto, y scroll si hay más filas que espacio.

### 4.7 D1 · Apps de la implantación

```text
kamino · Implantar · Apps                                    Paso 2 de 4   * * o o
--------------------------------------------------------------------------------

  Elige las apps que quieres implantar

  SEL  APP                     ARCHIVOS
  --------------------------------------------------------------------------
  > [ ] herdr                       12
    [ ] ghostty                      8
    [ ] git                          4
    [ ] bat                          3

  ^ 0 arriba   v 2 abajo
  Sin base en este Mac: claude-code, yazi

AVISO  Cierra las apps antes de continuar: pueden sobrescribir los cambios al cerrarse.
--------------------------------------------------------------------------------
^v mover  Espacio marcar  a/n todas/ninguna  Enter continuar  Esc volver  ? ayuda
```

Variante: preparación.

```text
  Preparando la implantación...

  Verificando el archivo y extrayendo el clon...
  Esc cancelar   Ctrl+C cancelar
```

Notas:

- Ninguna app empieza marcada.
- El aviso de cerrar apps va en la zona de mensajes, no en el body.
- Las apps no instaladas son informativas y no cuentan como omitidas.
- La preparación es cancelable y limpia el staging si se cancela.
- Un checksum fallido vuelve a D0 sin escribir.

#### 4.7.1 D1 · Pregunta de instalación

Aparece al pulsar Enter en D1 solo si alguna app ausente tiene una receta aplicable a este Mac. Si no hay ninguna, D1 pasa directamente al plan.

```text
kamino · Implantar · Apps                                    Paso 2 de 4   * * o o
--------------------------------------------------------------------------------

  Elige las apps que quieres implantar

  ¿Instalar las apps ausentes que tienen receta para este Mac?
  Marca las que quieras instalar. Las no marcadas quedan como «Sin base en este Mac».

  > [x] iTerm2  brew
    [ ] yazi  installer · desde https://example.com/yazi.sh

AVISO  ¿Instalar las apps ausentes que tienen receta para este Mac?
--------------------------------------------------------------------------------
^v mover  Espacio marcar  a/n todas/ninguna  Enter continuar  Esc volver  ? ayuda
```

Notas:

- Ninguna receta se marca por defecto. Rechazar una o todas no cancela la implantación: las no marcadas quedan como «Sin base en este Mac».
- El método se muestra siempre. Para `installer`, la URL exacta que se descargará.
- Esc vuelve a la elección de apps sin perder la selección.
- Al confirmar, la instalación se ejecuta durante la planificación, después de verificar el archivo y antes de clasificar los archivos y crear la instantánea.

### 4.8 D2 · Conflictos

```text
kamino · Implantar · Conflictos                              Paso 3 de 4   * * * o
--------------------------------------------------------------------------------

  Conflictos
  Nuevos 4   Clones perfectos 10   No soportados 1   Avisos 1

  ARCHIVO                          BLOQUES  LÍNEAS  ACCIÓN
  --------------------------------------------------------------------------
  > ~/.config/herdr/config.toml         2      14  Revisar
    ~/.config/ghostty/config            1       3  Aplicar clon
    ~/.config/git/config                0       0  Conservar original
    ~/.config/bat/config                3       7  Revisar

  ^ 0 arriba   v 0 abajo

--------------------------------------------------------------------------------
^v mover  r revisar  c/o clon/original  C/O todo  Enter confirmar  Esc volver  ? ayuda
```

Notas:

- Solo se listan los conflictos. Los conteos de nuevos, clones perfectos, no
  soportados y avisos van arriba.
- El cero se muestra como `.`.
- `r` abre D3; `c` aplica el clon; `o` conserva el original. `C`/`O` actúan
  sobre todos.
- Enter solo avanza si todos los conflictos tienen decisión. Si falta alguno, la
  zona de mensajes muestra un `AVISO` con el archivo pendiente.
- Sin conflictos se muestra «Sin conflictos: todo se puede aplicar» y Enter pasa
  a D4.
- Variantes: scroll, todas las filas decididas, y aviso de secretos en la zona de
  mensajes al abrir D3.

### 4.9 D3 · Revisión por líneas

Vista completa del archivo, dos columnas. El ejemplo muestra un bloque; la vista
recorre el archivo entero con scroll.

```text
kamino · Implantar · Conflictos                              Paso 3 de 4   * * * o
--------------------------------------------------------------------------------

  ~/.config/herdr/config.toml

  Original                                      Clon
  12 líneas elegidas                            13 líneas elegidas

  bloque 1 de 2
  >  12 * const a = 1                             12 o const a = 2
     13 o -- sin línea --                          13 * const b = 3
     14   contexto igual                           14   contexto igual
     15 * const some_long_key = "value_that..."    15 o const some_long_key = "value_that..."
     16 * const other = true                       16 o const other = false

  ^ 120 arriba   v 19850 abajo

AVISO  Los diffs pueden contener secretos.
--------------------------------------------------------------------------------
^v línea  <> / Espacio elegir  c/o bloque  C/O archivo  Tab bloque  Esc volver  ? ayuda
```

Notas:

- Cada fila cambiada tiene exactamente una versión elegida (`*`); la otra queda
  en `Faint`. No existe un estado sin decidir.
- Las líneas iguales no llevan casilla y se muestran como contexto.
- El lado sin línea muestra `-- sin línea --`; elegirlo omite la línea.
- Los contadores bajo cada título cuentan solo líneas cambiadas elegidas de ese
  lado.
- `↑↓` mueve la fila; `←→` o `Espacio` eligen versión; `c`/`o` actúan sobre el
  bloque; `C`/`O` sobre todo el archivo; `Tab`/`Shift+Tab` recorren bloques;
  `Esc` vuelve y valida el híbrido.
- Las líneas largas se truncan por la derecha con `...`. Las rutas conservan el
  final y abrevian el home con `~`.
- El aviso de secretos va en la zona de mensajes, no en el body.
- Solo se renderiza la ventana visible. El modelo de filas puede tener 20 000
  entradas; no se montan 20 000 widgets.

Variante: decisión de archivo entero (binario, texto de más de 5 MB o enlace).

```text
  Conflicto de archivo entero

  Ruta: ~/.config/herdr/state.bin
  Motivo: binario (contiene NUL)

  > Aplicar clon
    Conservar original

--------------------------------------------------------------------------------
^v mover  Enter aceptar  Esc volver  ? ayuda
```

### 4.10 D4 · Confirmación

```text
kamino · Implantar · Confirmación                            Paso 4 de 4   * * * *
--------------------------------------------------------------------------------

  Confirma la implantación

  APP               NUEVOS  REEMPLAZOS  HÍBRIDOS  SIN DESPLEGAR  SIN CAMBIOS
  --------------------------------------------------------------------------
  herdr                  2           3         1              1            4
  ghostty                1           2         .              .            3
  git                    .           1         1              .            2
  bat                    .           .         .              1            5
  --------------------------------------------------------------------------
  TOTAL                  3           6         2              2           14

  Protegidos: 8    Holocrón: ~/.config/kamino/vault/snapshot/20261113.tar.gz

  Abortar misión        > Retirada <        Implantar

AVISO  Cierra las apps antes de implantar: pueden sobrescribir los cambios.
--------------------------------------------------------------------------------
<> mover  Enter aceptar  Esc retroceder  ? ayuda
```

Notas:

- Filas por app; columnas Nuevos, Reemplazos, Híbridos, Sin implantar y Sin
  cambios. El cero es `.`.
- La fila de totales va separada por una línea. No hay líneas entre filas.
- `Protegidos = reemplazos + híbridos` contando solo archivos existentes que se
  escribirán. Un híbrido que compone idéntico al local cae en «Sin cambios» y no
  se cuenta como protegido.
- El aviso de cerrar apps va en la zona de mensajes, no en el body.
- La tabla se calcula con `ImplantSession.Preview(decisions)` y
  `ImplantPlan.Entries`, deduplicando por `Path` igual que `resolve`: la primera
  app gana. Así los totales coinciden con el núcleo.
- Las acciones son horizontales, con Retirada preseleccionada. No hay atajo de
  una tecla para implantar: hay que mover el foco y pulsar Enter.
- Si no hay originales que proteger, se muestra «No hay originales que proteger;
  no se creará holocrón» en lugar de la ruta.
- Variantes: muchas apps con scroll y tabla abreviada a 80 columnas.

### 4.11 D5 · Resumen de la implantación

```text
kamino · Implantar · Resumen
--------------------------------------------------------------------------------

  Implantación completada

  RESUMEN
  --------------------------------------------------------------------------
  Implantados                                                            9
  Híbridos                                                               2
  Sin implantar                                                          2
  Sin cambios                                                           14
  Sin base en este Mac                                                   2
  Archivos protegidos                                                    8

  Holocrón: ~/.config/kamino/vault/snapshot/20261113.tar.gz
  Para deshacer: kamino implant ~/.config/kamino/vault/snapshot/20261113.tar.gz

  c Copiar holocrón

--------------------------------------------------------------------------------
c copiar holocrón  Enter/Esc continuar  ? ayuda
```

Notas:

- «Implantados» incluye nuevos y reemplazos; el detalle puede mostrarse como
  `9 (3 nuevos, 6 reemplazos)`.
- Un híbrido cuenta como éxito. El código 3 se reserva a omitidos, no soportados
  y `--apps` no cumplidos.
- La variante «Implantación incompleta» usa el título en rojo, un `ERROR` en la
  zona de mensajes, muestra aplicados, fallido y holocrón, y termina con código 1.
  No hay rollback automático.
- Si no se creó holocrón, se muestra «No había originales que proteger».
- `c` copia la ruta del holocrón; el aviso `INFO` va en la zona de mensajes.

### 4.12 U1 · Actualización

```text
kamino · Actualización
--------------------------------------------------------------------------------

  Actualizar kamino
  Versión actual: 0.9.0
  Última versión: 0.9.1

  Descargando y verificando...

  [=========>          ]  60%

  Ctrl+C cancela antes de reemplazar el binario.

--------------------------------------------------------------------------------
Enter continuar  Esc cancelar/salir  ? ayuda
```

Notas:

- Estados: consultando, descargando, verificando, reemplazando y resultado.
- Resultados: actualizado, ya al día, instalada más reciente, error de red,
  integridad, plataforma no soportada o instalación no autocontenida.
- La barra se calcula con el ancho del body; `View` no usa `time.Now()`.
- Un error de actualización abre la pantalla de error común con la acción
  contextual que corresponda.
- La consulta y la descarga son comandos asíncronos; la cancelación se comprueba
  por chunk.

### 4.13 Configuración

```text
kamino · Configuración
--------------------------------------------------------------------------------

  Configuración
  El catálogo y el límite de holocrones también los aplica la CLI.

  > Catálogo    ~/.config/kamino/apps|
    Clones      ~/.config/kamino/vault
    Holocrones  ~/.config/kamino/vault/snapshot
    Límite      5

    Añadir app  Crea una definición TOML en el catálogo del usuario.

--------------------------------------------------------------------------------
Tab/Mayús+Tab mover  Ctrl+S guardar  Esc volver  ←→ cursor  F1/? ayuda
```

Notas:

- Las tres rutas y el límite son campos editables de una fila con etiqueta
  alineada, no una lista de solo lectura. El foco añade marcador `▌`, fondo ANSI,
  etiqueta en negrita y el cursor en vídeo inverso; los demás valores quedan
  atenuados. El cursor solo aparece en el campo enfocado.
- `Tab`/`Mayús+Tab` y `↑`/`↓` mueven el foco entre los cuatro campos; `←`/`→`
  mueven el cursor dentro del valor y no cambian de campo. El foco empieza en
  Catálogo.
- El campo del límite solo admite dígitos; un valor vacío, negativo o fuera de
  rango se rechaza con `ERROR` al guardar.
- `Ctrl+S` valida y guarda de forma atómica; `Esc` vuelve sin escribir.
- Los valores vacíos muestran un placeholder atenuado. Las rutas largas se
  desplazan dentro del campo con `…` para mantener visible el cursor, sin partir
  caracteres UTF-8 ni de doble ancho; sin foco conservan el estilo atenuado.
- Si el archivo es inválido, la pantalla se abre con los valores legibles y los
  defaults, muestra un `ERROR` en la zona de mensajes y marca el campo afectado
  sin desplazar las filas.
- El aviso «Configuración guardada» va en la zona de mensajes como `INFO`.
- Después de los cuatro campos hay una acción «Añadir app». `Tab`/`Mayús+Tab` y
  las flechas verticales la alcanzan; `Enter` abre la pantalla de autoría y `Esc`
  vuelve al menú. La acción no se edita como un campo y `Enter` no guarda.
- Guardar un `catalog.dir` distinto descarta el catálogo en memoria; el siguiente
  flujo lo carga desde la ruta nueva sin reiniciar.
- `?` y `F1` abren la ayuda desde un campo sin insertar caracteres.

### 4.14 Ayuda contextual

```text
kamino · Implantar · Ayuda                                   Paso 3 de 4   * * * o
--------------------------------------------------------------------------------

  Ayuda · Conflictos

  Atajos de esta pantalla
  ^v mover          r revisar          c/o clon/original
  C/O todo          Enter confirmar    Esc volver

  Atajos globales
  ? ayuda           Ctrl+C cancelar

  Flujo de implantación
  * clon   * apps   * conflictos   o confirmación

  ! Los diffs pueden contener secretos.

--------------------------------------------------------------------------------
Enter/Esc/? cerrar
```

Notas:

- Los atajos salen de `Keys()`, no de una lista escrita a mano.
- El flujo resalta el paso actual con `*` y acento; los pendientes van con `o`.
- El aviso de secretos solo aparece en D2 y D3.
- En campos de texto, `F1` y `?` abren esta pantalla sin insertar caracteres; al
  cerrarla, el campo conserva su valor, su foco y su cursor.
- La ayuda se apila sobre la pantalla actual; al cerrar se recupera su estado.

### 4.15 Error común

```text
kamino · Error
--------------------------------------------------------------------------------

  No se pudo planificar el clon

  No se escribió nada.

  Qué hacer: Revisa el destino y vuelve a intentarlo.

  [t Ver detalle]      [> Volver al menú]      [Salir]

--------------------------------------------------------------------------------
t detalle  c copiar  Enter aceptar  Esc volver  ? ayuda
```

Variante: detalle abierto.

```text
  Detalle técnico

  open /Users/test/.config/kamino/vault: permission denied
```

Notas:

- Cuatro partes: qué pasó, estado de seguridad, qué hacer y acciones.
- El detalle está plegado, se abre con `t` y se copia con `c`. Nunca incluye
  contenido de archivos.
- En modo directo no se ofrece «Volver al menú»; solo la acción contextual y
  «Salir».
- Las acciones se recorren con flechas y Enter; Esc vuelve a la pantalla anterior.
- Los textos de error siguen siendo literales y viven en `internal/texts`.

### 4.16 Aviso de terminal pequeña

Tamaño: 79×23. La guardia vive en el `Frame`; la pila y el estado no se tocan.

```text
kamino · Aviso
-------------------------------------------------------------------------------

  La terminal es demasiado pequeña
  Mínimo: 80 x 24   Actual: 79 x 23

  El flujo sigue en segundo plano. Amplía la ventana para continuar.
  Operación en curso: clonación, 12 de 20 archivos.

  ? ayuda
```

Notas:

- Se actualiza al redimensionar y recupera la pantalla anterior al volver a
  80×24.
- La operación en curso no se cancela ni se cierra la aplicación.
- No se pierde el estado de la pantalla inferior.

### 4.17 A1 · Añadir app

La pantalla crea un borrador `.toml.tmp` en el catálogo efectivo, abre el editor
del usuario y valida el archivo al volver. Un borrador sin editar no se guarda.

```text
kamino · Configuración · Añadir app
--------------------------------------------------------------------------------

  Nueva definición
  Se creará un borrador y se abrirá tu editor.

  Catálogo: ~/.config/kamino/apps
  Borrador: ~/.config/kamino/apps/.kamino-draft-123.toml.tmp

  Usa rutas relativas al HOME, sin ".." ni rutas absolutas.
  El borrador no se guardará hasta que sea válido.

--------------------------------------------------------------------------------
Enter/e editar   Esc descartar                                      ? ayuda
```

Notas:

- Se abre desde la acción «Añadir app» de Configuración; al terminar o cancelar
  se vuelve a Configuración.
- `Enter` o `e` abre `$VISUAL`, `$EDITOR` o `vi` con `tea.ExecProcess`; al volver,
  Kamino parsea y valida el borrador con las mismas reglas que `LoadCatalog`.
- Si el TOML es inválido o el editor falla, la pantalla sigue abierta, muestra el
  `ERROR` en la zona de mensajes y la ruta del borrador, y permite reeditar o
  descartar con `Esc`.
- Si el borrador es válido, el archivo se renombra de forma atómica a
  `<id>.toml` y el catálogo se recarga. Si el id reemplaza una app integrada o el
  archivo destino ya existe, se muestra una confirmación antes de escribir.
- La pantalla de éxito muestra la ruta creada y si la app se detectó. Si no se
  detectó, se avisa de que aparecerá en Clonar cuando esté instalada.
- `Esc` antes de editar no crea archivos; `Esc` con un borrador lo elimina.

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

## 7. Impacto en specs y documentación

| Artefacto | Cambio |
| --------- | ------ |
| `tui-navigation` | Frame, pila, contrato `Screen`, footer/ayuda desde `Keys()`, altura, menú, terminal pequeña |
| `tui-ansi-theme` | Solo ANSI, jerarquía, `theme.Styles`, `NO_COLOR`, ASCII, `ansi-light` |
| `clone-tui` | C1 como tabla y campo de filtro; C2 con tres opciones de carpeta, campos «Destino» y «Nombre» y diálogo Sobrescribir/Cambiar nombre/Volver; C3 absorbe C4: revisión, progreso y resultado en la misma tabla; el flujo sigue en tres pasos |
| `implant-conflicts` | D2 como tabla, D3 con archivo completo, D4 por app y protegidos |
| `docs/diseno-tecnico.md` | Sustituir «ANSI propio, sin framework» por Bubble Tea/Lip Gloss y el contrato de pantalla |
| `docs/diseno-funcional.md` | Actualizar C1, C3/C4, D3 y D4 |
| `AGENTS.md` | Retirar el invariante «sin framework» y describir el nuevo runtime |

## 8. Plan de implementación

1. `theme` (tokens, estilos, símbolos) y `Frame`; `App` con pila y
   `tea.WindowSizeMsg`; harness de golden files y test de contrato.
2. C1, C3, D3 y D4 como pantallas de referencia, con goldens a 80×24, 100×26,
   140×40 y 200×50, más `NO_COLOR` y ASCII.
3. Resto por derivación: menú, C2, D0–D2, D5, U1, configuración, ayuda, error y
   aviso de terminal pequeña.
4. Comandos asíncronos para recuento, planificación, escritura, implantación y
   actualización; pruebas de cancelación y códigos de salida.
5. Actualizar specs, `diseno-tecnico.md`, `diseno-funcional.md` y `AGENTS.md`.

Cada paso debe compilar y pasar los golden files. Los commits siguen
Conventional Commits (`feat(tui):`, `refactor(tui):`, `test(tui):`).

## 9. Puntos a validar antes de generar goldens

1. **D3:** ¿se acepta mostrar el archivo completo en dos columnas y sustituir la
   restricción actual «sin previsualización completa»? El aviso de secretos y los
   límites de 5 MB/20 000 líneas se mantienen.
2. **D4:** ¿la tabla por app y la fórmula `protegidos = reemplazos + híbridos`
   representan lo que quieres validar? Los totales se calculan con `Preview` y la
   misma deduplicación por ruta que el núcleo.
3. **Layout:** `ColGap=4` corrige el 3 de las reglas para que `46 | 4 | 46` sume
   96. ¿Se acepta la corrección?
4. **Altura:** «30+ layout completo» se interpreta como más altura para el
   contenido, sin una zona adicional por altura. ¿Es suficiente o quieres reservar
   la zona de mensajes a partir de 30 filas?
5. **Menú:** las ocho cápsulas pasan al lado derecho del header y el wordmark
   queda en el body. ¿Se mantiene así?
6. **Acciones de D4:** orden `Abortar misión`, `Retirada`, `Implantar`, con
   Retirada preseleccionada. ¿Se mantiene el orden de la spec?
7. **`KAMINO_NO_LOGO`:** además de ocultar el wordmark, fuerza símbolos ASCII.
   ¿Es el comportamiento deseado?
8. **Asincronía:** Path A incluye mover a `tea.Cmd` las operaciones que hoy
   bloquean la UI. ¿Confirmas que entra en este cambio y no se deja para después?
9. **C3/C4:** la pantalla única conserva `Paso 3 de 3` en el estado final y cambia
   el header a `Resultado`. ¿Prefieres ocultar el paso al completar?
10. **Mensajes:** la zona muestra hasta tres líneas, una por categoría, y el body
    se reduce. ¿Prefieres una sola línea con prioridad `ERROR` > `AVISO` > `INFO`?
    Los errores bloqueantes siguen usando la pantalla de error común.
11. **C2 (resuelto):** las tres opciones de carpeta sustituyen a `F2`/`F3`/`F4`,
    que se retiran. La sugerencia del nombre incluye `.tar.gz`.
