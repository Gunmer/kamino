# clone-tui Specification

## Purpose

Definir el flujo interactivo de clonación: elegir apps instaladas, fijar el destino, revisar el plan, ejecutarlo con progreso y cancelación, y presentar el resumen final.

## Requirements

### Requirement: Selección de apps para clonar

C1 DEBE (`MUST`) listar solo las apps instaladas, con su identificador y el número de archivos, sin marcar ninguna por defecto. Las flechas DEBEN recorrer las apps, `Espacio` DEBE marcar o desmarcar la app enfocada sin mover el cursor ni sacar el foco de la tabla, y `Enter` DEBE continuar. DEBE ofrecer filtro con `/`, seleccionar todas o ninguna con `a` y `n` sobre las filas visibles, y mostrar un contador; sin selección, `Enter` DEBE avisar y no avanzar. El pie DEBE mostrar la leyenda de teclas en lugar de botones. El recuento DEBE calcularse en segundo plano con las mismas exclusiones y el mismo confinamiento que el plan.

#### Scenario: Lista inicial sin selección

- **GIVEN** varias apps instaladas con archivos
- **WHEN** se abre C1
- **THEN** se muestran todas con su identificador y número de archivos, y ninguna está marcada

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
- **THEN** solo cambian las filas visibles y el contador refleja la selección total

#### Scenario: App instalada sin archivos

- **GIVEN** una app instalada cuyo recuento es cero
- **WHEN** se muestra C1
- **THEN** su casilla aparece deshabilitada y no se puede seleccionar

#### Scenario: Apps no detectadas

- **GIVEN** apps del catálogo que no están instaladas
- **WHEN** se muestra C1
- **THEN** se informa en una línea aparte y no aparecen como seleccionables

#### Scenario: Recuento en segundo plano

- **GIVEN** un HOME con muchos archivos
- **WHEN** se abre C1
- **THEN** la pantalla sigue respondiendo mientras el recuento termina y el resultado usa las mismas exclusiones que el plan

#### Scenario: --apps omite la selección

- **GIVEN** un clon iniciado con `--apps`
- **WHEN** el valor es válido
- **THEN** C1 se omite y se usan las apps indicadas

### Requirement: Destino del clon

C2 DEBE (`MUST`) proponer `clone-yyyyMMdd.tar.gz` con la fecha local, resolver la ruta a absoluta y validar en vivo. `Enter` DEBE continuar, `Esc` DEBE volver y `F2`, `F3` y `F4` DEBEN usar el directorio actual, Descargas y Escritorio. Si el clon existe, DEBE mostrar un diálogo con ruta, fecha y tamaño, con las opciones Volver, Otro nombre y Reemplazar clon navegables con flechas y `Enter`, y con Volver preseleccionada; Otro nombre DEBE proponer el primer sufijo libre. El destino NO DEBE crearse hasta C3 y NO DEBE limitarse al HOME.

#### Scenario: Nombre por defecto

- **GIVEN** una fecha local y un directorio actual sin `clone-yyyyMMdd.tar.gz`
- **WHEN** se abre C2
- **THEN** se propone `clone-yyyyMMdd.tar.gz` con la fecha local y la ruta absoluta resuelta

#### Scenario: Validación en vivo

- **GIVEN** un nombre o una ruta inválida
- **WHEN** se escribe en C2
- **THEN** se muestra el error sin crear el archivo ni el directorio

#### Scenario: Atajos de directorio

- **WHEN** se elige el directorio actual, Descargas o Escritorio
- **THEN** la ruta del destino se actualiza a la ubicación correspondiente

#### Scenario: Clon existente

- **GIVEN** un clon con el mismo nombre en el destino
- **WHEN** se intenta continuar
- **THEN** se muestra el diálogo con ruta, fecha, tamaño, Volver preseleccionada y las tres acciones

#### Scenario: Otro nombre propone sufijo

