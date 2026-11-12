# Diseño técnico: CLI de clonación y despliegue de configuraciones (macOS)

Oct 28, 2026 · @Cristiam

Este documento concreta cómo se construye la herramienta descrita en el diseño funcional, con las decisiones ya tomadas: Python, `tar.gz` con `manifest.json`, catálogo en TOML y distribución por `uv`, Homebrew y binario. Propone la estructura de módulos, los formatos de datos, los flujos, las pruebas y el empaquetado. Los puntos que necesitan tu decisión están en la sección 10.

## 1. Principios de diseño

- **Planificar y luego ejecutar.** `clone` y `deploy` primero construyen un plan (una lista de operaciones) y después lo ejecutan. `--dry-run` solo imprime el plan, así la simulación y la ejecución real comparten el mismo código.
- **Catálogo como datos.** Añadir una app es añadir un archivo TOML; el código no conoce ninguna app concreta.
- **Rutas relativas al home.** Todo se guarda relativo a `~`, lo que resuelve la migración entre usuarios (MVP).
- **Nunca destruir sin red.** Antes de sobrescribir se guarda una copia de lo actual, y las escrituras son atómicas.
- **Núcleo sin interfaz.** La lógica no imprime ni pregunta; solo `cli` y `ui` hablan con el usuario. Esto facilita las pruebas.

## 2. Stack y dependencias

| Necesidad | Propuesta | Motivo |
| --- | --- | --- |
| Lenguaje | Python 3.11.4 o superior | `tomllib` y filtros de extracción de `tarfile` en la librería estándar |
| Comandos y opciones | Typer (alternativa: `argparse`) | ayuda automática y tipado; `argparse` si prefieres cero dependencias |
| Interfaz interactiva | `textual` | una sola app con menú, flujos, tablas, progreso y pruebas headless con `run_test` |
| Salida formateada | `rich` | tablas, progreso y colores |
| Lectura del catálogo | `tomllib` (estándar) | TOML sin dependencias |
| Validación del catálogo | `pydantic` (alternativa: dataclasses y validación propia) | errores claros si una definición es inválida; añade peso al binario |
| Archivo del clon | `tarfile`, `hashlib`, `json` (estándar) | sin dependencias |
| Datos de las apps | `plistlib` (estándar) | leer `Info.plist` (bundle ID y versión) |
| Pruebas | `pytest` | HOME falso con `tmp_path` |
| Calidad | `ruff`, `mypy` | lint y tipos |

## 3. Arquitectura

El código se organiza en capas: la interfaz (`cli`, `ui`) llama a los casos de uso (`clone`, `deploy`), que se apoyan en módulos de dominio sin interfaz (catálogo, detección, archivo, rutas, seguridad).

&#91;embedded content: capas del paquete · 3 niveles\]

Cada capa se apoya en la de abajo: `cli` y `ui` llaman a `clone` y `deploy`, que usan módulos de dominio que nunca imprimen ni preguntan.

Estructura propuesta del paquete:

