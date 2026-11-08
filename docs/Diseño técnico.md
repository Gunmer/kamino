# Diseño técnico: CLI de respaldo y restauración de configuraciones (macOS)

Oct 28, 2026 · @Cristiam

Este documento concreta cómo se construye la herramienta descrita en el diseño funcional, con las decisiones ya tomadas: Python, `tar.gz` con `manifest.json`, catálogo en TOML y distribución por `uv`, Homebrew y binario. Propone la estructura de módulos, los formatos de datos, los flujos, las pruebas y el empaquetado. Los puntos que necesitan tu decisión están en la sección 10.

## 1. Principios de diseño

- **Planificar y luego ejecutar.** `backup` y `restore` primero construyen un plan (una lista de operaciones) y después lo ejecutan. `--dry-run` solo imprime el plan, así la simulación y la ejecución real comparten el mismo código.
- **Catálogo como datos.** Añadir una app es añadir un archivo TOML; el código no conoce ninguna app concreta.
- **Rutas relativas al home.** Todo se guarda relativo a `~`, lo que resuelve la migración entre usuarios (MVP).
- **Nunca destruir sin red.** Antes de sobrescribir se guarda una copia de lo actual, y las escrituras son atómicas.
- **Núcleo sin interfaz.** La lógica no imprime ni pregunta; solo `cli` y `ui` hablan con el usuario. Esto facilita las pruebas.

## 2. Stack y dependencias

| Necesidad | Propuesta | Motivo |
| --- | --- | --- |
| Lenguaje | Python 3.11.4 o superior | `tomllib` y filtros de extracción de `tarfile` en la librería estándar |
| Comandos y opciones | Typer (alternativa: `argparse`) | ayuda automática y tipado; `argparse` si prefieres cero dependencias |
| Selección interactiva | `questionary` (alternativa: `InquirerPy`) | lista de selección múltiple en terminal |
| Revisión interactiva de conflictos | `textual` | tabla navegable, detalle con scroll y pruebas headless con `run_test` |
| Salida formateada | `rich` | tablas, progreso y colores |
| Lectura del catálogo | `tomllib` (estándar) | TOML sin dependencias |
| Validación del catálogo | `pydantic` (alternativa: dataclasses y validación propia) | errores claros si una definición es inválida; añade peso al binario |
| Archivo de respaldo | `tarfile`, `hashlib`, `json` (estándar) | sin dependencias |
| Datos de las apps | `plistlib` (estándar) | leer `Info.plist` (bundle ID y versión) |
| Pruebas | `pytest` | HOME falso con `tmp_path` |
| Calidad | `ruff`, `mypy` | lint y tipos |

## 3. Arquitectura

El código se organiza en capas: la interfaz (`cli`, `ui`) llama a los casos de uso (`backup`, `restore`), que se apoyan en módulos de dominio sin interfaz (catálogo, detección, archivo, rutas, seguridad).

&#91;embedded content: capas del paquete · 3 niveles\]

Cada capa se apoya en la de abajo: `cli` y `ui` llaman a `backup` y `restore`, que usan módulos de dominio que nunca imprimen ni preguntan.

Estructura propuesta del paquete:

```
src/kamino/
├── cli.py            # comandos backup y restore, y menú TUI sin argumentos
├── ui.py             # selección interactiva, menú de comandos, tablas, progreso
├── backup.py         # caso de uso: construir plan y ejecutar respaldo
├── restore.py        # caso de uso: sesión, plan y ejecución de restauración
├── conflicts.py      # diff por líneas, composición y validación de mezclas
├── tui/
│   ├── app.py        # KaminoRestoreApp, estado y entrada run_restore_tui
│   ├── screens.py    # apps, conflictos, detalle, confirmación, resumen y ayuda
│   ├── widgets.py    # tabla, visor de líneas y avisos
│   └── styles.tcss   # estilos de la aplicación
├── plan.py           # Plan y Operation, compartidos por ambos
├── catalog/
│   ├── models.py     # AppDefinition (validación)
│   ├── loader.py     # catálogo embebido y directorio del usuario
│   └── apps/*.toml   # una definición por app
├── detect.py         # ¿está instalada la app? ¿qué versión?
├── archive.py        # leer y escribir tar.gz, manifiesto, checksums
├── paths.py          # normalización relativa al home y validación
├── snapshot.py       # copia previa a restaurar
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

Los valores son ilustrativos. `format_version` permite evolucionar el formato: `restore` rechaza un archivo con una versión mayor que la que entiende.

### 4.3 Estructura del archivo

```
respaldo.tar.gz
├── manifest.json
└── apps/
    └── vscode/
        └── Library/Application Support/Code/User/settings.json
