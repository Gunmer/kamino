# Tasks

## 1. Componente de campo

- [x] 1.1 Añadir a `theme.Styles` los estilos `FieldFocus`, `FieldIdle`, `FieldLabelFocus`, `FieldLabelIdle`, `FieldError` y `FieldCaret`, con fondo ANSI 8 en `ansi-dark`, ANSI 7 en `ansi-light`, acento para el valor y `Reverse(true)` para el cursor; verificar con tests del tema que solo se emiten secuencias ANSI, que `NO_COLOR` no emite color ni fondo y que no aparece truecolor.
- [x] 1.2 Crear `internal/tui/field.go` con el tipo `field` (`Label`, `Value`, `Placeholder`, `Cursor`, `Invalid`) y `SetValue`, `Insert`, `Backspace` y `MoveCursor`; verificar con tests de inserción, retroceso, límites y UTF-8 que el cursor es un índice de runas y nunca sale del valor.
- [x] 1.3 Implementar `field.Render` con una sola fila: marcador `▌`, etiqueta alineada, valor, foco, placeholder, error y ausencia de cursor sin foco; verificar con tests de render de campo enfocado, inactivo, vacío y con error.
- [x] 1.4 Implementar la ventana horizontal pura del valor con `…` al inicio y al final cuando corresponda, sin partir runas anchas y manteniendo visible el cursor; verificar con tests de cursor al inicio, en medio y al final, rutas largas y caracteres de doble ancho.
- [x] 1.5 Implementar `fieldOptions` y el cálculo de `labelWidth`, incluyendo el ancho extra de «editando» con `NO_COLOR`; verificar que las etiquetas y los valores quedan alineados en una pantalla con etiquetas de distinta longitud.
- [x] 1.6 Añadir `fieldHelpKey()` y el binding compartido de cursor en `keys.go`; verificar que el pie muestra `Enter`, `Esc` y `F1/?` sin duplicar `? ayuda` y que la ayuda conserva su lista global.

## 2. Integración por pantalla

- [x] 2.1 Migrar el filtro de C1 a `filter field`: `/` abre con el cursor al final, escribir edita en el cursor, `Enter` aplica y devuelve el foco a la tabla y `Esc` limpia y cierra; verificar con tests de apertura, edición en medio, aplicación, cierre y ayuda.
- [x] 2.2 Migrar `path` y `name` de C2 a campos «Destino» y «Nombre», eliminar `insert`/`deleteRune` y conectar `←`/`→` al cursor; verificar con tests de foco, edición en medio, placeholder, ruta larga, error y transiciones de `Enter`, y comprobar que el diálogo de clon existente no cambia.
- [x] 2.3 Migrar el filtro y la ruta manual de D0 a campos «Filtro» y «Ruta», con `Enter` de validación y `Esc` de vuelta a la tabla; verificar con tests de apertura, edición, ruta válida, ruta inválida que conserva el valor, ayuda y ruta larga.
- [x] 2.4 Migrar los cuatro valores de Configuración a `fields [fieldCount]field` con etiquetas «Catálogo», «Clones», «Holocrones» y «Límite»; verificar con tests de foco inicial, `Tab`/`Mayús+Tab`, flechas verticales, `←`/`→`, límite solo con dígitos, `Ctrl+S`, `Esc` y error sin mover filas, usando siempre un HOME temporal para guardar.
- [x] 2.5 Actualizar los `Keys()` de C1, C2, D0 y Configuración para que, con un campo enfocado, devuelvan aceptar, volver, cursor, campo y `F1/?` según corresponda; verificar con tests de pie y de ayuda que las leyendas salen de la misma lista de atajos.
- [x] 2.6 Regenerar y revisar los goldens de C2, D0 y Configuración, y añadir casos de C1 con filtro, C2 con foco en ruta y nombre, D0 con filtro y ruta manual, y Configuración vacía; verificar con `go test ./internal/tui -run Golden -update` y después sin `-update`.

