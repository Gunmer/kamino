# Decisiones de la migración

Cada entrada documenta una decisión que afecta a los tests dorados, a los
`*_test.go` de la Fase 2 o a una invariante de comportamiento. `scripts/guard.sh`
exige una entrada aquí cuando cambian esos archivos.

## D001 — Ejecutar las comprobaciones con las herramientas disponibles sin red

- **Fecha:** 2026-11-12
- **Contexto:** el entorno no tiene acceso a `proxy.golang.org` ni a Homebrew;
  `golangci-lint`, `gofumpt`, `goreleaser` y `hyperfine` no se pudieron instalar.
- **Decisión:** `make lint` usa `golangci-lint` si existe y, si no, `go vet`.
  `make fmt` usa `gofumpt` si existe y, si no, `gofmt`. La medición usa un arnés
  propio en lugar de `hyperfine`. GoReleaser se configura, pero la snapshot local
  se valida con `go build` y las mismas ldflags.
- **Consecuencia:** las puertas conservan el mismo significado; la herramienta
  exacta queda registrada como pendiente manual en `MIGRATION/BLOCKED.md`.

## D002 — Dependencias Go desde la caché local

- **Fecha:** 2026-11-12
- **Contexto:** sin red, el módulo solo puede usar paquetes ya presentes en
  `GOMODCACHE`.
- **Decisión:** se usan `charmbracelet/bubbletea`, `bubbles`, `lipgloss`,
  `BurntSushi/toml`, `golang.org/x/sys` y `atotto/clipboard` desde la caché.
  La CLI no usa Cobra porque no está disponible; se implementa un parser propio
  con el mismo contrato.
- **Consecuencia:** el binario conserva el contrato de opciones y códigos; la
  ayuda no reproduce el dibujo de Cobra.

## D003 — Formato de manifiesto

- **Fecha:** 2026-11-12
- **Contexto:** el manifiesto es el contrato entre Python y Go.
- **Decisión:** `encoding/json` con campos en el orden de Pydantic, `null`
  explícito para campos opcionales y `SetEscapeHTML(false)`. Los bytes del tar
  no se comparan entre lenguajes; se compara el modelo semántico.
- **Consecuencia:** Python puede leer clones Go y viceversa con
  `format_version = 1`.
