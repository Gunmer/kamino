# Spec Delta

## MODIFIED Requirements

### Requirement: Selección de apps para clonar

C1 DEBE (`MUST`) listar solo las apps instaladas con archivos para clonar en una tabla con columnas `SEL`, `APP` y `ARCHIVOS`, con el número de archivos alineado a la derecha, sin marcar ninguna por defecto. Las casillas DEBEN ser `[x]` y `[ ]`; el cursor DEBE ser `▌`. Las flechas DEBEN recorrer las apps, `Espacio` DEBE marcar o desmarcar la app enfocada sin mover el cursor ni sacar el foco de la tabla, y `Enter` DEBE continuar. DEBE ofrecer filtro con `/`, seleccionar todas o ninguna con `a` y `n`, y mostrar el contador `N de M marcadas`, donde `M` son las apps listadas. Sin selección, `Enter` DEBE avisar en la zona de mensajes y no avanzar. Las apps no detectadas y las instaladas sin archivos DEBEN ir cada una en una línea de nota aparte y no ser seleccionables. El pie DEBE mostrar la leyenda de teclas en lugar de botones. El recuento DEBE calcularse en segundo plano con las mismas exclusiones y el mismo confinamiento que el plan.

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

- **GIVEN** C1 con un filtro `/` activo y el cursor sobre una app
- **WHEN** se pulsa `a` o `n`
- **THEN** solo cambian las apps visibles y el contador sigue reflejando la selección total

#### Scenario: App instalada sin archivos

- **GIVEN** una app instalada cuyo recuento es cero
- **WHEN** se muestra C1 con el recuento terminado
- **THEN** no aparece en la tabla, no cuenta en `M` y se informa en la nota «Sin archivos: …» bajo la tabla

#### Scenario: Apps no detectadas

- **GIVEN** apps del catálogo que no están instaladas
- **WHEN** se muestra C1
- **THEN** se informan en la nota «No detectadas: …» y no aparecen como seleccionables

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

#### Scenario: Recuento que retira apps de la lista

- **GIVEN** C1 con el cursor sobre la última app mientras el recuento sigue en curso
- **WHEN** termina el recuento y esa app resulta no tener archivos
- **THEN** la app sale de la tabla, pasa a la nota «Sin archivos» y el cursor queda sobre una fila válida

#### Scenario: --apps omite la selección

- **GIVEN** un clon iniciado con `--apps`
- **WHEN** el valor es válido
- **THEN** C1 se omite y se usan las apps indicadas con todos sus archivos

## ADDED Requirements

### Requirement: Detalle de archivos por app en C1

C1 DEBE (`MUST`) permitir expandir cada app como un árbol en línea que muestra la ruta relativa al HOME y el tamaño de cada archivo que se va a clonar. Las apps DEBEN salir colapsadas. `→` DEBE expandir y `←` colapsar la app enfocada, sin alterar la selección. La lista de apps NO DEBE (`MUST NOT`) cambiar de aspecto mientras ninguna app esté expandida.

#### Scenario: Apps colapsadas por defecto

- **GIVEN** varias apps con archivos
- **WHEN** se abre C1
- **THEN** solo se ven las filas de app y ninguna muestra sus archivos

#### Scenario: Expandir y colapsar

- **GIVEN** el cursor sobre una app colapsada
- **WHEN** se pulsa `→` y después `←`
- **THEN** la app muestra bajo su fila cada archivo con ruta y tamaño y después vuelve a colapsarse, con la selección intacta

#### Scenario: Árbol que no cabe

- **GIVEN** una app expandida con más archivos de los que caben en el body
- **WHEN** se mueve el cursor fuera de la ventana
- **THEN** la vista se desplaza y muestra los indicadores de filas arriba y abajo

### Requirement: Selección de archivos por app

C1 DEBE (`MUST`) permitir marcar o desmarcar cada archivo de una app expandida con `Espacio`. La casilla de la app DEBE ser `[x]` con todos los archivos marcados, `[~]` con solo algunos y `[ ]` sin ninguno; en el último caso la app DEBE tratarse como no seleccionada. Marcar o desmarcar una app DEBE aplicarse a todos sus archivos. La columna `ARCHIVOS` DEBE mostrar `marcados/total` cuando la selección es parcial.

#### Scenario: Selección parcial

- **GIVEN** una app con 12 archivos expandida
- **WHEN** se desmarcan 3 archivos
- **THEN** su casilla es `[~]` y la columna `ARCHIVOS` muestra `9/12`

#### Scenario: Marcar la app completa

- **GIVEN** una app con selección parcial y el cursor sobre su fila
- **WHEN** se pulsa `Espacio`
- **THEN** todos sus archivos quedan marcados y su casilla pasa a `[x]`

#### Scenario: Todos los archivos desmarcados

- **GIVEN** una app expandida con todos sus archivos marcados
- **WHEN** se desmarca cada archivo
- **THEN** su casilla pasa a `[ ]`, no cuenta en `N de M marcadas` y no entra en el clon

### Requirement: Acciones de todas y ninguna según el ámbito

`a` y `n` DEBEN (`MUST`) actuar sobre el ámbito enfocado: con el cursor en una app, sobre las apps visibles según el filtro; con el cursor en un archivo, sobre todos los archivos de esa app. El filtro `/` DEBE seguir buscando solo por id y nombre de app.

#### Scenario: Todas con el cursor en un archivo

- **GIVEN** una app expandida con archivos desmarcados y el cursor en uno de ellos
- **WHEN** se pulsa `a`
- **THEN** todos los archivos de esa app quedan marcados y las demás apps no cambian

#### Scenario: Ninguna con el cursor en un archivo

- **GIVEN** una app expandida con archivos marcados y el cursor en uno de ellos
- **WHEN** se pulsa `n`
- **THEN** todos los archivos de esa app quedan desmarcados y su casilla pasa a `[ ]`

#### Scenario: El filtro no busca en archivos

- **GIVEN** una app cuyo id y nombre no contienen `config` pero que tiene un archivo `config.toml`
- **WHEN** se filtra por `config`
- **THEN** la app no aparece en la tabla

### Requirement: El clon respeta la selección por archivo

El plan, la simulación (`--dry-run`) y el archivo escrito DEBEN (`MUST`) incluir únicamente los archivos marcados en C1 de cada app seleccionada, y el manifiesto NO DEBE (`MUST NOT`) listar los desmarcados. Una app clonada con `--apps`, sin pasar por C1, DEBE incluir todos sus archivos.

#### Scenario: Clon con selección parcial

- **GIVEN** una app con 12 archivos de los que se marcan 9 en C1
- **WHEN** se llega a C3 y se clona
- **THEN** el plan, el manifiesto y el archivo contienen solo esos 9 archivos

#### Scenario: Simulación con selección parcial

- **GIVEN** una selección parcial en C1
- **WHEN** el clon termina en C3 con `--dry-run`
- **THEN** el plan muestra solo los archivos marcados y no se escribe nada

#### Scenario: Clon con --apps

- **GIVEN** `--apps` con una app válida
- **WHEN** se clona
- **THEN** el manifiesto incluye todos los archivos que el plan resuelve para esa app
