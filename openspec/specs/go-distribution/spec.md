# go-distribution Specification

## Purpose

Definir la distribución de Kamino como un binario Go estático para macOS Apple Silicon, con presupuestos de arranque, un asset de release compatible con el instalador y el actualizador, y un formato de clones y holocrones estable.

## Requirements

### Requirement: Binario Go estático y versión

La distribución de release DEBE (`MUST`) ser un ejecutable Go estático para macOS Apple Silicon (`darwin/arm64`) que no requiera intérpretes, gestores de paquetes externos ni una instalación previa. El ejecutable DEBE (`MUST`) exponer `--version` con el formato `kamino <versión>` y `--help` sin cargar el catálogo ni la configuración de la TUI.

#### Scenario: Ejecución sin dependencias

- **GIVEN** un macOS Apple Silicon sin intérpretes ni gestores de paquetes instalados
- **WHEN** se ejecuta el binario de release
- **THEN** `kamino --version` y `kamino --help` funcionan y no inician ningún proceso externo

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

La release DEBE (`MUST`) publicar el asset `kamino-arm64` con la versión inyectada y un SHA-256 verificable. `scripts/install.sh` y `kamino update` DEBEN (`MUST`) seguir usando el mismo nombre de asset, y la instalación NO DEBE (`MUST NOT`) requerir `sudo`. El binario autocontenido es la única vía de instalación.

#### Scenario: Instalación con el script

- **GIVEN** macOS Apple Silicon y una release publicada
- **WHEN** se ejecuta `scripts/install.sh`
- **THEN** descarga `kamino-arm64`, lo instala en `~/.local/bin/kamino` y no usa `sudo`

#### Scenario: Asset y digest publicados

- **GIVEN** una release creada por `cmd/release`
- **WHEN** se consulta el asset `kamino-arm64` en GitHub
- **THEN** el asset existe, tiene el SHA-256 publicado y la versión del binario coincide con el tag

#### Scenario: Plataforma no soportada

- **GIVEN** un sistema que no es macOS Apple Silicon
- **WHEN** se intenta instalar o actualizar
- **THEN** la operación falla sin descargar ni escribir el binario

### Requirement: Actualización del binario autocontenido

Un binario Go de release DEBE (`MUST`) poder actualizarse a sí mismo, siempre que la versión publicada sea superior y el tamaño y el SHA-256 coincidan. Una construcción de desarrollo (`go run` o binario no marcado como autocontenido) NO DEBE (`MUST NOT`) descargar ni reemplazar nada.

#### Scenario: El binario se actualiza a sí mismo

- **GIVEN** un binario Go de release con una versión anterior a la publicada
- **WHEN** se ejecuta `kamino update`
- **THEN** descarga, verifica y reemplaza el ejecutable real resolviendo los symlinks

#### Scenario: Construcción de desarrollo

- **GIVEN** un binario Go construido con `go run` o sin la marca de autocontenido
- **WHEN** se ejecuta `kamino update`
- **THEN** el comando falla con la vía de actualización correspondiente y no escribe nada

### Requirement: Compatibilidad del formato de clones y holocrones

El binario DEBE (`MUST`) leer y escribir clones y holocrones con `format_version = 1`, usando rutas POSIX relativas al HOME y validando el manifiesto y la integridad antes de escribir.

#### Scenario: Clon desplegado en otro HOME

- **GIVEN** un clon generado en un HOME temporal
- **WHEN** el binario lo despliega en otro HOME temporal
- **THEN** los archivos, enlaces, modos y categorías del resultado se aplican correctamente

#### Scenario: Holocrón restaurado

- **GIVEN** un holocrón generado por el binario
- **WHEN** se despliega en un HOME temporal
- **THEN** el holocrón se reconoce como la app sintética `snapshot` y restaura el estado anterior
