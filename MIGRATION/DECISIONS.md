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

## D004 — Ajustes de conformidad de la TUI

- **Fecha:** 2026-11-12
- **Contexto:** las pruebas de la TUI y la revisión contra `clone-tui` y
  `tui-navigation` detectaron diferencias: selección inicial vacía, recuento por
  app, diálogo de clon existente, numeración y menú degradado por altura.
- **Decisión:** C1 parte sin apps marcadas y deshabilita las de cero archivos;
  C2 muestra el diálogo de reemplazo con Volver preseleccionada; el menú no usa
  atajos de una tecla y degrada cabecera según la altura; los flujos iniciados
  desde el menú vuelven a él y actualizan la última misión.
- **Consecuencia:** los fixtures de texto y los tests de la TUI se actualizan de
  forma coherente; no cambia el formato del archivo ni los códigos de salida.

## D005 — Formato de gofumpt 0.12

- **Fecha:** 2026-11-12
- **Contexto:** la verificación con `gofumpt` real (antes no había red para
  instalarlo) marca tres archivos: `internal/archive/archive_test.go`,
  `internal/archive/manifest.go` y `internal/texts/texts.go`.
- **Decisión:** aplicar `gofumpt -w` a esos archivos. En el test protegido el
  cambio es solo de formato (coma final del parámetro agrupado); no altera
  aserciones ni comportamiento.
- **Consecuencia:** `gofumpt -l .` queda vacío y la guardia de tests protegidos
  conserva su propósito con este cambio documentado.

## D006 — Saltos de línea de la TUI en modo raw

- **Fecha:** 2026-11-12
- **Contexto:** el modo raw desactiva `OPOST`, así que un `\n` solo baja el
  cursor y no vuelve a la columna 0. La TUI unía las líneas del frame con `\n`
  y cada línea empezaba donde terminaba la anterior, produciendo el efecto de
  escalera y el texto desalineado.
- **Decisión:** `frame` emite `\r\n` al final de cada línea. Se añade
  `TestViewReturnsToColumnZero` para fijar el contrato del framebuffer.
- **Consecuencia:** todas las pantallas de la TUI empiezan en la columna 0 con
  independencia del ancho del terminal; no cambia el contenido ni la disposición
  de las pantallas.

## D007 — Fixture canónico del clon versionado

- **Fecha:** 2026-11-12
- **Contexto:** `testdata/golden/clone.tar.gz` no estaba versionado porque
  `.gitignore` ignoraba todos los `*.tar.gz`. Los tests pasaban en local pero
  fallaban en CI al no encontrar el fixture.
- **Decisión:** versionar el fixture (436 bytes, SHA-256
  `71db5ba0afd0d87c571d17662dc163e3f880164087f0973b0300e265e8cd20bc`) y
  excluir solo `testdata/golden/*.tar.gz` de la regla de ignorados. Añadir un
  README propio a `legacy/python/` para que Hatchling construya el paquete
  desde su directorio tras la migración.
- **Consecuencia:** CI dispone de todos los fixtures dorados y `uv sync`
  funciona en `legacy/python/` sin depender del README de la raíz.

## D008 — Nueva suite de la TUI con Bubble Tea

- **Fecha:** 2026-11-13
- **Contexto:** el cambio `tui-redesign` sustituye el `Model` monolítico, el
  lector de teclas, el modo raw y el render ANSI propio por Bubble Tea, Lip
  Gloss y `bubbles`. Los tests antiguos de la TUI dependían de esos tipos.
- **Decisión:** retirar `internal/tui/tui_test.go` y los tests acoplados al
  runner anterior; añadir pruebas de `App`/`Frame`/`theme`, C1–C3, D0–D5, U1,
  Configuración, ayuda, error y asincronía, además de goldens a 80×24, 100×26,
  140×40 y 200×50 con variantes `NO_COLOR` y ASCII. El contrato exige exactamente
  `h` líneas, ninguna más ancha que `w` y solo SGR ANSI.
- **Consecuencia:** los fixtures dorados de la TUI pasan a
  `internal/tui/testdata/golden/` y se regeneran con `-update`; el resto de
  fixtures y el formato del clon no cambian.
