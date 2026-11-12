# Diseño

## Context

Ver `proposal.md` para la motivación. El estado actual relevante:

- La rama `improvements` contiene el contrato ejecutable: Python 0.7.0, 9.908 líneas en `src/kamino`, 9.781 en `tests`, 26 definiciones TOML y 11 specs con 99 requisitos y 327 escenarios.
- No existe código Go, `go.mod`, `MIGRATION/`, `testdata/golden/` ni `scripts/gen_golden.py`. La Fase 1 parte de cero.
- El binario `dist/kamino-arm64` es un `onefile` de PyInstaller de 18 MB y tarda alrededor de 6,5 s en `--version` en este equipo; `uv run kamino --version` tarda alrededor de 0,27 s.
- Go 1.27.2 está instalado. Faltan `golangci-lint`, `gofumpt`, `goreleaser` y `hyperfine`.
- Python y Go deben convivir hasta la Fase 4. El código Python es el oráculo; los specs son el contrato.

Restricciones que condicionan el diseño:

- Todo test usa HOMEs temporales y rutas inyectadas; nunca el HOME real ni `/Applications`.
- El núcleo no imprime ni pregunta. La CLI y la TUI traducen resultados; los textos viven en `texts`.
- Se conservan el formato `format_version = 1`, el manifiesto, las rutas relativas al HOME, los códigos 0/1/2/3 y el orden verificar -> extraer -> decidir -> holocrón -> aplicar.
- `dry_run` no escribe nada. La cancelación es cooperativa y respeta el punto de no retorno.
- No se hace push, merge a `main`, release ni publicación.

## Goals / Non-Goals

**Goals:**

- Un binario Go estático `darwin/arm64` con arranque de CLI inferior a 30 ms y de TUI inferior a 150 ms.
- Paridad de comportamiento con Python para clonar, desplegar, conflictos, catálogo, detección, actualización y TUI.
- Compatibilidad cruzada de clones y holocrones entre Python y Go durante la transición.
- Una migración por fases, con tests primero, fixtures doradas, puertas de calidad y tags locales.
- Un corte reversible: Python se mueve a `legacy/python/`, no se borra.

**Non-Goals:**

- Portar Textual, Rich, Typer, Pydantic o PyInstaller a Go.
- Cambiar el formato del archivo, el manifiesto, el catálogo o el comportamiento de `clone`/`deploy`.
- Soportar Windows, Linux o macOS Intel como destino de release.
- Publicar, firmar, notarizar o crear un tap de Homebrew durante la migración.
- Portar todos los temas de Textual: la TUI Go solo garantiza `ansi-dark` y `ansi-light`.
- Reproducir byte a byte las tablas y cajas de Rich.

## Decisions

### 1. Layout Go y dirección de dependencias

Se usa un módulo raíz `github.com/Gunmer/kamino` y paquetes internos con dependencias en una sola dirección:

```text
cmd/kamino -> internal/cli -> internal/app
                                  |
             +--------------------+--------------------+
             |                    |                    |
          clone                deploy               update
             |                    |                    |
           plan <-- catalog    conflicts             config
             |      detect        |                    |
          ghostty  paths       snapshot             defaults
             |                    |
          archive <--------------+
             |
           paths (SafeHome)
```

`internal/app` sustituye a `commands.py` y devuelve resultados tipados. `internal/cli` solo traduce flags, gating de TUI, salida y códigos. `internal/ui` contiene la salida de texto; `internal/tui` contiene la app interactiva. `internal/texts` no importa `tui` ni `ui`.

**Alternativa descartada:** un único paquete `internal/kamino`. Evita decisiones de importación, pero mezcla dominio, CLI y TUI y dificulta los tests por capas.

### 2. `SafeHome` con `openat` y `O_NOFOLLOW`, no solo `os.Root`

`os.Root` impide escapar del root, pero sigue enlaces simbólicos internos. El contrato exige no atravesar enlaces intermedios aunque el destino quede dentro del HOME. Por eso `internal/paths` porta `SafeHome` con `golang.org/x/sys/unix`:

