# ghostty-config-references Specification

## Purpose

Conservar las configuraciones, los atajos y los temas globales de Ghostty que estén dentro del HOME, incluso cuando se distribuyen entre varios archivos relacionados.

## Requirements

### Requirement: Archivos y temas globales de Ghostty

Kamino DEBE (`MUST`) contemplar las variantes `config` y `config.ghostty` de las ubicaciones globales de Ghostty en macOS y sus temas personalizados bajo el HOME.

#### Scenario: Variantes de configuración y tema personalizado

- **GIVEN** Ghostty está instalado y existen ambas variantes de configuración en los directorios globales XDG y de soporte de aplicaciones de macOS, además de un tema personalizado
- **WHEN** se clona y despliega Ghostty
- **THEN** se conservan todos los archivos existentes, incluidos los atajos de sus configuraciones y el tema, sin alterar sus contenidos.

#### Scenario: Tema integrado de Ghostty

- **GIVEN** la configuración selecciona un tema integrado de Ghostty y no hay un archivo de tema personalizado con ese nombre
- **WHEN** se clona Ghostty
- **THEN** se conserva la selección en la configuración sin copiar recursos del paquete de la aplicación ni registrar una omisión por el tema integrado.

#### Scenario: Directorio XDG personalizado dentro del HOME

- **GIVEN** `XDG_CONFIG_HOME` señala un directorio dentro del HOME que contiene la configuración global de Ghostty
- **WHEN** se clona Ghostty
- **THEN** se incluyen las variantes existentes y los temas propios de ese directorio, sin recorrer ubicaciones fuera del HOME.

### Requirement: Referencias de archivos de configuración

Kamino DEBE (`MUST`) incluir los archivos globales referenciados mediante `config-file` que se encuentren dentro del HOME y descubrir sus referencias anidadas sin ejecutar su contenido.

#### Scenario: Referencia relativa anidada

- **GIVEN** una configuración global referencia un archivo relativo y este referencia otro archivo dentro del HOME
- **WHEN** se planifica y ejecuta el clon
- **THEN** se incluyen ambos archivos usando las rutas relativas al archivo que los referencia.

#### Scenario: Referencia opcional ausente

- **GIVEN** un archivo de Ghostty referencia mediante `?` una configuración opcional que no existe
- **WHEN** se clona la aplicación
- **THEN** la ausencia no se informa como omisión ni causa por sí sola el código de salida 3.

#### Scenario: Ciclo de referencias

- **GIVEN** dos archivos de configuración de Ghostty se incluyen mutuamente
- **WHEN** se planifica el clon
- **THEN** la exploración termina, conserva los archivos válidos una sola vez e informa del ciclo sin ejecutar instrucciones del archivo.

### Requirement: Referencias de temas personalizados

Kamino DEBE (`MUST`) incluir los temas personalizados referenciados por la configuración global de Ghostty cuando se encuentren dentro del HOME y no estén excluidos por seguridad.

#### Scenario: Temas claros y oscuros

- **GIVEN** la configuración de Ghostty referencia temas personalizados para los modos claro y oscuro dentro del HOME
- **WHEN** se clona Ghostty
- **THEN** se incluyen ambos archivos de tema y la configuración que los selecciona.

#### Scenario: Referencia fuera del HOME

- **GIVEN** una configuración de Ghostty referencia un tema o un archivo situado fuera del HOME
- **WHEN** se planifica el clon
- **THEN** no se lee ni se incluye el archivo externo y se informa de la omisión.

### Requirement: Referencias sin duplicados

Cada archivo descubierto para Ghostty DEBE (`MUST`) figurar una sola vez por aplicación en el plan, el manifiesto y el archivo de clon, aunque coincidan rutas declaradas y referencias.

#### Scenario: Referencia cubierta por un directorio

- **GIVEN** un archivo de tema ya está cubierto por una ruta del catálogo y también está referenciado desde la configuración
- **WHEN** se escribe y verifica el clon
- **THEN** se registra una sola entrada para ese archivo y la verificación permite desplegarlo.

### Requirement: Advertencia por rutas absolutas no portables

Kamino DEBE (`MUST`) conservar los contenidos originales de Ghostty y advertir al desplegar cuando una referencia absoluta pueda seguir apuntando al HOME de origen.

#### Scenario: Despliegue en otro HOME

- **GIVEN** una configuración clonada contiene una referencia absoluta dentro del HOME de origen y el HOME de destino es distinto
- **WHEN** se planifica o ejecuta el despliegue de Ghostty
- **THEN** se advierte que la referencia puede requerir ajuste manual.
- **AND** el contenido despliegado no se reescribe ni se modifica para cambiar la ruta.
