# Design

## Context

D0 es `DeployFileScreen` (`src/kamino/tui/deploy.py:157`). Hoy construye una tabla con `Path.cwd().glob("*.tar.gz")`, la ordena por nombre y arranca con el foco en el `Input` de ruta. `Enter` valida el valor del `Input`, lee el manifiesto y avanza a D1. La tabla solo sincroniza la fila resaltada con el `Input`; la selección real vive en el campo de texto.

`DeployState` ya transporta `catalog_dir` y `snapshots_dir`, pero no `clone_output_dir`. `start_deploy_from_menu` (`:1342`) y `build_state` (`:1372`) construyen el estado desde `AppConfig`, que sí tiene las tres rutas. La CLI exige un argumento posicional en `deploy`; D0 aparece desde el menú o cuando el archivo indicado no es válido.

Restricciones que condicionan el diseño:

- D0 solo lee: `stat` para listar y `read_manifest` al elegir. La verificación completa sigue en D1 y no se debe adelantar.
- Las pruebas usan HOME temporales y estados inyectados; la pantalla no debe leer el HOME real.
- El formato del clon, el manifiesto, la configuración y los códigos de salida no cambian.
- La TUI debe funcionar desde 80x24 columnas y toda la interacción debe ser posible con teclado.

## Goals / Non-Goals

**Goals:**

- Un único modelo de foco y selección: la tabla es la fuente de verdad.
- Ver y elegir clones de `clone.output_dir` y del directorio actual en una sola lista.
- Mantener visible el clon elegido al volver desde D1 y conservar el flujo directo con archivo inválido.
- Resolver nombres repetidos y rutas largas sin ambigüedad.
- Mantener la pantalla read-only y las pruebas aisladas del HOME real.

**Non-Goals:**

- Permitir `kamino deploy` sin argumento.
- Listar holocrones o añadir una sección de deshacer.
- Leer el manifiesto de cada fila al resaltarla.
- Cambiar la verificación, extracción o aplicación de D1 en adelante.
- Recordar la última selección entre sesiones.
- Cambiar el esquema de `config.toml` ni las rutas por defecto.

## Decisions

### 1. Lista plana construida desde dos fuentes

D0 combina los `.tar.gz` hijos directos de `state.clone_output_dir` y de `Path.cwd()`. No recorre subdirectorios y no incluye `snapshots_dir`. Cada candidato se representa con una estructura local con `path`, `origin`, `size` y `mtime`.

La deduplicación se hace por ruta resuelta. Si el mismo archivo aparece en ambas fuentes, se conserva el origen de la carpeta configurada. Si ambas rutas son el mismo directorio, se escanea una sola vez.

**Por qué:** es el conjunto que la persona puede elegir sin sorpresas; un escaneo recursivo podría mezclar archivos ajenos y los holocrones tienen otro flujo mental (deshacer).

**Alternativas descartadas:** recorrer subdirectorios (lento y propenso a falsos positivos); incluir holocrones (fuera de alcance y confunde “clon” con “copia de seguridad”).

### 2. Orden por fecha de modificación descendente

La lista se ordena por `mtime` descendente y, en empate, por nombre descendente. El nombre por defecto `clone-yyyyMMdd.tar.gz` ayuda, pero no cubre nombres personalizados, sufijos `-1` ni archivos copiados. La fecha de modificación es la mejor señal de “lo más reciente”.

**Por qué:** el caso más frecuente es desplegar el clon recién creado. El orden alfabético actual deja ese clon al final.

**Alternativa descartada:** ordenar por nombre (predecible pero deja lo nuevo al final); ordenar por tamaño (irrelevante para la elección).

### 3. Tabla plana con columna de origen

La tabla usa las columnas `Archivo`, `Origen` y `Tamaño`. Los valores de origen son `por defecto`, `este directorio` y `ruta manual`; este último solo para la fila final. La ruta completa, la fecha y el tamaño del elemento resaltado se muestran en una línea de preview debajo de la tabla.

**Por qué:** una lista plana ordenada por fecha no entierra una fuente bajo la otra, y la columna de origen más el preview resuelven nombres repetidos. Cabe en 80 columnas sin una cuarta columna de fecha.

**Alternativas descartadas:** secciones con encabezados (si la carpeta por defecto tiene muchos clones, el directorio actual queda fuera de pantalla); una columna de fecha (el nombre y el preview ya dan contexto y la tabla se estrecha demasiado).

### 4. Foco en la tabla y ruta manual como última fila

Al abrir, la tabla recibe el foco. Las flechas mueven la selección y `Enter` actúa sobre la fila resaltada, no sobre un `Input`. El campo de ruta se oculta por defecto.

La última fila es `Otro archivo...`. Al seleccionarla y pulsar `Enter`, se muestra el `Input`, se enfoca y se conserva el valor previo si lo había. `o` salta directamente a esa fila. `Esc` en el `Input` vuelve a la tabla sin cancelar; `Esc` en la tabla cancela el flujo, como hoy.

Si la lista no tiene ningún clon, el `Input` se abre automáticamente para no dejar una pantalla sin salida.

**Por qué:** elimina los dos cursores y hace que `↑↓` funcione sin `Tab`. La ruta manual sigue siendo la última opción, pero deja de competir con la lista.

**Alternativas descartadas:** mantener el `Input` siempre visible (dos modelos de foco); un diálogo modal para la ruta (más pasos y menos contexto).

### 5. Filtro `/`

`/` revela un campo de filtro sobre la tabla, con el mismo patrón de C1. El filtro compara el nombre del archivo y la etiqueta de origen, sin distinguir mayúsculas. `Esc` limpia el filtro y devuelve el foco a la tabla. La fila `Otro archivo...` se añade siempre, incluso cuando el filtro no tiene coincidencias.

