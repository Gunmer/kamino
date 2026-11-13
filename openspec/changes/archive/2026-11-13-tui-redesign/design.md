# Design

## Context

La TUI actual es un `Model` monolítico con un lector de teclas propio, modo raw y
renderizado ANSI escrito a mano. Las pantallas comparten un `frame()` que trunca
por runas, un `paint()` con colores fijos y una ayuda construida a mano. No hay
un tema centralizado, un contrato de pantalla ni golden files.

El diseño visual aprobado está en `docs/Diseño tui.md`: allí viven las maquetas
ASCII, la notación de símbolos, el presupuesto de layout y los ejemplos de cada
pantalla. Este documento explica la arquitectura para implementarlo. Ver
`proposal.md` para la motivación y las specs delta para el comportamiento.

Restricciones:

- Go 1.25, binario autocontenido y pruebas sin red ni HOME real.
- Solo Bubble Tea, Lip Gloss y `bubbles` (`key`, `help`). No se añaden
  `textinput`, `viewport` ni `table`; las listas y tablas se renderizan a mano.
- Texto visible en español y centralizado en `internal/texts`; código,
  identificadores y comentarios en inglés.
- No cambia la lógica de `clone`, `deploy`, `update`, `conflicts`, `paths` ni
  `snapshot`.

## Goals / Non-Goals

**Goals:**

- Un `Frame` único que posea cabecera, cuerpo, mensajes, pie, separadores,
  guardia de tamaño y `tea.WindowSizeMsg`.
- Un contrato `Screen` con estado local por pantalla y una pila que conserve el
  estado al volver.
- Un tema ANSI centralizado, un layout por tokens y una zona de mensajes
  `INFO`/`AVISO`/`ERROR` sobre el pie.
- C1, C2, C3, D3 y D4 como pantallas de referencia; el resto por derivación.
- Golden files deterministas y un test de contrato de límites y ANSI.
- Operaciones de planificación, escritura, despliegue y actualización sin
  bloquear el render.

**Non-Goals:**

- No cambiar la CLI, los códigos de salida, los textos de error ni el formato
  del archivo del clon o del manifiesto.
- No persistir mensajes, diffs ni contenido de archivos.
- No detectar apps abiertas; D4 solo avisa.
- No añadir un framework alternativo ni un feature flag permanente.
- No reescribir el diseño visual: la fuente es `docs/Diseño tui.md`.

## Decisions

### 1. Runtime: Bubble Tea + Lip Gloss + `bubbles`

Se migra el runner al modelo de Bubble Tea. `App` es la raíz; `tea.Program`
gestiona entrada, modo raw, tamaño y repintado. `lipgloss` aporta medición,
`Place` y estilos; `bubbles/key` y `bubbles/help` alimentan el pie y la ayuda.

Alternativas:

- Mantener el runner ANSI actual (Path B): menos dependencias, pero habría que
  reimplementar medición, layout, ayuda y el bucle de mensajes.
- Usar Lip Gloss sin Bubble Tea: no resuelve el `tea.WindowSizeMsg` ni los
  comandos asíncronos.

La migración se justifica porque las specs ya prometen progreso, cancelación y
recuento en segundo plano; hoy el `Model` bloquea el bucle mientras planifica o
aplica. Las versiones se fijan en `go.mod`; no se usa una API v2 si cambia los
tipos de `tea.Msg` o `key.Binding`.

### 2. `App`, `Frame`, `Screen` y pila

```text
App (raíz de Bubble Tea)
  stack []Screen
  flujo compartido (clone/deploy/update/config)
  Frame
    header · separador · body · mensajes · separador · footer
```

- `App` es el único dueño de la pila y del estado de dominio compartido.
- `Frame` calcula el tamaño del terminal y entrega `bodyW`/`bodyH` a la pantalla.
- `Screen` expone `Init`, `Update`, `View(w,h)`, `Title`, `Step`, `Keys`,
  `Messages` y `SetSize`.
- `View` es puro: sin E/S, sin `time.Now()` y sin leer entorno.
- Las pantallas emiten `pushMsg`, `popMsg` o `replaceMsg`; `Frame` los procesa.
- El enum actual `Screen` se renombra a `ScreenID`; `Screen` queda para la
  interfaz.

Alternativas:

