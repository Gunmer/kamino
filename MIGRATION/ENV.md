# Entorno de la migración

Registrado: 2026-11-12

| Herramienta | Versión | Estado |
|---|---|---|
| macOS | Apple Silicon (`darwin/arm64`) | disponible |
| Go | go1.27.2 darwin/arm64 | disponible |
| Python (`.venv`) | 3.12.7 | disponible |
| Python (sistema) | 3.11.1 | no usado |
| `uv` | en `~/.local/bin/uv` | sin red para sincronizar |
| `golangci-lint` | 2.15.0 | instalado con Homebrew |
| `gofumpt` | 0.12.0 | instalado con Homebrew |
| `goreleaser` | 2.20.0 | instalado con Homebrew |
| `hyperfine` | 2.0.0 | instalado con Homebrew |

Los binarios se instalaron omitiendo el proxy local caído (`127.0.0.1:8999`)
solo durante la descarga; el proxy sigue configurado en `~/.zshrc`.

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
