# Formatos de datos

## 4. Modelos de datos

### 4.1 Definición de app (TOML)

```toml
id = "herdr"
name = "herdr"
command = "herdr"
exclude = ["**/*.log", "**/session.json"]  # logs y estado de sesión, no configuración

paths = [
  ".config/herdr/config.toml",
  ".config/herdr/plugins/config",
]
```

- `id`: único, en minúsculas y guiones; es el nombre que se usa en `--apps` y dentro del archivo.
- `bundle_id`: sirve para detectar si la app está instalada.
- `command`: alternativa a `bundle_id` para herramientas de terminal sin `.app` (por ejemplo `git`, `yazi` o `herdr`); se detectan con `shutil.which`.
- `search_paths`: rutas adicionales donde buscar el binario de `command` si no está en el PATH (por ejemplo `/opt/homebrew/bin`). Opcional.
- `command_aliases`: nombres alternativos del ejecutable, como `gsh` para Gentle Shell. Opcional.
- `install_marker`: archivo relativo al home que también debe existir para considerar instalada la app, como `.oh-my-zsh/oh-my-zsh.sh`. Opcional.
- `paths`: rutas obligatorias relativas al home, archivos o carpetas. Su ausencia se informa como omisión. Las rutas fuera del home no se admiten.
- `optional_paths`: rutas que solo se incluyen si existen; su ausencia no es una omisión. Opcional.
- `optional_patterns`: patrones con un directorio fijo y comodines solo en el último componente, como `.codex/*.config.toml`. Opcional.
- `env_paths`: variables de entorno con una ruta absoluta que debe quedar dentro del home, como `HERDR_CONFIG_PATH`. Opcional.
- `xdg_paths`: rutas relativas a `XDG_CONFIG_HOME`, admitidas solo si ese directorio está dentro del home. Opcional.
- `exclude`: patrones glob que se descartan al expandir las rutas.
- `[[install]]`: recetas de instalación ordenadas, opcionales. Solo se admiten los métodos `brew`, `npm`, `installer`, `binary` y `github-release`; la lista completa de reglas está en la sección 4.1.1.
- Una definición necesita al menos una ruta obligatoria, opcional o un patrón.

#### 4.1.1 Recetas de instalación

Una receta permite que Kamino instale la app si falta al implantar un clon. Vive solo en el catálogo: el manifiesto del clon nunca aporta URLs, comandos ni métodos.

- `method`: `brew`, `npm`, `installer`, `binary` o `github-release`. Cualquier otro valor se rechaza.
- El orden de las recetas es la preferencia (convención del catálogo integrado: `brew`, `npm`, `installer`, `github-release`). Se usa la primera que aplica a la plataforma y cuya herramienta (`brew`, `npm`) está en el PATH.
- `brew`: exactamente una de `formula` o `cask`. Kamino nunca instala Homebrew; si `brew` no está en el PATH, se prueba la siguiente receta.
- `npm`: `package` con un nombre de paquete válido (`@ámbito/` opcional, sin espacios ni metacaracteres). Ejecuta `npm install --global <package>`, sin `sudo` ni `--prefix`, y por tanto la última versión publicada. Kamino nunca instala Node ni npm; si `npm` no está en el PATH, se prueba la siguiente receta.
- `installer`: `url` con `https`. Se descarga a un temporal y se ejecuta con `sh` desde ese archivo; la confirmación muestra la URL exacta.
- `binary`: `url` con `https`, `sha256` de 64 caracteres hexadecimales y `dest` relativo al home. Se descarga, se verifica el SHA-256 y solo entonces se mueve al destino con modo 0755.
- `github-release`: `repo` (`propietario/nombre`), `asset` (nombre de fichero; admite `{version}`), `format` (`raw`, `zip` o `tar.gz`) y `checksums` opcional. Con `raw` exige `dest`; con `zip` o `tar.gz` exige una lista `extract` con una o más entradas `file` (miembro del archivo) y `dest` (relativo al home), y no admite `dest`. No admite `url`, `sha256`, `package`, `formula` ni `cask`.
  - El último release se resuelve con la redirección de `https://github.com/<repo>/releases/latest` (sin API ni límite de peticiones) y se descarga `…/releases/download/<tag>/<asset>`. `{version}` es el tag sin `v`.
  - Antes de escribir se comprueba que todos los miembros nombrados existen y son ficheros regulares; luego se escriben con `SafeHome` (reemplazo atómico, modo 0755). Un miembro ausente o con ruta con `..` no instala nada.
  - Con `checksums` se descarga ese fichero del mismo release y se compara el SHA-256 del asset con su línea (`<hash>  <nombre>`) o con un hash suelto si el fichero es de un solo asset; si falta, no lista el asset o no coincide, no se instala nada. Sin `checksums` solo hay HTTPS y el origen GitHub (rebaja consciente respecto a `binary`), y la confirmación y `--dry-run` lo indican. Detecta corrupción, no un release comprometido.
- `os` (`darwin`) y `arch` (`arm64` o `amd64`): opcionales. Las recetas que no coinciden con el entorno se ignoran.

```toml
[[install]]
method = "brew"
cask = "ghostty"

[[install]]
method = "binary"
url = "https://example.com/tool-arm64"
sha256 = "<64 caracteres hexadecimales>"
dest = ".local/bin/tool"
arch = "arm64"

[[install]]
method = "npm"
package = "@openai/codex"

[[install]]
method = "github-release"
repo = "sxyazi/yazi"
asset = "yazi-aarch64-apple-darwin.zip"
format = "zip"
arch = "arm64"
extract = [
  { file = "yazi-aarch64-apple-darwin/yazi", dest = ".local/bin/yazi" },
  { file = "yazi-aarch64-apple-darwin/ya", dest = ".local/bin/ya" },
]
```

### 4.2 Manifiesto (`manifest.json`)

```json
{
  "format_version": 1,
  "tool_version": "0.1.0",
  "created_at": "2026-10-28T21:30:00Z",
  "macos_version": "15.1",
  "arch": "arm64",
  "hostname": "mac-origen",
  "apps": [
    {
      "id": "ghostty",
      "name": "Ghostty",
      "bundle_id": "com.mitchellh.ghostty",
      "version": "1.2.0",
      "files": [
        {"path": ".config/ghostty/config", "sha256": "…", "size": 1432, "mode": "0644"}
      ]
    }
  ]
}
```

Los valores son ilustrativos. `format_version` permite evolucionar el formato: `implant` rechaza un archivo con una versión mayor que la que entiende. `mode` es un permiso octal entre `0000` y `0777`; un valor distinto invalida el manifiesto en lugar de sustituirse por `0644`. Dos rutas distintas que solo difieren en mayúsculas también invalidan el manifiesto, porque colisionarían en un volumen APFS que no distingue mayúsculas.

### 4.3 Estructura del archivo

```
clon.tar.gz
├── manifest.json
└── apps/
    └── ghostty/
        └── .config/ghostty/config
```

Los checksums se calculan al construir el plan, así `manifest.json` puede escribirse primero y `inspect` (v2) lo lee sin recorrer todo el archivo.

