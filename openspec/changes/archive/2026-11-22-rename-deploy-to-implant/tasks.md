# Tasks

## 1. Comando y alias en la CLI

- [x] 1.1 Despachar `implant` y `deploy` al mismo flujo en `internal/cli/cli.go`, conservando el argumento posicional, los flags y los códigos de salida; verificar con una prueba que ambos producen el mismo resultado sobre un clon de fixture.
- [x] 1.2 Renombrar la ayuda de `deploy` a `printImplantHelp`, mostrar `implant` como comando principal y `deploy` como alias, y corregir `Show this message and exit.` y `No such command` al español; verificar `kamino -h`, `kamino implant -h`, `kamino deploy -h` y un comando desconocido.
- [x] 1.3 Incluir `implant` y `deploy` en `completionScript` para bash, zsh y fish; verificar la salida de `kamino --show-completion` para los tres shells.
- [x] 1.4 Añadir pruebas de equivalencia `implant`/`deploy`, del alias en el autocompletado, de la ayuda y del código 3 por elementos omitidos; ejecutar `go test ./internal/cli -race`.

## 2. Textos y TUI

- [x] 2.1 Actualizar los valores visibles de `internal/texts/texts.go` de «Desplegar/despliegue» a «Implantar/implantación», incluidos menú, títulos, botones, progreso, resumen, error parcial y orden de deshacer; verificar con `grep` que no quedan etiquetas principales con «Desplegar».
- [x] 2.2 Revisar las pantallas de `internal/tui` por etiquetas literales fuera de `texts` y sustituirlas por «Implantar» o «implantación»; verificar que el menú muestra Clonar, Implantar, Actualizar, Configuración y Abandonar Kamino.
- [x] 2.3 Regenerar los goldens de `internal/tui/testdata/golden/` con `go test ./internal/tui -run Golden -update` y ejecutar el test de contrato de la TUI en 80×24, 100×26, 140×40 y 200×50.
- [x] 2.4 Añadir o ajustar pruebas de la TUI para el menú, el título «Implantar un clon», el botón «Implantar», el resumen «Implantación completada» y la orden `kamino implant <ruta>`; ejecutar `go test ./internal/tui -race`.

## 3. Documentación

- [x] 3.1 Actualizar `README.md`: descripción, ejemplos, ayuda, menú, órdenes de deshacer y referencias al alias `deploy`; verificar que los ejemplos usan `implant` como forma principal y que `deploy` aparece documentado como alias.
- [x] 3.2 Actualizar `docs/diseno-funcional.md` y `docs/diseno-tecnico.md` con la terminología «implantar/implantación», manteniendo `deploy.snapshots_dir`, `deploy.snapshots_limit` y los identificadores internos; verificar que no queda terminología visible obsoleta fuera de los archivos históricos de `openspec/changes/archive/`.

## 4. Verificación de integración

- [x] 4.1 Ejecutar `gofmt -l .` (vacío), `go vet ./...`, `golangci-lint run ./...` si está instalado y `go test ./... -race`.
- [x] 4.2 Verificar que el cambio no altera el formato del clon, el manifiesto, los flags ni las claves de configuración: comparar el diff de `internal/archive`, `internal/deploy` y `internal/config` y comprobar que `deploy.snapshots_dir` y `deploy.snapshots_limit` siguen iguales.

## Workflow follow-up

- Archivar el cambio después de la revisión y de que CI pase en `main`.
- Verificar el resultado archivado y que `kamino deploy` sigue funcionando como alias.
