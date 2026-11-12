# Informe de la migración a Go

Fecha: 2026-11-12 · Rama: `migrate/go` · Cambio: `migrate-to-go`

## Estado por fase

| Fase | Estado | Tag |
|---|---|---|
| 1. Entorno, layout y fixtures | completada | `migration/phase-1` |
| 2. Tests por capas y cobertura | completada | `migration/phase-2` |
| 3. Implementación por capas y TUI | completada | `migration/phase-3` |
| 4. Release, rendimiento y corte | completada | `migration/phase-4` |

## Tiempos

| Medición | Python PyInstaller | Python `.venv` | Go | Presupuesto |
|---|---:|---:|---:|---:|
| `--version` | 5964.27 ms | 240.56 ms | 6.39 ms | < 30 ms |
| `--help` | 5966.94 ms | 268.40 ms | 5.72 ms | < 30 ms |
| `clone -n -a git` | 5967.18 ms | 252.80 ms | 8.72 ms | — |
| Primer frame TUI | 6045.31 ms | 236.10 ms | 25.97 ms | < 150 ms |

Detalle y método en `MIGRATION/baseline-python.md` y
`MIGRATION/benchmark-after.md`.

## Decisiones

- `SafeHome` con `openat`/`O_NOFOLLOW` en vez de `os.Root`.
- Manifiesto JSON con orden de Pydantic, `null` explícitos y `SetEscapeHTML(false)`.
- `SequenceMatcher` compatible con `autojunk` configurable, no una librería Myers.
- Parser de CLI propio porque Cobra no está disponible sin red.
- TUI de terminal crudo y ANSI propio porque Bubble Tea/Textual no son necesarios
  para el contrato y varias dependencias no están en la caché.
- Extracción segura con validación previa y sin atravesar enlaces.
- Actualización con `SelfContained` inyectado por `-ldflags`.

Todas las decisiones que afectan a tests protegidos están en
`MIGRATION/DECISIONS.md`.

## Bloqueos y mitigaciones

| Bloqueo | Mitigación |
|---|---|
| Sin red para `golangci-lint`, `gofumpt`, `goreleaser` y `hyperfine` | Resuelto: herramientas instaladas y verificaciones repetidas el 2026-11-12 |
| Firma GPG de commits no configurada | commits locales con `-c commit.gpgsign=false`; no se hizo push |
| Python/uv no sincronizan sin red | se usó el `.venv` existente y `PYTHONPATH=legacy/python/src` para el oráculo |

Las verificaciones exactas con esas herramientas quedan como pasos manuales.

## Cobertura de specs

`MIGRATION/spec-coverage.md` contiene 99 filas, una por requisito de los 11 specs
(además de `go-distribution`). Las pruebas viven en `internal/*/*_test.go`,
`internal/integration` y `internal/performance`.

## Riesgos residuales

- La TUI Go reproduce el contrato conductual (pantallas, teclas, estados, textos
  y códigos), no los píxeles de Textual.
- Los mensajes internos de parser TOML/JSON/INI/plist pueden diferir; el estado
  de validación es el contrato.
- La release no se firmó ni notarizó; macOS puede aplicar cuarentena.
- GoReleaser se ejecutó en modo snapshot; la release firmada y notarizada sigue
  pendiente.
- El workflow de release conserva semantic-release para el versionado; la
  decisión 0.8.0/1.0.0 sigue diferida.

## Pasos manuales pendientes

Completados el 2026-11-12: instalación de `golangci-lint`, `gofumpt`,
`goreleaser` y `hyperfine`, y `goreleaser release --snapshot --clean` en macOS
arm64.

1. Revisar `review-request-phase2.md`, `review-request-3a.md` y
   `review-request-3d.md` con otro modelo.
2. Notarizar el binario y crear el Homebrew tap cuando se decida publicar.
3. Retirar `legacy/python/` solo después de que la compatibilidad cruzada deje de
   ser necesaria.

## Verificación final

```sh
gofmt -l .
gofumpt -l .
go vet ./...
golangci-lint run ./...
go test ./... -race
go test ./internal/integration -run Cross -v
scripts/guard.sh
scripts/check_catalog_sync.sh
goreleaser release --snapshot --clean
```
