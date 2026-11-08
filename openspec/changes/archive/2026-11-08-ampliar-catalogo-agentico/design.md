# Diseño: catálogo global para desarrollo agéntico

## Contexto

La motivación y el alcance se describen en `proposal.md`; los contratos están en las tres especificaciones de este cambio. El catálogo actual valida rutas literales relativas al HOME, detecta comandos con `shutil.which` y permite que un TOML del usuario sustituya una definición integrada. `build_plan` expande cada entrada sin distinguir variantes opcionales ni eliminar solapamientos.

La validación de `paths.py` es léxica: un componente intermedio puede ser un enlace que redirija una lectura o escritura fuera del HOME. Además, `archive.py` vuelve a abrir las rutas durante el empaquetado y `restore.py` las usa para clasificar, crear la instantánea y reemplazar destinos. Validar solo al construir el plan no protege esas operaciones posteriores.

Ghostty guarda atajos en su configuración y admite `config-file` y temas en archivos separados. Herdr guarda `[keys]`, `[[keys.command]]` y `[theme]` en su `config.toml`; su configuración de plugins puede contener `.env`. El respaldo actual es un `tar.gz` sin cifrar y los archivos permitidos se conservan sin redactar.

## Objetivos y exclusiones

**Objetivos:** mantener el catálogo declarativo para rutas conocidas, acotar el descubrimiento especial a Ghostty, aplicar el límite del HOME en todas las operaciones y conservar las garantías y el formato de restauración existentes.

**Exclusiones:** no ejecutar shells ni agentes para descubrir configuración, no recorrer repositorios, no copiar binarios ni instalaciones de plugins, no reescribir archivos de configuración y no implementar cifrado ni exportación de bases de datos de memoria.

## Decisiones

### Definiciones explícitas y rutas opcionales

Mantener una definición por herramienta con ejecutable propio y usar los mecanismos existentes de PATH y `bundle_id`. Añadir al modelo una lista de rutas opcionales y, para nombres variables conocidos como los perfiles `*.config.toml` de Codex, patrones opcionales restringidos a un directorio declarado. Las rutas opcionales ausentes no generan omisiones; una ruta existente pero ilegible o insegura sí se informa. Los patrones se validan contra HOME, se aplican sin seguir enlaces y no convierten las rutas literales heredadas en patrones. Se permite una lista `paths` vacía solo cuando haya al menos una ruta o patrón opcional válido; una definición sin entradas sigue siendo inválida.

Las definiciones anteriores conservan su validación y el significado obligatorio de `paths`. En particular, las dos rutas `config` actualmente declaradas por Ghostty siguen teniendo su comportamiento previo; `config.ghostty`, temas y referencias opcionales se añaden sin reclasificar esas rutas históricas. Así se mantiene el código 3 para las omisiones que ya se informaban, aunque una configuración nueva de Ghostty pueda coexistir con ese aviso. El catálogo del usuario continúa sustituyendo completamente la definición integrada por id.

Para Oh My Zsh, ampliar la detección con un marcador opcional relativo al HOME, validado con las mismas reglas de seguridad. La entrada integrada requiere `zsh` y `~/.oh-my-zsh/oh-my-zsh.sh`; detectar solo `zsh` produciría falsos positivos en macOS. El HOME se inyecta también en la detección usada por la CLI, el respaldo y la restauración. Las personalizaciones habituales se toman de `.zshrc`, `.zprofile`, `.zshenv` y `.oh-my-zsh/custom`, excluyendo `custom/plugins` y metadatos `.git`. Si `ZSH` o `ZSH_CUSTOM` se establecen únicamente mediante código de shell o apuntan a otra ubicación, no se evalúa ese código: se documenta el límite y puede usarse una definición del usuario con rutas dentro del HOME.

### Matriz inicial de configuración

Las siguientes ubicaciones son relativas al HOME y solo se consideran cuando existen y corresponden al alcance seguro. Los directorios que contienen configuración de usuario se recorren con exclusiones; no se respalda un directorio completo que mezcle por defecto credenciales, sesiones o instalaciones. Las formas alternativas y los archivos personales soportados por cada herramienta se declaran explícitamente en su definición y se prueban con datos sintéticos.

