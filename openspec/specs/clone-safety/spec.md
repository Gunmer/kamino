# clone-safety Specification

## Purpose

Reducir la exposición accidental de credenciales y garantizar que las operaciones sobre configuraciones permanezcan dentro del HOME sin debilitar la recuperación existente.

## Requirements

### Requirement: Exclusión de credenciales independientes conocidas

Las nuevas definiciones integradas, las definiciones integradas ampliadas y las referencias descubiertas DEBEN (`MUST`) excluir los archivos de credenciales independientes conocidos para cada herramienta. La exclusión DEBE aplicarse antes de leer su contenido.

#### Scenario: Credenciales en la configuración de plugins de Herdr

- **GIVEN** la configuración global de plugins de Herdr contiene `.env` y un archivo de ajustes no excluido
- **WHEN** se planifica y escribe el clon
- **THEN** se excluye `.env` y se conserva el archivo de ajustes permitido.

#### Scenario: Almacenes de autenticación de herramientas

- **GIVEN** existen `auth.json` de Pi, `secrets.yaml` de Goose o `hosts.yml` de autenticación de GitHub CLI junto a configuraciones permitidas
- **WHEN** se clonan esas herramientas
- **THEN** se excluyen esos archivos de autenticación y se conservan las configuraciones permitidas.

#### Scenario: Referencia explícita a una credencial

- **GIVEN** un archivo de configuración de Ghostty referencia una credencial independiente conocida dentro del HOME
- **WHEN** se descubren las referencias
- **THEN** no se lee ni se incluye el contenido de esa credencial y se informa que la referencia no se clonó.

### Requirement: Exclusión de historiales y cachés

Las nuevas definiciones integradas y las rutas descubiertas NO DEBEN (`MUST NOT`) incluir historiales, sesiones, cachés ni bases de datos de memoria como parte de la configuración global.

#### Scenario: Configuración junto a historial

- **GIVEN** una herramienta tiene un archivo de ajustes, un historial de conversaciones y una caché bajo su directorio global
- **WHEN** se clona esa herramienta
- **THEN** se incluye el archivo de ajustes permitido y se excluyen el historial y la caché.

### Requirement: Aviso de contenido sensible

Antes de ejecutar un clon, la CLI DEBE (`MUST`) advertir que el archivo no está cifrado y que las configuraciones incluidas pueden contener secretos. El aviso DEBE aparecer también en modo no interactivo y en simulación, sin mostrar valores secretos ni pedir confirmación adicional.

#### Scenario: Clonación no interactiva

- **GIVEN** se ejecuta un clon sin terminal interactiva
- **WHEN** se prepara el clon
- **THEN** se muestra el aviso antes de crear el archivo y el comando conserva sus reglas de selección y códigos de salida.

#### Scenario: Secreto incrustado en ajustes

- **GIVEN** una configuración permitida contiene una clave incrustada y no es un archivo independiente excluido
- **WHEN** se clona la herramienta
- **THEN** se conserva el archivo íntegro y se advierte del riesgo sin imprimir ni redactar la clave.

### Requirement: Confinamiento del clon al HOME

La detección por marcadores, la planificación y la lectura de configuraciones para clonar NO DEBEN (`MUST NOT`) atravesar enlaces simbólicos en directorios intermedios bajo el HOME ni leer configuraciones fuera de este. Los enlaces finales solo se admiten con los destinos relativos seguros ya soportados.

#### Scenario: Enlace intermedio hacia fuera del HOME

- **GIVEN** un directorio intermedio de una ruta seleccionada o referenciada es un enlace hacia una ubicación externa al HOME
- **WHEN** se detecta o planifica el clon
- **THEN** no se lee el destino externo y se informa de la ruta no clonada.

#### Scenario: Enlace final relativo seguro

- **GIVEN** una ruta seleccionada termina en un enlace relativo sin componentes `..`
- **WHEN** se clona
- **THEN** se conserva el enlace sin seguirlo, como en el formato existente.

### Requirement: Confinamiento del despliegue al HOME

La clasificación, instantánea y escritura de destinos NO DEBEN (`MUST NOT`) atravesar enlaces simbólicos en directorios intermedios bajo el HOME ni modificar archivos fuera de este, incluso si un directorio cambia durante la operación.

#### Scenario: Destino con un enlace intermedio inseguro

- **GIVEN** un clon válido apunta a un destino cuyo directorio intermedio es un enlace fuera del HOME
- **WHEN** se solicita el despliegue
- **THEN** se rechaza la operación antes de crear una instantánea o modificar destinos.

#### Scenario: Cambio de directorio durante el despliegue

- **GIVEN** un directorio de destino se sustituye por un enlace después de planificar el despliegue
- **WHEN** el despliegue intenta aplicar los cambios
- **THEN** no se lee ni se modifica ningún archivo fuera del HOME y se informa del fallo.

### Requirement: Orden y compatibilidad del despliegue

Kamino DEBE (`MUST`) conservar la validación del manifiesto, la verificación completa antes de escribir destinos, la instantánea previa y los reemplazos atómicos. Las simulaciones NO DEBEN escribir y los clones de formato 1 anteriores DEBEN seguir siendo legibles.

#### Scenario: Clon alterado

- **GIVEN** un clon tiene un checksum incorrecto para un archivo
- **WHEN** se solicita un despliegue real
- **THEN** se rechaza antes de crear una instantánea o modificar cualquier destino.

#### Scenario: Simulación y clon anterior

- **GIVEN** un clon válido de formato 1 creado antes de este cambio
- **WHEN** se simula su despliegue
- **THEN** se genera el plan sin escribir destinos ni instantáneas.

#### Scenario: Instantánea completa al desplegar un clon anterior

- **GIVEN** un clon anterior contiene un archivo que ahora se excluye de los nuevos clones y su destino actual existe
- **WHEN** se despliega ese archivo
- **THEN** la instantánea previa conserva el contenido actual del destino antes de reemplazarlo.
