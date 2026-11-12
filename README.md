# kamino

CLI para macOS que clona las configuraciones de las aplicaciones elegidas en **un único
archivo portable** (`tar.gz` con `manifest.json`) y las despliega después, aplicando solo lo que
corresponda a las apps que estén instaladas.

## Estado

Distribución Go estática implementada: `clone` y `deploy` funcionales con catálogo declarativo,
detección de apps, archivado `tar.gz` con manifiesto y checksums, CLI interactiva y revisión de
conflictos por líneas dentro de cada bloque en una TUI ANSI propia. El binario no requiere Python.

## Requisitos

- **Instalación con el binario:** macOS con Apple Silicon (`arm64`) y `curl`. El binario es
  autocontenido y no requiere instalar Python.
- **Compilación desde el código fuente o desarrollo:** Go 1.25 o superior. Las pruebas de
  compatibilidad con el oráculo Python requieren `uv` y Python 3.11.4 o superior.

## Instalación con script

Descarga el instalador, revisa su contenido y ejecútalo:

```bash
curl -fsSL https://raw.githubusercontent.com/Gunmer/kamino/main/scripts/install.sh -o kamino-install.sh
less kamino-install.sh
sh kamino-install.sh
```

También puedes ejecutarlo directamente, sin guardarlo. Esta opción ejecuta el script remoto sin
revisión previa:

```bash
curl -fsSL https://raw.githubusercontent.com/Gunmer/kamino/main/scripts/install.sh | sh
```

Si ya descargaste el repositorio, puedes ejecutar el instalador incluido:

```bash
sh scripts/install.sh
```

El instalador comprueba que el sistema sea macOS con Apple Silicon, descarga el binario
`kamino-arm64` de la última release publicada y lo instala en `~/.local/bin/kamino`, sin usar
`sudo`. Si necesita crear `~/.local/bin` o reemplazar una instalación existente, pide confirmación;
si no hay una terminal disponible para responder, detiene la instalación. También marca el binario
como ejecutable e intenta quitar el atributo de cuarentena de macOS.

Comprueba la instalación ejecutando:

```bash
~/.local/bin/kamino --help
```

Si quieres invocar el comando como `kamino`, añade `~/.local/bin` al `PATH` de la sesión actual:

```bash
export PATH="$HOME/.local/bin:$PATH"
kamino --help
```

Para conservarlo en nuevas terminales, añade esa línea al archivo de inicio de tu shell (por
ejemplo, `~/.zshrc` si usas zsh).

## Actualización

Si instalaste el binario autocontenido, puedes actualizarlo a la última release publicada en
GitHub:

```bash
kamino update
```

El comando consulta la última release, compara su versión con `kamino --version` y, si hay una
más nueva, descarga `kamino-arm64`, verifica su tamaño y SHA-256 y reemplaza el binario en
ejecución de forma atómica. No requiere `sudo` y no toca configuraciones, clones ni
instantáneas. Si ya estás en la última versión, o tienes una más reciente, no descarga ni escribe
nada.

La actualización solo está disponible para el binario autocontenido. Una compilación de
desarrollo (`go run` o un binario sin la marca de autocontenido) rechaza `kamino update` e indica
que no se descargó nada.

El comando termina con código 0 tanto si actualiza como si ya está al día, y con 1 ante cualquier
error. La opción también aparece en el menú que se abre al ejecutar `kamino` sin argumentos.

## Instalación (desarrollo)

```bash
go build ./...
go run ./cmd/kamino --help
```

La distribución se publica como el asset `kamino-arm64`; `uv` y `pip` ya no son vías de
instalación. La referencia Python permanece en `legacy/python/` para ejecutar la matriz de
compatibilidad.

## Versionado y releases

