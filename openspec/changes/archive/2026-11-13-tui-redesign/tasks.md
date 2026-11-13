# Tasks

## 1. Base de presentación

- [x] 1.1 Añadir Bubble Tea, Lip Gloss y `bubbles` (`key`, `help`) a `go.mod`; verificar con `go mod tidy` y `go build ./...`.
- [x] 1.2 Implementar `theme.Tokens` y `theme.Styles` con solo colores ANSI; verificar con tests de tokens, jerarquía y ausencia de hex/RGB/`AdaptiveColor`.
- [x] 1.3 Implementar `Frame` con cabecera, body, mensajes, pie, separadores y guardia de tamaño; verificar con tests de ancho 80/100/140, alto 24/26/30, exactitud de `h` líneas y `NO_COLOR`.
- [x] 1.4 Implementar `App`, `ScreenID`, la interfaz `Screen`, la pila y `SetSize`; verificar con tests de push/pop que conservan cursor y filtro.
- [x] 1.5 Añadir el harness de golden files con `-update` y el test de contrato de límites/ANSI; verificar que una pantalla de prueba no supera `w×h` y solo usa SGR ANSI.
- [x] 1.6 Actualizar `docs/Diseño técnico.md` y `AGENTS.md` con Bubble Tea/Lip Gloss, `Frame`/`Screen` y la referencia a `docs/Diseño tui.md`; verificar que no queda «ANSI propio, sin framework».

## 2. C1 · Apps del clon

- [x] 2.1 Implementar C1 como tabla `SEL | APP | ARCHIVOS`, con contador, casillas `[x]/[ ]/[-]`, cursor `▌`, línea de no detectadas y mensajes; verificar con tests de `Espacio`, `a/n`, filtro, Enter sin selección y app sin archivos.
- [x] 2.2 Añadir goldens de C1 a 80x24, 100x26, 140x40 y 200x50, más `NO_COLOR` y ASCII; verificar con `TestGoldenC1` y el test de contrato.
- [x] 2.3 Actualizar la sección de C1 en `docs/Diseño funcional.md`; verificar que describe la tabla, el contador y la línea de no detectadas.

## 3. C3 · Revisión, progreso y resultado

- [x] 3.1 Implementar C3 como tabla única con filas Apps, Archivos, Tamaño, Ruta y Estado; verificar con tests de revisión, escritura, estado final, código 3, dry-run y copia de ruta.
- [x] 3.2 Mover la barra de progreso a la fila Estado y conservar `d`/`f` en revisión y `d`/`c` al final; verificar con tests de progreso, cancelación y estado final.
- [x] 3.3 Añadir goldens de C3 en sus estados de revisión, escritura, resultado y dry-run; verificar con `TestGoldenC3` y el test de contrato.
- [x] 3.4 Actualizar la sección C3/C4 de `docs/Diseño funcional.md`; verificar que C4 aparece como estado final de C3 y no como pantalla separada.

## 4. D3 · Archivo completo en dos columnas

- [x] 4.1 Implementar el modelo de filas de archivo completo y la alineación de hunks; verificar con tests de CRLF, sin salto final, inserciones, eliminaciones y contexto.
- [x] 4.2 Implementar las dos columnas, números de línea, casillas `●/○`, lado `— sin línea —`, contadores y truncado; verificar con tests de decisión única, línea omitida y línea larga.
- [x] 4.3 Implementar el viewport y los atajos `←→`/`Espacio`, `c/o`, `C/O` y `Tab`/`Mayús+Tab`; verificar con tests de foco, scroll y acciones por línea/bloque/archivo.
- [x] 4.4 Añadir goldens de D3 y un benchmark con 20 000 líneas; verificar que solo se renderiza la ventana visible y que el render no supera `w×h`.
- [x] 4.5 Actualizar la sección D3 de `docs/Diseño funcional.md`; verificar que describe el archivo completo, las dos columnas y el aviso de secretos en mensajes.

## 5. D4 · Confirmación por app

- [x] 5.1 Implementar la agregación con `DeploySession.Preview` y `DeployPlan.Entries`, deduplicando por `Path` como `resolve`; verificar con tests de rutas compartidas, híbridos idénticos y totales.
- [x] 5.2 Implementar la tabla por app con columnas Nuevos, Reemplazos, Híbridos, Sin desplegar y Sin cambios, cero `·`, fila de totales, protegidos, holocrón y acciones horizontales; verificar con tests de Retirada preseleccionada y ausencia de atajo de despliegue.
- [x] 5.3 Añadir goldens de D4 y el aviso de apps abiertas en la zona de mensajes; verificar con `TestGoldenD4` y el test de contrato.
- [x] 5.4 Actualizar la sección D4 de `docs/Diseño funcional.md`; verificar que describe la tabla por app y la fórmula de protegidos.

## 6. C2 · Destino del clon

