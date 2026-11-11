# Propuesta de diseño: pantalla de Configuración

## Objetivo

Definir la pantalla de Configuración de la TUI para que la persona usuaria pueda ver y editar las tres rutas de Kamino sin editar `~/.config/kamino/config.toml` a mano. La pantalla debe dejar claro qué valores usa la TUI, cuáles son los defaults y qué hace la CLI.

Ver `proposal.md` para el alcance y `specs/user-configuration/spec.md` para el contrato de comportamiento. Este documento concreta la experiencia visual, los estados y la estructura de la pantalla.

## Principios de diseño

- **Una sola tarea por pantalla:** editar tres rutas y guardar. Nada de opciones de tema, modo de texto ni confirmaciones destructivas.
- **Valores visibles:** los campos muestran el valor efectivo; si el archivo es inválido, conservan el valor legible para poder corregirlo. No se usan placeholders vacíos como única información.
- **La CLI no se confunde con la TUI:** un aviso fijo recuerda que la CLI no usa estas rutas.
- **Las rutas ausentes no son errores:** se avisa de que se crearán al usarse. Solo son errores las rutas vacías, relativas, con `~usuario` o que existen y no son directorios.
- **Guardar es explícito:** `Ctrl+S`. `Esc` nunca guarda.
- **El error no pierde datos:** si el archivo es inválido, se muestran los valores legibles y los defaults; el archivo no se sobrescribe hasta guardar.
- **Coherencia con la TUI existente:** título, subtítulo, campos con borde, línea de pista, estado y barra de atajos, como C2 y las demás pantallas.

## Ubicación y acceso

- Nueva opción **Configuración** en el menú principal, después de **Actualizar** y antes de **Abandonar**.
- Sin atajo de teclado: se selecciona con flechas y `Enter`.
- La pantalla se abre en modo menú. También se abre automáticamente si la TUI arranca con un `config.toml` inválido.
- `Esc` vuelve al menú si la pantalla se abrió desde el menú.
- `Esc` sale con código 1 si la pantalla se abrió para corregir una configuración inválida.
- Si se abrió desde un flujo directo (`kamino clone` o `kamino deploy`) por un error de configuración, al guardar se continúa con la primera pantalla de ese flujo.

## Layout

La pantalla usa el ancho completo del terminal y una columna única. La cabecera y el pie quedan fijos; el cuerpo con los campos se desplaza si el terminal no alcanza a mostrarlo completo.

### Wireframe: 80×24, estado normal con archivo existente

```text
Configuración
Rutas de Kamino · ~/.config/kamino/config.toml
La CLI no usa estos valores; solo la TUI.

Catálogo de aplicaciones
▸ ~/.config/kamino/apps

Destino de los clones
  ~/.config/kamino/vault

Directorio de holocrones
  ~/.config/kamino/vault/snapshot

Definiciones TOML de apps del usuario. Por defecto: ~/.config/kamino/apps.
✔ listo

Tab/Mayús+Tab mover · Ctrl+S guardar · Esc volver · ? ayuda
```

Los campos se dibujan con el borde estándar de Textual. El wireframe muestra solo el contenido; `▸` indica el campo con foco.

### Wireframe: 80×24, configuración inválida al arrancar

```text
Configuración
Rutas de Kamino · ~/.config/kamino/config.toml
La CLI no usa estos valores; solo la TUI.
✖ error: valor relativo en clone.output_dir; usa una ruta absoluta o que empiece por ~.

Catálogo de aplicaciones
  ~/.config/kamino/apps

Destino de los clones
▸ Downloads

Directorio de holocrones
  ~/.config/kamino/vault/snapshot

Corrige la ruta marcada. El archivo actual no se sobrescribirá hasta que guardes.
✖ error

Tab/Mayús+Tab mover · Ctrl+S guardar · Esc salir sin guardar · ? ayuda
```

El aviso de error ocupa una o dos líneas y el cuerpo se desplaza si hace falta. El campo inválido conserva el valor legible para que se pueda corregir; los campos no legibles muestran el default.

