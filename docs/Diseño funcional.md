# Diseño: CLI de clonación y despliegue de configuraciones (macOS)

## 1. Objetivo

Una herramienta de línea de comandos para macOS que guarda las configuraciones de las aplicaciones elegidas en **un único archivo portable**, y que permite desplegarlas después en el mismo Mac o en otro, aplicando solo lo que corresponda a las apps que estén instaladas.

**Fuera de alcance:** otros sistemas operativos, clonación de documentos personales o de los binarios de las apps.

## 2. Funcionalidades principales (requeridas)

### 2.0 Menú interactivo (TUI)

- Una sola app de Textual reúne menú, clonación, despliegue, actualización y pantallas
  transversales. Se abre solo si la entrada y la salida son terminales y no se definió
  `KAMINO_PLAIN`.
- El menú muestra el wordmark KAMINO, un separador de ocho cápsulas con la última marcada como el
  original, una frase fija por sesión y las opciones Clonar (`c`), Desplegar (`d`), Actualizar (`u`)
  y Abandonar Kamino (`q`). `q` solo sale desde el menú y Escape nunca cierra.
- La cabecera degrada por altura: 30+ filas muestra todo; 26-29 cabecera, separador y menú; 24-25
  cabecera y menú; por debajo de 24 aparece un aviso que conserva el estado.
- La ayuda contextual se abre con `?` y, en campos de texto, con `F1`. Todo error tiene cuatro
  partes: qué pasó, estado de seguridad, qué hacer y acciones contextuales más Volver al menú y
  Salir. El detalle se abre con `t` y se copia con `c`.
- `Ctrl+C` cancela antes del punto de no retorno de cada flujo y se ignora con aviso después.
- Al terminar un flujo iniciado desde el menú se vuelve a él y se actualiza la última misión. Un
  comando directo no vuelve al menú y sale con el código de la CLI.
- Sin terminal, o con `KAMINO_PLAIN`, se usa la salida de texto. `NO_COLOR` y `TEXTUAL_THEME`
  respetan la decisión de la persona usuaria; el valor por defecto es el tema `ansi-dark`.

### 2.1 Clonación (`clone`)

- Detecta las aplicaciones soportadas que están instaladas en el Mac.
- C1 muestra solo las apps instaladas con su número de archivos, sin marcar ninguna. El recuento se
  calcula en segundo plano con las mismas exclusiones y el mismo confinamiento que el plan.
  Permite filtrar con `/`, seleccionar todas o ninguna con `a` y `n` sobre las filas visibles, y
  deshabilita las apps sin archivos. Las flechas recorren las apps, `Espacio` marca o desmarca sin
  perder el foco y `Enter` continúa. Las apps no detectadas se informan en una línea aparte.
- C2 propone `clone-<aaaammdd>.tar.gz` con la fecha local, resuelve la ruta a absoluta y valida en
  vivo. `F2`, `F3` y `F4` usan el directorio actual, Descargas y Escritorio. Si el clon existe,
  muestra un diálogo con ruta, fecha y tamaño y las acciones Volver, Otro nombre y Reemplazar clon
  navegables con flechas y `Enter`, con Volver preseleccionada. El destino no se crea hasta C3.
- C3 pasa por planificando, revisión y en curso. Muestra apps, archivos, tamaño y destino; `d`
  enumera las rutas omitidas con su motivo y `f` los archivos incluidos agrupados por app. Advierte
  de que el clon no está cifrado y puede contener secretos. Con `--dry-run` termina en la revisión
  sin acción de clonar.
- En curso, la barra avanza por archivos. `Ctrl+C` antes del renombrado elimina el temporal y deja
  intacto el clon anterior; una vez iniciado el renombrado, la operación termina.
- C4 muestra apps, archivos, tamaño y ruta. `d` consulta los omitidos y `c` copia la ruta con OSC 52;
  si el terminal no lo admite, se avisa. La línea de salida solo aparece con código 3. No hay atajo
  para abrir el Finder.
