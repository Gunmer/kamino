# Diseño: CLI de respaldo y restauración de configuraciones (macOS)

## 1. Objetivo

Una herramienta de línea de comandos para macOS que guarda las configuraciones de las aplicaciones elegidas en **un único archivo portable**, y que permite restaurarlas después en el mismo Mac o en otro, aplicando solo lo que corresponda a las apps que estén instaladas.

**Fuera de alcance:** otros sistemas operativos, respaldo de documentos personales o de los binarios de las apps.

## 2. Funcionalidades principales (requeridas)

### 2.0 Menú interactivo (TUI)

- Al ejecutar `kamino` sin comandos, en una terminal interactiva, se muestra una lista con los comandos disponibles (`backup`, `restore` y `update`) para elegir uno.
- Sin terminal, se muestra la ayuda como antes.

### 2.1 Respaldo (`backup`)

- Detecta las aplicaciones soportadas que están instaladas en el Mac.
- Muestra una lista interactiva para **escoger qué apps respaldar** (selección múltiple, con opción de seleccionar todo o nada).
- Recolecta los archivos de configuración de cada app seleccionada.
- Genera un **único archivo de respaldo** (por ejemplo `.tar.gz` o un formato propio) con:
  - los archivos de configuración,
  - un manifiesto con metadatos (apps incluidas, versión de cada app, fecha, versión de macOS, versión de la herramienta, nombre del equipo).
- Permite modo no interactivo: `backup --apps vscode,iterm2,git`.

### 2.2 Restauración (`restore`)

- Lee el manifiesto del archivo de respaldo.
- **Comprueba qué apps del respaldo están instaladas** en el Mac actual y restaura solo esas.
- Informa de las apps omitidas por no estar instaladas.
- En una terminal interactiva, permite elegir, entre las restaurables, cuáles aplicar realmente y
  revisar los conflictos por líneas antes de escribir. `--apps` omite la selección de apps
  interactiva y `--yes` aplica todo sin preguntar.
- Antes de sobrescribir, guarda una copia de seguridad de la configuración actual (ver 3.2).

### 2.3 Catálogo de aplicaciones

- Cada app soportada se describe con una **definición declarativa** (archivo YAML/TOML/JSON): nombre, cómo detectar si está instalada (bundle ID o ruta en `/Applications`), y qué rutas respaldar (por ejemplo `~/Library/Preferences/...`, `~/Library/Application Support/...`, `~/.config/...`, dotfiles).
- Añadir soporte para una nueva app = añadir una definición, sin tocar el código.

### 2.4 Actualización (`update`)

- Solo está disponible para el binario autocontenido. En instalaciones desde código, `uv` o `pip` se indica la vía de actualización correspondiente.
- Consulta la última release publicada en GitHub y compara su versión con la instalada usando el orden SemVer. Si la instalada es igual o superior, no descarga ni escribe nada.
- Descarga `kamino-arm64`, verifica su tamaño y SHA-256 y reemplaza el binario en ejecución de forma atómica. Ante cualquier error, el binario actual queda intacto y el temporal se elimina.
- No requiere `sudo`, no pide confirmación adicional y no modifica configuraciones, respaldos ni instantáneas.
- El comando termina con código 0 si actualiza o ya está al día, y con 1 ante cualquier error. La opción aparece en el menú interactivo.

## 3. Funcionalidades sugeridas (que conviene considerar)

### 3.1 Simulación (`--dry-run`)

Mostrar qué se respaldaría o restauraría, sin tocar nada. Es casi imprescindible en la restauración.

### 3.2 Red de seguridad al restaurar

Guardar automáticamente la configuración actual antes de sobrescribirla, y ofrecer un comando `undo` para revertir la última restauración.

### 3.3 Inspección del respaldo (`list` / `inspect`)

Ver el contenido de un respaldo (apps, versiones, fecha, tamaño) sin restaurarlo. Útil antes de migrar a otro Mac.

### 3.4 Cifrado opcional

Las configuraciones pueden contener tokens, claves API o rutas sensibles. Ofrecer cifrado con contraseña (por ejemplo con `age`) y avisar al usuario cuando una app contenga datos potencialmente sensibles.

### 3.5 Exclusiones y patrones

Poder excluir cachés, logs, historiales o archivos grandes dentro de la configuración de una app. Definir excluidos por defecto sensatos en cada definición del catálogo.

### 3.6 Resolución de rutas entre equipos

Al llevar el respaldo a otro Mac, el nombre de usuario y las rutas pueden cambiar (`/Users/ana` vs `/Users/pedro`). Guardar rutas relativas al home y normalizarlas al restaurar.

### 3.7 Manejo de conflictos

Cuando el archivo destino ya existe y difiere:

- La restauración interactiva abre una TUI que integra la selección de apps, la tabla de conflictos,
  el detalle por líneas y la confirmación final.
- La TUI usa la paleta ANSI del terminal (`ansi-dark` por defecto); `TEXTUAL_THEME` permite elegir
  otro tema válido de Textual, como `ansi-light`.
- La tabla lista solo los conflictos y muestra los conteos de archivos nuevos, idénticos y no
  soportados. Cada archivo se puede revisar, aplicar entero desde el respaldo u omitir; también hay
  acciones globales para aplicar u omitir todos antes de la confirmación.
- En el detalle, la cabecera y el contexto de cada bloque no son decidibles. Cada línea cambiada
  empieza en la versión del respaldo y se alterna con la local; `A`/`L` afectan a todo el archivo y
  `Shift+A`/`Shift+L` solo al bloque actual, con `n`/`p` para saltar entre bloques. La composición
  puede mezclar líneas de un mismo bloque y no se muestra una vista previa completa.
