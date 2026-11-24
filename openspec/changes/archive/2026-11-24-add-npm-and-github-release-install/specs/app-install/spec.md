# Spec Delta

## MODIFIED Requirements

### Requirement: Recetas de instalación en el catálogo

Una definición del catálogo PUEDE (`MAY`) declarar una lista ordenada de recetas `[[install]]`. Cada receta DEBE (`MUST`) usar exactamente uno de estos métodos: `brew`, `npm`, `installer`, `binary` o `github-release`. Cualquier otro método DEBE (`MUST`) rechazarse al cargar la definición.

#### Scenario: Método no permitido

- **GIVEN** una definición declara `method = "script"`
- **WHEN** se carga el catálogo
- **THEN** la definición se rechaza con un error en español que nombra los cinco métodos permitidos.

#### Scenario: Definición sin recetas

- **GIVEN** una definición no declara `[[install]]`
- **WHEN** esa app está ausente al implantar
- **THEN** se informa «Sin base en este Mac» y no se ofrece instalarla.

## ADDED Requirements

### Requirement: Método npm

El método `npm` DEBE (`MUST`) exigir `package` con un nombre de paquete válido y no admitir `url`, `sha256`, `dest` ni `repo`. Kamino DEBE (`MUST`) ejecutar `npm install --global <package>`, que instala la última versión publicada. Si `npm` no está en el `PATH`, la receta DEBE (`MUST`) omitirse y NO DEBE (`MUST NOT`) instalarse Node ni npm.

#### Scenario: npm presente

- **GIVEN** una receta `npm` con `package = "@openai/codex"` y `npm` en el `PATH`
- **WHEN** se instala la app
- **THEN** se ejecuta `npm install --global @openai/codex` y la app queda instalada.

#### Scenario: npm ausente con receta alternativa

- **GIVEN** una receta `npm` seguida de una receta `installer` y sin `npm` en el `PATH`
- **WHEN** se selecciona la receta
- **THEN** se usa la receta `installer`.

#### Scenario: Nombre de paquete inválido

- **GIVEN** una receta `npm` con `package = "foo; rm -rf ~"`
- **WHEN** se carga el catálogo
- **THEN** la definición se rechaza con un error en español.

### Requirement: Método github-release

El método `github-release` DEBE (`MUST`) exigir `repo` con forma `propietario/nombre`, `asset` y `dest` o `extract`, y NO DEBE (`MUST NOT`) admitir `url`, `sha256`, `package`, `formula` ni `cask`. Kamino DEBE (`MUST`) descargar solo desde `https://github.com/<repo>/releases/` y siempre el último release, nunca una versión fijada por la receta.

#### Scenario: Último release

- **GIVEN** una receta con `repo = "sxyazi/yazi"` y un `asset` sin versión
- **WHEN** se instala la app
- **THEN** se descarga el asset del último release publicado en ese momento.

#### Scenario: Versión en el nombre del asset

- **GIVEN** una receta con `repo = "cli/cli"` y `asset = "gh_{version}_macOS_arm64.zip"`
- **WHEN** el último release es `v2.105.0`
- **THEN** se descarga `gh_2.105.0_macOS_arm64.zip` de ese release.

#### Scenario: Repositorio inválido

- **GIVEN** una receta con `repo = "../otro/sitio"`
- **WHEN** se carga el catálogo
- **THEN** la definición se rechaza con un error en español.

### Requirement: Extracción de la descarga

La receta `github-release` DEBE (`MUST`) declarar el formato del asset (`raw`, `zip` o `tar.gz`) y, salvo `raw`, los ficheros a extraer con su destino bajo el HOME. Kamino DEBE (`MUST`) extraer solo los ficheros regulares nombrados, rechazar rutas absolutas o con `..`, y escribirlos con permisos 0755 mediante reemplazo atómico.

#### Scenario: Varios ejecutables

- **GIVEN** una receta de un zip con dos ficheros a extraer, `yazi` y `ya`
- **WHEN** se instala la app
- **THEN** ambos quedan en sus destinos con permisos 0755.

#### Scenario: Fichero ausente en el archivo

- **GIVEN** una receta que nombra un fichero que el archivo no contiene
- **WHEN** se instala la app
- **THEN** no se escribe ningún destino y la app queda omitida con el motivo.

#### Scenario: Entrada con ruta peligrosa

- **GIVEN** un archivo cuyo miembro nombrado resuelve fuera del directorio de extracción
- **WHEN** se instala la app
- **THEN** no se instala nada.

### Requirement: Verificación con checksums del release

Una receta `github-release` PUEDE (`MAY`) declarar `checksums` con el nombre del fichero de sumas del mismo release. Si lo declara, Kamino DEBE (`MUST`) verificar el SHA-256 del asset descargado contra esa suma antes de extraer y NO DEBE (`MUST NOT`) instalar nada si falta, no lista el asset o no coincide. Sin `checksums`, la confirmación y `--dry-run` DEBEN (`MUST`) indicar que la descarga no se verifica.

#### Scenario: Suma correcta

- **GIVEN** una receta con `checksums = "checksums.txt"` y un asset cuya suma coincide
- **WHEN** se instala la app
- **THEN** se extrae e instala.

#### Scenario: Suma incorrecta

- **GIVEN** una receta con `checksums` y un asset cuya suma no coincide
- **WHEN** se instala la app
- **THEN** no se instala nada y el error indica que el SHA-256 no coincide.

#### Scenario: Receta sin checksums

- **GIVEN** una receta `github-release` sin `checksums`
- **WHEN** se muestra la confirmación o se ejecuta `--dry-run`
- **THEN** se indica que la descarga se hace solo por HTTPS, sin verificación de suma.

### Requirement: Preferencia entre métodos

Kamino DEBE (`MUST`) elegir la primera receta, en el orden del catálogo, que aplique a la plataforma y cuya herramienta esté disponible: `brew` exige `brew` y `npm` exige `npm` en el `PATH`. El catálogo integrado DEBE (`MUST`) ordenar las recetas como `brew`, `npm`, `installer` y `github-release`. Si ninguna puede ejecutarse, la app NO DEBE (`MUST NOT`) ofrecerse para instalar.

#### Scenario: Homebrew y npm disponibles

- **GIVEN** una app con recetas `brew` y `npm` y ambas herramientas en el `PATH`
- **WHEN** se selecciona la receta
- **THEN** se usa `brew`.

#### Scenario: Solo npm disponible

- **GIVEN** la misma app y solo `npm` en el `PATH`
- **WHEN** se selecciona la receta
- **THEN** se usa `npm`.

#### Scenario: Ninguna herramienta disponible

- **GIVEN** una app con recetas `brew` y `npm` y ninguna de las dos herramientas
- **WHEN** se prepara la instalación
- **THEN** la app no se ofrece para instalar.

### Requirement: Cobertura del catálogo integrado

Cada app del catálogo integrado DEBE (`MUST`) declarar todas las recetas que admita entre `brew`, `npm`, `installer` y `github-release`, con el último release como única versión posible. Las apps sin distribución por esos métodos (`git` sin binario de macOS, `oh-my-zsh`, `ghostty` y `gentle-shell`) PUEDEN (`MAY`) quedar sin la receta que no exista.

#### Scenario: App con las cuatro vías

- **GIVEN** una app disponible por brew, npm, instalador y release de GitHub
- **WHEN** se carga el catálogo integrado
- **THEN** su definición declara las cuatro recetas en ese orden.

#### Scenario: Catálogo válido

- **GIVEN** el catálogo integrado
- **WHEN** se ejecuta `apps validate`
- **THEN** todas las recetas pasan la validación.
