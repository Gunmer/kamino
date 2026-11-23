# Spec Delta

## Purpose

Permitir que Kamino instale, con confirmación y por un conjunto cerrado de métodos, las apps ausentes al implantar un clon, sin que el clon pueda dictar qué se descarga ni ejecutar.

## ADDED Requirements

### Requirement: Recetas de instalación en el catálogo

Una definición del catálogo PUEDE (`MAY`) declarar una lista ordenada de recetas `[[install]]`. Cada receta DEBE (`MUST`) usar exactamente uno de estos métodos: `brew`, `installer` o `binary`. Cualquier otro método DEBE (`MUST`) rechazarse al cargar la definición.

#### Scenario: Método no permitido

- **GIVEN** una definición declara `method = "script"`
- **WHEN** se carga el catálogo
- **THEN** la definición se rechaza con un error en español que nombra los tres métodos permitidos.

#### Scenario: Definición sin recetas

- **GIVEN** una definición no declara `[[install]]`
- **WHEN** esa app está ausente al implantar
- **THEN** se informa «Sin base en este Mac» y no se ofrece instalarla.

### Requirement: El clon no aporta recetas

El manifiesto y el contenido del clon NO DEBEN (`MUST NOT`) aportar URLs, comandos ni métodos de instalación. Solo el catálogo local DEBE (`MUST`) determinar qué se instala y desde dónde.

#### Scenario: Clon con datos de instalación

- **GIVEN** un clon cuyo manifiesto incluye campos de instalación o una URL
- **WHEN** se implanta con `--install-missing`
- **THEN** esos campos se ignoran y solo se usa la receta del catálogo local.

### Requirement: Filtro por plataforma

Cada receta PUEDE (`MAY`) declarar `os` y `arch`. Kamino DEBE (`MUST`) ignorar las recetas que no coincidan con el entorno actual. Si ninguna receta aplica, la app NO DEBE (`MUST NOT`) ofrecerse para instalar.

#### Scenario: Receta de otra arquitectura

- **GIVEN** una receta con `arch = "amd64"` en un entorno `arm64`
- **WHEN** se calculan las apps instalables
- **THEN** esa receta se ignora y, sin otra aplicable, la app no se ofrece.

### Requirement: Método brew

El método `brew` DEBE (`MUST`) instalar la fórmula o cask indicada con Homebrew. Si `brew` no está en el `PATH`, Kamino DEBE (`MUST`) pasar a la siguiente receta aplicable y NO DEBE (`MUST NOT`) instalar Homebrew.

#### Scenario: Homebrew ausente con receta alternativa

- **GIVEN** una app con recetas `brew` y `binary` y sin `brew` en el `PATH`
- **WHEN** se instala la app
- **THEN** se usa la receta `binary`.

#### Scenario: Homebrew ausente sin alternativa

- **GIVEN** una app con solo receta `brew` y sin `brew` en el `PATH`
- **WHEN** se instala la app
- **THEN** la app se informa como omitida y no se ejecuta ninguna instalación.

### Requirement: Método binary

El método `binary` DEBE (`MUST`) exigir una URL `https` y un `sha256` válido. Kamino DEBE (`MUST`) descargar con curl a un temporal, verificar el SHA-256 y solo entonces moverlo de forma atómica a un destino bajo el HOME con permisos 0755. Si la verificación falla, NO DEBE (`MUST NOT`) instalarse nada.

#### Scenario: Definición sin sha256

- **GIVEN** una receta `binary` sin `sha256`
- **WHEN** se carga el catálogo
- **THEN** la definición se rechaza con un error en español.

#### Scenario: Suma de verificación incorrecta

- **GIVEN** un binario descargado cuyo SHA-256 no coincide con el declarado
- **WHEN** se instala la app
- **THEN** se elimina el temporal, no se escribe el destino y la app se informa como omitida.

### Requirement: Método installer

El método `installer` DEBE (`MUST`) exigir una URL `https`. Kamino DEBE (`MUST`) descargar el script con curl a un temporal y ejecutarlo con `sh`, sin canalizar la descarga a un intérprete. La confirmación DEBE (`MUST`) mostrar la URL exacta antes de ejecutar.

#### Scenario: URL no HTTPS

- **GIVEN** una receta `installer` con una URL `http`
- **WHEN** se carga el catálogo
- **THEN** la definición se rechaza con un error en español.

#### Scenario: Confirmación con la URL

- **GIVEN** una app ausente con receta `installer`
- **WHEN** se pide confirmar la instalación
- **THEN** la confirmación muestra la URL exacta que se descargará.

### Requirement: Orden de la instalación en el despliegue

La instalación DEBE (`MUST`) ejecutarse después de validar el manifiesto y verificar el archivo completo, y antes de crear la instantánea y aplicar los reemplazos. Con `--dry-run` NO DEBE (`MUST NOT`) ejecutarse ninguna instalación ni escribirse nada.

#### Scenario: Archivo inválido

- **GIVEN** un clon que no supera la verificación
- **WHEN** se implanta con `--install-missing`
- **THEN** no se instala ninguna app y no se escribe ningún destino.

#### Scenario: Simulación

- **GIVEN** una app ausente con receta aplicable
- **WHEN** se implanta con `--dry-run --install-missing`
- **THEN** se informa qué se instalaría, sin descargar ni ejecutar nada.

### Requirement: Fallo de instalación

Si la instalación de una app falla, esa app DEBE (`MUST`) quedar como omitida, el despliegue DEBE (`MUST`) continuar con el resto y el código de salida DEBE (`MUST`) ser 3 si no hay otro error.

#### Scenario: Una de dos instalaciones falla

- **GIVEN** dos apps ausentes instalables y una instalación que falla
- **WHEN** se implanta con `--install-missing --yes`
- **THEN** la app instalada recibe su configuración, la fallida se informa como omitida y el código de salida es 3.

#### Scenario: App instalada tras éxito

- **GIVEN** una app ausente cuya instalación termina bien
- **WHEN** continúa el despliegue
- **THEN** la app se trata como instalada y su configuración se despliega con las reglas habituales.