### Wireframe: error al guardar

```text
Configuración
...
Directorio de holocrones
  ~/.config/kamino/vault/snapshot

No se pudo guardar la configuración. El archivo anterior no cambió.
✖ error: no se pudo escribir el archivo de configuración.

Tab/Mayús+Tab mover · Ctrl+S guardar · Esc volver · ? ayuda
```

La pantalla permanece abierta, conserva los valores introducidos y permite reintentar.

## Contenido y textos

### Cabecera

| Elemento | Texto | Estilo |
|---|---|---|
| Título | `Configuración` | `$primary`, negrita |
| Subtítulo | `Rutas de Kamino · ~/.config/kamino/config.toml` | `$text`, cursiva |
| Aviso fijo | `La CLI no usa estos valores; solo la TUI.` | atenuado |
| Error de carga | `✖ error: <mensaje>` | `$error`, solo si existe |

### Campos

| Clave | Etiqueta | Ayuda contextual | Default |
|---|---|---|---|
| `catalog.dir` | `Catálogo de aplicaciones` | `Definiciones TOML de apps del usuario.` | `~/.config/kamino/apps` |
| `clone.output_dir` | `Destino de los clones` | `Carpeta de clone-AAAAMMDD.tar.gz.` | `~/.config/kamino/vault` |
| `deploy.snapshots_dir` | `Directorio de holocrones` | `Copia del estado anterior antes de desplegar.` | `~/.config/kamino/vault/snapshot` |

Cada campo muestra su valor efectivo. La línea de pista inferior cambia con el foco y muestra la descripción y el default del campo; la línea de estado muestra la validación:

- `Definiciones TOML de apps del usuario. Por defecto: ~/.config/kamino/apps.`
- `Carpeta de clone-AAAAMMDD.tar.gz. Por defecto: ~/.config/kamino/vault.`
- `Copia del estado anterior antes de desplegar. Por defecto: ~/.config/kamino/vault/snapshot.`

No se usan placeholders como única etiqueta: la etiqueta siempre está visible encima del campo.

### Pie

| Elemento | Texto | Estilo |
|---|---|---|
| Pista | Descripción del campo enfocado, default y validación | atenuado |
| Estado | `✔ listo`, `▲ aviso: ...` o `✖ error: ...` | `$success`, `$warning`, `$error` |
| Atajos | `Tab/Mayús+Tab mover · Ctrl+S guardar · Esc volver · ? ayuda` | `ShortcutBar` |
| Atajos en modo error | `... · Esc salir sin guardar · ? ayuda` | `ShortcutBar` |

Los estados siempre incluyen icono y palabra, no solo color.

## Estados

| Estado | Disparador | Qué se ve | Qué puede hacer la persona usuaria |
|---|---|---|---|
| Normal, sin archivo | `config.toml` no existe | Valores por defecto; estado `▲ aviso: no existe config.toml; se creará al guardar.` | Editar, `Ctrl+S`, `Esc` |
| Normal, con archivo | `config.toml` válido | Valores del archivo y defaults en claves ausentes; `✔ listo` | Editar, `Ctrl+S`, `Esc` |
| Campo con foco | `Tab`, `Mayús+Tab`, `Enter` o clic | Borde de foco; la pista muestra su descripción y default | Editar |
| Aviso de ruta ausente | Ruta de salida, holocrones o catálogo inexistente | `▲ aviso`: se creará al usar o se usarán solo las definiciones integradas | Guardar igualmente |
| Error de campo | Ruta vacía, relativa, `~usuario`, o existente que no es directorio | `✖ error` en el estado; el foco va al primer campo inválido al guardar | Corregir |
| Guardando | `Ctrl+S` con valores válidos | Estado `Guardando…`; se ignoran pulsaciones repetidas | Esperar |
| Guardado | `save_config` termina correctamente | Se actualiza `app.config`; se vuelve al menú o se continúa el flujo directo | — |
| Error al guardar | Fallo de escritura | `✖ error`; los campos y el archivo anterior se conservan | Corregir y reintentar, o `Esc` |
| Carga inválida | `config.toml` inválido al arrancar | Banner de error, valores legibles y defaults; el archivo no se sobrescribe | Corregir y guardar, o `Esc` para salir con código 1 |
| Cancelado | `Esc` en modo normal | Vuelve al menú sin escribir | — |
| Cancelado en modo error | `Esc` con configuración inválida | Sale con código 1 sin escribir | — |

