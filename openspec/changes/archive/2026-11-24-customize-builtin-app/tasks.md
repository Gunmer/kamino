# Tasks

## 1. Catálogo y caso de uso

- [x] 1.1 Añadir en `internal/catalog` una función que devuelva el TOML integrado en bruto por id, con error en español si no existe, y su prueba
- [x] 1.2 Añadir `ExecuteAppsCustomize` en `internal/app/apps.go` (copia, `--print`, `--force`, id duplicado en otro archivo) con pruebas con HOME temporal

## 2. CLI

- [x] 2.1 Añadir el subcomando `kamino apps customize <id>` con `--print` y `--force`, códigos de salida 0/1/2 y salida en español
- [x] 2.2 Actualizar la ayuda de `apps` y las pruebas de `internal/cli`

## 3. TUI

- [x] 3.1 Añadir el selector de apps integradas y la entrada «Personalizar app» en Configuración, con textos en `internal/texts`
- [x] 3.2 Sembrar el borrador con el TOML integrado y reutilizar `appAuthoringScreen` (cancelar, editar, confirmar reemplazo)
- [x] 3.3 Añadir pruebas de pantalla y goldens (`go test ./internal/tui -run Golden -update`)

## 4. Documentación y verificación

- [x] 4.1 Documentar `apps customize`, la pantalla y la deriva en `docs/guia/catalogo.md` y `docs/tui/`
- [x] 4.2 Ejecutar `gofmt -l .`, `go vet ./...`, `golangci-lint run ./...` y `go test ./... -race`
