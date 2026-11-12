# self-update Specification

## Purpose

Mantener actualizado el binario autocontenido de kamino desde las releases oficiales de GitHub, sin reinstalar a mano ni usar privilegios elevados.

## Requirements

### Requirement: Actualización exclusiva del binario instalado

`kamino update` DEBE (`MUST`) actualizar únicamente un binario autocontenido en ejecución. Un binario Go de release DEBE (`MUST`) considerarse autocontenido. Una instalación desde código, `go run` o un binario Go sin la marca de autocontenido DEBE (`MUST`) fallar con un mensaje que indique la vía de actualización correspondiente, sin descargar ni escribir nada. Las instalaciones con `uv` o `pip` dejan de estar soportadas.

#### Scenario: Binario autocontenido

- **WHEN** se ejecuta `kamino update` desde el binario autocontenido, Go o Python legado
- **THEN** el comando actualiza el archivo ejecutable en ejecución

#### Scenario: Instalación desde código

- **WHEN** se ejecuta `kamino update` desde una instalación de código o un binario Go sin marca de autocontenido
- **THEN** el comando termina con error e indica la vía de actualización correspondiente
- **AND** no descarga ni escribe ningún archivo

#### Scenario: Construcción de desarrollo Go

- **GIVEN** un binario construido con `go run` o sin la marca de autocontenido
- **WHEN** se ejecuta `kamino update`
- **THEN** el comando termina con error y no descarga ni reemplaza el ejecutable

### Requirement: Destino del reemplazo

La actualización DEBE (`MUST`) reemplazar el archivo ejecutable real del proceso, resolviendo los symlinks que apunten a él, y NO DEBE (`MUST NOT`) modificar otros archivos de la instalación.

#### Scenario: Ejecutable alcanzado mediante symlink

- **GIVEN** un binario instalado detrás de un symlink
- **WHEN** se ejecuta `kamino update`
- **THEN** se reemplaza el archivo real y el symlink sigue apuntando a él

#### Scenario: Otros archivos de la instalación

- **WHEN** termina una actualización correcta
- **THEN** solo cambia el binario actualizado y no se modifican configuraciones, clones ni instantáneas

### Requirement: Consulta de la última release y comparación de versiones

El comando DEBE (`MUST`) consultar la última release publicada del repositorio oficial en GitHub y comparar su versión con la instalada usando el orden SemVer. Si la instalada es igual o superior, NO DEBE (`MUST NOT`) descargar ni escribir nada.

#### Scenario: Actualización disponible

- **GIVEN** una versión instalada anterior a la última publicada
- **WHEN** se ejecuta `kamino update`
- **THEN** el comando descarga e instala la versión publicada

#### Scenario: Versión ya actualizada

- **GIVEN** una versión instalada igual a la última publicada
- **WHEN** se ejecuta `kamino update`
- **THEN** el comando informa de que ya está al día
- **AND** no descarga ni escribe nada

#### Scenario: Versión instalada más reciente

- **GIVEN** una versión instalada posterior a la última publicada
- **WHEN** se ejecuta `kamino update`
- **THEN** el comando no degrada la instalación ni descarga el asset

### Requirement: Verificación de integridad

Antes de reemplazar el binario, el comando DEBE (`MUST`) verificar que el asset descargado coincide con el tamaño y el SHA-256 publicados por GitHub para la release. NO DEBE (`MUST NOT`) ejecutar el binario descargado antes de reemplazarlo.

#### Scenario: Asset verificado

- **WHEN** el tamaño y el SHA-256 del asset coinciden con los publicados
- **THEN** el comando continúa con el reemplazo

#### Scenario: Digest ausente o incorrecto

- **WHEN** el tamaño no coincide, el digest falta o el SHA-256 no coincide
- **THEN** el comando aborta con error
- **AND** el binario actual queda intacto

### Requirement: Reemplazo atómico y recuperación

La escritura DEBE (`MUST`) usar un archivo temporal en el mismo directorio que el binario y un reemplazo atómico. Ante cualquier error o cancelación antes del reemplazo, el binario actual DEBE (`MUST`) quedar intacto y el temporal DEBE (`MUST`) eliminarse. Una vez iniciado el reemplazo, la operación NO DEBE interrumpirse.

#### Scenario: Actualización correcta

- **WHEN** la descarga y la verificación terminan correctamente
- **THEN** el binario se reemplaza de forma atómica y el temporal desaparece

#### Scenario: Fallo durante la actualización

- **WHEN** falla la descarga, la verificación o el reemplazo
- **THEN** el binario anterior sigue siendo ejecutable y el temporal se elimina

#### Scenario: Cancelación antes del reemplazo

- **GIVEN** una descarga o verificación en curso
- **WHEN** se cancela
- **THEN** el binario actual sigue intacto y el temporal se elimina

#### Scenario: Cancelación durante el reemplazo

