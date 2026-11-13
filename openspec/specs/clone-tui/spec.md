# clone-tui Specification

## Purpose

Definir el flujo interactivo de clonación: elegir apps instaladas, fijar el destino, revisar el plan, ejecutarlo con progreso y cancelación, y presentar el resumen final.

## Requirements

### Requirement: Selección de apps para clonar

C1 DEBE (`MUST`) listar solo las apps instaladas en una tabla con columnas `SEL`, `APP` y `ARCHIVOS`, con el número de archivos alineado a la derecha, sin marcar ninguna por defecto. Las casillas DEBEN ser `[x]`, `[ ]` y `[-]` para las apps sin archivos; el cursor DEBE ser `▌`. Las flechas DEBEN recorrer las apps, `Espacio` DEBE marcar o desmarcar la app enfocada sin mover el cursor ni sacar el foco de la tabla, y `Enter` DEBE continuar. DEBE ofrecer filtro con `/`, seleccionar todas o ninguna con `a` y `n` sobre las filas visibles, y mostrar el contador `N de M marcadas`. Sin selección, `Enter` DEBE avisar en la zona de mensajes y no avanzar. Las apps no detectadas DEBEN ir en una línea aparte y no ser seleccionables. El pie DEBE mostrar la leyenda de teclas en lugar de botones. El recuento DEBE calcularse en segundo plano con las mismas exclusiones y el mismo confinamiento que el plan.

#### Scenario: Lista inicial sin selección

- **GIVEN** varias apps instaladas con archivos
- **WHEN** se abre C1
- **THEN** se muestran en una tabla con columnas `SEL`, `APP` y `ARCHIVOS`, con los conteos alineados a la derecha y ninguna app marcada

#### Scenario: Marcar sin perder el foco

- **GIVEN** C1 con el cursor sobre una app
- **WHEN** se pulsa `Espacio`
- **THEN** la app se marca o desmarca y el cursor y el foco permanecen en esa misma fila

#### Scenario: Enter continúa

- **GIVEN** C1 con al menos una app seleccionada
- **WHEN** se pulsa `Enter`
- **THEN** se avanza al destino o a la revisión según el flujo

#### Scenario: Filtro y acciones sobre filas visibles

- **GIVEN** C1 con un filtro `/` activo
- **WHEN** se pulsa `a` o `n`
- **THEN** solo cambian las filas visibles y el contador sigue reflejando la selección total

#### Scenario: App instalada sin archivos

- **GIVEN** una app instalada cuyo recuento es cero
- **WHEN** se muestra C1
- **THEN** su casilla aparece como `[-]`, deshabilitada, y no se puede seleccionar

#### Scenario: Apps no detectadas

- **GIVEN** apps del catálogo que no están instaladas
- **WHEN** se muestra C1
- **THEN** se informan en una línea aparte y no aparecen como seleccionables

#### Scenario: Aviso sin selección

- **GIVEN** C1 sin apps seleccionadas
- **WHEN** se pulsa `Enter`
- **THEN** la zona de mensajes muestra un `AVISO` y no se avanza

#### Scenario: Lista desplazada

- **GIVEN** más apps de las que caben en el body
- **WHEN** se mueve el cursor fuera de la ventana
- **THEN** la tabla se desplaza y muestra el indicador de filas arriba y abajo

#### Scenario: Recuento en segundo plano

- **GIVEN** un HOME con muchos archivos
- **WHEN** se abre C1
- **THEN** la pantalla sigue respondiendo mientras el recuento termina y el resultado usa las mismas exclusiones que el plan

#### Scenario: --apps omite la selección

- **GIVEN** un clon iniciado con `--apps`
- **WHEN** el valor es válido
- **THEN** C1 se omite y se usan las apps indicadas

### Requirement: Destino del clon

C2 DEBE (`MUST`) ofrecer tres opciones de carpeta: vault por defecto (o `clone.output_dir` si está configurado), carpeta actual (`pwd`) e indicar carpeta. «Indicar carpeta» DEBE convertirse en un input para la ruta; debajo DEBE mostrarse un input para el nombre, precargado con `clone-yyyyMMdd.tar.gz` y editable. `Enter` en el nombre DEBE fijar el destino; si el archivo existe, DEBE abrir un diálogo con ruta, fecha y tamaño y las opciones Sobrescribir, Cambiar nombre y Volver, con Volver preseleccionada. Cambiar nombre DEBE enfocar el nombre y proponer el primer sufijo libre. El destino NO DEBE (`MUST NOT`) crearse hasta que empiece la clonación y NO DEBE limitarse al HOME. Si el directorio no existe, C2 DEBE aceptarlo y la clonación DEBE crearlo al empezar. `F2`, `F3` y `F4` se retiran; las tres opciones los sustituyen.

