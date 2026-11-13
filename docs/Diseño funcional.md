# Diseño: CLI de clonación y despliegue de configuraciones (macOS)

## 1. Objetivo

Una herramienta de línea de comandos para macOS que guarda las configuraciones de las aplicaciones elegidas en **un único archivo portable**, y que permite desplegarlas después en el mismo Mac o en otro, aplicando solo lo que corresponda a las apps que estén instaladas.

**Fuera de alcance:** otros sistemas operativos, clonación de documentos personales o de los binarios de las apps.

## 2. Funcionalidades principales (requeridas)

### 2.0 Menú interactivo (TUI)

- Una sola app de terminal construida con Bubble Tea, Lip Gloss y `bubbles` reúne menú, clonación,
  despliegue, actualización y pantallas transversales. Se abre solo si la entrada y la salida son
  terminales y no se definió `KAMINO_PLAIN`.
- El menú muestra el wordmark KAMINO en el cuerpo, un separador de ocho cápsulas en el lado derecho
  de la cabecera con la última marcada como el original, una frase fija por sesión y las opciones
  Clonar, Desplegar, Actualizar, Configuración y Abandonar Kamino. Todas se recorren con flechas y
  `Enter`; no hay atajos de una tecla y Escape nunca cierra.
- La cabecera del marco está presente desde 24 filas. El cuerpo del menú degrada por altura: con 30
  o más muestra wordmark, subtítulo, frase, opciones y última misión; con 26 a 29, wordmark y
  opciones; con 24 o 25, wordmark y opciones sin fila de aire; por debajo de 24 aparece el aviso de
  terminal pequeña, que conserva el estado. Las ocho cápsulas permanecen en la cabecera.
- Configuración muestra y edita las rutas de catálogo, clones y holocrones como tres inputs. `Tab` y
  `Mayús+Tab` mueven el foco, `Ctrl+S` valida y guarda, y `Esc` vuelve sin escribir. El archivo
  `~/.config/kamino/config.toml` solo lo lee y escribe la TUI; la vía no interactiva de la CLI usa
  los valores integrados. El archivo se crea solo al guardar.
- Entre el cuerpo y el pie hay una zona de mensajes `INFO`, `AVISO` y `ERROR`, con icono, palabra y
  color ANSI; no se reserva cuando está vacía y no sustituye a la pantalla de error común.
- La ayuda contextual se abre con `?` y, en campos de texto, también con `F1`; se genera de los
  atajos activos de cada pantalla. Todo error tiene cuatro partes: qué pasó, estado de seguridad,
  qué hacer y acciones contextuales más Volver al menú y Salir. El detalle se abre con `t` y se copia
  con `c`.
- `Ctrl+C` cancela antes del punto de no retorno de cada flujo y se ignora con aviso después. Las
  operaciones largas se ejecutan en segundo plano y el pie nunca muestra botones.
- Al terminar un flujo iniciado desde el menú se vuelve a él y se actualiza la última misión. Un
  comando directo no vuelve al menú y sale con el código de la CLI.
- Sin terminal, o con `KAMINO_PLAIN`, se usa la salida de texto. `NO_COLOR` y `TEXTUAL_THEME`
  respetan la decisión de la persona usuaria; el valor por defecto es el tema `ansi-dark`.

### 2.1 Clonación (`clone`)

- Detecta las aplicaciones soportadas que están instaladas en el Mac.
- C1 muestra solo las apps instaladas en una tabla `SEL | APP | ARCHIVOS`, con el contador
  `N de M marcadas`, casillas `[x]`, `[ ]` y `[-]` y cursor `▌`. El recuento se calcula en segundo
  plano con las mismas exclusiones y el mismo confinamiento que el plan. Permite filtrar con `/`,
  seleccionar todas o ninguna con `a` y `n` sobre las filas visibles, y deshabilita las apps sin
  archivos. Las flechas recorren las apps, `Espacio` marca o desmarca sin perder el foco y `Enter`
  continúa; sin selección, la zona de mensajes muestra un `AVISO`. Las apps no detectadas se
  informan en una línea aparte.
- C2 ofrece tres opciones de carpeta: vault por defecto (o `clone.output_dir`), carpeta actual
  (`pwd`) e indicar carpeta, que se convierte en un input. Debajo hay un input de nombre precargado
  con `clone-<aaaammdd>.tar.gz` y editable. La ruta completa se actualiza y se valida en vivo; un
  valor inválido muestra un `ERROR` sin crear nada. Si el clon existe, muestra un diálogo con ruta,
  fecha y tamaño y las acciones Sobrescribir, Cambiar nombre y Volver, con Volver preseleccionada.
  Cambiar nombre propone el primer sufijo libre. `--force` omite el diálogo y avisa con un `AVISO`; los antiguos atajos de directorio se
  retiran. El destino no se crea hasta que empieza la clonación.
