# kamino

CLI para macOS que respalda las configuraciones de las aplicaciones elegidas en **un único
archivo portable** (`tar.gz` con `manifest.json`) y las restaura después, aplicando solo lo que
corresponda a las apps que estén instaladas.

## Estado

MVP v1 implementado: `backup` y `restore` funcionales con catálogo declarativo, detección de
apps, archivado `tar.gz` con manifiesto y checksums, CLI interactiva y revisión de conflictos por
líneas dentro de cada bloque en una TUI de `textual`.

## Requisitos

- **Instalación con el binario:** macOS con Apple Silicon (`arm64`) y `curl`. El binario es
  autocontenido y no requiere instalar Python.
- **Instalación desde el código fuente o desarrollo:** macOS y Python 3.11.4 o superior (`tarfile`
  usa `filter="data"`).

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
ejecución de forma atómica. No requiere `sudo` y no toca configuraciones, respaldos ni
instantáneas. Si ya estás en la última versión, o tienes una más reciente, no descarga ni escribe
nada.

La actualización solo está disponible para el binario autocontenido. En una instalación desde
código o con `uv`/`pip`, actualiza con `uv sync`, `uv tool upgrade kamino-cli` o el comando
equivalente de tu gestor.

El comando termina con código 0 tanto si actualiza como si ya está al día, y con 1 ante cualquier
error. La opción también aparece en el menú que se abre al ejecutar `kamino` sin argumentos.

## Instalación (desarrollo)

```bash
uv sync --all-extras
uv run kamino --help
```

Sin `uv`, con `pip`:

```bash
python3.11 -m venv .venv
.venv/bin/pip install -e ".[dev]"
.venv/bin/kamino --help
```

## Distribución

Instalación como herramienta con `uv` (diseño técnico §8):

```bash
uv tool install git+https://github.com/Gunmer/kamino
```

El paquete se publica como `kamino-cli` (el nombre `kamino` ya está ocupado en PyPI) y expone el
comando `kamino`.

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
4. Se actualizan `pyproject.toml`, `src/kamino/__init__.py`, `uv.lock` y `CHANGELOG.md`. El binario
   se construye y comprueba antes de empujar el commit de release y el tag `vX.Y.Z`.
5. Se crea una GitHub Release con las notas y el binario autocontenido `kamino-arm64`
   (PyInstaller `onefile`, Apple Silicon), incluido el catálogo de aplicaciones. No se distribuye
   actualmente un binario Intel. La release permanece en borrador hasta subir el binario.

