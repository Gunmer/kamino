# Design

## Context

C1 (`cloneAppsScreen`) solo conoce un recuento por app. `plan.CountFiles` ya recorre las apps con el mismo `expandApp` que `BuildPlan` en modo `countOnly`, y en ese modo `addFile`/`addLink` ya construyen `Operation{Path, Size, Mode, Link}`; el recorrido se descarta y solo se conserva `count`. `clone.PlanClone` recibe `Selected *[]string` (ids de app), construye el plan completo y deriva el manifiesto de `built.Operations` en `buildManifest`. C3 (`cloneReviewScreen`) y el archivo escrito salen de ese plan. Motivación y alcance: ver `proposal.md`; requisitos: ver `specs/clone-tui/spec.md`.

## Goals / Non-Goals

**Goals:**
- Que C1 conozca las rutas y tamaños de cada app sin un segundo recorrido.
- Que el filtrado por archivo viva en un único lugar del backend y sirva al plan, a `--dry-run` y a la escritura.
- Que la lista de apps se lea igual que hoy mientras no se expanda nada.

**Non-Goals:**
- Opciones de CLI para incluir o excluir archivos; `--apps` sigue clonando todo.
- Búsqueda por ruta de archivo en el filtro `/`.
- Cambios en el formato del manifiesto o del archivo.

## Decisions

**1. `CountFiles` devuelve los archivos por app.** `CountResult` gana `Files map[string][]FileInfo` (`Path`, `Size`, `Link bool`), construido desde `b.operations` al terminar. `Counts` se conserva derivado de `len(Files[id])` para no romper otros usos. Alternativa descartada: una función nueva `ListFiles`; duplicaría el recorrido y la lógica de exclusiones que el spec exige que sean idénticas a las del plan.

**2. Se guardan los archivos incluidos (decisión del usuario).** `appChoice` pasa a tener `Files []fileChoice{Path, Size, Link, Included bool}`. Al llegar `appsCountedMsg` se guardan los archivos de cada app con `Included=false`, porque ninguna app parte marcada. `Espacio` sobre una app incluye todos sus archivos si no estaban todos incluidos y, si lo estaban, los excluye. `Count` es el total y la casilla se deriva de cuántos archivos están incluidos (`[x]`/`[~]`/`[ ]`), de modo que no hay un `Selected` independiente que pueda desincronizarse. `selectedIDs()` devuelve las apps con al menos un archivo incluido.

**3. El filtrado vive en `clone.PlanClone`.** `PlanOptions` gana `Files map[string][]string` (id de app → rutas incluidas). Una app ausente del mapa incluye todos sus archivos (camino de `--apps` y de la CLI); una app presente incluye solo esas rutas. El filtro se aplica sobre `built.Operations` antes de `buildManifest`, así que plan, simulación y escritura comparten una sola fuente. Se hace sobre el plan ya construido, no dentro de `plan.BuildPlan`, para no tocar el recorrido ni sus invariantes de seguridad. Un archivo nuevo que aparezca entre C1 y C3 no estará en la lista y quedará fuera; es coherente con «lo que ves es lo que se clona».

**4. Árbol aplanado en filas.** `View` y la navegación trabajan sobre una lista plana de filas (`appRow` o `fileRow`) calculada a partir de las apps visibles y de qué apps están expandidas. `index` apunta a esa lista y `window(index, count, rows)` se reutiliza tal cual para el desplazamiento. Con todo colapsado la lista coincide con la actual. `toggle`, `a` y `n` miran el tipo de la fila enfocada para decidir el ámbito. El conjunto de apps expandidas es estado de la pantalla, no de `appChoice`.

**5. Apps sin archivos fuera de `visibleApps`.** `visibleApps` excluye `Count == 0` solo cuando el recuento ha terminado (`!counting`). `View` calcula la nota «Sin archivos: …» de las apps detectadas con `Count == 0` y la dibuja bajo «No detectadas». Se eliminan la casilla `[-]` y `C1NoFiles`. Al llegar `appsCountedMsg` se acota `index` y, si la fila enfocada era un archivo de una app que desaparece, se colapsa y vuelve a la fila de app.

**6. Textos y ayuda.** Se añaden `C1NoFilesNote` («Sin archivos: {apps}»), teclas `→/←` («expandir/colapsar») en `cloneAppsKeys` y `HelpC1`, y se ajusta `a/n` a «todas/ninguna del ámbito». Los goldens se regeneran con `-update`.

## Risks / Trade-offs

- [Lista que cambia de tamaño al terminar el recuento] → se acota el cursor y se prueba el caso de que la app enfocada desaparezca.
- [HOME con apps de miles de archivos hacen pesado el estado en memoria] → solo se guardan ruta, tamaño y un booleano por archivo; el recuento ya los recorría. Si resulta problemático, el árbol puede cargar perezosamente, pero no se hace de forma especulativa.
- [Ancho de terminal] → las rutas largas se truncan con `Truncate` por el inicio de la ruta para conservar el nombre del archivo; se verifica con goldens a 80 columnas.
- [Archivos nuevos entre C1 y C3 se omiten] → documentado en la decisión 3; el clon refleja lo marcado.
- [Desincronía entre `Selected` y archivos] → se evita eliminando el estado duplicado (decisión 2).

## Migration Plan

Sin migración: el formato del manifiesto y del archivo no cambia. Se despliega con la versión siguiente; revertir es revertir el commit.