- Permite modo no interactivo: `clone --apps vscode,iterm2,git`.
- Sin `--output`, el nombre por defecto es `clone-<aaaammdd>.tar.gz` en el directorio actual. Sin
  `--force`/`-f`, un clon existente es un error; con `--force` se reemplaza de forma atómica tras
  revalidar el destino, y cualquier error o cancelación elimina el temporal.
- Con terminal interactiva, `--apps` omite C1, `--output` omite C2 y ambos juntos pasan a C3. Un
  valor inválido abre la pantalla correspondiente precargada con el error.

### 2.2 Despliegue (`deploy`)

- D0 valida solo el manifiesto del clon: rechaza un `tar.gz` dañado, con más de un `manifest.json`,
  con un manifiesto de más de 4 MiB o con un formato más nuevo. Los avisos de arquitectura o macOS
  no bloquean. Un manifiesto válido no implica que la integridad esté verificada.
- D1 verifica el archivo completo antes de extraer, lista las apps instaladas con su número de
  archivos y las no instaladas aparte como “sin base en este Mac”. Ninguna app está marcada por
  defecto. La preparación es cancelable y limpia el staging; un checksum fallido vuelve a D0 sin
  escribir. `--apps` omite esta pantalla.
- D2 lista solo los conflictos y los conteos de nuevos, clones perfectos, no soportados y avisos.
  `r` revisa, `c` aplica el clon y `o` conserva el original; `C`/`O` actúan sobre todos. Confirmar
  queda bloqueado hasta que todos tengan decisión. Sin conflictos se pasa directo a D4.
- D3 muestra un bloque a la vez con el contexto una sola vez. Cada línea cambiada ofrece la versión
  del original y la del clon; `●` marca la que irá al archivo final y `○` la que no. `c`/`o` actúan
  sobre el bloque, `C`/`O` sobre todo el archivo y `Tab`/`Mayús+Tab` cambian de bloque. `Esc` vuelve
  y valida el híbrido. Los binarios, los textos de más de 5 MB y los enlaces con enlace se resuelven
  como archivo entero.
- D4 muestra los conteos de desplegados (nuevos y reemplazos), híbridos, sin desplegar y sin
  cambios, el número de archivos protegidos y el directorio del holocrón. Las acciones se recorren
  con flechas, Retirada está preseleccionada y no existe un atajo de una tecla para aplicar. Volver
  o cancelar no escribe ni crea holocrón.
- El holocrón se crea solo si hay al menos un destino existente que proteger. Antes de escribir se
  revalidan los destinos; si alguno cambió, se aborta sin escribir. Un fallo después del holocrón
  se informa como “Despliegue incompleto” con los aplicados, el fallido y el holocrón, y termina con
  código 1; no hay rollback automático.
- D5 informa de desplegados, híbridos, omitidos, sin cambios y de la ruta del holocrón. Un híbrido
  cuenta como éxito. La línea de salida aparece solo si el código no es 0 e incluye la orden para
  deshacer con `kamino deploy <ruta>` cuando existe holocrón.
- El código 3 se reserva para los conflictos conservados en el original, los no soportados y los
  `--apps` no cumplidos. Los clones perfectos, las apps no instaladas y los híbridos no producen
  código 3.
- `--yes` mantiene una vía no interactiva que aplica el clon a todos los conflictos.
  `--dry-run` imprime los diffs en la salida estándar sin abrir la TUI ni escribir nada.

### 2.3 Catálogo de aplicaciones

- Cada app soportada se describe con una **definición declarativa** TOML: nombre, cómo detectar si
  está instalada (bundle ID o ruta en `/Applications`) y qué rutas clonar.
- Añadir soporte para una nueva app = añadir una definición, sin tocar el código.

### 2.4 Actualización (`update`)

- Solo está disponible para el binario autocontenido. En instalaciones desde código, `uv` o `pip`
  se indica la vía de actualización correspondiente y no se descarga ni escribe nada.
- U1 muestra los estados consultando, descargando y verificando tamaño y SHA-256, reemplazando y
  resultado. La descarga reporta progreso y comprueba la cancelación por chunk; cancelar antes del
  reemplazo deja el binario actual intacto y elimina el temporal.
