# backup-safety Specification

## Purpose

Reducir la exposición accidental de credenciales y garantizar que las operaciones sobre configuraciones permanezcan dentro del HOME sin debilitar la recuperación existente.

## Requirements

### Requirement: Exclusión de credenciales independientes conocidas

Las nuevas definiciones integradas, las definiciones integradas ampliadas y las referencias descubiertas DEBEN (`MUST`) excluir los archivos de credenciales independientes conocidos para cada herramienta. La exclusión DEBE aplicarse antes de leer su contenido.

#### Scenario: Credenciales en la configuración de plugins de Herdr

- **GIVEN** la configuración global de plugins de Herdr contiene `.env` y un archivo de ajustes no excluido
- **WHEN** se planifica y escribe el respaldo
- **THEN** se excluye `.env` y se conserva el archivo de ajustes permitido.

#### Scenario: Almacenes de autenticación de herramientas

- **GIVEN** existen `auth.json` de Pi, `secrets.yaml` de Goose o `hosts.yml` de autenticación de GitHub CLI junto a configuraciones permitidas
- **WHEN** se respaldan esas herramientas
- **THEN** se excluyen esos archivos de autenticación y se conservan las configuraciones permitidas.

#### Scenario: Referencia explícita a una credencial

- **GIVEN** un archivo de configuración de Ghostty referencia una credencial independiente conocida dentro del HOME
- **WHEN** se descubren las referencias
- **THEN** no se lee ni se incluye el contenido de esa credencial y se informa que la referencia no se respaldó.

### Requirement: Exclusión de historiales y cachés

Las nuevas definiciones integradas y las rutas descubiertas NO DEBEN (`MUST NOT`) incluir historiales, sesiones, cachés ni bases de datos de memoria como parte de la configuración global.

#### Scenario: Configuración junto a historial

- **GIVEN** una herramienta tiene un archivo de ajustes, un historial de conversaciones y una caché bajo su directorio global
- **WHEN** se respalda esa herramienta
- **THEN** se incluye el archivo de ajustes permitido y se excluyen el historial y la caché.

### Requirement: Aviso de contenido sensible

Antes de ejecutar un respaldo, la CLI DEBE (`MUST`) advertir que el archivo no está cifrado y que las configuraciones incluidas pueden contener secretos. El aviso DEBE aparecer también en modo no interactivo y en simulación, sin mostrar valores secretos ni pedir confirmación adicional.

#### Scenario: Respaldo no interactivo

- **GIVEN** se ejecuta un respaldo sin terminal interactiva
- **WHEN** se prepara el respaldo
- **THEN** se muestra el aviso antes de crear el archivo y el comando conserva sus reglas de selección y códigos de salida.

#### Scenario: Secreto incrustado en ajustes

- **GIVEN** una configuración permitida contiene una clave incrustada y no es un archivo independiente excluido
- **WHEN** se respalda la herramienta
- **THEN** se conserva el archivo íntegro y se advierte del riesgo sin imprimir ni redactar la clave.

### Requirement: Confinamiento del respaldo al HOME

La detección por marcadores, la planificación y la lectura de configuraciones para respaldar NO DEBEN (`MUST NOT`) atravesar enlaces simbólicos en directorios intermedios bajo el HOME ni leer configuraciones fuera de este. Los enlaces finales solo se admiten con los destinos relativos seguros ya soportados.

#### Scenario: Enlace intermedio hacia fuera del HOME

- **GIVEN** un directorio intermedio de una ruta seleccionada o referenciada es un enlace hacia una ubicación externa al HOME
- **WHEN** se detecta o planifica el respaldo
- **THEN** no se lee el destino externo y se informa de la ruta no respaldada.

#### Scenario: Enlace final relativo seguro

- **GIVEN** una ruta seleccionada termina en un enlace relativo sin componentes `..`
- **WHEN** se respalda
- **THEN** se conserva el enlace sin seguirlo, como en el formato existente.

### Requirement: Confinamiento de la restauración al HOME

La clasificación, instantánea y escritura de destinos NO DEBEN (`MUST NOT`) atravesar enlaces simbólicos en directorios intermedios bajo el HOME ni modificar archivos fuera de este, incluso si un directorio cambia durante la operación.

#### Scenario: Destino con un enlace intermedio inseguro

- **GIVEN** un respaldo válido apunta a un destino cuyo directorio intermedio es un enlace fuera del HOME
- **WHEN** se solicita la restauración
- **THEN** se rechaza la operación antes de crear una instantánea o modificar destinos.

#### Scenario: Cambio de directorio durante la restauración

- **GIVEN** un directorio de destino se sustituye por un enlace después de planificar la restauración
- **WHEN** la restauración intenta aplicarla
- **THEN** no se lee ni se modifica ningún archivo fuera del HOME y se informa del fallo.

### Requirement: Orden y compatibilidad de restauración

Kamino DEBE (`MUST`) conservar la validación del manifiesto, la verificación completa antes de escribir destinos, la instantánea previa y los reemplazos atómicos. Las simulaciones NO DEBEN escribir y los respaldos de formato 1 anteriores DEBEN seguir siendo legibles.

#### Scenario: Respaldo alterado

- **GIVEN** un respaldo tiene un checksum incorrecto para un archivo
- **WHEN** se solicita una restauración real
- **THEN** se rechaza antes de crear una instantánea o modificar cualquier destino.

#### Scenario: Simulación y respaldo anterior

- **GIVEN** un respaldo válido de formato 1 creado antes de este cambio
- **WHEN** se simula su restauración
- **THEN** se genera el plan sin escribir destinos ni instantáneas.

#### Scenario: Instantánea completa al restaurar un respaldo anterior

- **GIVEN** un respaldo anterior contiene un archivo que ahora se excluye de los nuevos respaldos y su destino actual existe
- **WHEN** se restaura ese archivo
- **THEN** la instantánea previa conserva el contenido actual del destino antes de reemplazarlo.