```
src/kamino/
├── cli.py            # argumentos, gating de TUI, salida y códigos
├── commands.py       # flujos de texto compartidos con la CLI
├── texts.py          # única fuente de textos visibles y glosario
├── ui.py             # salida no interactiva con Rich
├── defaults.py       # rutas por defecto compartidas por CLI y TUI
├── config.py         # configuración TOML exclusiva de la TUI
├── clone.py          # plan_clone y write_clone (progreso y cancelación)
├── deploy.py         # sesión, plan, clasificación y aplicación
├── update.py         # consulta, descarga/verificación y reemplazo
├── conflicts.py      # diff por líneas, composición y validación de híbridos
├── progress.py       # contrato de progreso y cancelación
├── tui/
│   ├── app.py        # KaminoApp, SessionState, guardia de tamaño y salida
│   ├── menu.py       # menú principal, cápsulas y última misión
│   ├── common.py     # ayuda, error común, diálogos y terminal pequeña
│   ├── clone.py      # C1-C4
│   ├── deploy.py     # D0-D5
│   ├── update.py     # U1
│   ├── config.py     # pantalla de Configuración
│   ├── widgets.py    # render de bloques, marcadores y tamaños
│   └── styles.tcss   # estilos ANSI de la aplicación
├── plan.py           # Plan, Operation y recuento compartido
├── catalog/
│   ├── models.py     # AppDefinition (validación)
│   ├── loader.py     # catálogo embebido y directorio del usuario
│   └── apps/*.toml   # una definición por app
├── detect.py         # ¿está instalada la app? ¿qué versión?
├── archive.py        # leer y escribir tar.gz, manifiesto, checksums
├── paths.py          # normalización relativa al home y validación
├── snapshot.py       # holocrón de seguridad previo a desplegar
└── errors.py         # excepciones propias
```

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
- `command_aliases`: nombres alternativos del ejecutable, como `cave` para Caveman. Opcional.
- `install_marker`: archivo relativo al home que también debe existir para considerar instalada la app, como `.oh-my-zsh/oh-my-zsh.sh`. Opcional.
- `paths`: rutas obligatorias relativas al home, archivos o carpetas. Su ausencia se informa como omisión. Las rutas fuera del home no se admiten.
- `optional_paths`: rutas que solo se incluyen si existen; su ausencia no es una omisión. Opcional.
- `optional_patterns`: patrones con un directorio fijo y comodines solo en el último componente, como `.codex/*.config.toml`. Opcional.
- `env_paths`: variables de entorno con una ruta absoluta que debe quedar dentro del home, como `HERDR_CONFIG_PATH`. Opcional.
- `xdg_paths`: rutas relativas a `XDG_CONFIG_HOME`, admitidas solo si ese directorio está dentro del home. Opcional.
- `exclude`: patrones glob que se descartan al expandir las rutas.
- Una definición necesita al menos una ruta obligatoria, opcional o un patrón.

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
      "id": "vscode",
      "name": "Visual Studio Code",
      "bundle_id": "com.microsoft.VSCode",
      "version": "1.95.0",
      "files": [
        {"path": "Library/Application Support/Code/User/settings.json", "sha256": "…", "size": 1432, "mode": "0644"}
      ]
    }
  ]
}
```

Los valores son ilustrativos. `format_version` permite evolucionar el formato: `deploy` rechaza un archivo con una versión mayor que la que entiende.

### 4.3 Estructura del archivo

```
clon.tar.gz
├── manifest.json
└── apps/
    └── vscode/
        └── Library/Application Support/Code/User/settings.json