- Una línea sin salto final solo puede quedar al final: si la mezcla la deja en medio, recibe el
  final de línea dominante local.
- Los binarios, los textos de más de 5 MB y los enlaces con enlace se resuelven como archivo
  entero, sin diff de líneas.
- Antes de aplicar una mezcla se validan TOML, JSON, INI y plist. Si ambos originales eran válidos
  y la mezcla no, se bloquea; si alguno ya era inválido, se avisa y se permite aplicar. YAML y
  shell no se validan.
- La confirmación final muestra lo que se restaurará, mezclará y omitirá, y el directorio del
  snapshot; cancelar o volver no escribe ni crea snapshot.
- Si un destino cambia durante la revisión, se aborta toda la operación antes de escribir.
- `--yes` mantiene una vía no interactiva que aplica el respaldo a todos los conflictos.
  `--dry-run` imprime los diffs en la salida estándar sin abrir la TUI ni escribir nada, y
  `--apps` permite elegir apps concretas sin la pantalla de selección.
- El resumen distingue archivos restaurados, mezclados y omitidos; un archivo mezclado cuenta como
  éxito y el código 3 queda reservado a los elementos omitidos.

### 3.8 Compatibilidad de versiones

Avisar si la versión instalada de una app es muy distinta a la del respaldo, o si la versión de macOS difiere, ya que algunas configuraciones podrían no ser compatibles.

### 3.9 Cierre de apps antes de restaurar

Muchas apps reescriben sus preferencias al cerrarse y pisan lo restaurado. Detectar si la app está abierta y pedir cerrarla (o cerrarla con confirmación). Además, `cfprefsd` cachea las preferencias de macOS, así que puede ser necesario refrescarlo tras restaurar.

### 3.10 Verificación de integridad

Incluir checksums en el manifiesto y verificarlos antes de restaurar para detectar archivos corruptos.

### 3.11 Permisos de macOS

Algunas rutas requieren *Full Disk Access* o permisos específicos. Detectar el problema y explicar claramente al usuario cómo concederlos, en lugar de fallar en silencio. Se pospone a las fases finales (v4): el foco inicial son las apps y las configuraciones de usuario, y las que necesiten permisos especiales quedan para después.

### 3.12 Elementos que no son archivos de configuración

Considerar como funcionalidades opcionales:

- lista de paquetes de **Homebrew** (`brew bundle dump`) para reinstalar las apps en el Mac nuevo,
- extensiones de editores (VS Code, etc.),
- ajustes del sistema vía `defaults` (Dock, Finder, teclado, trackpad), fase final (v4),
- configuraciones de shell (zsh, aliases) y de Git,
- claves SSH (con mucha cautela y cifrado obligatorio). Se deja para la fase final (v4).

### 3.13 Automatización

Respaldos programados (con `launchd`) y política de retención (conservar los últimos N). Es útil, pero puede quedar para una versión posterior.

### 3.14 Perfiles / conjuntos

Guardar selecciones de apps como perfiles reutilizables (`--profile trabajo`) para no tener que escoger cada vez.

### 3.15 Salida clara

Barra de progreso, resumen final (qué se respaldó, qué se omitió y por qué), códigos de salida coherentes y registro (`--verbose`, archivo de log).

## 4. Comandos propuestos

| Comando | Función |
| --- | --- |
| `backup` | Crea el archivo de respaldo con las apps elegidas |
| `restore` | Restaura desde un archivo, solo apps instaladas |
| `list` / `inspect` | Muestra el contenido de un respaldo |
| `apps` | Lista las apps soportadas y su estado (instalada o no) |
| `undo` | Revierte la última restauración |
| `verify` | Comprueba la integridad de un respaldo |

Opciones comunes: `--dry-run`, `--apps`, `--output`, `--encrypt`, `--yes`, `--verbose`.

## 5. Flujo general

**Respaldo:** detectar apps instaladas → el usuario selecciona → recolectar archivos (aplicando exclusiones) → generar manifiesto → empaquetar (y cifrar si se pidió) → resumen.

**Restauración:** abrir y verificar el respaldo → leer manifiesto → comparar con apps instaladas → el usuario confirma → copia de seguridad de lo actual → restaurar → refrescar preferencias → resumen.

## 6. Decisiones de diseño

- **Lenguaje:** Python 3.11 o superior, para usar `tomllib` de la librería estándar.
- **Formato del archivo de respaldo:** `tar.gz` estándar con un `manifest.json` dentro. Conserva permisos y enlaces simbólicos, y se puede inspeccionar con herramientas comunes.
- **Formato de las definiciones de apps:** TOML, con comentarios y sin dependencias externas.
- **Distribución:** `uv`/`pipx` desde GitHub, Homebrew tap y binario autocontenido, todos a partir del mismo proyecto empaquetado con `pyproject.toml`.
- **Alcance de la v1:** MVP ajustado, tal como se describe en la sección 7.

## 7. Propuesta de alcance por versiones

**v1 (MVP):** `backup` interactivo, `restore` filtrado por apps instaladas, catálogo declarativo, manifiesto, `--dry-run`, copia de seguridad previa a restaurar y normalización de rutas entre usuarios.

**v2:** cifrado, `inspect`, verificación de integridad, manejo de conflictos, detección de apps abiertas.

**v3:** Homebrew y listado de apps a instalar, extensiones, perfiles.

**v4:** ajustes del sistema (`defaults`), respaldos programados, claves SSH y apps que requieren permisos especiales.
