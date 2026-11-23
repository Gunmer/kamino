# Plan de implementación y decisiones tomadas

## 9. Plan de implementación (v1)

1. Esqueleto Go: `go.mod`, `cmd/kamino`, `internal/`, `Makefile`, `golangci-lint`, `go test` y CI.
2. Modelos y carga del catálogo, con una app de ejemplo.
3. Detección de apps (`Info.plist` y `command`).
4. `internal/archive`: escritura y lectura del `tar.gz`, manifiesto y checksums.
5. `internal/plan` e `internal/clone`, con `--dry-run`.
6. `internal/implant`: staging, verificación, holocrón y aplicación.
7. `internal/cli` e `internal/ui`: selección interactiva y resumen.
8. Resto del catálogo inicial (Git, Ghostty, herdr y yazi) y pruebas de integración.
9. `cmd/release` y empaquetado del binario; después tap y binario.

## 10. Decisiones tomadas

- **Nombre:** `kamino` para el comando, el módulo y el repositorio.
- **Comandos del CLI:** parser propio con el contrato de opciones y alias. El verbo es `implant` y el paquete interno se llama igual.
- **Versión mínima de Go:** 1.25.
- **Catálogo inicial:** Git, Ghostty, herdr y yazi.
- **Ubicación de los clones:** `~/.config/kamino/vault/` por defecto, con el nombre `clone-<aaaammdd>.tar.gz` y reemplazo explícito mediante `--force`. La TUI puede cambiarla con `clone.output_dir`; la CLI mantiene el default.
- **Pendiente al escribir las definiciones:** confirmar las rutas exactas de Ghostty. En yazi se clonan la configuración, `init.lua`, `package.toml` y la carpeta `plugins`.
