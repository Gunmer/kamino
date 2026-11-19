# Design

## Context

La TUI ya tiene un `Frame` único, un contrato `Screen` y un tema ANSI
centralizado. Sin embargo, cada pantalla edita texto a su manera:

- C1 y D0 guardan el filtro en un `string` y lo dibujan como `Filtro: {valor}_`.
- C2 guarda `path` y `name` en `string`, añade `_` al final y solo permite
  escribir o borrar al final.
- Configuración guarda cuatro `string`, dibuja etiqueta y valor en filas
  separadas y usa `_` como cursor.

El resultado es un foco ambiguo, sin posición de inserción real y sin scroll de
rutas. Ver `proposal.md` para la motivación y las specs delta para el contrato
de comportamiento.

Restricciones que condicionan el diseño:

- Go 1.25; Bubble Tea v1.3.10, Lip Gloss v1.1.0 y `bubbles` ya están en
  `go.mod`. No se añaden frameworks ni dependencias. La restricción de «sin
  frameworks externos de TUI» se interpreta como no introducir ningún framework
  nuevo ni sustituir el runtime existente por terminal crudo: el repositorio ya
  usa Bubble Tea y reescribirlo quedaría fuera de este cambio.
- Los textos visibles van en español y en `internal/texts`; el código,
  identificadores y comentarios van en inglés.
- `View` sigue siendo puro: no lee entorno, reloj ni disco.
- Los goldens y el test de contrato exigen exactamente `h` líneas, ninguna línea
  más ancha que `w` y solo secuencias SGR ANSI.
- No cambian la CLI, los códigos de salida, el formato del clon, el manifiesto
  ni las rutas de seguridad.

## Goals / Non-Goals

**Goals:**

- Un único componente de campo en `internal/tui` que concentre render, edición,
  cursor, placeholder, scroll y estado de error.
- Un foco inequívoco: `▌`, fondo ANSI, etiqueta en negrita, valor en acento y
  cursor de edición; sin foco, texto atenuado y sin cursor.
- Edición en cualquier posición del valor, no solo al final.
- Scroll horizontal determinista con UTF-8 y caracteres anchos.
- Degradación por forma con `NO_COLOR` y por ASCII sin UTF-8.
- Integración en C1, C2, D0 y Configuración sin duplicar lógica.
- Pruebas de modelo y de render, goldens y documentación actualizada.

**Non-Goals:**

- No se implementa un cursor real del terminal; se usa el fallback de vídeo
  inverso (decisión 2).
- No se añaden `bubbles/textinput`, `viewport` ni un framework de formularios.
- No se cambia el modelo de validación de rutas ni los mensajes de error
  existentes.
- No se convierten en campos editables las tablas, los diálogos ni las
  pantallas sin entradas de texto.
- No se cambia el formato de `config.toml` ni el comportamiento de la CLI.

## Decisions

### 1. Componente `field` en `internal/tui`

Se añade `internal/tui/field.go` con un tipo no exportado:

```go
type field struct {
    Label       string
    Value       string
    Placeholder string
    Cursor      int  // rune index in [0, len([]rune(Value))]
    Invalid     bool
}

func (f *field) SetValue(value string)
func (f *field) Insert(text string)
func (f *field) Backspace()
func (f *field) MoveCursor(delta int)
func (f field) Render(theme *theme.Theme, options fieldOptions) string
```

`fieldOptions` contiene `Width`, `LabelWidth` y `Focused`. El foco no se guarda
en el campo: lo decide la pantalla y lo pasa al render, de modo que no puede
haber dos campos con foco por estado inconsistente. `Invalid` sí vive en el
campo porque es estado de validación de ese valor.

Las pantallas dejan de guardar `string` para los inputs y guardan `field`:

- C1: `filter field`.
- C2: `path field`, `name field`.
- D0: `filter field`, `manualPath field`.
- Configuración: `fields [fieldCount]field`.

La lógica de insertar, borrar y mover el cursor existe una sola vez. Las
pantallas solo deciden qué campo recibe la tecla, cuándo se abre o se cierra y
qué validación aplica.

Alternativas descartadas:

- Un `textinput` de `bubbles`: añade un modelo con estilos y keymap propios que
  no encajan con el tema ni con el contrato de una fila.