```

Los checksums se calculan al construir el plan, así `manifest.json` puede escribirse primero y `inspect` (v2) lo lee sin recorrer todo el archivo.

## 5. Flujos

### 5.1 Clonación

1. Cargar el catálogo (embebido y directorio del usuario).
2. Detectar las apps instaladas.
3. C1 cuenta archivos con el mismo recorrido y exclusiones que el plan, sin hashear. `--apps` omite C1.
4. C2 resuelve y valida el destino. `--output` omite C2; sin él se propone
   `clone-<aaaammdd>.tar.gz` con la fecha local.
5. `plan_clone` expande `paths`, aplica `exclude` y construye el plan con tamaño y checksum.
   Las rutas ausentes se registran con su motivo. C3 puede cancelar durante la planificación.
6. C3 muestra la revisión y `write_clone` escribe un temporal con progreso por archivos. La
   cancelación comprueba el callback antes de cada archivo y antes del renombrado.
7. Sin `--force` el commit usa `os.link` y falla si el destino existe; con `--force` usa
   `os.replace` y revalida la huella del destino justo antes. Cualquier error o cancelación elimina
   el temporal. C4 resume apps, archivos, tamaño, ruta y omitidos.

### 5.2 Despliegue

1. D0 descubre los `.tar.gz` hijos directos de `clone.output_dir` y del directorio actual,
   deduplica por ruta resuelta, ordena por `mtime` descendente y construye la tabla
   `Archivo`/`Origen`/`Tamaño` con preview de ruta, fecha y tamaño, filtro `/` y la fila
   `Otro archivo...` para la ruta manual. Abre el archivo y valida solo el manifiesto: un único
   `manifest.json` regular, tamaño máximo de 4 MiB, tope de miembros y `format_version`. Envuelve
   errores de `tarfile`/OS. Los avisos de arquitectura o macOS no bloquean.
2. D1 detecta las apps instaladas y cruza el manifiesto. Las no instaladas se informan como “sin
   base en este Mac”. La verificación completa ocurre antes de extraer; la preparación es
   cancelable y limpia el staging si falla.
3. D1 elige qué apps aplicar: ninguna marcada por defecto, todas/ninguna, o `--apps` omite la
   pantalla.
4. Construir el plan: el destino de cada archivo es `Path.home()` más su ruta relativa, clasificado
   como nuevo, sin cambios (clon perfecto, incluida una diferencia solo de CRLF), distinto o no
   soportado. Para cada archivo distinto se construye un conflicto: filas de texto para revisar, o
   archivo entero si es binario, grande o un enlace con enlace.
5. D2 lista solo conflictos y permite `r` revisar, `c` aplicar el clon, `o` conservar el original,
   `C`/`O` sobre todos. Confirmar queda bloqueado hasta cubrir todos. Sin conflictos se pasa a D4.
6. D3 muestra un bloque a la vez: contexto una vez, filas original/clon, marcadores `●`/`○`,
   `c`/`o` por bloque, `C`/`O` por archivo, `Tab`/`Mayús+Tab` entre bloques y validación al volver.
7. D4 revalida los destinos, cuenta protegidos y exige una acción explícita con foco en Retirada.
8. Aplicar en este orden: cobertura de decisiones, revalidación de híbridos, huellas de destinos,
   holocrón si hay destinos existentes y escrituras atómicas con progreso. La cancelación se
   comprueba hasta crear el holocrón; después se ignora. Un fallo posterior lanza
   `PartialApplyError` con aplicados, fallido y holocrón.
9. D5 muestra desplegados, híbridos, omitidos, sin cambios, protegidos y la ruta del holocrón. El
   código 3 se calcula solo con omitidos y `--apps` no cumplidos; un despliegue incompleto usa 1.

El orden de seguridad es `verificar -> extraer -> decidir -> holocrón -> aplicar`. `--yes` y
`--dry-run` conservan la vía de texto y no abren la TUI.

Pantallas y atajos:

```text
Menú -> C1 apps -> C2 destino -> C3 revisión -> C4 resumen
Menú -> D0 clon -> D1 apps -> D2 conflictos -> D3 bloques -> D4 confirmación -> D5 resumen
Menú -> U1 actualización
```

- **C1:** solo apps instaladas con archivos, filtro `/`, `a`/`n` sobre visibles, casillas
  deshabilitadas sin archivos, `Espacio` para marcar sin perder el foco, `Enter` para continuar y
  línea de no detectadas.
- **C2:** nombre por defecto, validación en vivo, `F2`/`F3`/`F4` para directorios y diálogo de
  reemplazo navegable por teclado con Volver preseleccionada.
- **C3:** planificando, revisión y en curso; `d` omitidos con motivo, `f` archivos incluidos, aviso
  de secretos y dry-run sin acción de clonar.
- **C4:** resumen, `d` omitidos, `c` copiar ruta y línea de salida solo con código 3.
- **D0:** tabla `Archivo`/`Origen`/`Tamaño` con los clones de `clone.output_dir` y del directorio
  actual, orden por fecha, deduplicación por ruta resuelta, preview con ruta/fecha/tamaño, filtro
  `/`, ruta manual en `Otro archivo...` (`o`) y validación del manifiesto.
- **D1:** apps instaladas con número de archivos, no instaladas informativas y preparación
  cancelable.
- **D2:** tabla de conflictos con conteos y acciones `r`/`c`/`o`/`C`/`O`.
- **D3:** un bloque a la vez, contexto único, marcadores y acciones por línea/bloque/archivo.
- **D4:** conteos, protegidos, directorio del holocrón, aviso de apps abiertas y foco en Retirada.
- **D5:** categorías, holocrón, `c`, orden de deshacer y variante “Despliegue incompleto”.
- **Ayuda:** `?` en todas las pantallas y `F1` en campos de texto.

### 5.3 Actualización

1. `query_update` comprueba que el proceso es un binario autocontenido, resuelve el ejecutable real
   y valida macOS arm64. Si no procede, lanza `UpdateError` con una categoría estable.
2. Consulta `https://api.github.com/repos/Gunmer/kamino/releases/latest` y lee `tag_name`, el asset
   `kamino-arm64`, su `size` y su `digest` SHA-256.