- C3 usa una tabla única con las filas Apps, Archivos, Tamaño, Ruta y Estado. La fila Estado pasa
  por Planificando, Listo para clonar, Escribiendo con barra de progreso, Completado, Simulación
  completada o Cancelado antes de escribir. En revisión, `d` enumera las rutas omitidas con su
  motivo y `f` los archivos incluidos; `Enter` clona y `Esc` vuelve. La zona de mensajes advierte de
  que el clon no está cifrado y puede contener secretos. El header cambia de Revisión a Resultado
  al terminar y conserva `Paso 3 de 3`.
- En curso, la barra avanza por archivos dentro de la fila Estado. `Ctrl+C` antes del renombrado
  elimina el temporal y deja intacto el clon anterior; una vez iniciado el renombrado, la operación
  termina y la pantalla indica que ya no se puede cancelar.
- El final de C3 absorbe la antigua C4: muestra Completado o Simulación completada, `d` consulta los
  omitidos, `c` copia la ruta con OSC 52 si se creó el clon y `Enter`/`Esc` continúan. El código 3
  aparece como `AVISO` en la zona de mensajes; no hay una pantalla separada ni un paso adicional.
  `--dry-run` termina en Resultado sin escribir y sin ofrecer copiar la ruta.
- Permite modo no interactivo: `clone --apps vscode,iterm2,git`.
- Sin `--output`, el nombre por defecto es `clone-<aaaammdd>.tar.gz` en
  `~/.config/kamino/vault/`; el directorio se crea al escribir y no en simulación. Sin `--force`/`-f`,
  un clon existente es un error; con `--force` se reemplaza de forma atómica tras revalidar el
  destino, y cualquier error o cancelación elimina el temporal.
- Con terminal interactiva, `--apps` omite C1, `--output` omite C2 y ambos juntos pasan a C3. Un
  valor inválido abre la pantalla correspondiente precargada con el error.

### 2.2 Despliegue (`deploy`)

- D0 combina en una tabla `ARCHIVO | ORIGEN | TAMAÑO` los `.tar.gz` hijos directos de
  `clone.output_dir` (por defecto `~/.config/kamino/vault/`) y del directorio actual, sin recorrer
  subdirectorios. Los ordena por fecha de modificación descendente, deduplica por ruta resuelta y
  muestra el origen (`por defecto` o `este directorio`) y el tamaño. El preview de la fila resaltada
  muestra la ruta completa con `~`, la fecha y el tamaño; si una fuente no existe o no se puede
  leer, se informa en la zona de mensajes sin ocultar la otra. `/` filtra, `o` abre la ruta manual y
  `Esc` vuelve a la tabla, cierra el filtro o cancela según el foco.
- D0 valida solo el manifiesto del clon: rechaza un `tar.gz` dañado, con más de un `manifest.json`,
  con un manifiesto de más de 4 MiB o con un formato más nuevo. Los avisos de arquitectura o macOS
  no bloquean. Un manifiesto válido no implica que la integridad esté verificada.
- D1 muestra una tabla `SEL | APP | ARCHIVOS` con las apps instaladas y su número de archivos, y una
  línea aparte con las no instaladas como “sin base en este Mac”. Ninguna app empieza marcada;
  `Espacio` marca sin mover el cursor, `a`/`n` actúan sobre todas y `Enter` continúa. La preparación
  es cancelable y limpia el staging; un checksum fallido vuelve a D0 sin escribir. El aviso para
  cerrar las apps aparece en la zona de mensajes. `--apps` omite esta pantalla.
- D2 muestra solo los conflictos en una tabla `ARCHIVO | BLOQUES | LÍNEAS | ACCIÓN`, con el cero
  como `·`, cursor `▌` e indicador de scroll. Arriba van los conteos de nuevos, clones perfectos, no
  soportados y avisos. `r` revisa, `c` aplica el clon y `o` conserva el original; `C`/`O` actúan
  sobre todos. Confirmar queda bloqueado hasta que todos tengan decisión y, si falta alguna, la zona
  de mensajes muestra un `AVISO` con el archivo pendiente. Sin conflictos se pasa directo a D4.
- D3 muestra el archivo completo en dos columnas alineadas, `Original` y `Clon`, con el número de
  línea de cada lado y contexto sin casilla. Cada línea cambiada tiene exactamente una versión
  elegida (`●` en el lado elegido y `○` en el descartado); el lado sin línea muestra
  `— sin línea —` y elegirlo omite la línea. Bajo cada título aparece `N líneas elegidas`. Solo se
  renderiza la ventana visible, el archivo puede tener 20 000 líneas y las líneas largas se truncan
  con `…`. `↑↓` mueve, `←→` o `Espacio` eligen versión, `c`/`o` actúan sobre el bloque, `C`/`O`
  sobre el archivo y `Tab`/`Mayús+Tab` recorren bloques. `Esc` vuelve y valida el híbrido; el aviso
  de secretos va en la zona de mensajes. Los binarios, los textos de más de 5 MB y los enlaces se
  resuelven como archivo entero.