- Una interfaz `Input` con implementaciones por pantalla: sería una abstracción
  de una sola implementación y volvería a duplicar el render.
- Un `string` más un `cursor int` por pantalla: repite la misma lógica cuatro
  veces, que es justo lo que se quiere eliminar.

### 2. Cursor: fallback determinista en vídeo inverso

Bubble Tea v1.3.10 solo expone `tea.ShowCursor` y `tea.HideCursor`; no permite
posicionar el cursor. Su `standardRenderer` trunca cada línea, escribe el frame y
después ejecuta `ansi.CursorPosition(0, len(newLines))`, que devuelve el cursor
al inicio de la última línea. Un escape CUP añadido al final de `View` sería
sobrescrito y, además, rompería el test de contrato que solo admite SGR.

Posicionar el cursor real exigiría sustituir el renderer de Bubble Tea o
gestionar el terminal fuera de él; ambas opciones son un cambio desproporcionado
para este rediseño y afectan al repintado, al alt-screen y a los goldens.

Decisión: se implementa el fallback previsto por la petición. El componente
dibuja el cursor sobre el carácter que está en la posición de inserción con
vídeo inverso (`Reverse(true)`); si el cursor está al final del valor, dibuja un
espacio en vídeo inverso. Solo el campo enfocado dibuja ese cursor. El cursor
real del terminal queda oculto por el renderer de Bubble Tea y no se emite
ninguna secuencia de posicionamiento.

El marcador `▌` de las tablas sigue siendo el indicador de fila seleccionada,
no un cursor de texto; no se elimina ni se confunde con el cursor de edición.

### 3. Render de una fila

La fila se compone así:

```text
▌ Destino   ~/.config/kamino/vault/█
  Catálogo  ~/.config/kamino/apps
```

- Marcador: `theme.Symbols.Cursor` (`▌`/`>`) y un espacio; sin foco, dos
  espacios.
- Etiqueta: `PadRight(label, labelWidth)`; con foco en negrita, sin foco
  terciaria, con error en color de error.
- Separación: dos espacios.
- Valor: el ancho restante. Con foco, acento sobre fondo ANSI; sin foco,
  terciario/`Faint`; con error, color de error.
- Fondo: ANSI 8 en `ansi-dark` y ANSI 7 en `ansi-light`, aplicado desde el
  marcador hasta el final del valor visible. No se rellena el resto de la línea
  para no convertir la fila en una barra de ancho completo. Los tres segmentos
  de la fila enfocada comparten el mismo fondo: el marcador y el valor usan
  `FieldFocus`, y la etiqueta usa `FieldLabelFocus`, que añade negrita sobre ese
  fondo.
- `NO_COLOR`: sin fondo ni colores; el foco se distingue por `▌`, la etiqueta en
  negrita, el cursor en vídeo inverso y la palabra «editando».
- «editando»: se añade a la etiqueta solo con `NO_COLOR` y solo en el campo
  enfocado. Para no desalinear los valores, `labelWidth` reserva el ancho de
  `etiqueta + " · editando"` cuando `NO_COLOR` está activo.
- Placeholder: se dibuja con el estilo atenuado. Si el campo está vacío y tiene
  foco, el cursor en vídeo inverso va al inicio, antes del placeholder.
- ASCII: `theme.Symbols.Cursor` ya es `>` y `theme.Symbols.Ellipsis` ya es
  `...`; el vídeo inverso se conserva.

Los estilos nuevos viven en `theme.Styles` (`FieldFocus`, `FieldIdle`,
`FieldLabelFocus`, `FieldLabelIdle`, `FieldError`, `FieldCaret`). Ninguna
pantalla crea estilos ni colores propios.

### 4. Edición y desplazamiento horizontal

`Cursor` es un índice de runas, no de bytes. `Insert` inserta en esa posición y
avanza tantas runas como el texto insertado; `Backspace` borra la runa anterior;
`MoveCursor` se limita a `[0, len(runes)]`. `SetValue` coloca el cursor al final,
que es el comportamiento actual de C2 y Configuración al abrirse.

El scroll no se guarda: se calcula en cada render a partir de `Value`, `Cursor`
y el ancho disponible. Así `View` sigue siendo puro y los goldens son estables.

Algoritmo:

1. `valueWidth = width - markerWidth - labelWidth - gap`; mínimo 1.
2. Se mide cada runa con `lipgloss.Width` para no partir caracteres de doble
   ancho.
