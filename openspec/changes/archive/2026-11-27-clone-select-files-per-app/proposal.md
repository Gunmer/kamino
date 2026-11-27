# Proposal

## Why

En C1 solo se elige por app: el usuario ve un número de archivos pero no cuáles son, y no puede dejar fuera uno concreto (una configuración que no quiere llevarse, un archivo grande). Además, las apps instaladas sin archivos para clonar ocupan filas deshabilitadas `[-]` que no aportan nada y saturan la lista.

## What Changes

- C1 muestra el detalle de los archivos de cada app como un árbol en línea: las apps salen colapsadas y `→`/`←` expanden o colapsan la app enfocada. Cada archivo muestra su ruta relativa al HOME y su tamaño.
- Se puede marcar o desmarcar la app completa o cada archivo uno a uno con `Espacio`. La casilla de la app pasa a tener tres estados: `[x]` todo, `[~]` parcial y `[ ]` nada. La columna ARCHIVOS muestra `marcados/total` (`9/12`) cuando la selección es parcial.
- `a` y `n` actúan sobre el ámbito enfocado: con el cursor en una app, sobre las apps visibles; con el cursor en un archivo, sobre los archivos de esa app.
- Una app con todos sus archivos desmarcados queda desmarcada: no entra en el manifiesto ni cuenta en `N de M marcadas`.
- El clon (C3, `--dry-run` y escritura) solo incluye los archivos marcados.
- Las apps instaladas sin archivos para clonar dejan de mostrarse en la tabla y se informan en una nota «Sin archivos: …» bajo la tabla, junto a la de «No detectadas». Se elimina la casilla `[-]` y el aviso `C1NoFiles`. El contador `N de M marcadas` cuenta solo las apps listadas.
- El filtro `/` sigue buscando solo por id y nombre de app.
- Fuera de alcance: la CLI no cambia. Con `--apps` se omite C1 y se clonan todos los archivos de esas apps; no se añade ninguna opción para incluir o excluir archivos.

## Capabilities

### New Capabilities

Ninguna.

### Modified Capabilities

- `clone-tui`: cambia el requisito «Selección de apps para clonar» (árbol de archivos, casilla parcial, `a`/`n` por ámbito, apps sin archivos fuera de la tabla con nota) y se añade un requisito que garantiza que el clon respeta la selección por archivo.

## Impact

- `internal/plan`: `CountFiles` pasa a devolver las rutas y los tamaños por app, no solo el recuento.
- `internal/clone`: `PlanOptions` acepta los archivos incluidos por app y `PlanClone` filtra las operaciones antes de construir el manifiesto.
- `internal/tui` (`screens_clone.go`, `appChoice`): modelo de selección por archivo, árbol, ajuste del cursor cuando el recuento retira apps de la lista y render.
- `internal/texts`: textos nuevos (nota «Sin archivos», ayuda de teclas) y retirada de `C1NoFiles`.
- Docs y pruebas: `docs/tui/pantallas-clon.md`, `docs/guia/clonar-e-implantar.md`, goldens de C1 y pruebas de `plan`, `clone` y `tui`. La ayuda de la CLI no cambia.