- D4 muestra una tabla por app con `NUEVOS`, `REEMPLAZOS`, `HÍBRIDOS`, `SIN DESPLEGAR` y
  `SIN CAMBIOS`, el cero como `·`, una fila de totales, el número de protegidos y la ruta del
  holocrón. Los protegidos son los reemplazos más los híbridos que escribirán un destino existente;
  un híbrido idéntico al original cae en «Sin cambios». Las acciones son horizontales, con Retirada
  preseleccionada, y no existe un atajo de una tecla para aplicar. El aviso de cerrar apps va en la
  zona de mensajes. Volver o cancelar no escribe ni crea holocrón.
- El holocrón se guarda por defecto en `~/.config/kamino/vault/snapshot/`, o en el
  `deploy.snapshots_dir` configurado para la TUI, y se crea solo si hay al menos un destino existente
  que proteger. Antes de escribir se revalidan los destinos; si alguno cambió, se aborta sin
  escribir. Un fallo después del holocrón se informa como “Despliegue incompleto” con los aplicados,
  el fallido y el holocrón, y termina con código 1; no hay rollback automático.
- Los holocrones antiguos en `~/Library/Application Support/kamino/snapshots/` no se mueven ni se
  copian; siguen pudiendo desplegarse indicando su ruta.
- D5 informa de desplegados, híbridos, sin desplegar, sin cambios, sin base en este Mac, archivos
  protegidos y la ruta del holocrón; `c` copia el holocrón y `Enter`/`Esc` continúan. Un híbrido
  cuenta como éxito. Si no hubo originales que proteger, muestra que no se creó holocrón. La línea
  de salida aparece solo si el código no es 0 e incluye la orden para deshacer con
  `kamino deploy <ruta>` cuando existe holocrón. La variante incompleta usa el título en rojo, un
  `ERROR` en la zona de mensajes, muestra aplicados, fallido y holocrón, y termina con código 1.
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

- Solo está disponible para el binario autocontenido. En compilaciones desde código sin la marca de autocontenido
  se indica la vía de actualización correspondiente y no se descarga ni escribe nada.
- U1 muestra los estados consultando, descargando, verificando tamaño y SHA-256, reemplazando y
  resultado. La consulta, la descarga y la ejecución son comandos asíncronos; la barra usa el ancho
  del body y la cancelación se comprueba por chunk. `Esc`/`Ctrl+C` antes del reemplazo deja el
  binario actual intacto y elimina el temporal; después del punto de no retorno se ignora con un
  `AVISO`. Los errores de actualización abren la pantalla de error común con la acción contextual.
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
- D3 muestra el archivo completo en dos columnas alineadas: las líneas iguales son contexto sin
  casilla y cada fila cambiada ofrece la versión del original y la del clon. No existe un estado sin
  decidir: `●` marca la versión elegida y `○` la descartada. El lado sin línea muestra
  `— sin línea —` y elegirlo omite la línea. Solo se renderiza la ventana visible.
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
planificar y revisar en C3 → escribir el temporal con progreso → renombrar al destino → estado final
de C3 con resultado.

**Despliegue:** D0 valida el manifiesto → D1 verifica, extrae y clasifica → D2 resuelve conflictos →
D3 revisa el archivo completo en dos columnas → D4 confirma y crea el holocrón → D5 aplica con
progreso y presenta el resultado o el despliegue incompleto.

**Actualización:** U1 consulta la release → descarga y verifica tamaño y SHA-256 → reemplaza el
binario de forma atómica → pide reiniciar.

## 6. Decisiones de diseño

- **Lenguaje:** Go 1.25 o superior, con binario estático para macOS Apple Silicon.
- **Formato del archivo del clon:** `tar.gz` estándar con un `manifest.json` dentro. Conserva permisos y enlaces simbólicos, y se puede inspeccionar con herramientas comunes.
- **Formato de las definiciones de apps:** TOML, con comentarios y sin dependencias externas.
- **Distribución:** asset `kamino-arm64` publicado por GoReleaser, instalador `scripts/install.sh` y, más adelante, Homebrew tap. `uv` y `pip` dejan de ser vías de instalación.
- **Interfaz:** una sola app ANSI con modos `menu` y `direct`; la salida de texto se conserva
  para scripts y terminales no interactivas.
- **Vocabulario visible:** original, holocrón de seguridad, híbrido, clon perfecto y “sin base en
  este Mac”. Los identificadores internos y el formato no cambian.
- **Alcance de la v1:** MVP ajustado, tal como se describe en la sección 7.

## 7. Propuesta de alcance por versiones

**v1 (MVP):** `clone` interactivo, `deploy` filtrado por apps instaladas, catálogo declarativo, manifiesto, `--dry-run`, copia de seguridad previa a desplegar y normalización de rutas entre usuarios.

**v2:** cifrado, `inspect`, verificación de integridad, manejo de conflictos, detección de apps abiertas.

**v3:** Homebrew y listado de apps a instalar, extensiones, perfiles.

**v4:** ajustes del sistema (`defaults`), clonaciones programadas, claves SSH y apps que requieren permisos especiales.