El versionado sigue [SemVer](https://semver.org/) y se automatiza con
[python-semantic-release](https://python-semantic-release.readthedocs.io/):

1. Los commits integrados en `main` deben seguir Conventional Commits. Con merge tradicional o
   rebase se analizan los commits de la rama; con squash, conserva un mensaje Conventional Commit
   adecuado en el commit resultante. CI no valida los títulos de los PRs.
2. Después de que CI pase en `main`, se calcula la siguiente versión desde el último tag:
   `feat:` aumenta minor, `fix:` aumenta patch y `!` o un footer `BREAKING CHANGE:` identifica
   cambios incompatibles. Si solo hay `docs:`, `ci:`, `chore:`, `perf:` u otros cambios sin ruptura
   de compatibilidad, no se genera versión ni release.
3. Durante las versiones `0.x`, los cambios incompatibles aumentan minor (`major_on_zero = false`).
   Desde `1.0.0`, aumentan major. Se toma el mayor incremento entre los commits pendientes.
4. Se actualizan `CHANGELOG.md`, `internal/version/version.go` y el tag `vX.Y.Z`. El binario Go se
   construye con `scripts/build_release.py` y se comprueba antes de empujar el commit de release.
5. Se crea una GitHub Release con las notas y el binario autocontenido `kamino-arm64`
   (Go estático, Apple Silicon), incluido el catálogo de aplicaciones. No se distribuye actualmente
   un binario Intel. La release permanece en borrador hasta subir el binario.

Por ejemplo, desde `0.1.0`, `fix: corrige el despliegue` produce `0.1.1`, mientras que
`feat: añade una aplicación` o `feat!: cambia el formato` produce `0.2.0`. Un merge que solo
contenga `ci: ajusta el workflow` no publica nada.

Las publicaciones se serializan. Si `main` avanza durante la construcción, se cancela el push
sin forzarlo; la ejecución del nuevo commit validado retomará los cambios pendientes. Los pushes
y tags del bot no disparan otro workflow: todo el proceso ocurre en **Release**.

Si falla la publicación después de empujar el tag, ejecuta manualmente **Release** desde la rama
`main` e indica ese tag en el campo `tag`. Esta recuperación reconstruye y comprueba la misma
versión y completa la release sin incrementar la versión, duplicar el changelog ni reemplazar
assets ya subidos. Si GitHub dejó un asset vacío por una subida fallida, lo elimina antes de
reintentar. No permite publicar tags ajenos a un commit de release integrado en `main`.

Los binarios descargados manualmente no están firmados y macOS puede aplicarles cuarentena. Para
quitarla manualmente:

```bash
xattr -d com.apple.quarantine ./kamino-arm64
```

El instalador intenta quitar ese atributo automáticamente.

El Homebrew tap se publicará tras la primera versión etiquetada.

## Uso

```bash
kamino                       # TUI: menú Clonar, Desplegar, Actualizar, Configuración y Abandonar
kamino --help
kamino clone                 # TUI: elige apps, destino y revisa el plan
kamino deploy clone-<aaaammdd>.tar.gz   # TUI: apps, conflictos y confirmación
kamino update                # TUI: consulta, descarga y reemplazo
```

`-h` es alias de `--help` en el nivel principal y en `clone`, `deploy` y `update`, y `-V` es
alias de `--version`:

```bash
kamino -h
kamino clone -h
kamino deploy -h
kamino update -h
kamino -V
```

En `clone`, `-a` equivale a `--apps`, `-o` a `--output`, `-n` a `--dry-run`, `-v` a `--verbose`
y `-f` a `--force`:

```bash
kamino clone -a vscode,iterm2,git -o clon.tar.gz -v
kamino clone -a vscode,iterm2,git -o clon.tar.gz -n   # simulación: no crea el archivo
kamino clone -f                                      # reemplaza clone-<aaaammdd>.tar.gz
```

Sin `--output`, el nombre por defecto es `clone-<aaaammdd>.tar.gz` con la fecha local dentro de
`~/.config/kamino/vault/`. Kamino crea ese directorio al escribir el clon; `--dry-run` no lo crea.
Sin `--force`, un clon existente es un error y no se sobrescribe; con `--force` se reemplaza de
forma atómica. `--dry-run --force` no modifica nada.

El cambio de destino es incompatible con scripts que esperaban el archivo en el directorio actual:
usa `--output` para conservar la ruta anterior. Los holocrones nuevos se guardan en
`~/.config/kamino/vault/snapshot/`; los antiguos en
`~/Library/Application Support/kamino/snapshots/` no se mueven ni se copian y siguen pudiéndose
desplegar indicando su ruta.

Con terminal interactiva, `clone` abre la TUI si no se definió `KAMINO_PLAIN`. `--apps` omite C1,
`--output` omite C2 y ambos juntos pasan directamente a la revisión C3. Un valor inválido abre la
pantalla correspondiente precargada con el error. `--dry-run` termina en C3 sin acción de clonar.

En `deploy`, `-a` equivale a `--apps`, `-n` a `--dry-run`, `-y` a `--yes` y `-v` a `--verbose`; el archivo del clon sigue siendo un argumento posicional:

```bash
kamino deploy clone-20261109.tar.gz            # TUI: apps, conflictos, confirmación
kamino deploy clone-20261109.tar.gz -a git,zsh # sin selección de apps
kamino deploy clone-20261109.tar.gz -y -v      # aplica todo sin preguntar
kamino deploy clone-20261109.tar.gz -n         # simulación: imprime diffs, no escribe
```

Desde el menú, **Desplegar** abre D0 con una sola lista que combina los clones de
`clone.output_dir` (por defecto `~/.config/kamino/vault/`) y los `.tar.gz` del directorio actual.
La lista se ordena por fecha de modificación, deduplica los archivos que aparecen en ambas rutas y
muestra el origen de cada fila, además de la ruta completa, la fecha y el tamaño del clon
resaltado. La tabla recibe el foco y deja seleccionado el clon más reciente; `/` filtra por nombre
u origen y `o` salta a `Otro archivo...`, que abre el campo de ruta manual. Si `kamino deploy`
recibe una ruta inexistente o inválida, D0 se abre con esa ruta precargada, el error visible y la
lista de clones disponible.

Sin terminal interactiva, un despliegue real requiere `-y` o `--yes`; la simulación con `-n`
no lo requiere. `deploy -n` y `deploy --yes` no abren la TUI aunque haya terminal. `-a` valida los
ids contra las apps desplegables: los desconocidos o no instalados se avisan y cuentan como
omitidos (código de salida 3). Las apps del manifiesto no instaladas se informan como “sin base en
este Mac” y no alteran el código.

### Glosario visible

| Nombre visible | Significado |
| --- | --- |
| original | configuración actual del Mac; antes “local” |
| holocrón de seguridad | copia previa a desplegar; antes “snapshot” |
| híbrido | archivo compuesto con líneas del original y del clon |
| clon perfecto | archivo idéntico al clon; no se escribe ni cuenta como omitido |
| sin base en este Mac | app del manifiesto que no está instalada aquí |
| perturbación en la Fuerza | el destino cambió durante la revisión |
| clon con mutaciones | la integridad del clon no coincide con el manifiesto |
| registro de los kaminoanos | manifiesto `manifest.json` del clon |

Los identificadores internos (`LocalFile`, `RowDecision.LOCAL`, `snapshot`, `SNAPSHOT_APP_ID`)
no cambian: el formato del archivo y los holocrones antiguos siguen siendo compatibles.

### La TUI

Una sola app de terminal con ANSI propio reúne menú, clonación, despliegue y actualización. El menú muestra el
wordmark KAMINO, ocho cápsulas con la última marcada como el original, una frase fija por sesión y
las opciones Clonar, Desplegar, Actualizar, Configuración y Abandonar Kamino. Las flechas mueven la
selección y `Enter` acepta la opción seleccionada; no hay atajos de una tecla. Escape nunca cierra. La línea de última misión aparece tras el primer flujo de la sesión y no se persiste.

Todas las pantallas se manejan con teclado y muestran al pie una leyenda de las teclas activas, sin
botoneras inferiores: `Esc` retrocede, `Enter` continúa o acepta y las flechas mueven el foco. En
C1, `Espacio` marca o desmarca una app sin perder el cursor ni el foco.

Cada flujo muestra el paso actual y el total visible. Los argumentos reducen el total cuando omiten
pantallas. La ayuda contextual se abre con `?` y, en campos de texto, también con `F1`. Todo error
usa cuatro partes —qué pasó, estado de seguridad, qué hacer y acciones— con el detalle plegado
(`t`) y copiable (`c`). Por debajo de 80×24 aparece un aviso que conserva el estado y bloquea la
salida; al redimensionar de vuelta se recupera la pantalla.

`Ctrl+C` cancela antes del punto de no retorno: en clonar, antes de renombrar el temporal; en
desplegar, antes de crear el holocrón; en actualizar, antes de `os.replace`. Después de ese punto
se ignora con aviso. La barra de progreso avanza por archivos y `Ctrl+C` elimina el temporal.

La TUI usa la paleta ANSI del terminal: el tema por defecto es `ansi-dark` y `TEXTUAL_THEME`
admite `ansi-light`; cualquier otro valor cae a `ansi-dark`. Con `NO_COLOR` se muestra en
monocromo y cada estado conserva icono y palabra. El wordmark se oculta sin UTF-8 o con
`KAMINO_NO_LOGO`. `KAMINO_PLAIN` fuerza la salida de texto para scripting.

### Configuración de la TUI

La opción **Configuración** del menú, situada después de Actualizar, permite
ver y editar las tres rutas en una pantalla. La pantalla se guarda con `Ctrl+S`; `Esc` vuelve sin
escribir. El archivo solo lo lee y escribe la TUI: la vía no interactiva de la CLI no aplica estos
valores y usa los defaults integrados.

```toml
[catalog]
dir = "~/.config/kamino/apps"

[clone]
output_dir = "~/.config/kamino/vault"

[deploy]
snapshots_dir = "~/.config/kamino/vault/snapshot"
```

| Clave | Valor por defecto | Uso en la TUI |
| --- | --- | --- |
| `catalog.dir` | `~/.config/kamino/apps` | Definiciones TOML de apps del usuario |
| `clone.output_dir` | `~/.config/kamino/vault` | Directorio propuesto en C2 y destino del clon |
| `deploy.snapshots_dir` | `~/.config/kamino/vault/snapshot` | Directorio de los holocrones |

El archivo es opcional y se crea solo al guardar desde la pantalla. Admite únicamente esas tres
claves; un TOML inválido, una sección o clave desconocida, un tipo incorrecto, una ruta relativa,
`~usuario` o una ruta existente que no sea un directorio se rechazan. Si el archivo es inválido al
arrancar, la TUI abre Configuración con el error y los valores legibles; no lo sobrescribe hasta
que se guarde. Los directorios configurados se crean al clonar o desplegar, no al guardar. Los
cambios guardados se aplican de inmediato a los flujos de la misma sesión. `--force`, `--yes` y
`--dry-run` no se configuran: son decisiones por operación. `ui.plain` y `ui.theme` tampoco existen
en este esquema.

### Conflictos en el despliegue

- D0 valida solo el manifiesto: rechaza un `tar.gz` dañado, con más de un `manifest.json`, con un
  manifiesto de más de 4 MiB o con un formato más nuevo. Los avisos de arquitectura o macOS no
  bloquean. La integridad completa se comprueba en D1 antes de extraer.
- D1 lista solo las apps instaladas con su número de archivos; las no instaladas aparecen aparte
  como “sin base en este Mac”. Ninguna está marcada por defecto y `--apps` omite esta pantalla.
- D2 lista solo los conflictos y muestra los conteos de nuevos, clones perfectos, no soportados y
  avisos. `r` revisa, `c` aplica el clon y `o` conserva el original; `C`/`O` actúan sobre todos.
  Confirmar queda bloqueado hasta que todos tengan decisión. Sin conflictos se pasa directo a D4.
- D3 muestra un bloque a la vez: el contexto se ve una vez y cada línea cambiada ofrece la versión
  del original y la del clon. `●` marca lo que irá al archivo final y `○` lo que no. `c`/`o` actúan
  sobre el bloque, `C`/`O` sobre todo el archivo y `Tab`/`Mayús+Tab` cambian de bloque. `Esc` vuelve
  y valida el híbrido.
- Antes de aplicar un híbrido se validan TOML, JSON, INI y plist con la biblioteca estándar. Si
  ambos originales eran válidos y el híbrido no, se bloquea; si alguno ya era inválido, se avisa y
  se permite aplicar. YAML y shell no se validan.
- D4 muestra los conteos antes de escribir, el número de archivos protegidos y el directorio del
  holocrón. Retirada está preseleccionada; las flechas recorren las acciones, `Enter` acepta la
  elegida y no hay atajo de una tecla para desplegar. Volver o cancelar no escribe ni crea holocrón.
  Si no hay destinos existentes, no se crea un holocrón vacío.
- D5 distingue desplegados, híbridos, omitidos y sin cambios, muestra la ruta del holocrón y la
  orden `kamino deploy <ruta>` para deshacer. Solo los conservados en el original, los no soportados
  y los `--apps` no cumplidos producen código 3. Un fallo después del holocrón se presenta como
  “Despliegue incompleto” con los aplicados, el fallido y el holocrón, y termina con código 1.
- `--yes` no abre la TUI y resuelve todos los conflictos aplicando el clon. `--dry-run` tampoco
  abre la TUI: usa `--apps` o todas las apps instaladas y vuelca los diffs a la salida estándar.

## Privacidad y aviso previo

- Los clones (`tar.gz`) y los holocrones de seguridad **no están cifrados**. Trátalos como datos
  sensibles y guárdalos en una ubicación protegida.
- Antes de `clone`, incluso sin terminal interactiva y con `--dry-run`, la CLI muestra un aviso
  en español: los ajustes incluidos pueden contener secretos incrustados. No se pide confirmación
  adicional ni se imprimen valores.
- Los clones nuevos excluyen archivos de credenciales conocidos (`.env`, `.env.*`, `auth.json`,
  `.credentials.json`, `secrets.yaml`, `secrets.yml` y `hosts.yml` de GitHub CLI), historiales,
  sesiones, cachés, registros y metadatos `.git`. El filtro se aplica antes de leer el contenido.
- Los archivos de configuración permitidos se conservan byte a byte, aunque contengan claves
  incrustadas: Kamino no redacta ni modifica ajustes.
- Las exclusiones rigen la creación de clones nuevos. Al desplegar un clon anterior, sus
  archivos se aplican igual y el holocrón previo conserva completos los destinos que cambian,
  aunque ahora estén excluidos.

## Catálogo de aplicaciones

Cada herramienta tiene un id que se usa con `--apps`. La detección solo comprueba el ejecutable o
el identificador de la aplicación; nunca ejecuta la herramienta.

| Grupo | Ids |
| --- | --- |
| Agentes y asistentes | `opencode`, `claude-code`, `codex`, `gemini-cli`, `copilot-cli`, `pi`, `aider`, `goose`, `gentle-shell`, `gentle-ai`, `gga`, `openspec`, `caveman` |
| Terminales y utilidades | `lazygit`, `micro`, `navi`, `glow`, `bat`, `eza`, `fd`, `github-cli`, `git`, `herdr`, `yazi`, `ghostty`, `oh-my-zsh` |

Ejemplo:

```bash
kamino clone --apps opencode,claude-code,herdr,ghostty,oh-my-zsh -o clon.tar.gz
kamino deploy clon.tar.gz --yes
```

Límites del catálogo:

- Solo se cubren las ubicaciones globales documentadas de cada herramienta. Kamino no descubre
  archivos de configuración de repositorios ni de proyectos.
- Una definición propia en `~/.config/kamino/apps/` sustituye por completo a la integrada con el
  mismo id. Sus rutas deben ser relativas al HOME; si una ruta es absoluta o contiene `..`, el
  catálogo se rechaza. Las definiciones del usuario no pueden desactivar el confinamiento.
- Las ubicaciones alternativas se admiten solo cuando están documentadas y dentro del HOME. Herdr
  admite `HERDR_CONFIG_PATH` y XDG; Ghostty admite su directorio XDG. Otras variables o rutas
  elegidas por scripts requieren una definición personalizada.
- No se copian binarios, instalaciones de plugins, bases de datos de sesiones o memoria, ni
  índices como los de Engram o CodeGraph. Tampoco se crean entradas para herramientas cuyos ajustes
  dependen de variables del shell (`fzf`, `rg`, `zoxide`).
- Un directorio de configuración puede mezclar ajustes y estado. Solo se incluyen las rutas
  declaradas y se excluyen credenciales, historiales, sesiones, cachés y registros.

### Oh My Zsh

La entrada `oh-my-zsh` requiere `zsh` **y** `~/.oh-my-zsh/oh-my-zsh.sh`. Incluye `.zshrc`,
`.zprofile`, `.zshenv` y `.oh-my-zsh/custom` (ajustes y temas personales). Se excluyen
`.oh-my-zsh/custom/plugins` y los metadatos `.git`; los plugins deben reinstalarse por separado en
el equipo de destino. Kamino no evalúa código de shell: si `ZSH` o `ZSH_CUSTOM` se definen en
`.zshrc` o apuntan fuera del HOME, esas rutas no se descubren; usa una definición personalizada con
rutas dentro del HOME.

### Herdr

Se conserva `.config/herdr/config.toml`, incluidos `[theme]`, `[theme.custom]`, `[keys]` y
`[[keys.command]]`, y la configuración permitida de plugins. También se incluye
`HERDR_CONFIG_PATH` y la variante XDG cuando apuntan dentro del HOME; un valor externo se informa
sin leerse. Los `.env` y credenciales de plugins se excluyen, y no se copian los programas de los
plugins.

### Ghostty

- Se contemplan `config` y `config.ghostty` en `~/.config/ghostty/` y en
  `~/Library/Application Support/com.mitchellh.ghostty/`, además de la variante XDG cuando
  `XDG_CONFIG_HOME` queda dentro del HOME.
- Se incluyen los temas personalizados dentro del HOME y los archivos referenciados por
  `config-file`. Las referencias pueden ser relativas al archivo que las declara, absolutas dentro
  del HOME, opcionales con `?`, repetidas o anidadas. Los ciclos y los límites se informan sin
  ejecutar instrucciones.
- Los temas integrados de Ghostty no se copian. Las referencias a credenciales conocidas o a rutas
  fuera del HOME se omiten antes de leer su contenido.
- Los atajos globales viven en las configuraciones incluidas. Si un ajuste conserva una ruta
  absoluta de otro HOME, el despliegue avisa sin reescribir el archivo: puede requerir un ajuste
  manual.
- Tras desplegar en otro equipo puede ser necesario volver a conceder permisos de accesibilidad de
  macOS a Ghostty para los atajos globales.

## Desarrollo

```bash
go build ./...               # compilar
go test ./... -race          # pruebas
go vet ./...                 # vet
gofmt -l .                   # formato (debe estar vacío)
golangci-lint run ./...      # lint (si está instalado)
make release-snapshot        # binario local dist/kamino-arm64
go test ./internal/integration -run Cross  # compatibilidad con Python
```

## Estructura

```
cmd/kamino/            # punto de entrada del binario
internal/
├── cli/               # argumentos, gating de TUI, salida y códigos
├── app/               # flujos de texto compartidos con la CLI
├── texts/             # única fuente de textos visibles y glosario
├── ui/                # salida no interactiva
├── defaults/          # rutas por defecto compartidas
├── config/            # configuración TOML exclusiva de la TUI
├── clone/             # plan_clone y write_clone
├── deploy/            # sesión, plan, clasificación y aplicación
├── update/            # consulta, descarga/verificación y reemplazo
├── conflicts/         # SequenceMatcher, diff, composición y validación
├── progress/          # contrato de progreso y cancelación
├── tui/               # app única: menú, C1-C4, D0-D5, U1, configuración
├── plan/              # Plan, Operation y recuento compartido
├── catalog/           # AppDefinition y apps/*.toml embebidos
├── detect/            # detección de apps y versiones
├── archive/           # tar.gz, manifiesto y checksums
├── ghostty/           # lectura acotada de config-file y temas
├── paths/             # SafeHome con openat/O_NOFOLLOW
├── snapshot/          # holocrón de seguridad previo a desplegar
└── errors/            # errores propios
```

`legacy/python/` conserva el oráculo Python y `testdata/golden/` los fixtures de compatibilidad.


Los diseños de referencia están en `docs/` (`Diseño funcional.md` y `Diseño técnico.md`).
