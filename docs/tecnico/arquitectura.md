# Arquitectura y decisiones técnicas

Este documento concreta cómo se construye la herramienta descrita en el diseño funcional, con las decisiones ya tomadas: Go estático, `tar.gz` con `manifest.json`, catálogo en TOML y distribución como asset `kamino-arm64`. Los formatos, flujos, pruebas y decisiones están en los demás documentos de `docs/tecnico/`.

## 1. Principios de diseño

- **Planificar y luego ejecutar.** `clone` e `implant` primero construyen un plan (una lista de operaciones) y después lo ejecutan. `--dry-run` solo imprime el plan, así la simulación y la ejecución real comparten el mismo código.
- **Catálogo como datos.** Añadir una app es añadir un archivo TOML; el código no conoce ninguna app concreta.
- **Rutas relativas al home.** Todo se guarda relativo a `~`, lo que resuelve la migración entre usuarios (MVP).
- **Nunca destruir sin red.** Antes de sobrescribir se guarda una copia de lo actual, y las escrituras son atómicas.
- **Núcleo sin interfaz.** La lógica no imprime ni pregunta; solo `cli` y `ui` hablan con el usuario. Esto facilita las pruebas.

## 2. Stack y dependencias

| Necesidad | Propuesta | Motivo |
| --- | --- | --- |
| Lenguaje | Go 1.25 o superior | binario estático, arranque inferior a 30 ms |
| Comandos y opciones | parser propio | mismo contrato de alias, ayuda y códigos 0/1/2/3 |
| Interfaz interactiva | Bubble Tea, Lip Gloss y `bubbles` (`key`, `help`) | marco, pila de pantallas, comandos asíncronos y goldens |
| Salida formateada | `internal/ui` | texto plano con tablas y diffs |
| Lectura del catálogo | `BurntSushi/toml` + `go:embed` | TOML estricto y catálogo dentro del binario |
| Validación del catálogo | `internal/catalog` | reglas estrictas sobre el TOML |
| Archivo del clon | `archive/tar`, `compress/gzip`, `crypto/sha256` | sin dependencias externas |
| Datos de las apps | `internal/plist` | XML y binario, sin ejecutar binarios |
| Pruebas | `go test ./... -race` | HOME temporales e inyección de dependencias |
| Calidad | `gofmt`, `gofumpt`, `go vet`, `golangci-lint` | formato, vet y lint |

## 3. Arquitectura

El código se organiza en capas: la interfaz (`cli`, `ui`) llama a los casos de uso (`clone`, `implant`), que se apoyan en módulos de dominio sin interfaz (catálogo, detección, archivo, rutas, seguridad).

&#91;embedded content: capas del paquete · 3 niveles\]

Cada capa se apoya en la de abajo: `cli` y `ui` llaman a `clone` y `implant`, que usan módulos de dominio que nunca imprimen ni preguntan. El comando `implant` usa el caso de uso `internal/implant`.

Estructura propuesta del paquete:

```
cmd/kamino/
cmd/release/
internal/
├── release/        # versionado, changelog, build, smoke y publicación
├── cli/            # argumentos, gating de TUI, salida y códigos
├── app/            # flujos de texto compartidos
├── texts/          # textos visibles y glosario
├── ui/             # salida no interactiva
├── clone/          # plan_clone y write_clone
├── implant/         # sesión, plan, clasificación y aplicación
├── update/         # consulta, descarga, verificación y reemplazo
├── conflicts/      # SequenceMatcher, diff, composición y validación
├── progress/       # progreso y cancelación cooperativa
├── tui/            # App, Frame, Screen, theme, pantallas y goldens
├── plan/           # Plan, Operation y recuento
├── catalog/        # AppDefinition y apps/*.toml embebidos
├── detect/         # Info.plist y búsqueda de comandos
├── archive/        # tar.gz, manifiesto y checksums
├── ghostty/        # config-file y temas
├── paths/          # SafeHome con openat/O_NOFOLLOW
├── snapshot/       # holocrón previo al implantación
└── errors/         # errores propios
```

## 6. Decisiones técnicas relevantes