3. Compara la versión publicada con `__version__` como `X.Y.Z`. Devuelve actualizado, ya al día o
   instalada más reciente sin descargar en los dos últimos casos.
4. `download_update` descarga a un temporal del mismo directorio, emite progreso por chunk y
   comprueba la cancelación. Verifica tamaño y digest antes de devolver la ruta.
5. `replace_binary` marca el temporal como ejecutable y lo reemplaza con `os.replace`. La
   cancelación se comprueba justo antes; una vez iniciado el reemplazo no se interrumpe.
6. Cualquier error o cancelación elimina el temporal y deja intacto el binario anterior.
7. U1 muestra consultando, descargando/verificando, reemplazando y resultado, pide reiniciar tras
   actualizar y usa la ayuda contextual de `texts.UPDATE_ERROR_ADVICE` según la categoría. El código
   es 0 si actualiza o ya está al día; 1 ante error o cancelación.

No se ejecuta el binario descargado antes de reemplazarlo. La actualización no usa `sudo` ni
modifica datos del usuario. La consulta usa `certifi` como bundle de CAs para no depender de las
rutas de certificados del Python de build, y el binario de release incluye `certifi/cacert.pem`.

## 6. Decisiones técnicas relevantes

- **Detección de apps.** Hay dos mecanismos. Para las apps con `.app` se lee el `Info.plist` en `/Applications`, `~/Applications` y `/System/Applications`, buscando por `CFBundleIdentifier`; es más rápido que `mdfind` y no lanza subprocesos, que queda como respaldo. Para las herramientas de terminal se usa `shutil.which` y, si no aparece, las rutas de `search_paths` de la definición.
- **El PATH no es fiable en un Mac nuevo.** Antes de configurar el shell, rutas como `/opt/homebrew/bin` pueden no estar en el PATH, y la herramienta aparecería como no instalada sin estarlo. Por eso cada definición puede declarar rutas adicionales donde buscar el binario.
- **Estado frente a configuración.** Algunas carpetas mezclan ambos: la de herdr guarda también logs y estado de sesión. Las definiciones deben clonar solo los archivos de configuración y excluir el resto (`**/*.log`, `session.json`).
- **Exclusiones.** Con Python 3.11 no existe `Path.full_match()` (llegó en 3.13), así que los patrones como `**/CachedData` se resuelven con `fnmatch` y `glob`, y se prueban con casos límite.
- **Seguridad al extraer.** Nunca usar `extractall` sin control: rechazar rutas absolutas o con `..` y usar `filter="data"` de `tarfile`. Por eso la versión mínima es 3.11.4.
- **Enlaces simbólicos.** Se guardan como enlaces y no se siguen, para no salir del home por accidente.
- **Permisos y atributos.** Se conserva el modo de cada archivo, no el propietario. `tarfile` no conserva atributos extendidos de macOS, algo que en configuraciones rara vez importa.
- **Escrituras atómicas.** Todo se escribe en un temporal y se renombra, tanto el archivo del clon como cada archivo desplegado.
- **Ubicación y nombre del clon.** Por defecto se crea en `~/.config/kamino/vault/` como `clone-<aaaammdd>.tar.gz` con la fecha local y el directorio se crea al escribir. La TUI puede cambiarlo con `clone.output_dir`; la CLI no lee esa configuración. Sin `--force` un destino existente es un error; con `--force` se revalida y se reemplaza de forma atómica. `--output` lo cambia.
- **Holocrón de seguridad.** Se guarda por defecto en `~/.config/kamino/vault/snapshot/<fecha>.tar.gz` con el mismo formato que un clon. La TUI puede cambiarlo con `deploy.snapshots_dir`; la CLI no lee esa configuración. Solo se crea si hay destinos existentes que proteger; el id sintético `snapshot` y el formato no cambian. Los holocrones anteriores en `~/Library/Application Support/kamino/snapshots/` no se migran.
- **Configuración de la TUI.** `src/kamino/config.py` carga y guarda `~/.config/kamino/config.toml` con las claves `catalog.dir`, `clone.output_dir` y `deploy.snapshots_dir`. La CLI no importa este módulo ni lee el archivo. El guardado es atómico, escribe siempre las tres claves y no crea los directorios configurados. Un archivo inválido abre la pantalla de Configuración con los valores legibles y defaults.
- **Catálogo.** Viene embebido en el paquete y se complementa con un directorio del usuario (`~/.config/kamino/apps/`), donde se pueden añadir o sobrescribir definiciones sin tocar el código.
- **Códigos de salida.** 0 éxito, 1 error, 2 uso incorrecto, 3 éxito con elementos omitidos.
- **Límite del HOME.** `SafeHome` abre el home una vez y recorre sus descendientes con `dir_fd` y `O_NOFOLLOW`. La validación léxica de `to_absolute`/`to_relative` sigue siendo la primera barrera; el acceso con descriptores impide que un enlace intermedio redirija una lectura o escritura fuera del home. Si la plataforma no ofrece estas primitivas, se aborta en lugar de degradar a `resolve()` seguido de una apertura insegura.
- **Enlaces finales.** Se conserva la regla existente: un enlace final se guarda como enlace, nunca se sigue y solo se admite si su destino es relativo y no contiene `..`. `SafeHome.read_bytes` rechaza el componente final si es un enlace; `lstat` y `readlink` permiten representarlo en el manifiesto.
- **Cambios concurrentes.** La planificación calcula el checksum con el mismo acceso seguro que usa el empaquetado, que reabre cada ruta y compara el contenido con el manifiesto. Si una ruta cambia a un enlace o a otro tipo entre el plan y la escritura, la clonación falla en vez de incluir datos externos. En el despliegue se valida cada destino antes del holocrón y se vuelve a abrir cada padre durante la aplicación; si la jerarquía cambia, se detiene sin escribir fuera del home. Además, la sesión guarda el hash del archivo regular, el destino del enlace o la ausencia del destino al planificar y los re-verifica justo antes del holocrón: si alguno cambió durante la revisión, aborta todo el despliegue sin escribir.
- **Conflictos e híbridos.** `conflicts.py` contiene el modelo y las funciones puras. La comparación decodifica UTF-8 y normaliza CRLF solo para comparar; un clon perfecto no se escribe. Un contenido con NUL o que no decodifica es binario; un texto de más de 5 MB o de más de 20 000 líneas se trata como grande para proteger `difflib` y la TUI. Los bloques se generan con `difflib.SequenceMatcher` y 3 líneas de contexto; dentro de cada bloque, las líneas iguales son contexto y cada par original/clon o línea de un solo lado es una fila decidible que empieza en el clon. `build_hunks` y `compose` comparten la misma generación de filas. La composición conserva las líneas iguales del original, adapta las del clon al estilo de final de línea original y, si una línea sin salto final queda en medio, le añade el EOL dominante; solo la última puede conservar la ausencia de salto final. El núcleo compone el archivo; la TUI nunca inyecta bytes arbitrarios y D3 renderiza un solo bloque con un `Static`, no miles de widgets.
- **Validación de híbridos.** Se valida por extensión con la biblioteca estándar: TOML (`tomllib`), JSON (`json` con detección de claves duplicadas), INI/CFG/Git config (`configparser` estricto) y plist XML (`plistlib`, con detección de claves duplicadas en el XML). Solo se bloquea si ambos originales parsean y el híbrido no; si alguno ya era inválido se avisa y se permite aplicar. YAML y shell no se validan. `apply` repite la validación como defensa del núcleo.
- **TUI única.** `KaminoApp` arranca en modo `menu` o `direct`; una pila de `Screen` conserva el estado inferior y `SessionState` guarda frase, última misión, último código y modo. Los flujos comparten `HelpScreen`, `ErrorScreen` y la guardia de tamaño. `Ctrl+C` se reparte por pantalla y respeta los puntos de no retorno. La app se abre solo con entrada y salida TTY y sin `KAMINO_PLAIN`; `deploy --yes` y `--dry-run` usan texto. La TUI muestra avisos de secretos en C3, D2 y D3 y no persiste su contenido.
- **Progreso y cancelación.** `progress.py` define `ProgressEvent`, `ProgressCallback` y `CancelCallback`. El núcleo emite fases y consulta `cancelled()` en bucles largos; la cancelación lanza `OperationCancelled`. Los workers de Textual son de hilo y actualizan con `call_from_thread`; `Worker.cancel()` no mata un hilo, por eso todos los bucles comprueban el callback.
- **Empaquetado de la TUI.** El build de PyInstaller incluye los datos de `textual` (CSS y recursos) con `--collect-data textual`, además del catálogo embebido.
- **Referencias de Ghostty.** El descubrimiento es específico de `ghostty` y se limita a `config-file` y a la selección de temas. Resuelve cada candidato con el acceso seguro, aplica las exclusiones antes de leer, mantiene límites de tamaño, profundidad y cantidad, y deduplica por `(app_id, ruta relativa)`.
- **Exclusiones y aviso.** Los clones nuevos excluyen credenciales independientes conocidas, historiales, sesiones, cachés y registros. El aviso de que el archivo no está cifrado se muestra siempre antes de clonar y no modifica los bytes de los ajustes. Las exclusiones no se aplican al despliegue de clones antiguos ni a la instantánea previa.
- **Actualización del binario.** `update.py` es un caso de uso independiente de la CLI. Solo actúa sobre binarios autocontenidos (`sys.frozen`), resuelve el ejecutable real y consulta la API de GitHub. La versión se compara como `X.Y.Z`; no se degrada ni se descarga si la instalada es igual o superior. La integridad se basa en el tamaño y el digest SHA-256 del asset. El reemplazo usa un temporal en el mismo directorio y `os.replace`, sin `sudo` y sin ejecutar el binario descargado. La consulta usa `urllib` y `certifi` para no introducir subprocesos en `src/`. No hay comprobación automática ni rollback.
- **Tema ANSI de la TUI.** `KaminoApp` usa `ansi-dark` por defecto para que fondo, texto, bordes, wordmark, cápsulas y estados salgan de la paleta ANSI del terminal. `TEXTUAL_THEME` conserva el override de Textual, por ejemplo `ansi-light`. `NO_COLOR` se lee antes de crear la app y todos los estados conservan icono y palabra. `◉`, `●`, `○` y `↪` tienen equivalentes ASCII cuando la codificación no es UTF-8 o se define `KAMINO_NO_LOGO`.