```

Los checksums se calculan al construir el plan, así `manifest.json` puede escribirse primero y `inspect` (v2) lo lee sin recorrer todo el archivo.

## 5. Flujos

### 5.1 Respaldo

1. Cargar el catálogo (embebido y directorio del usuario).
2. Detectar las apps instaladas.
3. Mostrar solo las instaladas y recoger la selección, o usar `--apps` en modo no interactivo.
4. Expandir `paths` de cada app, aplicar `exclude` y construir el plan: archivos con tamaño y checksum. Las rutas que no existen se registran como omitidas, no como error.
5. Con `--dry-run`: imprimir el plan y terminar.
6. Escribir el archivo en un temporal dentro del mismo directorio y renombrarlo al nombre final al terminar.
7. Mostrar el resumen: apps respaldadas, archivos, tamaño y omitidos.

### 5.2 Restauración

1. Abrir el archivo, leer `manifest.json` y validar `format_version`.
2. Detectar las apps instaladas y cruzarlas con las del manifiesto; las que no están instaladas se
   listan como omitidas.
3. Elegir qué apps aplicar: la TUI muestra todas las restaurables sin marcar ninguna por defecto, o
   `--apps` proporciona la selección y omite esa pantalla.
4. Construir el plan: el destino de cada archivo es `Path.home()` más su ruta relativa, clasificado
   como nuevo, idéntico (se omite, incluida una diferencia solo de CRLF), distinto o no soportado.
   Para cada archivo distinto se construye un conflicto: filas de texto para revisar, o archivo
   entero si es binario, grande o un enlace con enlace.
5. Verificar el `tar.gz` completo antes de extraerlo, extraerlo a un directorio temporal y leer el
   contenido de cada conflicto. Con `--dry-run` no se verifica ni se extrae: se lee el contenido
   directamente del archivo y se imprimen los diffs; no se abre la TUI, ni se crea snapshot, ni se
   escribe nada.
6. Revisar los conflictos en la TUI: tabla con solo los conflictos y los conteos, detalle por
   líneas y validación de la mezcla al volver. Ningún conflicto puede quedar sin decisión. La
   confirmación final exige una acción explícita; volver o cancelar no escribe ni crea snapshot.
7. Aplicar en este orden: exigir cobertura de decisiones, revalidar las mezclas, re-verificar los
   destinos que se van a escribir, crear el snapshot de esos destinos y escribir con reemplazo
   atómico. Si un destino cambió desde el plan, abortar toda la operación antes del snapshot.
8. Mostrar el resumen: restaurados, mezclados, omitidos y ruta del snapshot. Un archivo mezclado es
   éxito; el código 3 se reserva a los elementos omitidos.

El orden de seguridad es `verificar -> extraer -> decidir -> snapshot -> aplicar`. `--yes` resuelve
todos los conflictos aplicando el respaldo y conserva la vía no interactiva.

Pantallas y atajos de la TUI:

```text
Selección de apps -> Tabla de conflictos -> Detalle por líneas (por archivo)
                  -> Confirmación -> Resumen