- **Detección de apps.** Hay dos mecanismos. Para las apps con `.app` se lee el `Info.plist` en `/Applications`, `~/Applications` y `/System/Applications`, buscando por `CFBundleIdentifier`; es más rápido que `mdfind` y no lanza subprocesos, que queda como respaldo. Para las herramientas de terminal se busca el ejecutable en el PATH sin ejecutarlo y, si no aparece, en las rutas de `search_paths` de la definición.
- **El PATH no es fiable en un Mac nuevo.** Antes de configurar el shell, rutas como `/opt/homebrew/bin` pueden no estar en el PATH, y la herramienta aparecería como no instalada sin estarlo. Por eso cada definición puede declarar rutas adicionales donde buscar el binario.
- **Estado frente a configuración.** Algunas carpetas mezclan ambos: la de herdr guarda también logs y estado de sesión. Las definiciones deben clonar solo los archivos de configuración y excluir el resto (`**/*.log`, `session.json`).
- **Exclusiones.** Los patrones como `**/CachedData` se resuelven con un `fnmatch` propio (`*` también coincide con `/`) y se prueban con casos límite.
- **Seguridad al extraer.** `internal/archive` rechaza rutas absolutas o con `..`, tipos especiales y enlaces que salen del destino; verifica todos los miembros antes de extraer en un directorio temporal.
- **Enlaces simbólicos.** Se guardan como enlaces y no se siguen, para no salir del home por accidente.
- **Permisos y atributos.** Se conserva el modo de cada archivo, no el propietario. `archive/tar` no conserva atributos extendidos de macOS, algo que en configuraciones rara vez importa.
- **Escrituras atómicas.** Todo se escribe en un temporal y se renombra, tanto el archivo del clon como cada archivo implantado.
- **Ubicación y nombre del clon.** Por defecto se crea en `~/.config/kamino/vault/` como `clone-<aaaammdd>.tar.gz` con la fecha local y el directorio se crea al escribir. La TUI puede cambiarlo con `clone.output_dir`; la CLI no lee esa configuración. Sin `--force` un destino existente es un error; con `--force` se revalida y se reemplaza de forma atómica. `--output` lo cambia.
- **Holocrón de seguridad.** Se guarda por defecto en `~/.config/kamino/vault/snapshot/<fecha>.tar.gz` con el mismo formato que un clon. La TUI puede cambiar el directorio con `implant.snapshots_dir`; la CLI no lee esa clave. Solo se crea si hay destinos existentes que proteger; el id sintético `snapshot` y el formato no cambian. `snapshot.Apply` no existe: la implantación toma `snapshot.Lock` –`flock` exclusivo sobre `<directorio>/.snapshot.lock`– antes de crear el holocrón y lo suelta al terminar de escribir los destinos, así dos implantaciones concurrentes no se podan el respaldo. `internal/snapshot.PruneSnapshots` borra después los holocrones más antiguos hasta dejar `implant.snapshots_limit` (5 por defecto, `0` ilimitado) y solo considera archivos regulares con el nombre que escribe el paquete; un fallo de escritura del holocrón nuevo no borra nada y un fallo de poda aborta la implantación antes de escribir destinos. `--dry-run` no poda. Los holocrones anteriores en `~/Library/Application Support/kamino/snapshots/` no se migran.
- **Configuración compartida.** `internal/config` carga y guarda `~/.config/kamino/config.toml` con las claves `catalog.dir`, `clone.output_dir`, `implant.snapshots_dir` y `implant.snapshots_limit`. `internal/cli` no importa este paquete: `internal/app` resuelve `catalog.dir` y `implant.snapshots_limit` para la vía de texto y pasa el directorio del catálogo a `implant.Options`; `clone.output_dir` y `implant.snapshots_dir` siguen siendo exclusivos de la TUI. Un archivo ausente o inválido usa el catálogo integrado y el límite por defecto; `clone`/`implant` callan y los comandos `apps` avisan y muestran la ruta efectiva. El guardado es atómico, escribe siempre las cuatro claves y no crea los directorios configurados. La TUI invalida el catálogo en memoria al guardar un `catalog.dir` distinto.
- **Catálogo.** Viene embebido en el paquete y se complementa con las definiciones de `catalog.dir` (por defecto `~/.config/kamino/apps/`), donde se pueden añadir o sobrescribir definiciones sin tocar el código.
- **Autoría de apps.** `internal/catalog` expone `Parse`, `InspectDir`, `RenderTemplate`, `DefaultExcludes` y `CreateDefinition`; `internal/app` añade los flujos `apps` y la TUI reutiliza `CreateDefinition`. `apps new` exige detección y al menos una ruta, valida antes de escribir y usa temporal más rename; `--print` no toca el disco. `apps list` y `apps validate` usan `InspectDir`, que informa por archivo y detecta ids duplicados sin cambiar el fail-fast de `LoadCatalog`. La plantilla comenta los campos avanzados y activa la lista estándar de exclusiones. La pantalla «Añadir app» se abre desde Configuración (el menú conserva sus cinco opciones), crea un borrador `.toml.tmp` en el catálogo, lo abre con `$VISUAL`/`$EDITOR`/`vi`, valida al volver, confirma override o sobrescritura y recarga el catálogo tras el rename atómico; al terminar se vuelve a Configuración. `apps customize` y la pantalla «Personalizar app» parten del TOML integrado en bruto (`catalog.EmbeddedSource`, por id declarado) en lugar de la plantilla, y reutilizan `CreateDefinition` y la pantalla de autoría. `tea.ExecProcess` suspende la TUI durante el editor.
- **Códigos de salida.** 0 éxito, 1 error, 2 uso incorrecto, 3 éxito con elementos omitidos.
- **Límite del HOME.** `SafeHome` abre el home una vez y recorre sus descendientes con `dir_fd` y `O_NOFOLLOW`. La validación léxica de `to_absolute`/`to_relative` sigue siendo la primera barrera; el acceso con descriptores impide que un enlace intermedio redirija una lectura o escritura fuera del home. Si la plataforma no ofrece estas primitivas, se aborta en lugar de degradar a `resolve()` seguido de una apertura insegura.
- **Enlaces finales.** Se conserva la regla existente: un enlace final se guarda como enlace, nunca se sigue y solo se admite si su destino es relativo y no contiene `..`. `SafeHome.read_bytes` rechaza el componente final si es un enlace; `lstat` y `readlink` permiten representarlo en el manifiesto.
- **Cambios concurrentes.** La planificación calcula el checksum con el mismo acceso seguro que usa el empaquetado, que reabre cada ruta y compara el contenido con el manifiesto. Si una ruta cambia a un enlace o a otro tipo entre el plan y la escritura, la clonación falla en vez de incluir datos externos. En la implantación se valida cada destino antes del holocrón y se vuelve a abrir cada padre durante la aplicación; si la jerarquía cambia, se detiene sin escribir fuera del home. Además, la sesión guarda el hash del archivo regular, el destino del enlace o la ausencia del destino al planificar y los re-verifica justo antes del holocrón: si alguno cambió durante la revisión, aborta toda la implantación sin escribir.
- **Conflictos e híbridos.** `internal/conflicts` contiene el modelo y las funciones puras. La comparación decodifica UTF-8 y normaliza CRLF solo para comparar; un clon perfecto no se escribe. Un contenido con NUL o que no decodifica es binario; un texto de más de 5 MB o de más de 20 000 líneas se trata como grande para proteger el diff y la TUI. Los bloques se generan con un `SequenceMatcher` compatible y 3 líneas de contexto; dentro de cada bloque, las líneas iguales son contexto y cada par original/clon o línea de un solo lado es una fila decidible que empieza en el clon. `build_hunks` y `compose` comparten la misma generación de filas. La composición conserva las líneas iguales del original, adapta las del clon al estilo de final de línea original y, si una línea sin salto final queda en medio, le añade el EOL dominante; solo la última puede conservar la ausencia de salto final. El núcleo compone el archivo; la TUI nunca inyecta bytes arbitrarios y D3 renderiza un solo bloque con un `Static`, no miles de widgets.
- **Validación de híbridos.** Se valida por extensión: TOML (`BurntSushi/toml`), JSON con detección de claves duplicadas, INI/CFG/Git config estricto y plist XML/binario (`internal/plist`, con detección de claves duplicadas en XML). Solo se bloquea si ambos originales parsean y el híbrido no; si alguno ya era inválido se avisa y se permite aplicar. YAML y shell no se validan. `apply` repite la validación como defensa del núcleo.
- **TUI.** `internal/tui.App` es la raíz de Bubble Tea; una pila de `Screen` conserva el estado inferior y la app guarda frase, última misión, último código y modo. `Frame` posee cabecera, separadores, body, zona de mensajes y pie, y entrega a cada pantalla el ancho y alto del body; desde 100 columnas el bloque de contenido se centra dentro del body. `theme` centraliza tokens, símbolos y estilos ANSI; `layout` mide, trunca y garantiza exactamente `h` líneas. El pie y la ayuda salen de `Keys()`; no hay leyendas duplicadas. `Ctrl+C` se reparte por pantalla y respeta los puntos de no retorno. Las operaciones largas (recuento, planificación, escritura, implantación y actualización) se ejecutan en `tea.Cmd` con progreso y cancelación. La app se abre solo con entrada y salida TTY y sin `KAMINO_PLAIN`; `implant --yes` y `--dry-run` usan texto. La TUI muestra avisos de secretos en C2, C3, D2 y D3 y no persiste su contenido.
- **Campos de texto de la TUI.** `internal/tui/field.go` es el único componente de entrada: `field` guarda etiqueta, valor, placeholder, cursor como índice de runas y el estado `Invalid`, y expone `SetValue`, `Insert`, `Backspace`, `MoveCursor` y un `Render` de una sola fila. C1, C2, D0 y Configuración guardan `field` y solo deciden qué campo recibe cada tecla, cuándo se abre o se cierra y qué validación aplica; `fieldOptions` lleva ancho, ancho de etiqueta y foco, y `fieldLabelWidth` reserva la columna de etiquetas (incluido « · editando» con `NO_COLOR`) para que los valores queden alineados. No se usa el cursor real del terminal: Bubble Tea v1.3.10 solo expone `tea.ShowCursor`/`tea.HideCursor` y su renderer devuelve el cursor al inicio de la última línea tras cada frame, así que el cursor de edición se dibuja con vídeo inverso sobre el carácter de inserción (o un espacio al final), y solo en el campo enfocado. La ventana horizontal no guarda estado: se calcula en cada `Render` a partir del valor, el cursor y el ancho, reserva el ancho real de `theme.Symbols.Ellipsis` en cada lado oculto y avanza runa a runa con `lipgloss.Width`, sin partir UTF-8 ni caracteres de doble ancho. Los errores siguen en la zona de mensajes del `Frame` y el campo se marca con `Invalid`; no se añaden filas de error. `?` y `F1` se interceptan en `App.Update`, abren la ayuda y no llegan al campo.
- **Progreso y cancelación.** `internal/progress` define `Event` y `Callback`. La cancelación viaja como `context.Context` en el primer parámetro de los casos de uso (`archive`, `plan`, `snapshot`, `clone`, `implant`, `update`); el núcleo emite fases y consulta `ctx.Err()` en los bucles largos, y una cancelación devuelve `OperationCancelled`. La CLI crea un contexto raíz por invocación y la TUI crea un hijo por operación, cuyo `cancel` se guarda bajo mutex y se llama desde `Esc` y `Ctrl+C`; las peticiones HTTP de la actualización usan `http.NewRequestWithContext`.
- **Empaquetado de la TUI.** La TUI usa Bubble Tea, Lip Gloss y `bubbles` (`key`, `help`); el catálogo viaja embebido con `go:embed` y no hay recursos externos. Los golden files y el test de contrato viven en `internal/tui`.
- **Referencias de Ghostty.** El descubrimiento es específico de `ghostty` y se limita a `config-file` y a la selección de temas. Resuelve cada candidato con el acceso seguro, aplica las exclusiones antes de leer, mantiene límites de tamaño, profundidad y cantidad, y deduplica por `(app_id, ruta relativa)`.
- **Exclusiones y aviso.** Los clones nuevos excluyen credenciales independientes conocidas, historiales, sesiones, cachés y registros. El aviso de que el archivo no está cifrado se muestra siempre antes de clonar y no modifica los bytes de los ajustes. Las exclusiones no se aplican al implantación de clones antiguos ni a la instantánea previa.
- **Actualización del binario.** `internal/update` es un caso de uso independiente de la CLI. Solo actúa sobre binarios marcados como autocontenidos por `-ldflags`, resuelve el ejecutable real y consulta la API de GitHub. La versión se compara como `X.Y.Z`; no se degrada ni se descarga si la instalada es igual o superior. La integridad se basa en el tamaño y el digest SHA-256 del asset. El reemplazo usa un temporal en el mismo directorio y `os.Rename`, sin `sudo` y sin ejecutar el binario descargado. No hay comprobación automática ni rollback.
- **Tema ANSI de la TUI.** `internal/tui/theme` usa `ansi-dark` por defecto para que fondo, texto, separadores, wordmark, cápsulas y estados salgan de la paleta ANSI del terminal. `KAMINO_THEME` admite `ansi-light`; otros valores caen a `ansi-dark`. `NO_COLOR` se inyecta antes de crear la app y todos los estados conservan icono y palabra. Los campos usan `FieldFocus` (acento sobre ANSI 8 en `ansi-dark`, ANSI 7 en `ansi-light`), `FieldLabelFocus` (negrita sobre el mismo fondo), `FieldIdle`, `FieldLabelIdle`, `FieldError` y `FieldCaret` (vídeo inverso); con `NO_COLOR` no emiten color ni fondo y el foco se distingue por `▌`, el cursor inverso y «editando». `◉`, `●`, `○`, `◀`, `▶`, `▌`, `▸`, `▾`, `─`, `·`, `⚠`, `ℹ` y `✕` tienen equivalentes ASCII cuando la codificación no es UTF-8 o se define `KAMINO_NO_LOGO`; esa variable además oculta el wordmark.