| Herramienta | Configuración global y personalizaciones principales |
| --- | --- |
| OpenCode | `.config/opencode/opencode.json`, `opencode.jsonc`, `cli.json`, `themes`, `commands`, `agents` y `skills`. `cli.json` contiene tema y atajos de la TUI. |
| Claude Code | `.claude/settings.json`, `keybindings.json`, `CLAUDE.md`, `themes`, `commands`, `agents` y `skills`. Se excluyen credenciales y `~/.claude.json`, que mezcla cuenta y estado de proyectos. |
| Codex | `.codex/config.toml`, perfiles `*.config.toml`, `AGENTS.md`, `themes` y skills globales en `.agents/skills`. |
| Gemini CLI | `.gemini/settings.json`, `keybindings.json`, instrucciones globales y `commands`; los temas configurados en ajustes permanecen en ellos. |
| Copilot CLI | `.copilot/settings.json`, instrucciones, agentes, skills y configuración personal de MCP o proveedores; se excluye `config.json`, que contiene autenticación y estado interno. |
| Pi | `.pi/agent/settings.json`, `keybindings.json`, `themes`, instrucciones, `prompts`, `skills` y configuración MCP; no `auth.json` ni sesiones. |
| Aider | `.aider.conf.yml`, `.aider.model.settings.yml` y `.aider.model.metadata.json`; el primero puede contener claves incrustadas. |
| Goose | `.config/goose/config.yaml`, `permission.yaml` y plantillas personales de `prompts`; no `secrets.yaml`. |
| Gentle Shell | `.gentle-shell/config.json`, ajustes y temas de `.gentle-shell/agent`, y las preferencias personales documentadas de sus directorios de configuración y de `.pi/gentle-ai`. La instalación aislada y `--link` no se confunden. |
| Gentle AI | `.gentle-ai/state.json` para la selección y ajustes globales, sin respaldos internos, bloqueos, registros ni telemetría. |
| GGA | `.config/gga/config`. |
| OpenSpec | `.config/openspec/config.json`; no se descubren almacenes ni proyectos referenciados. |
| Caveman | `.config/caveman/config.json`, `.caveman-cloud/config.json` y `.caveman/caveman.yaml`; `cave` y `caveman` representan una sola entrada. |
| LazyGit | `Library/Application Support/lazygit/config.yml` y la alternativa XDG. |
| Micro | `.config/micro/settings.json`, `bindings.json` y `colorschemes`; no `backups` ni `buffers`. |
| Navi | `.config/navi/config.yaml`. |
| Glow | `Library/Preferences/glow/glow.yml` o `glow.yaml`, y su alternativa XDG. |
| bat | `.config/bat/config` y `themes`; no la caché generada. |
| eza | `.config/eza/theme.yml`. |
| fd | `.config/fd/ignore`. |
| GitHub CLI | `.config/gh/config.yml`; no `.config/gh/hosts.yml`, que puede contener autenticación. |
| Git y delta | Mantener `.gitconfig` y `.config/git/ignore` y añadir `.config/git/config` como alternativa opcional. |
| Herdr | `.config/herdr/config.toml` y configuración de plugins permitida; no programas de plugins ni sus credenciales. |

Las entradas de habilidades compartidas pueden pertenecer a más de una aplicación para que cada selección sea restaurable de forma independiente. Se deduplican dentro de cada aplicación; entre aplicaciones se conservan los miembros separados del formato actual. `fzf`, `rg` y `zoxide` no reciben una entrada ficticia: sus ajustes pueden depender de variables del shell o de ubicaciones escogidas por el usuario. Engram y los índices de CodeGraph no se copian como configuración.