- `rootFd` abierto una vez.
- `Openat` con `O_NOFOLLOW | O_DIRECTORY` en cada componente intermedio.
- `Fstatat` con `AT_SYMLINK_NOFOLLOW` y `Readlinkat` para el componente final.
- `Mkdirat`, `Renameat`, `Symlinkat`, `Unlinkat`, `Fchmod` y `Fsync` para escrituras atómicas.
- `walk` con `ReadDir` sobre un fd sin seguir enlaces y orden lexicográfico.

**Alternativa descartada:** `os.Root` puro. Es más simple, pero cambia una invariante de seguridad y no cumple `clone-safety`.

### 3. Archivo y manifiesto: semántica compatible, bytes deterministas por lenguaje

`internal/archive` usa `archive/tar` y `compress/gzip`. Para escribir:

- `manifest.json` primero, con el mismo orden de campos y `null` explícito de Pydantic.
- `tar.FormatPAX`, modo, uid/gid, mtime y enlaces del `lstat`.
- `gzip.Header.ModTime` cero y orden de miembros estable.
- Commit atómico con `os.Link` cuando no se sobrescribe y `os.Rename` con revalidación cuando hay `--force`.

Para leer:

- Escanear todo el tar para detectar `manifest.json` duplicado y el tope de 100.000 miembros.
- Limitar el manifiesto a 4 MiB.
- Validar rutas relativas, `sha256`, `size`, `link` y `format_version`.
- Implementar el equivalente de `tarfile` `filter="data"`: rechazar rutas absolutas, `..`, enlaces fuera del destino, archivos especiales y bits peligrosos; admitir enlaces internos.
- Verificar todos los miembros antes de extraer; extraer solo en staging temporal.

No se comparan bytes Python vs Go: se compara el modelo semántico (manifiesto, nombres, tipos, contenidos, enlaces y modos). Go sí debe ser determinista entre ejecuciones.

### 4. Diff: portar `SequenceMatcher`, `unified_diff` y `splitlines`

Hay dos caminos que deben conservarse:

```text
D3: build_hunks -> SequenceMatcher(autojunk=False) -> Hunk/HunkRow
Dry-run: unified_diff -> SequenceMatcher(autojunk=True) -> ---/+++/@@
```

Se porta `SequenceMatcher` con `autojunk` configurable, `get_grouped_opcodes(n)` y `unified_diff`. También se porta `splitlines(keepends=True)` con los separadores Unicode de Python (`\v`, `\f`, `\x1c`, `\x1d`, `\x1e`, `\x85`, `\u2028`, `\u2029`), porque `strings.SplitAfter` solo parte por `\n`.

**Alternativa descartada:** una librería Myers. Produce hunks distintos y rompe los golden fixtures de conflictos.

### 5. CLI: Cobra con contrato de presentación propio

`internal/cli` usa Cobra, pero no acepta su ayuda por defecto:

- `--version`/`-V` imprime `kamino <versión>`.
- `-h`/`--help` en todos los niveles.
- Códigos 2 para uso incorrecto, 1 para error/cancelación, 3 para resultado parcial.
- `--apps`, `--output`, `--dry-run`, `--verbose`, `--yes`, `--force` con alias.
- `--install-completion` y `--show-completion` se mantienen como compatibilidad; Cobra no los trae.
- La ayuda conserva comandos, opciones, alias y descripciones; no se promete el mismo dibujo de cajas de Rich.

La salida de texto se implementa en `internal/ui`. Las tablas y resúmenes conservan campos y contenido; el formato decorativo puede diferir. Los snapshots crudos de Python quedan como referencia.

**Alternativa descartada:** `flag` estándar. Es más pequeño, pero no ofrece subcomandos, ayuda ni completion con el mismo coste de mantenimiento.

### 6. TUI: Bubble Tea + Lip Gloss + Bubbles + teatest