- Tras actualizar se pide reiniciar kamino porque el proceso en marcha sigue siendo el anterior.
- Los errores se clasifican en instalación no autocontenida, plataforma no soportada, red,
  integridad y reemplazo para que la TUI elija la acción contextual sin parsear mensajes.
- El comando termina con código 0 si actualiza o ya está al día, y con 1 ante cualquier error o
  cancelación. La opción aparece en el menú interactivo.

## 3. Funcionalidades sugeridas (que conviene considerar)

### 3.1 Simulación (`--dry-run`)

Mostrar qué se clonaría o desplegaría, sin tocar nada. En `deploy` conserva la vía de texto; en
`clone` con terminal interactiva termina en la revisión C3 sin acción de clonar.

### 3.2 Red de seguridad al desplegar

El holocrón de seguridad guarda automáticamente la configuración actual antes de sobrescribirla.
La orden para deshacer se muestra en el resumen.

### 3.3 Inspección del clon (`list` / `inspect`)

Ver el contenido de un clon (apps, versiones, fecha, tamaño) sin desplegarlo. Útil antes de migrar a otro Mac.

### 3.4 Cifrado opcional

Las configuraciones pueden contener tokens, claves API o rutas sensibles. Ofrecer cifrado con contraseña (por ejemplo con `age`) y avisar al usuario cuando una app contenga datos potencialmente sensibles.

### 3.5 Exclusiones y patrones

Poder excluir cachés, logs, historiales o archivos grandes dentro de la configuración de una app. Definir excluidos por defecto sensatos en cada definición del catálogo.

### 3.6 Resolución de rutas entre equipos

Al llevar el clon a otro Mac, el nombre de usuario y las rutas pueden cambiar (`/Users/ana` vs `/Users/pedro`). Guardar rutas relativas al home y normalizarlas al desplegar.

### 3.7 Manejo de conflictos

Cuando el archivo destino ya existe y difiere:

- D2 lista solo los conflictos; cada fila permite revisar, aplicar el clon completo o conservar el
  original, y muestra cuántos bloques y líneas cambiadas contiene cuando es texto.
- D3 muestra un bloque a la vez: la cabecera y el contexto no son decidibles y cada fila cambiada
  ofrece la versión del original y la del clon. No existe un estado sin decidir: lo mostrado es la
  decisión. No se muestra una vista previa completa del archivo.
- Una línea sin salto final solo puede quedar al final: si el híbrido la deja en medio, recibe el
  final de línea dominante del original.
- Antes de aplicar un híbrido se validan TOML, JSON, INI y plist. Si ambos originales eran válidos
  y el híbrido no, se bloquea; si alguno ya era inválido, se avisa y se permite aplicar. YAML y
  shell no se validan.
- La confirmación final muestra lo que se desplegará, hibridará, omitirá y quedará sin cambios, el
  número de protegidos y el directorio del holocrón. Volver o cancelar no escribe ni crea holocrón.
- Si un destino cambia durante la revisión, se aborta toda la operación antes de escribir.
- `--yes` mantiene una vía no interactiva que aplica el clon a todos los conflictos.
  `--dry-run` imprime los diffs en la salida estándar sin abrir la TUI ni escribir nada.
- El resumen distingue desplegados, híbridos, omitidos y sin cambios; un híbrido cuenta como éxito
  y el código 3 queda reservado a los elementos omitidos y a los `--apps` no cumplidos.

### 3.8 Avisos de compatibilidad

Si la arquitectura o la versión de macOS del manifiesto difieren de las actuales, D0 lo avisa sin
bloquear y sin alterar el código de salida.

### 3.9 Cierre de apps antes de desplegar

Muchas apps reescriben sus preferencias al cerrarse y pisan lo desplegado. D4 recuerda cerrar las
apps abiertas; la detección automática queda fuera de este cambio.

### 3.10 Verificación de integridad

Incluir checksums en el manifiesto y verificarlos antes de desplegar para detectar archivos corruptos.

### 3.11 Permisos de macOS

