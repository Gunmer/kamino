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
├── cli.py            # comandos backup y restore
├── ui.py             # selección interactiva, tablas, progreso
├── backup.py         # caso de uso: construir plan y ejecutar respaldo
├── restore.py        # caso de uso: construir plan y ejecutar restauración
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
- `paths`: rutas relativas al home, archivos o carpetas. Las rutas fuera del home no se admiten en la v1.
- `exclude`: patrones glob que se descartan al expandir `paths`.

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
2. Detectar las apps instaladas y cruzarlas con las del manifiesto; las que no están instaladas se listan como omitidas.
3. El usuario confirma qué apps aplicar.
4. Construir el plan: el destino de cada archivo es `Path.home()` más su ruta relativa, clasificado como nuevo, idéntico (se omite) o distinto (se sobrescribe).
5. Con `--dry-run`: imprimir el plan y terminar.
6. Extraer a un directorio temporal y comparar cada archivo con su checksum; si algo no coincide, abortar antes de tocar nada.
7. Crear la copia previa (snapshot) de los destinos que van a cambiar.
8. Aplicar con reemplazo atómico (`os.replace`), conservando permisos.
9. Mostrar el resumen: restauradas, omitidas y ruta del snapshot.

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

## 7. Pruebas

- **Unitarias:** carga del catálogo (definiciones válidas e inválidas), expansión de rutas y exclusiones, normalización de rutas y manifiesto.
- **Integración:** ciclo completo respaldo y restauración entre dos HOME falsos distintos, con apps "instaladas" simuladas mediante un `Info.plist` falso, comprobando que solo se restauran las instaladas.
- **Seguridad:** archivos tar con rutas `..`, rutas absolutas y enlaces simbólicos maliciosos, y un checksum alterado.
- **Aislamiento:** una fixture que reemplaza `HOME` y hace fallar la prueba si alguna ruta resuelve fuera de `tmp_path`, para no tocar nunca el home real.
- **CI:** GitHub Actions con runner de macOS para ejecutar `pytest`.

## 8. Empaquetado y distribución

| Vía | Cómo | Cuándo |
| --- | --- | --- |
| `uv tool install git+https://github.com/<usuario>/<repo>` | `pyproject.toml` con un entry point de consola | v1 |
| Homebrew tap | Fórmula que instala el paquete en un virtualenv; se actualiza al publicar una versión etiquetada | tras la primera versión etiquetada |
| Binario autocontenido | PyInstaller `onefile`, uno por arquitectura (arm64 e Intel), publicado en GitHub Releases | en cada merge a `main` |

El binario sin firmar recibe la cuarentena de macOS: hay que documentar cómo quitarla (`xattr -d com.apple.quarantine`) o firmarlo y notarizarlo con una cuenta de desarrollador.

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