## Navegación y teclado

| Tecla | Acción |
|---|---|
| `Tab` | Siguiente campo; al final, vuelve al primero |
| `Mayús+Tab` | Campo anterior; al principio, va al último |
| `Enter` | Siguiente campo; en el último, no guarda |
| `Ctrl+S` | Valida y guarda |
| `Esc` | Vuelve al menú o sale sin guardar si la pantalla se abrió por un error |
| `F1` o `?` | Abre la ayuda contextual |
| `Ctrl+C` | Cancela la pantalla con el mismo criterio que `Esc` |

`Enter` no guarda: el guardado siempre es `Ctrl+S`. Así se evita escribir el archivo por accidente al pasar de un campo a otro.

## Validación y mensajes

La validación ocurre en dos momentos:

1. **Al cambiar el valor de un campo:** se actualiza el estado con el aviso o error correspondiente.
2. **Al guardar:** se validan los tres campos; si hay errores, no se escribe nada y el foco va al primero.

Reglas:

- Cadena vacía: `✖ error: la ruta no puede estar vacía.`
- Ruta relativa: `✖ error: usa una ruta absoluta o que empiece por ~.`
- `~usuario`: `✖ error: solo se admite ~ o ~/ al inicio.`
- Ruta existente que no es directorio: `✖ error: la ruta existe y no es un directorio.`
- Ruta ausente en catálogo: `▲ aviso: no existe; se usarán solo las definiciones integradas.`
- Ruta ausente en salida: `▲ aviso: no existe; se creará al clonar.`
- Ruta ausente en holocrones: `▲ aviso: no existe; se creará al desplegar.`
- Sin archivo: `▲ aviso: no existe config.toml; se creará al guardar.`
- Guardado correcto: `✔ configuración guardada.`
- Fallo al guardar: `✖ error: no se pudo guardar. El archivo anterior no cambió.`
- Error de carga: `✖ error: <detalle del archivo y la clave>.`

Los avisos no impiden guardar. Los errores sí.

## Guardado

- `Ctrl+S` valida y llama a `save_config`.
- Se escriben siempre las tres claves, con los valores efectivos.
- La escritura es atómica: temporal en `~/.config/kamino` y `os.replace`.
- Se crea `~/.config/kamino` si falta.
- No se crean los directorios de catálogo, salida ni holocrones al guardar.
- Si el archivo no existía, se crea con el primer guardado.
- Si el archivo existía, se reemplaza solo después de una escritura correcta.
- Tras guardar, `KaminoApp.config` se actualiza antes de volver al menú o continuar el flujo directo.
- Si el guardado falla, el archivo anterior conserva sus bytes.

## Accesibilidad y estilo

- Etiquetas visibles encima de cada campo; no se depende de placeholders.
- Foco visible con el borde de Textual y el cursor del campo.
- Estados con icono y palabra además del color.
- Orden de foco lógico: catálogo, salida, holocrones.
- La barra de atajos está siempre visible; el modo error cambia `Esc volver` por `Esc salir sin guardar`.
- La pantalla respeta el tema ANSI de la TUI; no fija colores truecolor.
- Por debajo de 80×24 se usa la guardia de terminal pequeña existente.
- La ayuda contextual muestra los atajos y el nombre de la pantalla con el mismo patrón que el resto de la TUI; los defaults y la creación de directorios se explican en la pista y en el aviso fijo.

## Estructura técnica propuesta