**Por qué:** con muchos clones, las flechas no escalan; el patrón ya existe en C1 y evita inventar otra interacción.

**Alternativa descartada:** no incluir filtro en este cambio (deja la pantalla a medias para listas largas).

### 6. Estado y cableado de configuración

`DeployState` incorpora `clone_output_dir: Path | None = None`. `start_deploy_from_menu` lo rellena con `app.config.clone_output_dir`; `build_state` lo rellena con `config.clone_output_dir`; `after_config` reconstruye el estado y hereda el valor nuevo. La pantalla solo lee `state.clone_output_dir`.

Si el valor es `None`, D0 lista únicamente el directorio actual. Eso mantiene las pruebas que construyen `DeployState()` directamente sin tocar el HOME real. No se deriva la carpeta por defecto de `state.home`, porque en el flujo directo `home` es el destino del despliegue y no el HOME de la persona.

**Por qué:** el estado es la fuente de verdad del flujo, las pruebas pueden inyectar rutas y el cambio de configuración se propaga sin casos especiales.

**Alternativa descartada:** que la pantalla lea `self.app.config` directamente (mezcla la pantalla con la app y dificulta las pruebas aisladas).

### 7. Compatibilidad con el flujo directo y el retorno desde D1

En `on_mount`:

- Si `state.file_error` existe, o `state.clone_file` no pertenece a la lista de candidatos, se muestra el `Input` precargado y enfocado, y el error queda visible. La lista sigue disponible.
- Si `state.clone_file` coincide con un candidato, se preselecciona esa fila y la tabla recibe el foco.
- Si no hay ni error ni selección previa, se selecciona el clon más reciente.

Al reanudar D0 desde D1, se vuelve a aplicar la misma regla sobre `state.clone_file`; si el archivo era una ruta manual fuera de las fuentes, se muestra el `Input` precargado. El error de D1 se conserva en `#status`.

**Por qué:** `kamino deploy missing.tar.gz` debe seguir mostrando la ruta que falló, y volver desde D1 no debe perder el clon que se estaba revisando.

**Alternativa descartada:** añadir una fila sintética “ruta elegida” a la tabla (complica la lista y la deduplicación para un caso que el `Input` ya resuelve).

### 8. Solo manifiesto, sin verificación adelantada

Al pulsar `Enter` sobre una fila o una ruta manual se llama a `read_manifest` y, si es válido, a `prepare_deploy`; no se llama a `verify_archive` ni se extrae nada. Un fallo de manifiesto se muestra en `#status` y D0 permanece abierta. La verificación de integridad, la extracción, el holocrón y la aplicación siguen en D1-D5.

**Por qué:** conserva el orden de seguridad y evita que elegir un clon tenga efectos laterales.

### 9. Textos, ayuda y pruebas

Se actualizan el título, los orígenes, los estados vacíos, el preview, las leyendas de atajos y `HELP_D0`. La leyenda cambia según el estado: tabla, filtro o ruta manual. Las pruebas de D0 usan `tmp_path` y `clone_output_dir` inyectado; nunca el HOME real. Se actualizan las pruebas existentes por la fila adicional y el foco inicial, y se añaden casos para las dos fuentes, deduplicación, orden, estados vacíos, filtro, ruta manual, flujo directo y retorno desde D1. También se comprueba que el menú y `after_config` propagan `clone_output_dir`.

## Risks / Trade-offs

- **Directorios enormes** → el escaneo y el `stat` de cada `.tar.gz` pueden tardar. El filtro y el orden por fecha mitigan la elección; si aparece lentitud real, se añadirá un tope o carga diferida en otro cambio.
- **Rutas largas en 80 columnas** → el preview puede ocupar más de una línea y la tabla se queda con poco ancho para el nombre. Se abrevia el HOME con `~`, se usa elipsis en la tabla y se prueba el tamaño mínimo 80x24.
- **Nombres repetidos** → la columna de origen y la ruta completa del preview los desambiguan. Si el mismo archivo está en ambas fuentes, la deduplicación evita una elección ambigua.
- **Enlaces y archivos que desaparecen** → se incluyen solo candidatos que sean archivos legibles; un `OSError` al leer una fuente o un candidato se informa o se omite sin tumbar la pantalla. Si el archivo desaparece entre el listado y `Enter`, `read_manifest` muestra el error.
- **Semántica de `Esc`** → cambia respecto a hoy cuando el `Input` está abierto: primero vuelve a la tabla y después cancela. La leyenda y la ayuda deben reflejarlo para no sorprender.
- **Pruebas existentes** → el foco inicial y la fila `Otro archivo...` cambian los conteos y las pulsaciones esperadas. Se actualizan en el mismo cambio para que la suite siga siendo significativa.
- **Configuración inválida al arrancar** → `after_config` reconstruye el estado; si se olvida `clone_output_dir`, D0 listaría solo el cwd. Se cubre con una prueba específica.

## Migration Plan

No hay migración de datos ni de formato. Los clones, holocrones, manifiestos y `config.toml` existentes siguen siendo válidos. El cambio es solo de la TUI; la CLI y sus códigos de salida no se tocan. La reversión consiste en volver al comportamiento anterior de D0; no deja estado persistido nuevo.

La verificación del cambio sigue el flujo del repositorio: `uv run ruff check .`, `uv run ruff format --check .` y `uv run pytest`. La documentación afectada es el README y las secciones de D0 de `docs/Diseño funcional.md` y `docs/Diseño técnico.md`.

## Open Questions

- ¿Conviene poner un tope de filas o una carga diferida para directorios con cientos de `.tar.gz`? Se deja sin tope en este cambio; si aparece lentitud medible, se resuelve en un cambio aparte sin alterar el contrato de la pantalla.
