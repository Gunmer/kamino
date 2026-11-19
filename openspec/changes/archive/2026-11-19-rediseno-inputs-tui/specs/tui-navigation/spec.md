# Spec Delta

## ADDED Requirements

### Requirement: Campo de texto compartido

Todos los campos de entrada de la TUI DEBEN (`MUST`) compartir el mismo render y la misma lógica de edición. Cada campo DEBE ocupar una sola fila con marcador, etiqueta y valor. El foco DEBE ser único por pantalla y DEBE mostrarse con el marcador, el fondo resaltado y el cursor de edición. NO DEBE (`MUST NOT`) existir una implementación de campo distinta por pantalla.

#### Scenario: Fila única por campo

- **GIVEN** una pantalla con varios campos
- **WHEN** se renderiza
- **THEN** cada campo ocupa exactamente una fila con su etiqueta y su valor

#### Scenario: Un solo campo enfocado

- **GIVEN** una pantalla con varios campos
- **WHEN** uno de ellos tiene el foco
- **THEN** solo ese campo muestra marcador, fondo y cursor, y los demás aparecen sin foco

#### Scenario: Campo vacío

- **GIVEN** un campo sin valor
- **WHEN** se renderiza con o sin foco
- **THEN** se muestra un placeholder atenuado y el cursor solo aparece cuando el campo tiene el foco

#### Scenario: Etiquetas alineadas

- **GIVEN** varios campos de una misma pantalla
- **WHEN** se renderizan
- **THEN** sus valores comienzan en la misma columna y las etiquetas se alinean a la izquierda

### Requirement: Edición con cursor

Cada campo DEBE (`MUST`) mantener un cursor como índice de inserción sobre sus caracteres. Escribir DEBE insertar en la posición del cursor, `Retroceso` DEBE borrar el carácter anterior al cursor y `←`/`→` DEBEN moverlo un carácter sin salir del valor. El cursor de edición NO DEBE (`MUST NOT`) mostrarse en campos sin foco ni en pantallas sin campos; el marcador `▌` de selección de las tablas no es un cursor de edición.

#### Scenario: Inserción en medio del texto

- **GIVEN** un campo con el valor `ab cd` y el cursor entre `b` y el espacio
- **WHEN** se escribe `X`
- **THEN** el valor pasa a `abX cd` y el cursor queda después de `X`

#### Scenario: Retroceso en medio del texto

- **GIVEN** un campo con el cursor después de una `X` intermedia
- **WHEN** se pulsa `Retroceso`
- **THEN** se borra esa `X` y no el último carácter del valor

#### Scenario: Flechas laterales

- **GIVEN** un campo con foco y varios caracteres
- **WHEN** se pulsan `←` o `→`
- **THEN** el cursor se mueve un carácter y se detiene al llegar al inicio o al final del valor

#### Scenario: Cursor fuera de foco

- **GIVEN** un campo sin foco o una pantalla sin campos
- **WHEN** se renderiza
- **THEN** no aparece ningún cursor de edición

### Requirement: Desplazamiento horizontal del valor

Cuando el valor no cabe en el ancho disponible, el campo DEBE (`MUST`) desplazarlo horizontalmente para mantener visible el cursor. DEBE mostrar `…` al inicio cuando oculte el comienzo del valor y `…` al final cuando oculte su final. NO DEBE (`MUST NOT`) partir caracteres UTF-8 ni caracteres de doble ancho.

#### Scenario: Ruta larga con cursor al final

- **GIVEN** una ruta más ancha que el campo y el cursor al final del valor
- **WHEN** se renderiza con foco
- **THEN** el comienzo se abrevia con `…`, el final de la ruta queda visible y el cursor sigue visible

#### Scenario: Cursor en medio de una ruta larga

- **GIVEN** una ruta más ancha que el campo y el cursor en una posición intermedia
- **WHEN** se renderiza con foco
- **THEN** el campo ajusta la ventana para mantener visible el cursor y señala con `…` cada lado oculto

#### Scenario: Caracteres anchos y UTF-8