La TUI se modela como una pila de pantallas sobre un `tea.Model` raíz:

```text
menu
  +-- C1 apps -> C2 destino -> C3 revisión -> C4 resumen
  +-- D0 archivo -> D1 apps -> D2 conflictos -> D3 detalle -> D4 confirmar -> D5 resumen
  +-- U1 actualización
  +-- Configuración
  +-- Ayuda / error común / terminal pequeña
```

- `bubbles/list` o un modelo propio para C1/D1, `bubbles/table` para D0/D2, `textinput` para C2/filtros, `progress` para operaciones y `viewport` para D3.
- La pila conserva estado al volver; los diálogos son pantallas modales.
- `Ctrl+C` se enruta a la pantalla activa y respeta el punto de no retorno.
- La guardia de tamaño 80x24 conserva la pantalla inferior y bloquea la salida.
- La copia usa OSC 52 con la misma heurística de terminal que Python.
- El tema por defecto es `ansi-dark`; `ansi-light` se admite; `NO_COLOR` usa un renderer monocromo.
- `TEXTUAL_THEME` con otros valores cae a `ansi-dark`.

**Alternativa descartada:** `tview`. Tiene tablas y formularios, pero menos control de layout, estilos y tests; `teatest` no aplica.

### 7. Actualización: autocontenido por ldflags

Go no tiene `sys.frozen`. La construcción de release inyecta:

```text
-X github.com/Gunmer/kamino/internal/version.Version=<versión>
-X github.com/Gunmer/kamino/internal/buildinfo.SelfContained=true
```

`go run`, `go test` y builds de desarrollo quedan en `false`. `update` resuelve `os.Executable()` con `filepath.EvalSymlinks`, consulta GitHub, descarga a un temporal del mismo directorio, verifica tamaño y SHA-256, y reemplaza con `os.Rename`. Un binario Python legado puede actualizar al asset Go porque el nombre, el digest y el reemplazo no dependen del lenguaje.

### 8. Catálogo, detección, plan y Ghostty

- `internal/catalog` usa `go:embed` sobre una copia de `src/kamino/catalog/apps/`. Un test y `scripts/check_catalog_sync.sh` fallan si divergen.
- `internal/detect` usa `howett.net/plist` para `Info.plist` XML/binario y un `which` inyectable que no ejecuta binarios.
- `internal/plan` porta `fnmatch` de Python: `*` también coincide con `/`, y `**/X` coincide con `X` en el nivel superior. `path.Match` no sirve tal cual.
- `internal/ghostty` porta `iter_directives`, `config_reference`, `theme_references`, `resolve_reference` y `absolute_references`, con límites de 500 archivos y profundidad 12.
- `internal/config` solo lo usa la TUI. La CLI no lee ni escribe `config.toml`.

### 9. Validación de híbridos

Se conservan los validadores por extensión:

- TOML: `pelletier/go-toml/v2` o equivalente estricto.
- JSON: `encoding/json` con detección de claves duplicadas.
- INI: parser estricto compatible con `configparser(strict=True, interpolation=None)`; si una librería no coincide, se implementa el subconjunto necesario.
- plist: `howett.net/plist` para XML y binario; detección de claves duplicadas solo en XML, como Python.

El estado (`valid`, `invalid`, `warning`, `unvalidated`) es el contrato. Los mensajes internos de parser pueden diferir; se conserva el mensaje de alto nivel.

### 10. Estrategia de pruebas y fixtures

- Fase 2 es TDD con stubs: `go vet ./...` compila y `go test ./...` falla por funcionalidad no implementada.
- `scripts/gen_golden.py` genera fixtures desde Python en HOMEs sintéticos: manifiestos, tar.gz, conflictos, ayuda, resúmenes y códigos 0/1/2/3.
- Los tar.gz se normalizan (mtime, uid/gid, gzip header) para que dos ejecuciones produzcan los mismos bytes.
- Los tests de compatibilidad cruzada ejecutan Python con `HOME` temporal y `KAMINO_PLAIN=1`; en CI hay un job con Python/uv.
- `scripts/guard.sh` falla si cambian `testdata/golden/` o los `*_test.go` de la Fase 2 sin una entrada en `DECISIONS.md`.
- La TUI se prueba con `teatest` o un arnés propio: 80x24, 79x23, redimensionado, navegación, `NO_COLOR`, ayuda, error de cuatro partes y `Ctrl+C` antes/después del punto de no retorno.