Por ejemplo, desde `0.1.0`, `fix: corrige la restauración` produce `0.1.1`, mientras que
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
kamino                       # TUI: lista interactiva con los comandos disponibles
kamino --help
kamino backup --apps vscode,iterm2,git
kamino restore kamino-<equipo>-<fecha>.tar.gz
kamino update                # actualiza el binario a la última release
```

`-h` es alias de `--help` en el nivel principal y en `backup`, `restore` y `update`, y `-V` es
alias de `--version`:

```bash
kamino -h
kamino backup -h
kamino restore -h
kamino update -h
kamino -V
```

En `backup`, `-a` equivale a `--apps`, `-o` a `--output`, `-n` a `--dry-run` y `-v` a `--verbose`:

```bash
kamino backup -a vscode,iterm2,git -o respaldo.tar.gz -v
kamino backup -a vscode,iterm2,git -o respaldo.tar.gz -n   # simulación: no crea el archivo
```

En `restore`, `-a` equivale a `--apps`, `-n` a `--dry-run`, `-y` a `--yes` y `-v` a `--verbose`; el archivo de respaldo sigue siendo un argumento posicional:

```bash
kamino restore respaldo.tar.gz               # TUI: elige apps y revisa conflictos
kamino restore respaldo.tar.gz -a git,zsh    # sin selección de apps; revisa conflictos
kamino restore respaldo.tar.gz -y -v         # aplica todo sin preguntar
kamino restore respaldo.tar.gz -n            # simulación: imprime diffs, no escribe
```

Sin terminal interactiva, una restauración real requiere `-y` o `--yes`; la simulación con `-n`
no lo requiere. `-a` valida los ids contra las apps restaurables: los desconocidos o no instalados
se avisan y cuentan como omitidos (código de salida 3).

### Conflictos en la restauración

- En una terminal interactiva, `restore` abre una TUI. La primera pantalla elige apps (ninguna
  marcada por defecto); después aparece solo la tabla de archivos en conflicto, con los conteos de
  nuevos, idénticos y no soportados.
- Cada conflicto de texto se revisa por líneas dentro de cada bloque: la cabecera y el contexto no
  son decidibles y cada línea cambiada empieza en la versión del respaldo. `Espacio` o `Enter`
  alterna la línea resaltada, `A`/`L` toman todo el archivo del respaldo o del local, y `Shift+A`/
  `Shift+L` solo el bloque actual; `n`/`p` saltan entre bloques. La composición puede mezclar líneas
  de un mismo bloque y no se muestra una vista previa completa del archivo.
- Si una línea sin salto final queda antes de otra línea elegida, la composición le añade el final
  de línea dominante local; solo la última línea puede conservar la ausencia de salto final.
- Los binarios, los textos de más de 5 MB y los enlaces con enlace se resuelven como archivo entero:
  aplicar el respaldo u omitir.
- Antes de aplicar una mezcla se valida TOML, JSON, INI y plist con la biblioteca estándar. Si ambos
  originales eran válidos y la mezcla no, se bloquea y se vuelve al detalle; si algún original ya
  era inválido, se avisa y se permite aplicar. YAML y shell no se validan.
- La confirmación final muestra los conteos de restaurados, mezclados y omitidos y el directorio
  del snapshot; hay que confirmar explícitamente antes de escribir. Volver o cancelar no escribe ni
  crea snapshot. El resumen final incluye la ruta del snapshot creado.
- Si un destino cambia mientras la revisión está abierta, se aborta toda la operación antes de
  escribir y no se crea snapshot.
- `--yes` no abre la TUI y resuelve todos los conflictos aplicando el respaldo. `--dry-run` tampoco
  abre la TUI: usa `--apps` o todas las apps instaladas y vuelca los diffs a la salida estándar.

La TUI usa la paleta ANSI del terminal: el tema por defecto es `ansi-dark` y `TEXTUAL_THEME`
permite elegir otro tema válido de Textual, por ejemplo `ansi-light`. Así el fondo, el texto, los
bordes y los botones siguen el esquema de colores de la terminal.

Al ejecutar `kamino` sin comandos (en una terminal interactiva) se muestra una lista con los
comandos disponibles (`backup`, `restore` y `update`) para elegir uno. Sin terminal, se muestra la
ayuda.

## Privacidad y aviso previo

- Los respaldos (`tar.gz`) y las instantáneas **no están cifrados**. Trátalos como datos
  sensibles y guárdalos en una ubicación protegida.
- Antes de `backup`, incluso sin terminal interactiva y con `--dry-run`, la CLI muestra un aviso
  en español: los ajustes incluidos pueden contener secretos incrustados. No se pide confirmación
  adicional ni se imprimen valores.
- Los respaldos nuevos excluyen archivos de credenciales conocidos (`.env`, `.env.*`, `auth.json`,
  `.credentials.json`, `secrets.yaml`, `secrets.yml` y `hosts.yml` de GitHub CLI), historiales,
  sesiones, cachés, registros y metadatos `.git`. El filtro se aplica antes de leer el contenido.
- Los archivos de configuración permitidos se conservan byte a byte, aunque contengan claves
  incrustadas: Kamino no redacta ni modifica ajustes.
- Las exclusiones rigen la creación de respaldos nuevos. Al restaurar un respaldo anterior, sus
  archivos se aplican igual y la instantánea previa conserva completos los destinos que cambian,
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
kamino backup --apps opencode,claude-code,herdr,ghostty,oh-my-zsh -o respaldo.tar.gz
kamino restore respaldo.tar.gz --yes
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
  absoluta de otro HOME, la restauración avisa sin reescribir el archivo: puede requerir un ajuste
  manual.
- Tras restaurar en otro equipo puede ser necesario volver a conceder permisos de accesibilidad de
  macOS a Ghostty para los atajos globales.

## Desarrollo

```bash
uv run pytest          # pruebas
uv run ruff check .    # lint
uv run ruff format .   # formato
```

## Estructura

```
src/kamino/
├── cli.py            # comandos backup, restore y update, y menú TUI sin argumentos
├── ui.py             # selección interactiva, menú de comandos, tablas, progreso
├── backup.py         # caso de uso: plan y ejecución del respaldo
├── restore.py        # caso de uso: sesión, plan y ejecución de la restauración
├── update.py         # actualización del binario desde GitHub
├── conflicts.py      # diff por líneas, composición y validación de mezclas
├── tui/              # TUI de restauración: apps, conflictos, detalle, confirmación y resumen
├── plan.py           # Plan y Operation, compartidos
├── catalog/
│   ├── models.py     # AppDefinition (validación)
│   ├── loader.py     # catálogo embebido y del usuario
│   └── apps/*.toml   # una definición por app
├── detect.py         # ¿está instalada? ¿qué versión?
├── archive.py        # tar.gz, manifiesto, checksums
├── ghostty.py        # lectura acotada de config-file y temas de Ghostty
├── paths.py          # normalización y acceso seguro relativo al home
├── snapshot.py       # copia previa a restaurar
└── errors.py         # excepciones propias
```

Los diseños de referencia están en `docs/` (`Diseño funcional.md` y `Diseño técnico.md`).