Algunas rutas requieren *Full Disk Access* o permisos específicos. Detectar el problema y explicar claramente al usuario cómo concederlos, en lugar de fallar en silencio. Se pospone a las fases finales (v4).

### 3.12 Elementos que no son archivos de configuración

Considerar como funcionalidades opcionales:

- lista de paquetes de **Homebrew** (`brew bundle dump`) para reinstalar las apps en el Mac nuevo,
- extensiones de editores (VS Code, etc.),
- ajustes del sistema vía `defaults` (Dock, Finder, teclado, trackpad), fase final (v4),
- configuraciones de shell (zsh, aliases) y de Git,
- claves SSH (con mucha cautela y cifrado obligatorio). Se deja para la fase final (v4).

### 3.13 Automatización

Clonaciones programadas (con `launchd`) y política de retención (conservar los últimos N). Es útil, pero puede quedar para una versión posterior.

### 3.14 Perfiles / conjuntos

Guardar selecciones de apps como perfiles reutilizables (`--profile trabajo`) para no tener que escoger cada vez.

### 3.15 Salida clara

Barra de progreso, resumen final (qué se clonó, qué se omitió y por qué), códigos de salida coherentes y registro (`--verbose`, archivo de log).

## 4. Comandos propuestos

| Comando | Función |
| --- | --- |
| `clone` | Crea el archivo del clon con las apps elegidas |
| `deploy` | Despliega desde un archivo, solo apps instaladas |
| `list` / `inspect` | Muestra el contenido de un clon |
| `apps` | Lista las apps soportadas y su estado (instalada o no) |
| `undo` | Revierte el último despliegue |
| `verify` | Comprueba la integridad de un clon |

Opciones comunes: `--dry-run`, `--apps`, `--output`, `--force`, `--encrypt`, `--yes`, `--verbose`.

## 5. Flujo general

**Clonación:** detectar apps instaladas → el usuario selecciona en C1 → elegir destino en C2 →
planificar y revisar en C3 → escribir el temporal con progreso → renombrar al destino → resumen C4.

**Despliegue:** D0 valida el manifiesto → D1 verifica, extrae y clasifica → D2 resuelve conflictos →
D3 revisa los híbridos por bloques → D4 confirma y crea el holocrón → D5 aplica con progreso y
presenta el resultado o el despliegue incompleto.

**Actualización:** U1 consulta la release → descarga y verifica tamaño y SHA-256 → reemplaza el
binario de forma atómica → pide reiniciar.

## 6. Decisiones de diseño

- **Lenguaje:** Python 3.11 o superior, para usar `tomllib` de la librería estándar.
- **Formato del archivo del clon:** `tar.gz` estándar con un `manifest.json` dentro. Conserva permisos y enlaces simbólicos, y se puede inspeccionar con herramientas comunes.
- **Formato de las definiciones de apps:** TOML, con comentarios y sin dependencias externas.
- **Distribución:** `uv`/`pipx` desde GitHub, Homebrew tap y binario autocontenido, todos a partir del mismo proyecto empaquetado con `pyproject.toml`.
- **Interfaz:** una sola app de Textual con modos `menu` y `direct`; la salida de texto se conserva
  para scripts y terminales no interactivas.
- **Vocabulario visible:** original, holocrón de seguridad, híbrido, clon perfecto y “sin base en
  este Mac”. Los identificadores internos y el formato no cambian.
- **Alcance de la v1:** MVP ajustado, tal como se describe en la sección 7.

## 7. Propuesta de alcance por versiones

**v1 (MVP):** `clone` interactivo, `deploy` filtrado por apps instaladas, catálogo declarativo, manifiesto, `--dry-run`, copia de seguridad previa a desplegar y normalización de rutas entre usuarios.

**v2:** cifrado, `inspect`, verificación de integridad, manejo de conflictos, detección de apps abiertas.

**v3:** Homebrew y listado de apps a instalar, extensiones, perfiles.

**v4:** ajustes del sistema (`defaults`), clonaciones programadas, claves SSH y apps que requieren permisos especiales.
