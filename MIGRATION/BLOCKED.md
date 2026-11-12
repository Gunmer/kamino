# Bloqueos

## B001 — Sin acceso a red para instalar herramientas

- **Fecha:** 2026-11-12
- **Afecta a:** tareas 1.3, 1.7, 12.2, 12.4 y 12.8.
- **Detalle:** `http_proxy`/`https_proxy` apuntan a `127.0.0.1:8999`, que rechaza
  conexiones; `go install`, `brew install`, `uv sync` y las descargas de módulos
  fallan. El módulo usa únicamente la caché local de Go.
- **Mitigación aplicada:** `go vet`, `gofmt`, `go test`, un arnés de medición
  propio y `go build` con las ldflags de release.
- **Estado:** resuelto el 2026-11-12. Se instalaron `golangci-lint` 2.15.0,
  `gofumpt` 0.12.0, `goreleaser` 2.20.0 y `hyperfine` 2.0.0 omitiendo el proxy
  caído solo durante la descarga, y se repitieron las verificaciones exactas.

## B002 — Firma de commits

- **Fecha:** 2026-11-12
- **Detalle:** `commit.gpgsign=true` apunta a una clave inexistente
  (`/Users/T040541/...`). Los commits locales se crean con
  `git -c commit.gpgsign=false`; no se hace push.