```

- **Selección de apps:** casillas sin marcar por defecto, botones para seleccionar todas o ninguna,
  continuar y cancelar. Se omite con `--apps`.
- **Tabla de conflictos:** solo conflictos, conteos de nuevos/idénticos/no soportados, acciones por
  fila (revisar, aplicar, omitir), acciones globales (aplicar todo, omitir todo), confirmar
  deshabilitado hasta cubrir todos los conflictos y cancelar.
- **Detalle:** `Espacio` o `Enter` alterna la línea resaltada; `A`/`L` toman todo el archivo del
  respaldo o del local y `Shift+A`/`Shift+L` solo el bloque actual; `n`/`p` saltan entre bloques y
  `Escape` vuelve y valida la mezcla. Las cabeceras y el contexto no son decidibles. Si la mezcla es
  inválida, se muestra el error y se permanece en el detalle; si algún original era inválido, se
  avisa y se permite aplicar.
- **Confirmación:** conteos de restaurados, mezclados y omitidos, directorio del snapshot y botones
  aplicar, volver y cancelar.
- **Resumen:** conteos finales y ruta del snapshot; salir.
- **Ayuda:** `?` abre la ayuda en español con los atajos, el flujo y el aviso de secretos.

### 5.3 Actualización

1. `kamino update` comprueba que el proceso es un binario autocontenido. Si no lo es, indica la vía
   de actualización de desarrollo (`uv sync`, `uv tool upgrade kamino-cli` o la de `pip`).
2. Resuelve el archivo real del ejecutable (`sys.executable` con symlinks resueltos) y comprueba
   que el sistema es macOS arm64.
3. Consulta `https://api.github.com/repos/Gunmer/kamino/releases/latest` y lee `tag_name`, el asset
   `kamino-arm64`, su `size` y su `digest` SHA-256.
4. Compara la versión publicada con `__version__` como `X.Y.Z`. Si la instalada es igual o superior,
   termina sin descargar ni escribir.
5. Descarga el asset a un temporal en el mismo directorio que el binario, calculando el SHA-256 y el
   tamaño mientras escribe.
6. Verifica tamaño y digest, marca el temporal como ejecutable y lo reemplaza con `os.replace`. Si
   algo falla, elimina el temporal y deja intacto el binario anterior.
7. Informa de la versión anterior, la nueva y la ruta actualizada. El código es 0 si actualiza o ya
   está al día; 1 ante cualquier error.

No se ejecuta el binario descargado antes de reemplazarlo. La actualización no usa `sudo` ni
modifica datos del usuario. La consulta usa `certifi` como bundle de CAs para no depender de las
rutas de certificados del Python de build, y el binario de release incluye `certifi/cacert.pem`.

## 6. Decisiones técnicas relevantes