```text
ConfigScreen(Screen[None])
├── Static#settings-title          "Configuración"
├── Static#settings-subtitle       "Rutas de Kamino · ~/.config/kamino/config.toml"
├── Static#settings-note           "La CLI no usa estos valores; solo la TUI."
├── Static#settings-error          Banner de error de carga, oculto si no hay error
├── VerticalScroll#settings-body
│   ├── Static.settings-label       Catálogo de aplicaciones
│   ├── Input#catalog-dir
│   ├── Static.settings-label       Destino de los clones
│   ├── Input#clone-output-dir
│   ├── Static.settings-label       Directorio de holocrones
│   └── Input#snapshots-dir
├── Static#settings-hint           Pista del campo enfocado
├── Static#settings-status         Estado de validación o guardado
└── ShortcutBar
```

### Responsabilidades

- `on_mount`: rellenar los campos con `ConfigLoadResult.raw` y, en su defecto, con `ConfigLoadResult.config`; mostrar el banner si `error` existe; mostrar el aviso de archivo ausente si `exists` es `False`; enfocar el primer campo y actualizar la pista y el estado.
- `on_input_changed`: validar el campo y actualizar el estado.
- `on_descendant_focus`: actualizar la pista y el estado al cambiar de campo.
- `action_next_field` (Enter): mover el foco al siguiente campo sin guardar.
- `action_save` (Ctrl+S): validar los tres campos; si hay errores, enfocar el primero; si no, llamar a `save_config`, actualizar `app.config` y volver o continuar el flujo.
- `action_back` (Esc): volver al menú, o salir con código 1 si la pantalla se abrió por un error.
- `action_interrupt` (Ctrl+C): aplicar el mismo criterio que `action_back`.
- `action_help` (F1/?): abrir la ayuda contextual con `SCREEN_SETTINGS` y `SHORTCUTS_SETTINGS`.

### Textos y estilos

- Añadir a `texts.py`: `SCREEN_SETTINGS`, `SETTINGS_TITLE`, `SETTINGS_SUBTITLE`, `SETTINGS_CLI_NOTE`, las etiquetas y pistas de los tres campos, `SHORTCUTS_SETTINGS`, `SHORTCUTS_SETTINGS_ERROR` y `HELP_SETTINGS`.
- Añadir a `styles.tcss`: `#settings-title`, `#settings-subtitle`, `#settings-note`, `#settings-error`, `#settings-body`, `.settings-label`, `#settings-hint`, `#settings-status` y el estado de error del `Input`.
- Reutilizar `ShortcutBar`, `HelpScreen` y los estilos ANSI existentes.

## Casos de prueba de la pantalla

- Se abre desde el menú con flechas y `Enter`, y `s` no la abre.
- Muestra los defaults cuando no existe `config.toml`.
- Muestra los valores del archivo y defaults en claves ausentes.
- `Ctrl+S` crea el archivo con las tres claves si no existía.
- `Esc` no crea ni modifica el archivo.
- Una ruta relativa, vacía o `~usuario` bloquea el guardado.
- Una ruta existente que no es directorio bloquea el guardado.
- Una ruta ausente muestra aviso y permite guardar.
- Un fallo de escritura deja el archivo anterior intacto y mantiene la pantalla.
- Un `config.toml` inválido abre la pantalla con banner, valores legibles y defaults.
- Cancelar desde el modo error sale con código 1 sin escribir.
- Guardar desde un flujo directo continúa la primera pantalla del flujo.
- Guardar desde el menú vuelve al menú y los flujos siguientes usan los valores nuevos.
- El menú de cinco opciones sigue siendo legible a 80×24.
- Los estados no dependen solo del color.

## Fuera de alcance

- Editar `ui.plain` o `ui.theme`: ya no forman parte de la configuración.
- Editar el TOML como texto libre.
- Restaurar valores por defecto con un botón o atajo adicional.
- Crear un comando de configuración en la CLI.
- Migrar holocrones antiguos.
- Añadir otras opciones de configuración.
