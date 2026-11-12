# go-distribution Specification

## Purpose

Definir la distribución de Kamino como un binario Go estático para macOS Apple Silicon, con presupuestos de arranque, un asset de release compatible con el instalador y el actualizador existentes, y una transición segura desde el binario Python legado.

## Requirements

### Requirement: Binario Go estático y versión

La distribución de release DEBE (`MUST`) ser un ejecutable Go estático para macOS Apple Silicon (`darwin/arm64`) que no requiera Python, `uv`, `pip` ni una instalación previa. El ejecutable DEBE (`MUST`) exponer `--version` con el formato `kamino <versión>` y `--help` sin cargar el catálogo ni la configuración de la TUI.

#### Scenario: Ejecución sin Python

- **GIVEN** un macOS Apple Silicon sin Python instalado
- **WHEN** se ejecuta el binario de release
- **THEN** `kamino --version` y `kamino --help` funcionan y no inician un intérprete Python

#### Scenario: Versión inyectada en la construcción

- **GIVEN** una construcción de release con la versión `X.Y.Z`
- **WHEN** se ejecuta `kamino --version`
- **THEN** la salida es `kamino X.Y.Z` y termina con código 0

### Requirement: Presupuestos de arranque

El binario de release DEBE (`MUST`) arrancar `--version` y `--help` en menos de 30 ms, y mostrar el primer frame de la TUI en menos de 150 ms, medidos en el equipo de referencia de macOS Apple Silicon. Las mediciones DEBEN (`MUST`) excluir la primera ejecución tras una cuarentena o un escaneo de seguridad del sistema y registrarse con `hyperfine` o un arnés PTY equivalente.

#### Scenario: Arranque de la CLI

- **GIVEN** el binario de release instalado en macOS Apple Silicon
- **WHEN** se miden `kamino --version` y `kamino --help` con `hyperfine`
- **THEN** el tiempo medio es inferior a 30 ms

#### Scenario: Arranque de la TUI

- **GIVEN** una terminal de 80x24 y el binario de release
- **WHEN** se abre el menú y se mide hasta el primer frame
- **THEN** el tiempo es inferior a 150 ms

### Requirement: Asset de release, digest e instalación

La release DEBE (`MUST`) publicar el asset `kamino-arm64` con la versión inyectada y un SHA-256 verificable. `scripts/install.sh` y `kamino update` DEBEN (`MUST`) seguir usando el mismo nombre de asset, y la instalación NO DEBE (`MUST NOT`) requerir `sudo`. `uv` y `pip` dejan de ser vías de instalación.

#### Scenario: Instalación con el script

- **GIVEN** macOS Apple Silicon y una release publicada
- **WHEN** se ejecuta `scripts/install.sh`
- **THEN** descarga `kamino-arm64`, lo instala en `~/.local/bin/kamino` y no usa `sudo`

#### Scenario: Asset y digest publicados

- **GIVEN** una release de GoReleaser
- **WHEN** se consulta el asset `kamino-arm64` en GitHub
- **THEN** el asset existe, tiene el SHA-256 publicado y la versión del binario coincide con el tag

#### Scenario: Plataforma no soportada

- **GIVEN** un sistema que no es macOS Apple Silicon
- **WHEN** se intenta instalar o actualizar
- **THEN** la operación falla sin descargar ni escribir el binario

### Requirement: Actualización compatible con el binario Python legado

Un binario Python legado DEBE (`MUST`) poder ejecutar `kamino update` y reemplazarse por el asset Go, siempre que la versión publicada sea superior y el tamaño y el SHA-256 coincidan. Un binario Go de release DEBE (`MUST`) poder actualizarse a sí mismo; una construcción de desarrollo (`go run` o binario no marcado como autocontenido) NO DEBE (`MUST NOT`) descargar ni reemplazar nada.

#### Scenario: Python legado actualiza a Go

- **GIVEN** un binario Python autocontenido con una versión anterior a la release Go
- **WHEN** se ejecuta `kamino update`
- **THEN** descarga `kamino-arm64`, verifica el digest y reemplaza el binario por el ejecutable Go

#### Scenario: Go se actualiza a sí mismo

- **GIVEN** un binario Go de release con una versión anterior a la publicada
- **WHEN** se ejecuta `kamino update`
- **THEN** descarga, verifica y reemplaza el ejecutable real resolviendo los symlinks

#### Scenario: Construcción de desarrollo

- **GIVEN** un binario Go construido con `go run` o sin la marca de autocontenido
- **WHEN** se ejecuta `kamino update`
- **THEN** el comando falla con la vía de actualización correspondiente y no escribe nada

### Requirement: Compatibilidad de clones y holocrones durante la transición

El binario Go DEBE (`MUST`) leer y escribir clones y holocrones con `format_version = 1` compatibles con el binario Python legado. Un clon generado por Python DEBE (`MUST`) poder desplegarse con Go, y un clon generado por Go DEBE (`MUST`) poder desplegarse con Python, usando HOMEs temporales y sin tocar el HOME real.

#### Scenario: Clon de Python desplegado por Go

- **GIVEN** un clon generado por Python en un HOME temporal
- **WHEN** el binario Go lo despliega en otro HOME temporal
- **THEN** los archivos, enlaces, modos y categorías del resultado coinciden con el comportamiento de Python

#### Scenario: Clon de Go desplegado por Python

- **GIVEN** un clon generado por Go en un HOME temporal
- **WHEN** el binario Python lo despliega en otro HOME temporal
- **THEN** el manifiesto se valida, la integridad se verifica y los archivos se aplican correctamente

#### Scenario: Holocrón cruzado

- **GIVEN** un holocrón generado por Python o por Go
- **WHEN** el otro binario lo despliega en un HOME temporal
- **THEN** el holocrón se reconoce como la app sintética `snapshot` y restaura el estado anterior

### Requirement: Coexistencia y corte reversible

Durante la migración, Python DEBE (`MUST`) permanecer en `src/`, `tests/` y `scripts/` como referencia ejecutable, y el catálogo embebido de Go DEBE (`MUST`) mantenerse sincronizado con `src/kamino/catalog/apps/`. El corte final DEBE (`MUST`) mover Python a `legacy/python/` con `git mv`, sin borrarlo, y dejar la raíz como proyecto Go.

#### Scenario: Catálogo sincronizado

- **GIVEN** una definición de app en `src/kamino/catalog/apps/`
- **WHEN** se ejecuta la comprobación de sincronización
- **THEN** la copia en `internal/catalog/apps/` coincide byte a byte y la comprobación falla si divergen

#### Scenario: Corte sin pérdida

- **GIVEN** todas las pruebas Go y la compatibilidad cruzada en verde
- **WHEN** se ejecuta el corte de la Fase 4
- **THEN** Python se mueve a `legacy/python/` y no se elimina ningún archivo del historial