## 7. Pruebas

- **Unitarias:** carga del catálogo (definiciones válidas e inválidas), expansión de rutas y exclusiones, normalización de rutas y manifiesto.
- **Conflictos:** `tests/test_conflicts.py` cubre detección de texto, binario, grande y enlace; archivos nuevos, idénticos, solo modo, CRLF y cambios de tipo; generación de filas con contexto, emparejamiento e inserciones/eliminaciones; composición con híbridos dentro de un bloque, todo clon, todo original, CRLF, salvaguarda de salto final y longitud de decisiones; y validación de TOML, JSON, INI, plist, original inválido y formatos sin validar.
- **Integración:** ciclo completo clonación y despliegue entre dos HOME falsos distintos, con apps "instaladas" simuladas mediante un `Info.plist` falso, comprobando que solo se despliegan las instaladas. Las pruebas de conflicto cubren híbridos, holocrón, undo, `--yes` y `--dry-run` con diffs.
- **TUI:** `tests/test_tui.py` usa `App.run_test()` (sin TTY) para menú, degradación por altura, guardia de terminal pequeña, navegación por pila, C1-C4, D0-D5, U1, ayuda, error de cuatro partes, copia, cancelación y despliegue parcial. Comprueba 80×24, 79×23 y el redimensionado de vuelta, y que D3 maneja 20 000 líneas sin montar miles de widgets.
- **Actualización:** `tests/test_update.py` cubre el parseo y la comparación de versiones, la consulta de la release, el asset ausente, el digest ausente o incorrecto, la descarga y el reemplazo atómico, los symlinks, la plataforma no soportada, una instalación no frozen y la paridad de `REPOSITORY`/`ASSET_NAME` con `scripts/install.sh`. Ninguna prueba accede a la red ni al HOME real.
- **Seguridad:** archivos tar con rutas `..`, rutas absolutas y enlaces simbólicos maliciosos, y un checksum alterado. `tests/test_paths.py` cubre el rechazo de enlaces intermedios y la conservación de enlaces finales; `tests/test_deploy.py` cubre el rechazo antes de la instantánea, la sustitución de un directorio durante la aplicación y un destino cambiado entre plan y apply.
- **Aislamiento:** una fixture que reemplaza `HOME` y hace fallar la prueba si alguna ruta resuelve fuera de `tmp_path`, para no tocar nunca el home real.
- **CI:** GitHub Actions con runner de macOS para ejecutar `pytest`.