#### Scenario: Nombre por defecto

- **GIVEN** una fecha local, `~/.config/kamino/vault/` existente o ausente y la TUI sin `clone.output_dir` configurado
- **WHEN** se abre C2
- **THEN** la opción «Vault por defecto» está seleccionada, muestra `~/.config/kamino/vault/`, y el nombre propone `clone-20261113.tar.gz`

#### Scenario: Directorio configurado en la TUI

- **GIVEN** `clone.output_dir = "~/Descargas"` en la configuración de la TUI
- **WHEN** se abre C2
- **THEN** la opción «Vault por defecto» muestra `~/Descargas` y el nombre propone `clone-20261113.tar.gz`

#### Scenario: Carpeta actual

- **GIVEN** C2 abierta
- **WHEN** se elige «Carpeta actual»
- **THEN** la ruta del destino usa el directorio de trabajo actual (`pwd`) y el nombre conserva la sugerencia

#### Scenario: Indicar carpeta

- **GIVEN** C2 abierta
- **WHEN** se elige «Indicar carpeta»
- **THEN** la opción se convierte en un input para la ruta y el foco pasa a ese input

#### Scenario: Nombre editable

- **GIVEN** C2 con una carpeta elegida
- **WHEN** se edita el input del nombre
- **THEN** la ruta completa se actualiza y el nombre puede confirmarse con `Enter`

#### Scenario: Directorio por defecto ausente

- **GIVEN** `~/.config/kamino/vault/` no existe y la TUI no tiene `clone.output_dir` configurado
- **WHEN** se abre C2 y se continúa a la revisión
- **THEN** C2 no muestra un error por el directorio ausente y la clonación crea el directorio al empezar a escribir

#### Scenario: Directorio configurado ausente

- **GIVEN** `clone.output_dir` apunta a un directorio que no existe
- **WHEN** se abre C2 y se continúa a la revisión
- **THEN** C2 no muestra un error por el directorio ausente y la clonación crea el directorio configurado al empezar a escribir

#### Scenario: Validación en vivo

- **GIVEN** una carpeta o un nombre escrito manualmente en C2 que no es válido
- **WHEN** se escribe en C2
- **THEN** se muestra un `ERROR` en la zona de mensajes sin crear el archivo ni el directorio

#### Scenario: Clon existente

- **GIVEN** un clon con el mismo nombre en el destino
- **WHEN** se pulsa `Enter` en el nombre
- **THEN** se muestra el diálogo con ruta, fecha, tamaño, Volver preseleccionada y las tres acciones

#### Scenario: Sobrescribir

- **GIVEN** el diálogo de clon existente abierto
- **WHEN** se elige Sobrescribir
- **THEN** se confirma el reemplazo y se continúa a la revisión sin escribir todavía

#### Scenario: Otro nombre propone sufijo

- **GIVEN** el diálogo de clon existente abierto
- **WHEN** se elige Cambiar nombre
- **THEN** el foco vuelve al input del nombre y se propone el primer sufijo libre para el mismo nombre base

#### Scenario: Volver desde el diálogo

- **GIVEN** el diálogo de clon existente abierto
- **WHEN** se elige Volver o se pulsa `Esc`
- **THEN** el diálogo se cierra y el foco vuelve al input del nombre

#### Scenario: Reemplazo forzado

- **GIVEN** un clon iniciado con `--force` y un destino existente
- **WHEN** se llega a C2
- **THEN** no se muestra el diálogo y la zona de mensajes avisa de que el clon existente se reemplazará

#### Scenario: --output omite el destino

- **GIVEN** un clon iniciado con `--output`
- **WHEN** el valor es válido
- **THEN** C2 se omite y se usa esa ruta

#### Scenario: --output inválido

- **GIVEN** un clon iniciado con `--output` y un valor inválido
- **WHEN** se inicia el flujo con terminal interactiva
- **THEN** C2 se abre con «Indicar carpeta», los inputs precargados y un `ERROR` en la zona de mensajes

#### Scenario: Atajos de directorio

- **GIVEN** C2 abierta
- **WHEN** se elige una de las tres opciones de carpeta
- **THEN** la ruta del destino se actualiza a la opción elegida