- **GIVEN** un valor con caracteres de doble ancho o multibyte
- **WHEN** el campo calcula su ventana visible
- **THEN** no parte ningún carácter y la suma de columnas no supera el ancho disponible

#### Scenario: Valor que cabe completo

- **GIVEN** un valor que cabe en el ancho disponible
- **WHEN** se renderiza
- **THEN** se muestra completo, sin elipsis ni desplazamiento

### Requirement: Error de validación en campos

Un error de validación DEBE (`MUST`) mostrarse en la línea inferior de mensajes con icono, palabra y color de error, y DEBE marcar el campo afectado sin añadir ni desplazar filas de campos. Al corregir el valor, el error DEBE desaparecer.

#### Scenario: Error visible sin mover las filas

- **GIVEN** un campo con un valor inválido
- **WHEN** se renderiza la pantalla
- **THEN** la zona de mensajes muestra un `ERROR` con icono y palabra, el campo se marca como inválido y las filas de los demás campos no cambian de posición

#### Scenario: Error corregido

- **GIVEN** un campo marcado como inválido
- **WHEN** se corrige el valor y se vuelve a validar
- **THEN** el campo deja de estar marcado y el mensaje de error desaparece

#### Scenario: Varios campos

- **GIVEN** una pantalla con varios campos y un error de validación
- **WHEN** se muestra el error
- **THEN** solo se marca el campo afectado y los demás conservan su estado de foco o de campo inactivo

### Requirement: Pie de atajos y ayuda desde un campo

Con un campo enfocado, el pie DEBE (`MUST`) mostrar las teclas activas de esa pantalla, incluidos `Enter`, `Esc` y la ayuda, generadas desde la misma lista de atajos que usa la pantalla. `F1` y `?` DEBEN (`MUST`) abrir la ayuda sin insertar caracteres en el valor. Al cerrar la ayuda, el campo DEBE conservar su valor, su foco y su cursor.

#### Scenario: Pie con campo enfocado

- **GIVEN** una pantalla con un campo enfocado
- **WHEN** se muestra el pie
- **THEN** aparecen las teclas de aceptar, volver y ayuda correspondientes a esa pantalla

#### Scenario: F1 desde un campo

- **GIVEN** un campo con el foco y un valor escrito
- **WHEN** se pulsa `F1`
- **THEN** se abre la ayuda y el valor no cambia

#### Scenario: Interrogación desde un campo

- **GIVEN** un campo con el foco y un valor escrito
- **WHEN** se pulsa `?`
- **THEN** se abre la ayuda y el valor no cambia

#### Scenario: Volver de la ayuda

- **GIVEN** la ayuda abierta desde un campo con foco
- **WHEN** se cierra la ayuda
- **THEN** el campo recupera el foco y el cursor en la misma posición

### Requirement: Foco único y repintado limpio

Cada pantalla DEBE (`MUST`) tener como máximo un campo enfocado. Al mover el foco, la fila anterior DEBE perder el marcador, el fondo y el cursor, y la nueva DEBE mostrarlos. Al cambiar de pantalla NO DEBE (`MUST NOT`) quedar ningún resto de foco de la pantalla anterior. Las filas de campo DEBEN caber en 80x24 sin añadir filas adicionales por campo.

#### Scenario: Cambio de foco

- **GIVEN** una pantalla con dos campos y el foco en el primero
- **WHEN** el foco pasa al segundo
- **THEN** el primer campo pierde el marcador, el fondo y el cursor y el segundo los gana

#### Scenario: Cambio de pantalla

- **GIVEN** una pantalla con un campo enfocado
- **WHEN** se abre otra pantalla
- **THEN** la pantalla nueva se repinta completa y no conserva marcador, fondo ni cursor del campo anterior

#### Scenario: Presupuesto a 80x24

- **GIVEN** una terminal de 80x24
- **WHEN** se muestra una pantalla con varios campos
- **THEN** cada campo ocupa una sola fila y la pantalla cabe sin desplazar el pie ni desbordar el body