## 8. Empaquetado y distribución

| Vía | Cómo | Cuándo |
| --- | --- | --- |
| `uv tool install git+https://github.com/<usuario>/<repo>` | `pyproject.toml` con un entry point de consola | v1 |
| Homebrew tap | Fórmula que instala el paquete en un virtualenv; se actualiza al publicar una versión etiquetada | tras la primera versión etiquetada |
| Binario autocontenido | PyInstaller `onefile`, uno por arquitectura (arm64 e Intel), publicado en GitHub Releases | en cada merge a `main` |

El binario sin firmar recibe la cuarentena de macOS: hay que documentar cómo quitarla (`xattr -d com.apple.quarantine`) o firmarlo y notarizarlo con una cuenta de desarrollador.

El binario incluye el bundle de CAs de `certifi`, necesario para que `kamino update` verifique TLS sin depender de las rutas de certificados del Python de build.

El versionado y la publicación se automatizan con `python-semantic-release`: los títulos de los PRs siguen Conventional Commits (el CI los valida) y, al hacer merge a `main`, se calcula el bump semver, se actualiza `CHANGELOG.md`, se crea el tag `vX.Y.Z` y se publica la GitHub Release con los binarios adjuntos (workflow `.github/workflows/release.yml`). El tap depende de tener versiones etiquetadas, por eso `uv` va primero.