- Un `Model` con `switch` por pantalla: es el estado actual y concentra todo.
- Un programa Bubble Tea por pantalla: pierde la pila y el estado compartido.
- Estilos globales: impiden inyectar `NO_COLOR`/tema en las pruebas.

### 3. Estado compartido y estado local

`App` posee el flujo (`ClonePlan`, `DeploySession`, configuración, resultado,
última misión, código de salida). Cada pantalla posee su estado de presentación:
cursor, filtro, scroll, foco, fila y bloque. Al apilar una pantalla se conserva
la anterior; al volver, su cursor y su filtro siguen intactos y el estado de
dominio refleja lo que hizo la hija.

Esto evita duplicar planes o sesiones y permite probar cada pantalla con un
`App` de fixture.

### 4. Tema, tokens y layout

- `theme.Tokens`: `Gutter=2`, `PadX=2`, `PadY=1`, `Gap=1`, `ColGap=4`.
  `ColGap=4` corrige el 3 de las reglas para que `46 | 4 | 46` sume 96.
- `theme.Styles`: primario, secundario, terciario, acento, éxito/info, aviso,
  error, selección, tabla y mensajes. Ninguna pantalla crea estilos.
- Solo `lipgloss.ANSIColor(n)`; prohibidos hex, RGB y `AdaptiveColor`.
- Renderer con perfil ANSI forzado para que los goldens sean deterministas.
- `layout.Fit` garantiza exactamente `h` líneas y nunca más de `w` columnas;
  `Truncate` usa `lipgloss.Width` y termina en `…`; `TruncatePath` conserva el
  final y abrevia el home con `~`.
- Ancho del body: `termW-4` (<100), 96 (100–139), 120 (140+).
- Alto: 24–25 sin aire; 26–29 con aire; 30+ con el body extra para contenido.
- Sin borde exterior; solo el separador bajo la cabecera y el que está sobre el
  pie.

### 5. Zona de mensajes

`Messages()` devuelve hasta un mensaje por categoría. `Frame` los muestra entre
el body y el separador del pie, en orden `INFO`, `AVISO`, `ERROR`, con icono,
etiqueta y color ANSI. La zona no se reserva vacía; cada línea reduce el body.
Los errores bloqueantes siguen abriendo la pantalla de error común.

Alternativas:

- Una sola línea de status (actual): no distingue información, aviso y error.
- Colorear todo el texto: pierde contraste y depende del color.
- Reservar siempre tres filas: desperdicia altura en 24–25.

### 6. D3: archivo completo en dos columnas

Se construye un modelo de filas de archivo completo a partir de
`Conflict.LocalText`, `Conflict.CloneText` y los hunks:

- fuera de los hunks, original y clon avanzan juntos y la fila es contexto;
- dentro de un hunk se conserva la alineación de `RowReplace`, `RowLocalOnly`,
  `RowCloneOnly` y `RowEqual`;
- cada fila cambiada tiene un índice de decisión y exactamente una versión
  elegida; el lado sin línea usa `— sin línea —` y elegirlo omite la línea;
- los contadores suman las líneas cambiadas elegidas de cada lado;
- solo se renderiza la ventana visible; el modelo puede tener 20 000 filas;
- las líneas largas se truncan por la derecha con `…`.

Esto sustituye la vista de un bloque a la vez y la prohibición de previsualizar
el archivo completo. El aviso de secretos pasa a la zona de mensajes. Los
binarios, los textos de más de 5 MB y los enlaces siguen resolviéndose como
archivo entero.

Alternativas:

- Renderizar cada hunk por separado (actual): no permite comparar el archivo
  completo ni alinear el contexto.
- Renderizar todas las líneas como widgets: inviable con 20 000 líneas.
- Usar `bubbles/viewport`: no está permitido y no aporta nada sobre un índice de
  scroll propio.

### 7. D4: agregación por app y protegidos

D4 se calcula con `DeploySession.Preview(decisions)` y `DeployPlan.Entries`:

- se deduplica por `Path` igual que `resolve`; la primera app gana;
- las columnas son Nuevos, Reemplazos, Híbridos, Sin desplegar y Sin cambios;
- el cero se muestra como `·`; hay fila de totales;
- protegidos = reemplazos que escriben un destino existente + híbridos que
  escriben un destino existente;
- la ruta del holocrón se muestra antes de aplicar; el aviso de cerrar apps va
  en la zona de mensajes.

No se añade una API nueva a `deploy`; `Preview` ya existe y es de solo lectura.