## 3. Textos

- [x] 3.1 Añadir a `internal/texts` las etiquetas «Destino», «Nombre», «Filtro», «Ruta», «Límite», los placeholders, «editando» y las descripciones de atajo del cursor; verificar que las pantallas no contienen texto visible hardcodeado y que `go build ./...` pasa.
- [x] 3.2 Retirar o reutilizar `C1FilterLine` y las pistas por campo que dejen de mostrarse, y comprobar que no quedan referencias con `grep` ni símbolos muertos con `go vet ./...`.
- [x] 3.3 Revisar que la ayuda y el pie usan las mismas constantes de `internal/texts` y que `F1`/`?` siguen abriendo la ayuda sin insertar caracteres; verificar con los tests de ayuda y de campo.

## 4. Pruebas de render y contrato

- [x] 4.1 Añadir tests de render para foco y sin foco, cursor en medio del texto, ruta larga con scroll, vacío con placeholder, error, `NO_COLOR` y ASCII; verificar que cada caso comprueba el marcador, el estilo del cursor, la elipsis y la ausencia de color cuando corresponde.
- [x] 4.2 Añadir un test que cuente un solo cursor de edición por pantalla y que no haya cursor en tablas, diálogos, ayuda ni pantallas sin campos; verificar que el marcador `▌` de selección de tablas no se confunde con el cursor de texto.
- [x] 4.3 Añadir el caso de Configuración y C2 a 80x24 al test de contrato y a los goldens; verificar que el render tiene exactamente 24 líneas, ninguna supera 80 columnas, no hay truecolor, `NO_COLOR` no emite colores y el test rechaza también fondos ANSI 40–47 y 100–107 en monocromo.
- [x] 4.4 Ejecutar `go test ./internal/tui -race` y corregir cualquier regresión de foco, edición, scroll o goldens antes de continuar.

## 5. Documentación

- [x] 5.1 Actualizar la sección de la TUI y la de Configuración de `README.md` con el campo de una fila, el foco, el cursor por vídeo inverso, el scroll, `NO_COLOR` y ASCII; verificar que el texto coincide con el comportamiento implementado.
- [x] 5.2 Actualizar `docs/diseno-funcional.md` en la introducción de la TUI, C1 (filtro), C2 (destino y nombre), D0 (filtro y ruta manual) y Configuración; verificar que describe una fila por campo y el error en la zona de mensajes.
- [x] 5.3 Actualizar `docs/diseno-tecnico.md` con el componente `field`, la decisión de no usar el cursor real de Bubble Tea, el fallback de vídeo inverso, el algoritmo de scroll y las pruebas añadidas.
- [x] 5.4 Actualizar `docs/diseno-tui.md`: sustituir las maquetas de inputs por la fila única, documentar foco, cursor, placeholder y scroll, y corregir la nota que dice que `?` se inserta como carácter para que refleje que `?` y `F1` abren la ayuda.
- [x] 5.5 Revisar que README y los tres documentos no se contradicen entre sí ni con las specs delta; verificar con una búsqueda de `_`, `F2/F3/F4`, `? se inserta` y descripciones antiguas de los inputs.

## 6. Verificación final

- [x] 6.1 Ejecutar `gofmt -l .` y confirmar que no devuelve archivos.
- [x] 6.2 Ejecutar `go vet ./...` y `golangci-lint run ./...` si está instalado; corregir cualquier aviso introducido por el cambio.
- [x] 6.3 Ejecutar `go test ./... -race` y confirmar que pasan todas las pruebas, incluidas las de HOME temporal y los goldens.
- [x] 6.4 Revisar el diff para confirmar que no cambiaron la CLI, los códigos de salida, el formato del clon, el manifiesto ni las rutas de seguridad.

## Workflow follow-up

- Revisar los artefactos de este change antes de ejecutar `/opsx-apply`.
- Archivar el change después de implementar y verificar, cuando el proyecto lo requiera.
