# Spec Delta

## Purpose

Mantener actualizado el binario autocontenido de kamino desde las releases oficiales de GitHub, sin reinstalar a mano ni usar privilegios elevados.

## ADDED Requirements

### Requirement: Actualización exclusiva del binario instalado

`kamino update` DEBE (`MUST`) actualizar únicamente un binario autocontenido en ejecución. En instalaciones desde código, `uv` o `pip` DEBE (`MUST`) fallar con un mensaje que indique la vía de actualización correspondiente, sin descargar ni escribir nada.

#### Scenario: Binario autocontenido

- **WHEN** se ejecuta `kamino update` desde el binario autocontenido
- **THEN** el comando actualiza el archivo ejecutable en ejecución

#### Scenario: Instalación desde código

- **WHEN** se ejecuta `kamino update` desde una instalación de código, `uv` o `pip`
- **THEN** el comando termina con error e indica la vía de actualización correspondiente
- **AND** no descarga ni escribe ningún archivo

### Requirement: Destino del reemplazo

La actualización DEBE (`MUST`) reemplazar el archivo ejecutable real del proceso, resolviendo los symlinks que apunten a él, y NO DEBE (`MUST NOT`) modificar otros archivos de la instalación.

#### Scenario: Ejecutable alcanzado mediante symlink

- **GIVEN** un binario instalado detrás de un symlink
- **WHEN** se ejecuta `kamino update`
- **THEN** se reemplaza el archivo real y el symlink sigue apuntando a él

#### Scenario: Otros archivos de la instalación

- **WHEN** termina una actualización correcta
- **THEN** solo cambia el binario actualizado y no se modifican configuraciones, respaldos ni instantáneas

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

La escritura DEBE (`MUST`) usar un archivo temporal en el mismo directorio que el binario y un reemplazo atómico. Ante cualquier error, el binario actual DEBE (`MUST`) quedar intacto y el temporal DEBE (`MUST`) eliminarse.

#### Scenario: Actualización correcta

- **WHEN** la descarga y la verificación terminan correctamente
- **THEN** el binario se reemplaza de forma atómica y el temporal desaparece

#### Scenario: Fallo durante la actualización

- **WHEN** falla la descarga, la verificación o el reemplazo
- **THEN** el binario anterior sigue siendo ejecutable
- **AND** el temporal se elimina

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

El comando DEBE (`MUST`) informar de la versión actual, la versión encontrada y la ruta actualizada. DEBE (`MUST`) terminar con código 0 cuando actualiza o ya está al día, y con código 1 ante cualquier error.

#### Scenario: Actualización completada

- **WHEN** el comando actualiza el binario
- **THEN** informa de la versión anterior, la nueva y la ruta actualizada
- **AND** termina con código 0

#### Scenario: Sin actualización

- **WHEN** el comando determina que la versión instalada es la última
- **THEN** informa de ello y termina con código 0

#### Scenario: Error

- **WHEN** el comando no puede consultar, descargar, verificar o reemplazar el binario
- **THEN** informa del error y termina con código 1

### Requirement: Descubrimiento en el menú

El menú que aparece al ejecutar `kamino` sin argumentos en una terminal interactiva DEBE (`MUST`) incluir la opción de actualización, que ejecuta el mismo flujo que `kamino update`.

#### Scenario: Menú principal

- **WHEN** se ejecuta `kamino` sin argumentos en una terminal interactiva
- **THEN** el menú incluye una opción de actualización

#### Scenario: Selección de la actualización

- **WHEN** se elige la opción de actualización en el menú
- **THEN** se ejecuta el mismo flujo que `kamino update`