### 11. Release y corte

- `.goreleaser.yaml` produce un asset binario `kamino-arm64` con `CGO_ENABLED=0`, `-trimpath`, `-s -w` y versión inyectada.
- `goreleaser release --snapshot --clean` se ejecuta solo en local; no publica.
- Se conserva `semantic-release` para tag/changelog o se adapta en Fase 4; no se mezcla con GoReleaser en el mismo commit.
- El corte mueve Python a `legacy/python/` con `git mv`. `legacy/` no se elimina.
- El commit final es `feat!: migrate to Go` con `BREAKING CHANGE:`; no se publica.

## Risks / Trade-offs

| Riesgo | Mitigación |
|---|---|
| La TUI Go no reproduce píxeles de Textual | Contrato conductual: pantallas, teclas, estados, textos y códigos; tests de comportamiento |
| `os.Root` no cumple la invariante de no seguir enlaces | `SafeHome` con `openat`/`O_NOFOLLOW`; tests de enlaces intermedios y carreras |
| Los hunks de Myers no coinciden con Python | Portar `SequenceMatcher` con `autojunk` configurable y golden fixtures |
| Los tar.gz Python no son deterministas | Normalización en el generador; comparación semántica Python vs Go |
| Rich/Typer no se reproduce byte a byte | Contrato semántico de ayuda/tablas; snapshots crudos como referencia |
| `TEXTUAL_THEME` arbitrario no es viable | Spec modificado: `ansi-dark`/`ansi-light`; el resto cae a `ansi-dark` |
| `update` no distingue un build de desarrollo | Marca `SelfContained` por ldflags; tests de `go run`/build normal |
| El cross-compat necesita Python en CI | Job dedicado con `uv sync`, `KAMINO_PYTHON` y HOMEs temporales |
| El baseline está distorsionado por macOS | Medir con `hyperfine` y PTY, registrar entorno y primera ejecución |
| El release de GitHub no publica `digest` | Verificar el asset con `gh api` y el smoke test de release antes de publicar |

## Migration Plan

1. **Fase 1:** crear `migrate/go` desde `improvements`, `MIGRATION/`, `go.mod`, layout, Makefile, `.golangci.yml`, copia del catálogo, baseline, golden fixtures y CI.
2. **Fase 2:** tests por capas con stubs, cobertura de los 99 requisitos, compatibilidad cruzada y presupuestos de arranque. Tag `migration/phase-2`.
3. **Fase 3:** implementar `paths`/`archive`, `catalog`/`detect`/`plan`/`ghostty`, `snapshot`/`clone`, `conflicts`/`deploy`, `texts`/`ui`/`cli`, `update` y `tui`. Cada capa en verde antes de la siguiente. Tag `migration/phase-3`.
4. **Fase 4:** GoReleaser, benchmark, documentación, corte a `legacy/python/` y commit final. Tag `migration/phase-4`.

**Rollback:** mientras no se haga el corte, la rama `migrate/go` conserva Python intacto y los tags de fase. Si una capa falla, se revierte el commit de esa capa o se abandona la rama; `main` no se toca. Después del corte, `legacy/python/` permite volver a ejecutar el oráculo.

## Decisiones diferidas a Fase 4

- Versión final: `0.8.0` si se mantiene `major_on_zero = false`, o `1.0.0` si se decide que el fin de `uv`/`pip` es un cambio mayor.
- Propiedad del tooling de release: conservar `semantic-release` o portarlo a Go; no cambia el enfoque de la migración.