3. Si el valor completo y su cursor caben, no hay scroll.
4. Si no caben, se avanza el inicio de la ventana runa a runa hasta que el
   cursor quede dentro. Se reserva el ancho real de `theme.Symbols.Ellipsis` al
   inicio si se oculta el comienzo y al final si se oculta el final (`…` mide 1
   columna; `...` mide 3).
5. Si el cursor está al final, se reserva además una columna para el espacio en
   vídeo inverso.
6. Se construye la ventana sin partir runas; si una runa ancha no cabe, se deja
   fuera y se marca el lado oculto con la elipsis.
7. La elipsis se toma siempre de `theme.Symbols.Ellipsis`; en ASCII ya vale
   `...`, así que la ventana reserva su ancho real y el `Frame` no la vuelve a
   ensanchar al convertir símbolos.

Casos límite: anchos muy pequeños muestran solo el cursor o la elipsis; un valor
vacío no genera scroll; un cursor en el límite de una runa ancha se dibuja sobre
la runa completa.

### 5. Errores y presupuesto de layout

Los errores de validación siguen usando la zona de mensajes del `Frame`, que ya
incluye icono, palabra y color. El campo afectado se marca con `Invalid` para
que la etiqueta y el valor usen el estilo de error. No se añade una línea de
error debajo de cada campo: la fila del campo nunca cambia de altura y las filas
restantes no se reordenan. La zona de mensajes puede aparecer o desaparecer como
ya hace hoy; lo que no cambia es la posición relativa de las filas de campo.

«Sin mover el layout» se interpreta así: el campo sigue ocupando una fila y el
error no inserta filas nuevas entre campos. No se reserva una línea vacía
permanente porque la spec de la zona de mensajes prohíbe reservarla cuando está
vacía.

A 80x24 el body mide 76x19 (o menos si hay mensajes). Configuración usa cuatro
filas de campo, un título y un subtítulo: cabe con holgura. C2 usa dos filas de
campo además de las opciones y el nombre; D0 usa una fila de filtro o de ruta.
Ninguna pantalla añade una segunda fila por campo.

### 6. Integración por pantalla

- **C1.** `filtering` sigue siendo el booleano de modo; el valor vive en
  `filter field`. `/` abre el campo y coloca el cursor al final; `Enter` aplica
  y devuelve el foco a la tabla; `Esc` limpia y cierra. `a`/`n` y el resto de la
  navegación no cambian.
- **C2.** `path` y `name` pasan a `field` con etiquetas «Destino» y «Nombre».
  `insert`/`deleteRune` desaparecen; `Update` envía las teclas al campo activo.
  `Enter` conserva sus transiciones actuales (ruta → nombre → validación).
  `validate` marca el campo correspondiente y mantiene los mensajes actuales.
  El diálogo de clon existente no cambia.
- **D0.** `filter` y `manualPath` pasan a `field` con etiquetas «Filtro» y
  «Ruta». `Enter` valida la ruta y `Esc` vuelve a la tabla; una ruta inválida
  conserva el valor y marca el campo. La fila `Otro archivo...` y el atajo `o`
  conservan su comportamiento.
- **Configuración.** Los cuatro valores usan `field`; las etiquetas son
  «Catálogo», «Clones», «Holocrones» y «Límite». El foco empieza en Catálogo.
  `Tab`/`Mayús+Tab` y las flechas verticales mueven el foco; `←`/`→` mueven el
  cursor. El límite filtra las runas no numéricas antes de insertar. `Ctrl+S`
  valida y guarda; `Esc` vuelve sin escribir. Las pistas por campo desaparecen
  del body; el subtítulo existente y la ayuda contextual conservan la
  explicación.

### 7. Pie de atajos y ayuda

Las pantallas siguen generando el pie y la ayuda desde `Keys()`. Cuando un campo
tiene foco, `Keys()` devuelve las teclas de esa pantalla más:

- `Enter` con la etiqueta que corresponda (continuar, fijar, aplicar, abrir).
- `Esc` volver.
- `←→` cursor, cuando hay más de un carácter.
- `Tab` campo, cuando hay varios campos.
- `F1/?` ayuda.

