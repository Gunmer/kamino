# Tasks

## 1. Backend: recuento con detalle y filtrado por archivo

- [x] 1.1 Hacer que `plan.CountFiles` devuelva en `CountResult.Files` la ruta, el tamaño y la marca de enlace por app, derivando `Counts` de ello; verificar con una prueba en `internal/plan` que las rutas coinciden con las de `BuildPlan` para el mismo HOME temporal y que `go test ./internal/plan -race` pasa.
- [x] 1.2 Añadir `PlanOptions.Files` y filtrar `built.Operations` en `clone.PlanClone` antes de `buildManifest`; verificar con pruebas en `internal/clone` que una app presente solo incluye sus rutas, que una app ausente incluye todo, que un archivo desconocido se ignora y que `RunClone` con `dryRun` no escribe nada.
- [x] 1.3 Probar el ciclo plan → escritura → lectura del manifiesto con una selección parcial y comprobar que el archivo y el manifiesto contienen solo los archivos incluidos; `go test ./internal/clone -race` pasa.

## 2. TUI: modelo de selección y recuento

- [x] 2.1 Añadir `Files []fileChoice` a `appChoice`, sembrarlo en `appsCountedMsg` con `Included=false` y derivar el estado de la casilla (`[x]`/`[~]`/`[ ]`) y `selectedIDs()` de los archivos incluidos; verificar con pruebas unitarias del estado en `internal/tui`.
- [x] 2.2 Excluir de `visibleApps` las apps con `Count == 0` cuando el recuento ha terminado, acotar `index` al llegar `appsCountedMsg` y retirar la casilla `[-]` y el aviso `C1NoFiles`; verificar con una prueba en la que la app enfocada desaparece tras el recuento y el cursor queda en una fila válida.
- [x] 2.3 Pasar `planCloneCmd` a enviar `PlanOptions.Files` con las rutas incluidas de cada app seleccionada; verificar con una prueba del flujo C1 → C3 que el plan contiene solo los archivos marcados.

## 3. TUI: árbol, teclas y render

- [x] 3.1 Implementar la lista plana de filas (app y archivo), el conjunto de apps expandidas y las teclas `→`/`←`; verificar con pruebas de navegación que con todo colapsado la lista coincide con la actual y que `window` desplaza el árbol con los indicadores.
- [x] 3.2 Hacer que `Espacio`, `a` y `n` actúen según el ámbito de la fila enfocada (app o archivo) y que la app con todos los archivos desmarcados quede desmarcada; verificar con pruebas de los escenarios de `specs/clone-tui/spec.md`, incluido el filtro `/` solo por app.
- [x] 3.3 Dibujar las filas de archivo (ruta truncada por el inicio y tamaño), la columna `marcados/total` en selección parcial y la nota «Sin archivos: …» bajo «No detectadas»; verificar regenerando y revisando los goldens de C1 con `go test ./internal/tui -run Golden -update` a 80 columnas y con un árbol que no cabe.
- [x] 3.4 Añadir los textos nuevos y actualizar `cloneAppsKeys`, `HelpC1` y `ShortcutsC1`; verificar con `go test ./internal/tui` (incluidas las pruebas de terminología) y la ayuda de C1 en pantalla.

## 4. Documentación y verificación final

- [x] 4.1 Actualizar `docs/tui/pantallas-clon.md` (C1, variantes y notas) y `docs/guia/clonar-e-implantar.md`; verificar que los dibujos coinciden con los goldens y que ningún texto menciona `[-]`.
- [x] 4.2 Ejecutar `gofmt -l .`, `go vet ./...`, `golangci-lint run ./...` (si está instalado) y `go test ./... -race`; verificar que todo pasa y que `openspec validate clone-select-files-per-app --strict` no da errores.

## Workflow follow-up

- Archivar el change tras la revisión del proyecto.
- Verificar que `openspec/specs/clone-tui/spec.md` refleja los requisitos tras archivar.
