# Diseño

## Contexto

Ver `proposal.md` para la motivación y `specs/self-update/spec.md` y `specs/tui-ansi-theme/spec.md` para los requisitos. Este documento fija cómo se implementan.

Estado actual relevante:

- `kamino` es una CLI de Typer. `cli.py` es una capa fina; la lógica vive en módulos como `backup.py` y `restore.py`, y `ui.py` concentra la presentación. `src/` no lanza subprocesos.
- El binario se construye con PyInstaller (`onefile`, macOS arm64) y se publica como `kamino-arm64` en cada GitHub Release. La API de GitHub expone `tag_name`, `size` y `digest: sha256:...` del asset.
- `scripts/install.sh` es la vía de bootstrap: comprueba macOS arm64, descarga `releases/latest/download/kamino-arm64`, instala en `~/.local/bin/kamino`, pide confirmación si ya existe, marca el archivo como ejecutable e intenta quitar la cuarentena.
- El binario puede ejecutarse a través de un symlink (por ejemplo `~/.local/bin/kamino -> dist/kamino-arm64`). `sys.executable` apunta al ejecutable del proceso; resolverlo permite reemplazar el archivo real sin romper el enlace.
- Textual 8.2.8 incluye los temas `ansi-dark` y `ansi-light`. `KaminoRestoreApp` usa el tema por defecto (`textual-dark`), que es truecolor. `styles.tcss` ya usa tokens semánticos (`$surface`, `$primary`, `$warning`, `$error`, etc.) y `widgets.py` ya usa colores ANSI (`cyan`, `yellow`, `red`, `green`) para los diffs.

## Objetivos / No objetivos

**Objetivos:**

- Actualizar únicamente el binario autocontenido, de forma segura, atómica y sin `sudo`.
- No degradar la instalación ni descargar si ya se está en la última versión.
- Verificar integridad antes de reemplazar y dejar el binario actual intacto ante cualquier error.
- Mantener la actualización testeable sin red y sin depender de la máquina del desarrollador.
- Hacer que la TUI use la paleta ANSI del terminal, conservando `TEXTUAL_THEME` como escape.

**No objetivos:**

- Actualizar instalaciones de código, `uv`, `pip` o Homebrew.
- Comprobar actualizaciones automáticamente en cada arranque.
- Rollback o conservación del binario anterior.
- Barra de progreso de descarga, `--dry-run`/`--check` o selección de versión.
- Cambiar el formato del respaldo, el catálogo, la restauración, las instantáneas o el workflow de release.
- Firmar o notarizar el binario.

## Decisiones

### 1. Módulo `update.py` y capa CLI

La lógica vive en un nuevo `src/kamino/update.py`; `cli.py` solo declara el subcomando, resuelve el binario y traduce errores a códigos de salida. `ui.py` imprime el inicio y el resultado.

Alternativas descartadas:

- **Implementarlo en `cli.py`:** rompe la regla de capa fina y mezcla red, ficheros y presentación.
- **Ejecutar `scripts/install.sh` desde el comando:** el script no se empaqueta en el binario, instala en una ruta fija, pide confirmación y obliga a ejecutar código remoto. Además no permite actualizar un binario instalado en otra ruta.

### 2. Solo binario autocontenido

`installed_binary_path()` comprueba `getattr(sys, "frozen", False)`. Si no es un binario, lanza `UpdateError` con la vía correspondiente: `uv sync` para desarrollo, `uv tool upgrade kamino-cli` para una herramienta de `uv`, o la actualización de `pip` que corresponda. No se intenta detectar ni modificar otros gestores.

El destino es `Path(sys.executable).resolve()`. Así se reemplaza el archivo real y se conserva cualquier symlink. Se imprime la ruta real en el resumen.

### 3. Consulta de la release y comparación de versiones

Se consulta `https://api.github.com/repos/Gunmer/kamino/releases/latest` con `Accept: application/vnd.github+json` y `User-Agent: kamino/<version>`, timeout de 30 s.

Del JSON se extrae:

- `tag_name`, con formato `vX.Y.Z` (el mismo que valida `scripts/release.py`).
- El asset cuyo `name` sea `kamino-arm64`: `browser_download_url`, `size` y `digest`.