- **GIVEN** el diálogo de clon existente abierto
- **WHEN** se elige Otro nombre
- **THEN** se propone el primer sufijo libre para el mismo nombre base

#### Scenario: Reemplazo forzado

- **GIVEN** un clon iniciado con `--force` y un destino existente
- **WHEN** se llega a C2
- **THEN** no se muestra el diálogo y se avisa de que el clon existente se reemplazará al final

#### Scenario: --output omite el destino

- **GIVEN** un clon iniciado con `--output`
- **WHEN** el valor es válido
- **THEN** C2 se omite y se usa esa ruta

#### Scenario: --output inválido

- **GIVEN** un clon iniciado con `--output` y un valor inválido
- **WHEN** se inicia el flujo con terminal interactiva
- **THEN** C2 se abre precargada con el error

### Requirement: Revisión del plan de clonación

C3 DEBE (`MUST`) pasar por tres estados: planificando, revisión y en curso. En revisión DEBE mostrar apps, archivos, tamaño y destino; `d` DEBE mostrar los omitidos con su motivo; `f` DEBE mostrar la lista de archivos incluidos agrupada por app y solo rutas; `Enter` DEBE iniciar la clonación cuando el plan esté listo; y DEBE advertirse que el clon no está cifrado y puede contener secretos. Con `--dry-run`, C3 DEBE terminar en la revisión sin acción de clonar y `Enter` o `Esc` DEBEN cerrar la simulación sin escribir.

#### Scenario: Revisión con datos del plan

- **GIVEN** un plan calculado con apps y archivos
- **WHEN** se muestra C3
- **THEN** se muestran apps, archivos, tamaño, destino y el aviso de contenido no cifrado

#### Scenario: Omitidos con motivo

- **GIVEN** un plan con rutas omitidas
- **WHEN** se pulsa `d`
- **THEN** se muestra cada ruta omitida con su motivo

#### Scenario: Lista de archivos incluidos

- **GIVEN** un plan con archivos incluidos
- **WHEN** se pulsa `f`
- **THEN** se muestran agrupados por app y solo las rutas

#### Scenario: Dry-run termina en revisión

- **GIVEN** una clonación con `--dry-run` y terminal interactiva
- **WHEN** termina la planificación
- **THEN** C3 muestra la revisión sin acción de clonar, la leyenda ofrece terminar la simulación y no se escribe nada

### Requirement: Progreso y cancelación de la clonación

En curso, C3 DEBE (`MUST`) mostrar una barra de progreso por archivos. La operación DEBE poder cancelarse hasta el renombrado final; al cancelar, el temporal DEBE borrarse y el clon anterior DEBE quedar intacto. Una vez iniciado el renombrado final, la cancelación NO DEBE interrumpirlo.

#### Scenario: Progreso por archivos

- **GIVEN** una clonación en curso con varios archivos
- **WHEN** se escribe cada archivo
- **THEN** la barra avanza y muestra el archivo actual

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

### Requirement: Resumen de la clonación

C4 DEBE (`MUST`) mostrar apps, archivos, tamaño y ruta del clon; `d` DEBE mostrar los omitidos; la línea de salida DEBE aparecer solo si el código es 3; `c` DEBE copiar la ruta con OSC 52 y avisar si el terminal no lo soporta; `Enter` o `Esc` DEBEN continuar. NO DEBE existir un atajo para abrir el Finder.

#### Scenario: Resumen completo

- **GIVEN** una clonación completada
- **WHEN** se muestra C4
- **THEN** se muestran apps, archivos, tamaño y ruta, sin línea de salida si el código es 0

#### Scenario: Resumen con omitidos

- **GIVEN** una clonación con código 3
- **WHEN** se muestra C4
- **THEN** se muestra la línea de salida y `d` permite consultar los omitidos

#### Scenario: Copiar la ruta

- **GIVEN** C4 abierto
- **WHEN** se pulsa `c`
- **THEN** se intenta copiar la ruta con OSC 52 y se avisa si el terminal no lo soporta
