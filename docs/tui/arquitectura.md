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