La comparación es SemVer numérica (`X.Y.Z` como tupla). Si la versión instalada es igual o mayor, se devuelve un resultado sin descarga. Si la release no tiene el asset esperado, el tag no es válido o falta el digest, se lanza `UpdateError`.

Alternativas descartadas:

- **Descargar siempre y leer `--version` del binario nuevo:** gasta 20 MB en cada consulta y ejecuta código no verificado.
- **Seguir el redirect de `/releases/latest`:** evita la API, pero no da tamaño ni digest.
- **Añadir `packaging`:** el formato del proyecto es estricto (`vX.Y.Z`), una comparación de tres enteros es suficiente.

### 4. HTTP y TLS con `urllib` y `certifi`

Se usa `urllib.request` de la biblioteca estándar con un contexto `ssl.create_default_context(cafile=certifi.where())`. `certifi` pasa a ser dependencia directa; el hook de PyInstaller ya incluye `certifi/cacert.pem`.

El motivo es no depender de las rutas de certificados compiladas en el Python de build (en esta máquina apuntan a `/opt/homebrew/etc/openssl@3/cert.pem`), que pueden no existir en el Mac de destino. La alternativa de llamar a `curl` evita la dependencia, pero introduce subprocesos en `src/` y ata la actualización a que `curl` exista.

### 5. Verificación de integridad

Mientras se descarga en bloques de 64 KiB se calcula el SHA-256 y se cuenta el tamaño. Al terminar:

1. El tamaño debe coincidir con `size` de la API.
2. El digest debe tener el formato `sha256:<64 hex>` y coincidir con `hmac.compare_digest`.

Si falta el digest o no coincide, se aborta. El binario descargado no se ejecuta antes de reemplazarlo; la confianza está en TLS más el digest publicado por GitHub. Esto protege frente a truncamientos y corrupción del CDN, no frente a una cuenta de GitHub comprometida; el binario sigue sin firma, igual que en el instalador.

### 6. Reemplazo atómico y limpieza

Se crea un temporal con `tempfile.NamedTemporaryFile(prefix=".kamino-update-", dir=target.parent, delete=False)`. El mismo directorio garantiza que `os.replace` sea atómico. Al terminar la descarga y la verificación:

1. `chmod` a `0o755` (o se conserva el modo del binario actual si ya tenía bits de ejecución).
2. `os.replace(temp, target)`.

Un `try/finally` elimina el temporal si algo falla. En macOS, reemplazar un ejecutable en marcha es seguro: el proceso actual sigue con el inode anterior y el siguiente arranque usa el nuevo. No se necesita reiniciar ni pedir al usuario que cierre kamino.

### 7. Errores y códigos de salida

Se añade `UpdateError(KaminoError)` en `errors.py`. `update.py` envuelve `OSError`, `urllib.error.URLError`, `ssl.SSLError`, JSON inválido y respuestas inesperadas en `UpdateError` con mensajes en español. `cli.py` mantiene el patrón existente: captura `KaminoError`, imprime `str(exc)` en stderr y sale con 1.

`UpdateResult` es un `dataclass(frozen=True)` con `current_version`, `latest_version`, `target` y `updated`. El comando sale con 0 tanto si actualiza como si ya está al día; 1 ante cualquier error.

### 8. Sin prompt y sin `--dry-run`

Ejecutar `kamino update` es la confirmación explícita; no se pide otra. El comando no modifica datos del usuario, solo el propio binario. No se añade `--dry-run` en esta versión: el caso "ya estás al día" ya cubre la consulta y el comando es barato de repetir. Si más adelante se necesita para scripts, se puede añadir sin romper nada.

### 9. Menú

`_run_menu` añade `("update", "Actualización", "Actualiza a la última versión publicada.")`. La opción invoca el mismo comando mediante `ctx.invoke(update)`. En instalaciones de desarrollo el menú sigue mostrando la opción y el comando explica que solo aplica al binario.

### 10. Tema ANSI de la TUI

`KaminoRestoreApp` define el tema mediante un helper evaluado al importar el módulo, para que el valor por defecto sea testeable sin recargar la app:

```python
def _default_theme() -> str:
    return os.environ.get("TEXTUAL_THEME", "ansi-dark")


class KaminoRestoreApp(App[RestoreResult | None]):
    theme = _default_theme()
```

