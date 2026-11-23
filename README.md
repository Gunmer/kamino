# kamino

CLI para macOS que clona las configuraciones de las aplicaciones elegidas en **un único
archivo portable** (`tar.gz` con `manifest.json`) y las implanta después, aplicando solo lo que
corresponda a las apps que estén instaladas.

- **Clonar** guarda las configuraciones elegidas en un archivo.
- **Implantar** las aplica en el mismo Mac o en otro, revisando los conflictos y creando antes un
  holocrón de seguridad para poder deshacer.
- **Catálogo declarativo**: añadir una app es añadir un archivo TOML.
- **TUI interactiva** con Bubble Tea, y salida de texto para scripts.

## Requisitos

- **Binario:** macOS con Apple Silicon (`arm64`) y `curl`. No necesita intérpretes ni gestores de
  paquetes.
- **Compilar o desarrollar:** Go 1.25 o superior.

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

## Uso rápido

```bash
kamino                                   # TUI: Clonar, Implantar, Actualizar, Configuración
kamino --help
kamino clone                             # elige apps, destino y revisa el plan
kamino clone -a opencode,ghostty,git -o clon.tar.gz
kamino implant clone-<aaaammdd>.tar.gz   # apps, conflictos y confirmación
kamino implant clon.tar.gz --yes         # sin preguntar: aplica el clon a todo
kamino implant clon.tar.gz --dry-run     # simulación: imprime diffs, no escribe
kamino update                            # actualiza el binario
```

Alias: `-h` (`--help`), `-V` (`--version`). En `clone`: `-a` `--apps`, `-o` `--output`,
`-n` `--dry-run`, `-v` `--verbose`, `-f` `--force`. En `implant`: `-a`, `-n`, `-y` `--yes`, `-v`.

Códigos de salida: `0` éxito, `1` error o cancelación, `3` éxito parcial con
elementos omitidos.

Sin `--output`, el clon se guarda en `~/.config/kamino/vault/clone-<aaaammdd>.tar.gz`.

## Guías

| Guía | Contenido |
| --- | --- |
| [Clonar e implantar](docs/guia/clonar-e-implantar.md) | opciones, destinos, holocrones, conflictos e instalación de apps ausentes |
| [Catálogo de aplicaciones](docs/guia/catalogo.md) | apps incluidas, límites, `kamino apps`, y notas de Oh My Zsh, Herdr y Ghostty |
| [Configuración](docs/guia/configuracion.md) | `config.toml` y la pantalla de Configuración |
| [La TUI](docs/guia/tui.md) | pantallas, teclas, temas y glosario visible |
| [Versionado y releases](docs/guia/releases.md) | cómo se versiona y publica |
| [Diseño](docs/README.md) | base de conocimiento técnica y de producto |

## Privacidad

- Los clones (`tar.gz`) y los holocrones **no están cifrados**. Trátalos como datos sensibles.
- Antes de `clone`, incluso con `--dry-run`, la CLI avisa de que los ajustes pueden contener
  secretos incrustados. No se imprimen valores.
- Los clones nuevos excluyen credenciales conocidas (`.env`, `.env.*`, `auth.json`,
  `.credentials.json`, `secrets.yaml`, `secrets.yml`, `hosts.yml` de GitHub CLI), historiales,
  sesiones, cachés, registros y metadatos `.git`. El filtro se aplica antes de leer el contenido.
- Los archivos permitidos se conservan byte a byte: Kamino no redacta ajustes.
- Las exclusiones rigen solo al crear clones. Al implantar un clon anterior se aplican sus
  archivos igualmente, y el holocrón previo conserva completos los destinos que cambian.

## Desarrollo

```bash
go build ./...               # compilar
go test ./... -race          # pruebas
go vet ./...                 # vet
gofmt -l .                   # formato (debe estar vacío)
golangci-lint run ./...      # lint (si está instalado)
make release-snapshot        # binario local dist/kamino-arm64
```

La estructura de paquetes está en [docs/tecnico/arquitectura.md](docs/tecnico/arquitectura.md).
