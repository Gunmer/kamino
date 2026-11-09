# global-app-config Specification

## Purpose

Clonar y desplegar personalizaciones globales de herramientas de desarrollo y agentes de terminal sin incluir sus programas ni los archivos propios de cada proyecto.

## Requirements

### Requirement: Catálogo integrado de herramientas de desarrollo

El catálogo integrado DEBE (`MUST`) ofrecer las herramientas de desarrollo agéntico y de terminal acordadas, además de conservar las definiciones ya existentes. La cobertura de cada herramienta DEBE limitarse a su configuración global documentada, sus temas y atajos personalizables cuando existan.

#### Scenario: Herramientas agénticas disponibles en el catálogo

- **WHEN** se carga el catálogo integrado sin definiciones del usuario
- **THEN** contiene entradas para OpenCode, Claude Code, Codex, Gemini CLI, Copilot CLI, Pi, Aider y Goose.
- **AND** contiene entradas para Gentle Shell, Gentle AI, GGA, OpenSpec y Caveman.

#### Scenario: Herramientas de terminal disponibles en el catálogo

- **WHEN** se carga el catálogo integrado sin definiciones del usuario
- **THEN** contiene entradas para LazyGit, Micro, Navi, Glow, bat, eza, fd y GitHub CLI, además de Git, Ghostty, Herdr y yazi.
- **AND** ofrece la configuración de Git utilizada por herramientas como delta sin exigir una entrada duplicada para ella.

#### Scenario: Una herramienta no está instalada en el equipo de prueba

- **GIVEN** una herramienta detectada solo por ejecutable no está presente en el PATH simulado
- **WHEN** se consultan las aplicaciones instaladas
- **THEN** su definición sigue disponible en el catálogo, pero no aparece como instalada ni se clona automáticamente.

### Requirement: Detección sin ejecutar herramientas

Kamino DEBE (`MUST`) detectar las herramientas con ejecutable propio mediante ese ejecutable o su identificador de aplicación, sin ejecutarlas, y DEBE mantener la selección y despliegue filtradas por instalación.

#### Scenario: Ejecutable disponible

- **GIVEN** `opencode` existe y es ejecutable en un PATH simulado
- **WHEN** se detectan las herramientas instaladas
- **THEN** OpenCode se considera instalado sin ejecutar el archivo.

#### Scenario: Herramienta ausente al desplegar

- **GIVEN** un clon contiene configuraciones de una herramienta que no está instalada en el equipo de destino
- **WHEN** se despliega el clon
- **THEN** se omiten e informan sus archivos sin instalarlos ni modificar sus destinos.

#### Scenario: Dos nombres para Caveman

- **GIVEN** los ejecutables `cave` y `caveman` están presentes y corresponden a Caveman
- **WHEN** se presenta la selección de aplicaciones
- **THEN** Caveman aparece como una sola herramienta.

### Requirement: Conservación de personalizaciones globales

Kamino DEBE (`MUST`) preservar los archivos globales incluidos de configuración, temas y atajos sin alterar su contenido. No DEBE buscar automáticamente configuraciones de repositorios ni clonar ejecutables de las herramientas.

#### Scenario: Configuración y tema de OpenCode

- **GIVEN** OpenCode está instalado y existen su configuración global, `cli.json` con atajos y un tema personalizado
- **WHEN** se clona y despliega OpenCode en otro HOME con la herramienta instalada
- **THEN** los archivos incluidos conservan sus bytes y rutas relativas al HOME.

#### Scenario: Configuración y atajos de Pi

- **GIVEN** Pi está instalado y existen `settings.json`, `keybindings.json` y un tema propio en su directorio global
- **WHEN** se completa el ciclo de clon y despliegue
- **THEN** se conservan los tres archivos sin incluir `auth.json` ni el ejecutable.

#### Scenario: Archivos locales de proyectos

- **GIVEN** un repositorio contiene sus propios archivos de configuración de agentes
- **WHEN** se clona una herramienta del catálogo
- **THEN** Kamino no descubre ni incluye esos archivos por el mero hecho de pertenecer al repositorio.

### Requirement: Configuración global de Oh My Zsh

Kamino DEBE (`MUST`) detectar Oh My Zsh mediante `zsh` y el marcador del framework en el HOME, sin considerarlo instalado solo porque exista `zsh`. DEBE incluir ajustes y temas personales y excluir las carpetas de plugins.

#### Scenario: Framework presente

- **GIVEN** `zsh` está instalado y el HOME contiene el archivo de inicio de Oh My Zsh, archivos globales de configuración del shell y un tema personalizado
- **WHEN** se selecciona Oh My Zsh para clonar
- **THEN** se incluyen los ajustes existentes, atajos y tema dentro del HOME.
- **AND** se excluyen los archivos de las carpetas de plugins y sus directorios `.git`.

#### Scenario: Solo está instalado zsh

- **GIVEN** `zsh` está disponible pero el framework Oh My Zsh no está presente en el HOME
- **WHEN** se detectan las aplicaciones o se despliega su configuración
- **THEN** Oh My Zsh no se considera instalado y sus archivos no se aplican.

### Requirement: Tema y atajos globales de Herdr

Kamino DEBE (`MUST`) conservar la configuración global de Herdr, incluidos el tema, sus colores personalizados, los atajos y los comandos asociados a estos, sin copiar programas de plugins.

#### Scenario: Preferencias personalizadas

- **GIVEN** Herdr está instalado y su `config.toml` contiene `[theme]`, `[theme.custom]`, `[keys]` y `[[keys.command]]`
- **WHEN** se clona y despliega Herdr
- **THEN** el archivo despliegado conserva íntegramente esas preferencias.

#### Scenario: Ubicación global personalizada de Herdr

- **GIVEN** `HERDR_CONFIG_PATH` señala un archivo de configuración global dentro del HOME
- **WHEN** se clona Herdr
- **THEN** se incluye ese archivo sin alterar su contenido y no se leen rutas fuera del HOME.

### Requirement: Rutas opcionales y omisiones reales

La ausencia de una ruta declarada como opcional en una definición nueva o ampliada NO DEBE (`MUST NOT`) contarse como una omisión. Las rutas obligatorias heredadas y las omisiones reales DEBEN conservar la semántica y los códigos de salida existentes.

#### Scenario: Tema personalizado no creado

- **GIVEN** una herramienta instalada tiene su archivo de configuración y no tiene un directorio de temas marcado como opcional
- **WHEN** se clona sin otras omisiones
- **THEN** la ausencia del tema opcional no produce por sí sola el código de salida 3.

#### Scenario: Ruta obligatoria heredada ausente

- **GIVEN** una definición existente o del usuario declara una ruta obligatoria que no existe
- **WHEN** se clona esa definición
- **THEN** la ruta se informa como omitida y la CLI conserva el código de salida 3.

### Requirement: Prioridad del catálogo del usuario

Las definiciones del usuario DEBEN (`MUST`) seguir sustituyendo las integradas con el mismo identificador y las definiciones existentes DEBEN seguir siendo válidas.

#### Scenario: Sustitución de una herramienta integrada

- **GIVEN** el usuario define una aplicación con el mismo identificador que una entrada integrada
- **WHEN** se carga el catálogo
- **THEN** la definición del usuario sustituye a la integrada, sin perder las validaciones de rutas ni las protecciones generales de seguridad.