Fuentes de las ubicaciones: [OpenCode](https://opencode.ai/v2/docs/config), [Claude Code](https://code.claude.com/docs/en/claude-directory), [Codex](https://developers.openai.com/codex/config-file/config-advanced), [Gemini CLI](https://geminicli.com/docs/reference/configuration/), [Copilot CLI](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference), [Pi](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/configuration.md), [Aider](https://aider.chat/docs/config/aider_conf.html), [Goose](https://goose-docs.ai/docs/guides/config-files/), [Gentle Shell](https://github.com/Gentleman-Programming/gentle-shell/blob/main/docs/readme-reference.md), [Gentle AI](https://github.com/Gentleman-Programming/gentle-ai/blob/main/docs/usage.md), [GGA](https://github.com/Gentleman-Programming/gentleman-guardian-angel/blob/main/docs/configuration.md), [OpenSpec](https://github.com/Fission-AI/OpenSpec/blob/main/docs-lab/reference/configuration/config-json.md), [Caveman](https://github.com/JuliusBrussee/caveman/blob/main/docs/technical/configuration.md), [LazyGit](https://github.com/jesseduffield/lazygit/blob/master/docs/Config.md), [Micro](https://github.com/zyedidia/micro/blob/master/runtime/help/options.md), [Navi](https://github.com/denisidoro/navi/blob/master/docs/configuration/README.md), [Glow](https://github.com/charmbracelet/glow/blob/main/main.go), [bat](https://github.com/sharkdp/bat), [eza](https://github.com/eza-community/eza) y [GitHub CLI](https://cli.github.com/manual/gh_help_environment).

### Descubrimiento limitado de referencias de Ghostty

Añadir una función específica de Ghostty en la planificación, activada solo para esa aplicación y limitada por la definición efectiva del catálogo. Esto es una excepción explícita al catálogo puramente declarativo, necesaria para interpretar su formato; se descarta introducir un sistema genérico de plugins o ejecutar `ghostty` para consultar su configuración.

Partir de las cuatro ubicaciones globales habituales `config` y `config.ghostty`, el directorio XDG de temas y, si `XDG_CONFIG_HOME` se proporciona dentro del HOME, su equivalente. Procesar `config-file` con la sintaxis documentada: relativo al archivo que lo declara, absoluto, opcional con `?`, repetido y anidado. Procesar la selección de temas por nombre y por ruta absoluta, incluidos pares claro/oscuro. No se copian los temas integrados de la aplicación; se incluyen archivos personalizados dentro del HOME.

Resolver cada candidato mediante el acceso seguro al HOME antes de abrirlo, aplicar exclusiones antes de leer y mantener un conjunto de visitados y límites internos de tamaño, profundidad y cantidad. Una referencia opcional ausente se ignora; una referencia requerida ausente, excluida por seguridad o fuera del HOME se informa como omisión. Un ciclo se comunica y se corta sin perder los demás archivos. Deduplicar por `(app_id, ruta relativa)` después de combinar rutas y referencias para no escribir miembros repetidos que `verify_archive` rechazaría.

Para Herdr, reconocer `HERDR_CONFIG_PATH` y el directorio XDG cuando estén establecidos dentro del HOME; un valor externo se informa sin leerlo. Para las demás herramientas se cubren las ubicaciones globales indicadas en la matriz; rutas alternativas definidas solo mediante scripts, variables no soportadas o archivos locales de proyectos requieren una definición personalizada, siempre dentro del HOME.

### Confinamiento seguro de lectura y escritura

Concentrar las operaciones seguras en `paths.py` y reutilizarlas desde detección por marcador, planificación, lectura del archivo, clasificación de destinos, instantáneas y aplicación. Abrir el HOME de referencia y recorrer sus descendientes con descriptores de directorio y operaciones relativas que rechacen enlaces intermedios; verificar la identidad de los componentes y volver a comprobarla al usarlos. En macOS y Ubuntu se pueden usar primitivas de la biblioteca estándar con `dir_fd` y `O_NOFOLLOW`; si no se puede garantizar el límite, abortar en vez de recurrir a una comprobación léxica o a `resolve()` seguida de una apertura insegura.

Los enlaces finales conservan la regla actual: se guardan como enlaces, nunca se siguen y se aceptan solo destinos relativos sin `..`. Reabrir mediante el mismo mecanismo al calcular checksums y al empaquetar evita que una sustitución posterior al plan cambie el origen. En restauración, rechazar un destino inseguro antes de crear la instantánea; realizar la creación de padres y los reemplazos relativos a directorios seguros. Si cambia la jerarquía durante la operación, detenerse sin acceder fuera del HOME. No restringir la ubicación del archivo de respaldo que el usuario entrega: la restricción afecta a los archivos de configuración y a los destinos.

### Exclusiones y avisos sin alterar contenidos

Usar listas positivas de configuración por aplicación y exclusiones para credenciales documentadas y datos no configurables. Como mínimo, excluir `.env` y `.env.*`, `auth.json`, `.credentials.json`, `secrets.yaml`, `secrets.yml`, historiales, sesiones, cachés y registros en los árboles cubiertos; excluir también `hosts.yml` de GitHub CLI y ubicaciones conocidas de claves como `.ssh` y `.aws/credentials` cuando aparezcan como referencias automáticas. Aplicar el filtro a las referencias antes de abrirlas. Los TOML personalizados mantienen su precedencia y rutas explícitas, pero no pueden desactivar el confinamiento del HOME; las referencias automáticas tampoco pueden eludir las exclusiones.

No se afirma que la lista reconozca todos los secretos: un archivo de configuración permitido puede contener claves, y las exclusiones por nombre no detectan todas las credenciales. La CLI muestra un aviso general en español antes de respaldar, incluso sin TTY y en `--dry-run`, sin pedir confirmación adicional ni imprimir valores. No se modifican los bytes de los ajustes para eliminar secretos. Las exclusiones de respaldos nuevos no se aplican a la restauración de archivos antiguos ni a su instantánea de seguridad: esta debe conservar todos los destinos existentes que vayan a cambiar, aunque sean sensibles. Las instantáneas tampoco están cifradas y deben documentarse como datos sensibles.

En restauración, analizar de forma acotada y sin extraer las configuraciones de Ghostty presentes en el manifiesto para detectar referencias absolutas que no correspondan al HOME de destino. Emitir un aviso genérico sin exponer contenidos; no cambiar la ruta. La simulación puede leer el respaldo para mostrar el aviso, pero no crea temporales, instantáneas ni destinos. La restauración real mantiene la verificación completa antes de cualquier escritura, seguida de la instantánea de los destinos cambiantes y los reemplazos atómicos.

### Compatibilidad y verificación

Conservar `format_version = 1` y la estructura existente del manifiesto: las referencias descubiertas pasan a ser entradas normales por aplicación. La restauración de respaldos anteriores no se filtra con las nuevas exclusiones; estas regulan la creación de nuevos respaldos. Mantener `--apps`, alias, `--yes`, `--dry-run`, los códigos 1 y 3 y la selección no interactiva actuales. En particular, un `backup` no interactivo sin `--apps` incluirá las nuevas herramientas detectadas y deberá mostrar el aviso.

Las pruebas crean dos HOME temporales, ejecutables simulados, directorios de aplicaciones y de instantáneas inyectados. Cubren la matriz de rutas, ausencias opcionales, prioridad del catálogo del usuario, detección sin ejecutar, enlaces intermedios y sustituciones durante la operación, credenciales excluidas, referencia Ghostty anidada/cíclica y fuera del HOME, deduplicación, avisos, restauración íntegra y respaldos de formato 1. En las pruebas de CLI se sustituyen la detección y los casos de uso; nunca se leen el HOME real ni `/Applications`.

## Riesgos y compensaciones

- **Secretos incrustados o con nombres desconocidos** → copiar ajustes íntegros con aviso; la exclusión por rutas conocidas no garantiza ausencia de secretos ni sustituye al cifrado.
- **Rutas absolutas de Ghostty al migrar de usuario** → preservar los bytes y advertir; el usuario ajusta manualmente las referencias que ya no funcionen.
- **Plugins o dependencias no instalados en destino** → documentar su reinstalación, en particular los plugins de Oh My Zsh. Los atajos globales de Ghostty pueden requerir conceder otra vez permisos de accesibilidad de macOS.
- **Rutas alternativas o archivos externos** → admitir las variables explícitamente indicadas solo dentro del HOME y documentar el uso de definiciones personalizadas y las omisiones externas.
- **Rutas obligatorias históricas de Ghostty ausentes** → conservar el código 3 anterior aunque existan rutas nuevas opcionales; no redefinir silenciosamente la semántica de catálogos heredados.
- **Compatibilidad de versiones de herramientas** → basar cada entrada en ubicaciones documentadas y mantener pruebas y documentación por herramienta, sin suponer que todas tienen temas o atajos separados.

## Despliegue y reversión

Publicar el catálogo y las protecciones con la distribución normal, verificando que las definiciones TOML estén presentes en el paquete y en el binario. Los respaldos antiguos de formato 1 permanecen legibles y los nuevos usan el mismo formato. Una reversión de la ampliación deja de reconocer nuevas aplicaciones en el catálogo anterior; se deben conservar los respaldos y reinstalar una versión compatible antes de restaurarlas. No eliminar ni revertir la protección de rutas como parte de una reversión rutinaria.