#### Scenario: Sin atajos F2, F3 y F4

- **GIVEN** C2 abierta
- **WHEN** se pulsa `F2`, `F3` o `F4`
- **THEN** no se cambia la carpeta y las tres opciones siguen siendo la única forma de elegirla

### Requirement: Revisión del plan de clonación

C3 DEBE (`MUST`) usar una tabla única con filas Apps, Archivos, Tamaño, Ruta y Estado. La fila Estado DEBE mostrar Planificando, Listo para clonar, Escribiendo con barra de progreso, Completado, Simulación completada o Cancelado antes de escribir. En revisión DEBE permitir `d` para omitidos y `f` para archivos incluidos; `Enter` DEBE iniciar la clonación cuando el plan esté listo; el aviso de secretos DEBE ir en la zona de mensajes. En el estado final DEBE mostrar Completado, permitir `d` para omitidos y `c` para copiar la ruta, y `Enter`/`Esc` continuar. El código 3 DEBE mostrarse como `AVISO`. El header DEBE cambiar de Revisión a Resultado al terminar y conservar `Paso 3 de 3`. Con `--dry-run`, C3 DEBE terminar en Resultado con `Simulación completada` y no escribir.

#### Scenario: Revisión con datos del plan

- **GIVEN** un plan calculado con apps y archivos
- **WHEN** se muestra C3
- **THEN** la tabla muestra Apps, Archivos, Tamaño, Ruta y Estado `Listo para clonar`, y la zona de mensajes muestra el aviso de contenido no cifrado

#### Scenario: Omitidos con motivo

- **GIVEN** un plan con rutas omitidas
- **WHEN** se pulsa `d`
- **THEN** se muestra cada ruta omitida con su motivo

#### Scenario: Lista de archivos incluidos

- **GIVEN** un plan con archivos incluidos
- **WHEN** se pulsa `f`
- **THEN** se muestran agrupados por app y solo las rutas

#### Scenario: Escritura en la misma tabla

- **GIVEN** una clonación en curso
- **WHEN** se escribe cada archivo
- **THEN** la fila Estado muestra el progreso y la barra avanza dentro de la misma tabla

#### Scenario: Estado final

- **GIVEN** una clonación completada
- **WHEN** se muestra C3
- **THEN** la fila Estado muestra Completado y el header muestra `Resultado` con `Paso 3 de 3`

#### Scenario: Resumen con omitidos

- **GIVEN** una clonación con código 3
- **WHEN** se muestra el estado final
- **THEN** la zona de mensajes muestra un `AVISO` con el código 3 y `d` permite consultar los omitidos

#### Scenario: Copiar la ruta

- **GIVEN** C3 en el estado final con un clon creado
- **WHEN** se pulsa `c`
- **THEN** se intenta copiar la ruta con OSC 52 y se avisa del resultado en la zona de mensajes

#### Scenario: Dry-run termina en revisión

- **GIVEN** una clonación con `--dry-run` y terminal interactiva
- **WHEN** termina la planificación
- **THEN** C3 muestra `Simulación completada` en la fila Estado, no ofrece copiar la ruta y `Enter` o `Esc` cierran sin escribir

### Requirement: Progreso y cancelación de la clonación

En curso, C3 DEBE (`MUST`) mostrar la barra de progreso en la fila Estado de la misma tabla. La operación DEBE poder cancelarse hasta el renombrado final; al cancelar, el temporal DEBE borrarse y el clon anterior DEBE quedar intacto. Una vez iniciado el renombrado final, la cancelación NO DEBE interrumpirlo.

#### Scenario: Progreso por archivos

- **GIVEN** una clonación en curso con varios archivos
- **WHEN** se escribe cada archivo
- **THEN** la fila Estado muestra el archivo actual y la barra avanza

#### Scenario: Cancelar antes del renombrado

- **GIVEN** una clonación en curso antes del renombrado final
- **WHEN** se cancela
- **THEN** el temporal se borra y el clon anterior sigue intacto

#### Scenario: Cancelar durante el renombrado

- **GIVEN** una clonación que ya inició el renombrado final
- **WHEN** se intenta cancelar
- **THEN** la operación termina y la pantalla indica que ya no se puede cancelar

#### Scenario: Error de escritura

- **GIVEN** un fallo al escribir el temporal
- **WHEN** la clonación termina con error
- **THEN** el temporal se elimina y el clon anterior sigue intacto
