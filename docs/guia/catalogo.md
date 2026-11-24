# Catálogo de aplicaciones

Cada herramienta tiene un id que se usa con `--apps`. La detección solo comprueba el ejecutable o
el identificador de la aplicación; nunca ejecuta la herramienta.

| Grupo | Ids |
| --- | --- |
| Agentes y asistentes | `opencode`, `claude-code`, `codex`, `gemini-cli`, `copilot-cli`, `pi`, `gentle-shell`, `openspec` |
| Terminales y utilidades | `lazygit`, `micro`, `glow`, `bat`, `github-cli`, `git`, `herdr`, `yazi`, `ghostty`, `oh-my-zsh` |

Métodos de instalación de cada app (en orden de preferencia; ver «Añadir tus propias apps»):

| App | Métodos |
| --- | --- |
| `opencode`, `claude-code`, `copilot-cli` | `brew`, `npm`, `github-release` |
| `codex`, `openspec` | `brew`, `npm`, `installer`, `github-release` |
| `gemini-cli` | `brew`, `npm` |
| `pi` | `npm`, `installer` |
| `lazygit`, `micro`, `glow`, `bat`, `github-cli`, `yazi` | `brew`, `github-release` |
| `herdr` | `brew`, `installer`, `github-release` |
| `git`, `ghostty` | `brew` |
| `oh-my-zsh` | `installer` |
| `gentle-shell` | ninguno |

`github-release` se verifica con el fichero de sumas del release salvo en `bat`, `yazi`,
`opencode`, `herdr` y `codex`, que no publican uno que cubra el binario descargado: para esas la
confirmación y `--dry-run` avisan «sin verificación de suma». `gemini-cli` no publica binarios de
macOS y el archivo de `pi` necesita ficheros auxiliares, por lo que no tienen esa receta.

Ejemplo:

```bash
kamino clone --apps opencode,claude-code,herdr,ghostty,oh-my-zsh -o clon.tar.gz
kamino implant clon.tar.gz --yes
```

Límites del catálogo:

- Solo se cubren las ubicaciones globales documentadas de cada herramienta. Kamino no descubre
  archivos de configuración de repositorios ni de proyectos.
- Una definición propia en `catalog.dir` (por defecto `~/.config/kamino/apps/`) sustituye por
  completo a la integrada con el mismo id. Sus rutas deben ser relativas al HOME; si una ruta es
  absoluta o contiene `..`, el catálogo se rechaza. Las definiciones del usuario no pueden
  desactivar el confinamiento.
- Las ubicaciones alternativas se admiten solo cuando están documentadas y dentro del HOME. Herdr
  admite `HERDR_CONFIG_PATH` y XDG; Ghostty admite su directorio XDG. Otras variables o rutas
  elegidas por scripts requieren una definición personalizada.
- No se copian binarios, instalaciones de plugins, bases de datos de sesiones o memoria, ni
  índices como los de Engram o CodeGraph. Tampoco se crean entradas para herramientas cuyos ajustes
  dependen de variables del shell (`fzf`, `rg`, `zoxide`).
- Un directorio de configuración puede mezclar ajustes y estado. Solo se incluyen las rutas
  declaradas y se excluyen credenciales, historiales, sesiones, cachés y registros.

## Añadir tus propias apps

Kamino carga las definiciones de `catalog.dir` (por defecto `~/.config/kamino/apps/`) además del
catálogo integrado. Una definición con el mismo id reemplaza la integrada.

```bash
kamino apps new miapp --command miapp --path .config/miapp/config.toml
kamino apps validate
kamino apps list
```

- `apps new` exige `--command` o `--bundle-id` y al menos un `--path` o `--optional-path`. Las
  rutas son relativas al HOME y los flags de ruta se pueden repetir. La definición se valida
  antes de escribir; si algo falla, no queda ningún archivo.
- `apps new --print` escribe el TOML en la salida estándar sin crear archivos. Es útil para
  preparar una definición del catálogo embebido.
- `apps new --force` solo reemplaza `<id>.toml`. Si otro archivo del usuario ya declara ese id, la
  operación se rechaza y se nombra el archivo en conflicto.
- `apps validate` valida el catálogo efectivo; con un archivo o un directorio, valida esa ruta.
  Termina con código 0 si todo es válido y 1 si hay errores. Avisa, sin fallar, cuando una
  definición no declara `exclude` o su id no coincide con el nombre del archivo.
- Las recetas `[[install]]` se validan con las mismas reglas al cargar el catálogo y en
  `apps validate`: `brew`, `npm`, `installer`, `binary` o `github-release`; `https` obligatorio;
  `sha256` de 64 caracteres en `binary`. Una receta inválida se muestra con su número y su motivo. Ver `docs/tecnico/formatos.md`
  (sección 4.1.1).