- [x] 6.1 Implementar las tres opciones de carpeta, el input de «Indicar carpeta» y el input de nombre; verificar con tests de foco, `Enter`, ruta actual, ruta por defecto y nombre editable.
- [x] 6.2 Implementar el diálogo de clon existente con `Sobrescribir`, `Cambiar nombre` y `Volver`, y los flujos `--force`/`--output`; verificar con tests de sufijo libre, reemplazo y precarga inválida.
- [x] 6.3 Añadir goldens de C2 y sus variantes; verificar con `TestGoldenC2` y el test de contrato.
- [x] 6.4 Actualizar la sección C2 de `docs/Diseño funcional.md` y documentar la retirada de `F2/F3/F4`; verificar que la leyenda y la ayuda ya no los muestran.

## 7. Menú y pantallas transversales

- [x] 7.1 Implementar el menú con wordmark en el body, cápsulas en la cabecera, opciones y última misión; verificar con tests de navegación, Escape y ausencia de atajos de una tecla.
- [x] 7.2 Implementar Ayuda, Error común y aviso de terminal pequeña sobre el `Frame` y la zona de mensajes; verificar con tests de `?`/`F1`, cuatro partes del error y conservación de estado al redimensionar.
- [x] 7.3 Añadir goldens del menú a 24/26/30/40 filas, de la ayuda, del error y del aviso de terminal pequeña; verificar con los tests de contrato.
- [x] 7.4 Actualizar las secciones de menú, ayuda y error en `docs/Diseño funcional.md` y `docs/Diseño técnico.md`; verificar que describen la cabecera nueva y la zona de mensajes.

## 8. Pantallas derivadas del despliegue

- [x] 8.1 Implementar D0 como tabla con preview, filtro, ruta manual y mensajes de fuentes; verificar con tests de orden, deduplicación y avisos.
- [x] 8.2 Implementar D1 como tabla de selección con preparación cancelable y mensajes; verificar con tests de selección y apps no instaladas.
- [x] 8.3 Implementar D2 como tabla de conflictos con columnas, cero `·`, scroll y aviso de decisión pendiente; verificar con tests de acciones y bloqueo de confirmación.
- [x] 8.4 Implementar D5 con resumen, holocrón, copia y variante incompleta; verificar con tests de categorías, código 3 y snapshot.
- [x] 8.5 Añadir goldens de D0, D1, D2 y D5; verificar con los tests de contrato.
- [x] 8.6 Actualizar las secciones D0–D2 y D5 de `docs/Diseño funcional.md`; verificar que describen las tablas y los mensajes.

## 9. Configuración con inputs

- [x] 9.1 Implementar los tres inputs editables, el foco, `Tab`/`Mayús+Tab`, `Ctrl+S`, la validación y los mensajes `INFO`/`ERROR`; verificar con tests de edición, guardado y error.
- [x] 9.2 Añadir goldens de Configuración con foco en cada input y con error de validación; verificar con `TestGoldenSettings`.
- [x] 9.3 Actualizar la sección de configuración de `docs/Diseño funcional.md`; verificar que describe los inputs y la zona de mensajes.

## 10. Asincronía y U1

- [x] 10.1 Envolver recuento de C1, planificación/escritura del clon y preparación/planificación/aplicación del despliegue en `tea.Cmd`; verificar con tests de progreso, cancelación y errores.
- [x] 10.2 Implementar U1 con estados consultando, descargando, verificando, reemplazando y resultado; verificar con tests de actualización usando comandos falsos, sin red.
- [x] 10.3 Verificar `Ctrl+C` antes y después del punto de no retorno y los códigos de salida en modo menú y directo; añadir tests de regresión.
- [x] 10.4 Añadir goldens de U1 en consulta, descarga y resultado; verificar con `TestGoldenU1` y el test de contrato.
- [x] 10.5 Actualizar `docs/Diseño técnico.md` con la ejecución asíncrona y la sección U1 de `docs/Diseño funcional.md`; verificar que describen el progreso y la cancelación.

## 11. Integración y limpieza

- [x] 11.1 Retirar el runner raw, el lector de teclas y el render ANSI antiguo; verificar con `grep` que no quedan referencias y con `go build ./...`.
- [x] 11.2 Ejecutar `gofmt -l .` (vacío), `go vet ./...`, `golangci-lint run ./...` y `go test ./... -race`; verificar que todo pasa.
- [x] 11.3 Ejecutar `openspec validate tui-redesign --strict`; verificar que el cambio sigue siendo válido.
- [x] 11.4 Revisar los goldens de todas las pantallas y variantes, incluidos `NO_COLOR`, ASCII y mensajes; verificar que no hay desbordes ni glifos incorrectos.
- [x] 11.5 Verificar que `docs/Diseño técnico.md`, `docs/Diseño funcional.md` y `AGENTS.md` no conservan referencias a «ANSI propio, sin framework» ni a `F2/F3/F4` en C2.

## Workflow follow-up

- Archivar el cambio después de la revisión y de que `openspec validate` y la suite completa pasen.
- Verificar el resultado archivado y que las specs principales incorporan las deltas.
