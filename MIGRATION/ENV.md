# Entorno de la migración

Registrado: 2026-11-12

| Herramienta | Versión | Estado |
|---|---|---|
| macOS | Apple Silicon (`darwin/arm64`) | disponible |
| Go | go1.27.2 darwin/arm64 | disponible |
| Python (`.venv`) | 3.12.7 | disponible |
| Python (sistema) | 3.11.1 | no usado |
| `uv` | en `~/.local/bin/uv` | sin red para sincronizar |
| `golangci-lint` | — | no instalado (sin red) |
| `gofumpt` | — | no instalado (sin red) |
| `goreleaser` | — | no instalado (sin red) |
| `hyperfine` | — | no instalado (sin red) |

## Dependencias Go disponibles en caché

- `github.com/charmbracelet/bubbletea@v1.3.10`
- `github.com/charmbracelet/bubbles@v1.0.0`
- `github.com/charmbracelet/lipgloss@v1.1.0`
- `github.com/BurntSushi/toml@v1.5.0`
- `golang.org/x/sys@v0.47.0`
- `github.com/atotto/clipboard@v0.1.4`

## Comandos de referencia

```sh
go version
go vet ./...
go test ./... -race
gofmt -l .
.venv/bin/python -m kamino --version
```