### 8. Operaciones asíncronas

Se envuelven en `tea.Cmd`:

- recuento de archivos de C1;
- planificación y escritura del clon;
- preparación y planificación del despliegue;
- aplicación del despliegue;
- consulta y ejecución de la actualización.

Los comandos llaman a las mismas funciones de dominio y devuelven mensajes de
progreso, resultado o error. La cancelación usa `progress.Cancel`; `Ctrl+C`
antes del punto de no retorno cancela y después se ignora con un `AVISO`. No se
duplica lógica de dominio ni se mueven archivos fuera de los casos de uso.

### 9. Teclas, pie y ayuda

Cada pantalla construye sus `key.Binding` con textos de `internal/texts`.
`Keys()` alimenta el pie (`help.ShortHelpView`) y la ayuda (`help.FullHelpView`).
La ayuda añade los atajos globales, el flujo con el paso actual y el aviso de
secretos en D2/D3. `?` abre la ayuda en todas las pantallas; en campos de texto,
`F1` también. No se escriben leyendas a mano.

### 10. Golden files y pruebas

- Tamaños: 80x24, 100x26, 140x40 y 200x50.
- Variantes: color + Unicode; `NO_COLOR` y ASCII a 100x26; un golden de mensajes
  con `INFO`/`AVISO`/`ERROR`.
- Fixtures deterministas: `Home=/Users/test`, fecha fija, catálogo, planes,
  conflictos y resultados fijos. `View` no toca el disco.
- Test de contrato: todas las pantallas, tamaños y modos; exactamente `h` líneas,
  ninguna supera `w`, sin `\r`, sin tabs, sin truecolor y solo SGR ANSI
  permitidos.
- Modelo: teclas, foco, selección, pila, mensajes asíncronos, cancelación y
  códigos de salida, con HOME temporal.
- D3: benchmark con 20 000 líneas y comprobación de que solo se renderiza la
  ventana visible.

## Risks / Trade-offs

- **Dependencias nuevas y tamaño del binario** → fijar versiones, medir tamaño y
  arranque antes de cerrar; si el coste supera el presupuesto, revisar Path B.
- **Perfil de color y `NO_COLOR` no deterministas en tests** → renderer ANSI
  inyectado, contrato de color y golden de `NO_COLOR`.
- **D3 muestra el archivo completo** → aviso de secretos en mensajes, límite de
  5 MB/20 000 líneas, resolución de archivo entero y pruebas de no persistencia.
- **D3 con 20 000 líneas** → modelo de filas y ventana visible, sin widgets por
  línea; benchmark en CI sin umbral de tiempo frágil.
- **D4 con rutas compartidas o híbridos idénticos** → misma deduplicación que
  `resolve` y pruebas de totales/protegidos.
- **Asincronía y códigos de salida** → envoltorios finos, pruebas de Ctrl+C antes
  y después del no retorno, y de salida en modo menú/directo.
- **Goldens frágiles por ANSI y rutas** → fixtures fijos, `-update`, comparación
  por líneas y una vista sin ANSI para revisión.
- **C2 retira F2/F3/F4** → cambio marcado como BREAKING, footer y ayuda
  actualizados, y prueba de que las tres opciones son la única vía.
- **Menú con cápsulas en la cabecera** → goldens a 24, 26, 30 y 40 filas.
- **`KAMINO_NO_LOGO` fuerza ASCII** → cambio documentado y golden ASCII.

## Migration Plan

1. Añadir dependencias, `theme`, `Frame`, `App` y el harness de goldens; dejar el
   binario compilando.
2. Implementar C1, C3, D3 y D4 como pantallas de referencia, con goldens a los
   cuatro tamaños y variantes.
3. Derivar menú, C2, D0–D2, D5, U1, Configuración, Ayuda, Error y aviso de
   terminal pequeña.
4. Mover recuento, planificación, escritura, despliegue y actualización a
   `tea.Cmd`, con pruebas de cancelación y códigos.
5. Retirar el runner raw, el lector de teclas y el render ANSI antiguo; actualizar
   specs, `docs/Diseño técnico.md`, `docs/Diseño funcional.md` y `AGENTS.md`.
6. Ejecutar `gofmt`, `go vet`, `golangci-lint` y `go test ./... -race`.

No hay migración de datos ni de formato. El rollback es revertir el merge; el
archivo del clon, el manifiesto y los holocrones no cambian.