- **Detección de apps.** Hay dos mecanismos. Para las apps con `.app` se lee el `Info.plist` en `/Applications`, `~/Applications` y `/System/Applications`, buscando por `CFBundleIdentifier`; es más rápido que `mdfind` y no lanza subprocesos, que queda como respaldo. Para las herramientas de terminal se usa `shutil.which` y, si no aparece, las rutas de `search_paths` de la definición.
- **El PATH no es fiable en un Mac nuevo.** Antes de configurar el shell, rutas como `/opt/homebrew/bin` pueden no estar en el PATH, y la herramienta aparecería como no instalada sin estarlo. Por eso cada definición puede declarar rutas adicionales donde buscar el binario.
- **Estado frente a configuración.** Algunas carpetas mezclan ambos: la de herdr guarda también logs y estado de sesión. Las definiciones deben respaldar solo los archivos de configuración y excluir el resto (`**/*.log`, `session.json`).
- **Exclusiones.** Con Python 3.11 no existe `Path.full_match()` (llegó en 3.13), así que los patrones como `**/CachedData` se resuelven con `fnmatch` y `glob`, y se prueban con casos límite.
- **Seguridad al extraer.** Nunca usar `extractall` sin control: rechazar rutas absolutas o con `..` y usar `filter="data"` de `tarfile`. Por eso la versión mínima es 3.11.4.
- **Enlaces simbólicos.** Se guardan como enlaces y no se siguen, para no salir del home por accidente.
- **Permisos y atributos.** Se conserva el modo de cada archivo, no el propietario. `tarfile` no conserva atributos extendidos de macOS, algo que en configuraciones rara vez importa.
- **Escrituras atómicas.** Todo se escribe en un temporal y se renombra, tanto el archivo de respaldo como cada archivo restaurado.
- **Ubicación y nombre del respaldo.** Por defecto se crea en el directorio actual con el nombre `kamino-<equipo>-<fecha>.tar.gz`; si ya existe, se añade un sufijo y nunca se sobrescribe. `--output` lo cambia.
- **Snapshots.** Se guardan en `~/Library/Application Support/kamino/snapshots/<fecha>.tar.gz` con el mismo formato que un respaldo, así el futuro `undo` es simplemente un `restore` del snapshot.
- **Catálogo.** Viene embebido en el paquete y se complementa con un directorio del usuario (`~/.config/kamino/apps/`), donde se pueden añadir o sobrescribir definiciones sin tocar el código.
- **Códigos de salida.** 0 éxito, 1 error, 2 uso incorrecto, 3 éxito con elementos omitidos.
- **Límite del HOME.** `SafeHome` abre el home una vez y recorre sus descendientes con `dir_fd` y `O_NOFOLLOW`. La validación léxica de `to_absolute`/`to_relative` sigue siendo la primera barrera; el acceso con descriptores impide que un enlace intermedio redirija una lectura o escritura fuera del home. Si la plataforma no ofrece estas primitivas, se aborta en lugar de degradar a `resolve()` seguido de una apertura insegura.
- **Enlaces finales.** Se conserva la regla existente: un enlace final se guarda como enlace, nunca se sigue y solo se admite si su destino es relativo y no contiene `..`. `SafeHome.read_bytes` rechaza el componente final si es un enlace; `lstat` y `readlink` permiten representarlo en el manifiesto.
- **Cambios concurrentes.** La planificación calcula el checksum con el mismo acceso seguro que usa el empaquetado, que reabre cada ruta y compara el contenido con el manifiesto. Si una ruta cambia a un enlace o a otro tipo entre el plan y la escritura, el respaldo falla en vez de incluir datos externos. En restauración se valida cada destino antes de la instantánea y se vuelve a abrir cada padre durante la aplicación; si la jerarquía cambia, se detiene sin escribir fuera del home. Además, la sesión guarda el hash del archivo regular, el destino del enlace o la ausencia del destino al planificar y los re-verifica justo antes del snapshot: si alguno cambió durante la revisión, aborta toda la restauración sin escribir.
- **Conflictos y mezcla.** `conflicts.py` contiene el modelo y las funciones puras. La comparación decodifica UTF-8 y normaliza CRLF solo para comparar; un archivo que solo difiere en CRLF es idéntico. Un contenido con NUL o que no decodifica es binario; un texto de más de 5 MB o de más de 20 000 líneas se trata como grande para proteger `difflib` y la TUI. Los bloques se generan con `difflib.SequenceMatcher` y 3 líneas de contexto; dentro de cada bloque, las líneas iguales son contexto y cada par local/respaldo o línea de un solo lado es una fila decidible que empieza en el respaldo. `build_hunks` y `compose` comparten la misma generación de filas, así que las decisiones se alinean por posición. La composición conserva las líneas iguales del local, adapta las líneas del respaldo al estilo de final de línea local y, si una línea sin salto final queda en medio, le añade el EOL dominante local; solo la última línea puede conservar la ausencia de salto final. El núcleo compone el archivo; la TUI nunca inyecta bytes arbitrarios.
- **Validación de mezclas.** Se valida por extensión con la biblioteca estándar: TOML (`tomllib`), JSON (`json` con detección de claves duplicadas), INI/CFG/Git config (`configparser` estricto) y plist XML (`plistlib`, con detección de claves duplicadas en el XML). Solo se bloquea si ambos originales parsean y la mezcla no; si alguno ya era inválido se avisa y se permite aplicar. YAML y shell no se validan. `apply` repite la validación como defensa del núcleo.
- **TUI de restauración.** `textual` aporta la tabla navegable, el detalle con scroll y las pruebas headless con `run_test`. `restore` abre la TUI solo en una terminal interactiva y sin `--yes`; `--apps` omite la pantalla de selección y `--dry-run` imprime los diffs en stdout sin abrirla. La TUI muestra un aviso de que los diffs pueden contener secretos y no persiste ni registra su contenido.
- **Empaquetado de la TUI.** El build de PyInstaller incluye los datos de `textual` (CSS y recursos) con `--collect-data textual`, además del catálogo embebido.
- **Referencias de Ghostty.** El descubrimiento es específico de `ghostty` y se limita a `config-file` y a la selección de temas. Resuelve cada candidato con el acceso seguro, aplica las exclusiones antes de leer, mantiene límites de tamaño, profundidad y cantidad, y deduplica por `(app_id, ruta relativa)`.
- **Exclusiones y aviso.** Los respaldos nuevos excluyen credenciales independientes conocidas, historiales, sesiones, cachés y registros. El aviso de que el archivo no está cifrado se muestra siempre antes de respaldar y no modifica los bytes de los ajustes. Las exclusiones no se aplican a la restauración de respaldos antiguos ni a la instantánea previa.
- **Actualización del binario.** `update.py` es un caso de uso independiente de la CLI. Solo actúa sobre binarios autocontenidos (`sys.frozen`), resuelve el ejecutable real y consulta la API de GitHub. La versión se compara como `X.Y.Z`; no se degrada ni se descarga si la instalada es igual o superior. La integridad se basa en el tamaño y el digest SHA-256 del asset. El reemplazo usa un temporal en el mismo directorio y `os.replace`, sin `sudo` y sin ejecutar el binario descargado. La consulta usa `urllib` y `certifi` para no introducir subprocesos en `src/`. No hay comprobación automática ni rollback.
- **Tema ANSI de la TUI.** `KaminoRestoreApp` usa `ansi-dark` por defecto para que fondo, texto, bordes y botones salgan de la paleta ANSI del terminal. `TEXTUAL_THEME` conserva el override de Textual, por ejemplo `ansi-light`. `styles.tcss` ya usa tokens semánticos y no necesita cambios; los diffs de `widgets.py` ya usan colores ANSI.