El paquete se publica como `kamino-cli`, porque el nombre `kamino` ya está ocupado en PyPI, y expone el comando `kamino`. Por eso la instalación con `uv` usa la URL de GitHub y no el nombre del paquete.

## 9. Plan de implementación (v1)

1. Esqueleto del proyecto: `pyproject.toml`, estructura de paquetes, `ruff`, `pytest` y CI.
2. Modelos y carga del catálogo, con una app de ejemplo.
3. Detección de apps (`Info.plist` y `command`).
4. `archive.py`: escritura y lectura del `tar.gz`, manifiesto y checksums.
5. `plan.py` y `clone.py`, con `--dry-run`.
6. `deploy.py`: staging, verificación, holocrón y aplicación.
7. `cli.py` y `ui.py`: selección interactiva y resumen.
8. Resto del catálogo inicial (Git, Ghostty, herdr y yazi) y pruebas de integración.
9. Empaquetado con `uv`; después tap y binario.

## 10. Decisiones tomadas

- **Nombre:** `kamino` para el comando y el repositorio; el paquete de PyPI se llama `kamino-cli`.
- **Comandos del CLI:** Typer.
- **Versión mínima de Python:** 3.11.4.
- **Catálogo inicial:** Git, Ghostty, herdr y yazi.
- **Ubicación de los clones:** `~/.config/kamino/vault/` por defecto, con el nombre `clone-<aaaammdd>.tar.gz` y reemplazo explícito mediante `--force`. La TUI puede cambiarla con `clone.output_dir`; la CLI mantiene el default.
- **Pendiente al escribir las definiciones:** confirmar las rutas exactas de Ghostty. En yazi se clonan la configuración, `init.lua`, `package.toml` y la carpeta `plugins`.