- **GIVEN** un reemplazo ya iniciado
- **WHEN** se intenta cancelar
- **THEN** el reemplazo termina y la pantalla indica que ya no se puede cancelar

### Requirement: Compatibilidad de plataforma

La actualización DEBE (`MUST`) rechazar sistemas que no sean macOS sobre Apple Silicon (`arm64`), igual que el instalador, sin descargar el asset.

#### Scenario: Plataforma no soportada

- **WHEN** se ejecuta `kamino update` en un sistema que no es macOS arm64
- **THEN** el comando termina con error y no descarga ni escribe nada

### Requirement: Sin privilegios elevados

La actualización NO DEBE (`MUST NOT`) requerir `sudo` ni pedir confirmación adicional: ejecutar `kamino update` es la confirmación explícita.

#### Scenario: Actualización sin privilegios

- **WHEN** se ejecuta `kamino update` como el usuario propietario del binario
- **THEN** la actualización se completa sin solicitar contraseña ni privilegios elevados

### Requirement: Salida y códigos de salida

El comando DEBE (`MUST`) informar de la versión actual, la versión encontrada y la ruta actualizada, o de que ya está al día o tiene una versión más reciente. DEBE (`MUST`) terminar con código 0 cuando actualiza o ya está al día, y con código 1 ante cualquier error o cancelación.

#### Scenario: Actualización completada

- **WHEN** el comando actualiza el binario
- **THEN** informa de la versión anterior, la nueva y la ruta actualizada, y termina con código 0

#### Scenario: Sin actualización

- **WHEN** el comando determina que la versión instalada es la última
- **THEN** informa de ello y termina con código 0

#### Scenario: Versión instalada más reciente

- **WHEN** la versión instalada es posterior a la última publicada
- **THEN** informa de ello, no descarga ni escribe y termina con código 0

#### Scenario: Error

- **WHEN** el comando no puede consultar, descargar, verificar o reemplazar el binario
- **THEN** informa del error y termina con código 1

### Requirement: Descubrimiento en el menú

El menú que aparece al ejecutar `kamino` sin argumentos en una terminal interactiva DEBE (`MUST`) incluir la opción de actualización, que ejecuta el mismo flujo que `kamino update` en la TUI U1. Al terminar, DEBE volver al menú y actualizar la última misión.

#### Scenario: Menú principal

- **WHEN** se ejecuta `kamino` sin argumentos en una terminal interactiva
- **THEN** el menú incluye una opción de actualización

#### Scenario: Selección de la actualización

- **WHEN** se elige la opción de actualización en el menú
- **THEN** se ejecuta el mismo flujo que `kamino update`, con sus estados y resultados, y se vuelve al menú al terminar

### Requirement: Progreso y cancelación en la TUI

U1 DEBE (`MUST`) mostrar los estados consultando, descargando y verificando tamaño y SHA-256, reemplazando y resultado. La operación DEBE poder cancelarse hasta el reemplazo; al cancelar, el binario actual DEBE quedar intacto y el temporal eliminado. Tras actualizar, DEBE pedirse reiniciar porque el proceso en marcha sigue siendo el anterior.

#### Scenario: Consulta de la última versión

- **GIVEN** U1 abierto
- **WHEN** se consulta la última release
- **THEN** se muestra el estado “consultando” y, si no hay actualización, el resultado sin descargar

#### Scenario: Descarga y verificación

- **GIVEN** una actualización disponible
- **WHEN** se descarga el asset
- **THEN** se muestra el progreso y se verifica el tamaño y el SHA-256 antes de reemplazar

#### Scenario: Cancelar antes del reemplazo

- **GIVEN** una descarga o verificación en curso
- **WHEN** se cancela
- **THEN** el binario actual sigue intacto, el temporal se elimina y se vuelve al menú o se sale con código 1

#### Scenario: Reinicio tras actualizar

- **GIVEN** una actualización completada
- **WHEN** se muestra el resultado
- **THEN** se pide reiniciar kamino para usar la versión nueva

### Requirement: Errores de actualización

U1 DEBE (`MUST`) presentar el error común para una instalación no autocontenida, una plataforma no soportada, un fallo de red o un fallo de integridad. En la instalación no autocontenida DEBE indicar la vía de actualización correspondiente y NO DEBE descargar ni escribir.

#### Scenario: Instalación no autocontenida

- **GIVEN** una instalación desde código o un binario Go sin marca de autocontenido
- **WHEN** se ejecuta `kamino update`
- **THEN** se muestra la vía de actualización correspondiente y no se descarga ni escribe nada

#### Scenario: Plataforma no soportada

- **GIVEN** un sistema que no es macOS arm64
- **WHEN** se ejecuta `kamino update`
- **THEN** se muestra el error y no se descarga ni escribe nada

#### Scenario: Fallo de red o integridad

- **GIVEN** un fallo al consultar, descargar o verificar
- **WHEN** termina la operación
- **THEN** se muestra el error común, el binario actual sigue intacto y el temporal se elimina