## 7. Pruebas

- **Unitarias:** carga del catálogo (definiciones válidas e inválidas), expansión de rutas y exclusiones, normalización de rutas y manifiesto.
- **Conflictos:** `tests/test_conflicts.py` cubre detección de texto, binario, grande y enlace; archivos nuevos, idénticos, solo modo, CRLF y cambios de tipo; generación de filas con contexto, emparejamiento e inserciones/eliminaciones; composición con mezcla dentro de un bloque, todos-respaldo, todos-local, CRLF, salvaguarda de salto final y longitud de decisiones; y validación de TOML, JSON, INI, plist, original inválido y formatos sin validar.
- **Integración:** ciclo completo respaldo y restauración entre dos HOME falsos distintos, con apps "instaladas" simuladas mediante un `Info.plist` falso, comprobando que solo se restauran las instaladas. Las pruebas de conflicto cubren mezcla, snapshot, undo, `--yes` y `--dry-run` con diffs.
- **TUI:** `tests/test_tui.py` usa `App.run_test()` (sin TTY) para abrir y cerrar la app, recorrer la selección de apps, la tabla, el detalle por líneas (cursor que salta cabeceras y contexto, toggle, acciones globales y por bloque, saltos y reapertura), la validación bloqueada, la confirmación, el resumen, la ayuda y el aborto por cambio concurrente. También comprueba que el tema por defecto es ANSI (`ansi-dark`) y que `TEXTUAL_THEME` puede sobrescribirlo.
- **Actualización:** `tests/test_update.py` cubre el parseo y la comparación de versiones, la consulta de la release, el asset ausente, el digest ausente o incorrecto, la descarga y el reemplazo atómico, los symlinks, la plataforma no soportada, una instalación no frozen y la paridad de `REPOSITORY`/`ASSET_NAME` con `scripts/install.sh`. Ninguna prueba accede a la red ni al HOME real.
- **Seguridad:** archivos tar con rutas `..`, rutas absolutas y enlaces simbólicos maliciosos, y un checksum alterado. `tests/test_paths.py` cubre el rechazo de enlaces intermedios y la conservación de enlaces finales; `tests/test_restore.py` cubre el rechazo antes de la instantánea, la sustitución de un directorio durante la aplicación y un destino cambiado entre plan y apply.
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
5. `plan.py` y `backup.py`, con `--dry-run`.
6. `restore.py`: staging, verificación, snapshot y aplicación.
7. `cli.py` y `ui.py`: selección interactiva y resumen.
8. Resto del catálogo inicial (Git, Ghostty, herdr y yazi) y pruebas de integración.
9. Empaquetado con `uv`; después tap y binario.

## 10. Decisiones tomadas

- **Nombre:** `kamino` para el comando y el repositorio; el paquete de PyPI se llama `kamino-cli`.
- **Comandos del CLI:** Typer.
- **Versión mínima de Python:** 3.11.4.
- **Catálogo inicial:** Git, Ghostty, herdr y yazi.
- **Ubicación de los respaldos:** directorio actual, con el nombre `kamino-<equipo>-<fecha>.tar.gz` y sin sobrescribir.
- **Pendiente al escribir las definiciones:** confirmar las rutas exactas de Ghostty y yazi, y si los plugins de yazi se respaldan o se reinstalan.