`ansi-dark` hace que fondo y texto usen `ansi_default` (los del terminal), `$surface` sea transparente y los tokens de énfasis (`$primary`, `$warning`, `$error`, etc.) sean colores ANSI. `styles.tcss` no cambia porque ya usa tokens. `widgets.py` ya usa colores ANSI para los diffs.

`TEXTUAL_THEME` conserva el comportamiento de Textual: si se define, se usa ese tema. No se implementa detección de fondo claro/oscuro con OSC 11 ni `COLORFGBG`: ambos temas ANSI comparten fondo y texto del terminal y solo cambian detalles menores (flag `dark`, cursor, scrollbar), así que `ansi-dark` funciona también en terminales claros. Si un usuario necesita `ansi-light`, lo selecciona con la variable de entorno.

### 11. Pruebas sin red

`tests/test_update.py` inyecta o monkeypatchea la función de apertura HTTP y usa `tmp_path`. Casos:

- Parseo y comparación de versiones (`0.5.0`, `v0.5.0`, `0.10.0 > 0.9.0`, formatos inválidos).
- Release con asset, sin asset, sin digest, JSON inválido y error HTTP.
- Actualización correcta: escribe el contenido, modo ejecutable, reemplaza el destino y devuelve `updated=True`.
- Misma versión o versión local superior: no descarga y no escribe.
- Digest o tamaño incorrectos: el destino conserva sus bytes y el temporal se elimina.
- Destino detrás de un symlink: se reemplaza el archivo real y el enlace sigue apuntando a él.
- `installed_binary_path()` sin `sys.frozen`: `UpdateError` con la vía de actualización.
- Plataforma no soportada: `UpdateError`.
- `scripts/install.sh` sigue usando el mismo `REPO` y `ASSET` que `update.py` (guardia contra deriva).

`tests/test_cli.py` cubre el subcomando, los códigos 0/1 y el menú. `tests/test_tui.py` comprueba que la app usa `ansi-dark` y `native_ansi_color=True` en headless. `tests/test_ui.py` cubre el resumen de actualización.

## Riesgos / Compromisos

- **Límite de peticiones de GitHub (60/hora por IP sin token)** → el comando es manual y poco frecuente; ante 403 se muestra un error claro con la página de releases. No se añade token ni caché.
- **El digest y el asset vienen del mismo origen** → protege de corrupción/truncamiento, no de una cuenta comprometida. El binario sigue sin firma; es el mismo modelo de confianza que el instalador.
- **Binario gestionado por Homebrew en el futuro** → `update` reemplazaría el archivo del Cellar. Cuando exista el tap habrá que detectarlo y rechazarlo; hoy no aplica.
- **Destino sin permisos de escritura** → `os.replace` falla; se informa de la ruta y se recomienda reinstalar con el instalador. No se usa `sudo`.
- **`sys.executable.resolve()` en instalaciones personalizadas** → actualiza el archivo real, que puede estar fuera de `~/.local/bin`. El resumen imprime la ruta para que sea transparente.
- **Contraste en terminales claros** → los colores ANSI amarillo/verde pueden tener poco contraste según la paleta. Es el comportamiento pedido; `TEXTUAL_THEME=ansi-light` permite cambiarlo.
- **`certifi` añade tamaño al binario** → el hook de PyInstaller incluye su bundle; se verifica en el build de release.
- **Dos actualizaciones simultáneas** → `os.replace` es atómico y gana la última; no hay datos que corromper.
- **Cuarentena de macOS** → los archivos escritos por la propia app no reciben `com.apple.quarantine`; si una política externa la aplicara, el binario se comportaría como el descargado por el instalador. No se añade `xattr` para no introducir subprocesos.

## Plan de migración

- Cambio aditivo: no hay migración de datos ni de formato de respaldo.
- Orden de implementación: dependencia `certifi` → `UpdateError` → `update.py` → comando y menú → UI → tema de la TUI → pruebas → documentación.
- El workflow de release no cambia; el siguiente `feat:` genera una versión minor y el binario incluye `certifi` automáticamente.
- Rollback: revertir el cambio y volver a publicar. Para una instalación concreta, `update` no degrada; habría que reinstalar manualmente el asset de la release anterior.
- Verificación final: `uv run ruff check .`, `uv run ruff format --check .`, `uv run pytest`; build de PyInstaller y comprobación de que `certifi/cacert.pem` está incluido.