Se añade un binding compartido `fieldHelpKey()` con claves `f1` y `?` para que el
pie muestre la ayuda del campo y el `Frame` no añada el sufijo `? ayuda`
duplicado. La ayuda contextual sigue mostrando `?` y `F1` en su lista global.

`?` y `F1` se interceptan en `App.Update` antes de llegar al campo, así que
abren la ayuda sin insertar caracteres. El valor, el foco y el cursor no cambian
porque la pantalla no recibe la tecla.

### 8. Pruebas y goldens

- `field_test.go`: inserción, retroceso, movimiento, límites, UTF-8, runas
  anchas, ventana visible, placeholder, error, `NO_COLOR` y ASCII.
- Pruebas por pantalla: C1 filtro, C2 ruta/nombre, D0 filtro/ruta manual,
  Configuración foco/navegación/dígitos/error, con HOME temporal.
- Render: una sola fila por campo, un solo cursor, cursor en medio, ruta larga
  con `…` inicial, vacío con placeholder, error sin mover filas, `NO_COLOR` con
  `▌`/vídeo inverso/«editando», ASCII con `>`/`...` y 80x24.
- Goldens: se regeneran C2, D0 y Configuración, y se añaden casos de C1 con
  filtro, C2 con foco en ruta y nombre, D0 con filtro y ruta manual, y
  Configuración vacía. Todos los casos pasan por color, `NO_COLOR` y ASCII.
- El test de contrato se amplía con los casos nuevos y comprueba que el render
  sigue midiendo exactamente `w×h` y solo usa SGR.

### 9. Documentación

- `README.md`: comportamiento de los campos, foco, cursor, `NO_COLOR`, ASCII y
  ayuda.
- `docs/diseno-funcional.md`: TUI, C1, C2, D0 y Configuración.
- `docs/diseno-tecnico.md`: componente `field`, decisión del cursor y pruebas.
- `docs/diseno-tui.md`: sustituir las maquetas de inputs por la fila única y
  corregir la nota que dice que `?` se inserta como carácter; el
  comportamiento real y el pedido es que `?` abra la ayuda.

## Risks / Trade-offs

- **El cursor no es el real del terminal** → se usa vídeo inverso, que es
  determinista y funciona en los goldens; la posición es exacta porque se
  calcula sobre las runas visibles.
- **El fondo ANSI 8/7 depende de la paleta del terminal** → el marcador, la
  negrita y el cursor siguen distinguiendo el foco aunque el contraste del fondo
  sea bajo; con `NO_COLOR` no se usa fondo.
- **La etiqueta «editando» ensancha la columna de etiquetas con `NO_COLOR`** →
  el ancho se reserva para todos los campos de la pantalla, así los valores
  siguen alineados; a 80 columnas el espacio restante es suficiente.
- **El scroll calculado en cada render puede cambiar de ventana al mover el
  cursor** → es intencional: la ventana sigue al cursor y no guarda estado
  obsoleto; las pruebas cubren los extremos y las runas anchas.
- **Las pistas por campo de Configuración desaparecen** → el subtítulo y la
  ayuda contextual conservan la explicación; se documenta en el README y en el
  diseño funcional.
- **Regenerar goldens oculta cambios accidentales** → se revisan los diffs y el
  test de contrato valida altura, ancho, ANSI y ausencia de color con
  `NO_COLOR`.

## Migration Plan

No hay migración de datos ni de formato. El cambio es interno a la TUI:

1. Añadir el componente y sus estilos sin cambiar pantallas.
2. Migrar Configuración, C1, C2 y D0 en ese orden; regenerar goldens por
   pantalla.
3. Actualizar textos y documentación.
4. Ejecutar `gofmt -l .`, `go vet ./...`, `golangci-lint run ./...` y
   `go test ./... -race`.

Rollback: revertir el cambio completo; no hay estado persistido ni archivos de
configuración nuevos. Los goldens anteriores se recuperan del historial.

## Open Questions

Ninguna. Las decisiones que podrían haber quedado abiertas se resuelven así:

- El campo del límite de holocrones también usa el componente, para que todos
  los inputs compartan el mismo foco y render.
- Los errores usan la zona de mensajes existente, no una línea nueva por campo.
- `←`/`→` mueven el cursor; `↑`/`↓`, `Tab` y `Mayús+Tab` mueven el foco.
- El cursor real se sustituye por el fallback de vídeo inverso.
