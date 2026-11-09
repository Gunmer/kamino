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

Antes de ejecutar un clon, la CLI DEBE (`MUST`) advertir que el archivo no está cifrado y que las configuraciones incluidas pueden contener secretos. El aviso DEBE aparecer también en modo no interactivo, en simulación y en la revisión C3 de la TUI, sin mostrar valores secretos ni pedir confirmación adicional.

#### Scenario: Clonación no interactiva

- **GIVEN** se ejecuta un clon sin terminal interactiva
- **WHEN** se prepara el clon
- **THEN** se muestra el aviso antes de crear el archivo y el comando conserva sus reglas de selección y códigos de salida.

#### Scenario: Secreto incrustado en ajustes

- **GIVEN** una configuración permitida contiene una clave incrustada y no es un archivo independiente excluido
- **WHEN** se clona la herramienta
- **THEN** se conserva el archivo íntegro y se advierte del riesgo sin imprimir ni redactar la clave.

#### Scenario: Aviso en la revisión de la TUI

- **GIVEN** la revisión C3 con el plan de clonación
- **WHEN** se muestra la pantalla
- **THEN** se advierte de que el clon no está cifrado y puede contener secretos, sin mostrar valores de archivos.

### Requirement: Confinamiento del clon al HOME

La detección por marcadores, la planificación y la lectura de configuraciones para clonar NO DEBEN (`MUST NOT`) atravesar enlaces simbólicos en directorios intermedios bajo el HOME ni leer configuraciones fuera de este. Los enlaces finales solo se admiten con los destinos relativos seguros ya soportados. El archivo de salida del clon PUEDE estar fuera del HOME si el usuario lo indica; eso no habilita leer configuraciones fuera del HOME.

#### Scenario: Enlace intermedio hacia fuera del HOME

- **GIVEN** un directorio intermedio de una ruta seleccionada o referenciada es un enlace hacia una ubicación externa al HOME
- **WHEN** se detecta o planifica el clon
- **THEN** no se lee el destino externo y se informa de la ruta no clonada.

#### Scenario: Enlace final relativo seguro

- **GIVEN** una ruta seleccionada termina en un enlace relativo sin componentes `..`
- **WHEN** se clona
- **THEN** se conserva el enlace sin seguirlo, como en el formato existente.

#### Scenario: Destino de salida fuera del HOME

- **GIVEN** un destino de clonación fuera del HOME indicado explícitamente
- **WHEN** se planifica y escribe el clon
- **THEN** las lecturas de configuración siguen confinadas al HOME y el archivo se escribe en el destino indicado.

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

Kamino DEBE (`MUST`) conservar la validación del manifiesto, la verificación completa antes de escribir destinos, el holocrón de seguridad previo y los reemplazos atómicos. Las simulaciones NO DEBEN escribir y los clones de formato 1 anteriores DEBEN seguir siendo legibles. El holocrón DEBE crearse solo cuando haya al menos un destino existente que proteger.

#### Scenario: Clon alterado

- **GIVEN** un clon tiene un checksum incorrecto para un archivo
- **WHEN** se solicita un despliegue real
- **THEN** se rechaza antes de crear un holocrón o modificar cualquier destino.

#### Scenario: Simulación y clon anterior

- **GIVEN** un clon válido de formato 1 creado antes de este cambio
- **WHEN** se simula su despliegue
- **THEN** se genera el plan sin escribir destinos ni holocrones.

#### Scenario: Instantánea completa al desplegar un clon anterior

- **GIVEN** un clon anterior contiene un archivo que ahora se excluye de los nuevos clones y su destino actual existe
- **WHEN** se despliega ese archivo
- **THEN** el holocrón previo conserva el contenido actual del destino antes de reemplazarlo.

#### Scenario: Sin originales no hay holocrón

- **GIVEN** un despliegue cuyos únicos destinos que cambiarán no existen todavía
- **WHEN** se aplica el despliegue
- **THEN** no se crea un holocrón vacío y se informa de que no había originales que proteger.

### Requirement: Reemplazo explícito y atómico del clon

Un clon existente NO DEBE (`MUST NOT`) reemplazarse sin `--force`. Con `--force`, la escritura DEBE usar un temporal en el mismo directorio y un reemplazo atómico, y DEBE revalidarse que el destino no cambió entre la confirmación y el renombrado. Ante cualquier error o cancelación, el clon anterior DEBE quedar intacto y el temporal DEBE eliminarse.

#### Scenario: Clon existente sin fuerza

- **GIVEN** un clon existente en el destino
- **WHEN** se intenta clonar sin `--force`
- **THEN** el clon anterior no se modifica y el comando informa del conflicto

#### Scenario: Reemplazo con fuerza

- **GIVEN** un clon existente y `--force`
- **WHEN** termina la escritura
- **THEN** el clon anterior se reemplaza de forma atómica y no queda un temporal

#### Scenario: Destino cambiado antes del renombrado

- **GIVEN** un destino que cambia después de la confirmación
- **WHEN** se intenta el renombrado final
- **THEN** la operación se aborta sin reemplazar el destino y se informa del cambio

#### Scenario: Error durante la escritura

- **GIVEN** un fallo al escribir el temporal
- **WHEN** termina la operación
- **THEN** el temporal se elimina y el clon anterior sigue intacto

### Requirement: Cancelación sin efectos del clon

La clonación DEBE (`MUST`) poder cancelarse durante la planificación y la escritura del temporal. Al cancelar, el temporal DEBE eliminarse y el clon anterior DEBE quedar intacto. Una vez iniciado el renombrado final, la cancelación NO DEBE interrumpirlo.

#### Scenario: Cancelar durante la planificación

- **GIVEN** una clonación que aún planifica
- **WHEN** se cancela
- **THEN** no se crea ningún archivo ni temporal

#### Scenario: Cancelar durante la escritura

- **GIVEN** una clonación que escribe el temporal
- **WHEN** se cancela
- **THEN** el temporal se elimina y el clon anterior sigue intacto

#### Scenario: Cancelar durante el renombrado

- **GIVEN** una clonación que ya inició el renombrado final
- **WHEN** se intenta cancelar
- **THEN** la operación termina y la pantalla indica que ya no se puede cancelar