- El orden de las `[[install]]` es la preferencia: se usa la primera aplicable cuya herramienta
  (`brew`, `npm`) esté en el `PATH`. El catálogo integrado las ordena `brew`, `npm`, `installer`,
  `github-release`.
- `npm` (`package = "@openai/codex"`) ejecuta `npm install --global <package>`: siempre la última
  versión, sin `sudo` y respetando el `prefix` de npm del usuario. Si `npm` no está, se omite la
  receta; Kamino no instala Node.
- `github-release` descarga el asset del último release de `repo` (`propietario/nombre`) solo desde
  `https://github.com/<repo>/releases/`. `asset` puede llevar `{version}` (el tag sin `v`) y
  `format` es `raw` (con `dest`), `zip` o `tar.gz` (con una lista `extract` de `file` y `dest`).
  Solo se extraen esos ficheros regulares, con modo 0755. Con `checksums` (nombre del fichero de
  sumas del mismo release) se verifica el SHA-256 antes de escribir y, si falta, no lista el asset
  o no coincide, no se instala nada. Sin `checksums` solo hay HTTPS y el origen GitHub, y la
  confirmación y `--dry-run` lo avisan. El fichero de sumas viene del mismo origen: detecta
  corrupción, no un release comprometido.
- `apps customize <id>` copia la definición integrada de `<id>` (con sus `[[install]]` y
  comentarios) a `<id>.toml` en el catálogo del usuario, para editarla. Admite `--print` (TOML a
  stdout, sin archivos) y `--force` (reemplaza `<id>.toml`). Un id que no es integrado termina con
  código 2; un archivo existente sin `--force` o un id declarado en otro archivo, con código 1.
  La copia reemplaza por completo a la integrada y no recibe sus cambios futuros (p. ej. recetas
  nuevas): revisa `kamino apps customize <id> --print` tras actualizar Kamino si quieres
  incorporarlas.
- `apps list` muestra el catálogo fusionado con el origen (`integrado` o `usuario`) y el estado de
  cada archivo. Un archivo inválido se muestra con `ERROR` y no impide listar el resto.
- `apps new` usa el código 2 para uso incorrecto (`ID` ausente, falta de detección o de rutas) y el
  código 1 para validación, duplicados, un archivo existente sin `--force` o un error de escritura.
- En la TUI, «Añadir app» es una acción dentro de Configuración, después de los cuatro campos.
  Crea un borrador oculto, abre `$VISUAL`, `$EDITOR` o `vi`, valida al volver y solo guarda si la
  definición es válida. Si el id reemplaza una app integrada o el archivo ya existe, pide
  confirmación. El borrador no se guarda si se cancela y el flujo vuelve a Configuración.
- En la TUI, «Personalizar app» (la fila siguiente en Configuración) lista las apps integradas y
  abre el mismo flujo con el borrador ya relleno con su definición integrada.
- La plantilla incluye la lista estándar de exclusiones de credenciales, cachés e historiales. Los
  campos avanzados (`optional_patterns`, `env_paths`, `xdg_paths`, `install_marker`,
  `command_aliases`) aparecen comentados como ejemplo.

## Oh My Zsh

La entrada `oh-my-zsh` requiere `zsh` **y** `~/.oh-my-zsh/oh-my-zsh.sh`. Incluye `.zshrc`,
`.zprofile`, `.zshenv` y `.oh-my-zsh/custom` (ajustes y temas personales). Se excluyen
`.oh-my-zsh/custom/plugins` y los metadatos `.git`; los plugins deben reinstalarse por separado en
el equipo de destino. Kamino no evalúa código de shell: si `ZSH` o `ZSH_CUSTOM` se definen en
`.zshrc` o apuntan fuera del HOME, esas rutas no se descubren; usa una definición personalizada con
rutas dentro del HOME.

## Herdr

Se conserva `.config/herdr/config.toml`, incluidos `[theme]`, `[theme.custom]`, `[keys]` y
`[[keys.command]]`, y la configuración permitida de plugins. También se incluye
`HERDR_CONFIG_PATH` y la variante XDG cuando apuntan dentro del HOME; un valor externo se informa
sin leerse. Los `.env` y credenciales de plugins se excluyen, y no se copian los programas de los
plugins.

## Ghostty

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
  absoluta de otro HOME, la implantación avisa sin reescribir el archivo: puede requerir un ajuste
  manual.
- Tras implantar en otro equipo puede ser necesario volver a conceder permisos de accesibilidad de
  macOS a Ghostty para los atajos globales.
