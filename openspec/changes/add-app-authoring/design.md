# Design

## Context

Ver `proposal.md` para la motivación. El estado actual que condiciona el diseño:

- `internal/catalog.LoadCatalog` carga las definiciones embebidas con `go:embed`, lee `*.toml` del directorio de usuario, las parsea con `BurntSushi/toml` en modo estricto (`Undecoded`) y aplica `AppDefinition.Validate`. `loadUser` ordena los nombres y falla ante el primer error; un id duplicado entre archivos también falla. No hay modo de inspeccionar un catálogo roto.
- `internal/app.ExecuteClone` resuelve `defaults.CatalogDir(home)` cuando no recibe un catálogo inyectado. `ExecuteDeploy` deja `Catalog` nil y `deploy.PrepareDeploy` vuelve a resolver el directorio integrado. `internal/clone` y `internal/deploy` aceptan `CatalogDir` como inyección, pero no conocen `config.toml`.
- `internal/tui.App.loadCatalog` usa siempre `defaults.CatalogDir(a.home)` e ignora `catalog.dir` guardado en `config.toml`. La pantalla de Configuración sí guarda el valor, pero no invalida el catálogo en memoria.
- `internal/config.LoadConfig` resuelve cada ruta contra el HOME, rechaza valores inválidos y conserva los valores legibles en `LoadResult.Config` aunque haya error. `app.snapshotsLimit` usa la política contraria: ante cualquier error usa el valor por defecto.
- La TUI usa Bubble Tea 1.3.10, que ya ofrece `tea.ExecProcess`. `Screen.View` debe seguir siendo puro: no lee entorno, reloj ni disco. Los campos compartidos son de una sola fila; no hay editor multilínea.
- Las specs existentes que cambian son `user-configuration` y `cli-options`; la capability nueva es `app-authoring`. El menú conserva sus cinco opciones y la acción «Añadir app» vive en Configuración.

## Goals / Non-Goals

**Goals:**

- Una sola resolución de `catalog.dir` para TUI, `clone`, `deploy` y `apps`.
- Crear definiciones válidas por construcción desde la CLI y un flujo guiado por editor en la TUI.
- Inspeccionar y validar el catálogo sin cambiar la semántica fail-fast de `clone` y `deploy`.
- Mantener el formato TOML, el manifiesto, la detección y el confinamiento al HOME sin cambios.

**Non-Goals:**

- Autodescubrimiento de rutas o generación de definiciones a partir de apps instaladas.
- Editar o borrar definiciones existentes desde la TUI.
- Incluir definiciones de usuario dentro del archivo de clon.
- Salida `--json`, wizard interactivo en la CLI o cambios al esquema `AppDefinition`.
- Aplicar exclusiones por defecto a definiciones existentes que no las declaren.

## Decisions

### 1. Resolución única de `catalog.dir` en `internal/app`

Se añade `app.CatalogDir(home) (string, error)` con la misma forma que `snapshotsLimit`:

```
loaded := config.LoadConfig(home, "")
if loaded.Error != nil { return defaults.CatalogDir(home) }
return loaded.Config.CatalogDir
```

La política de "todo o nada" ante un archivo inválido se mantiene: un error en cualquier clave descarta también un `catalog.dir` legible. Es predecible y coincide con el comportamiento actual de `snapshots_limit`.

Usos:

- `ExecuteClone`: sustituye `defaults.CatalogDir` por `CatalogDir`.
- `ExecuteDeploy`: pasa `CatalogDir` en `deploy.Options`; `PrepareDeploy` deja de caer al valor integrado cuando el llamador ya lo resolvió.
- TUI: `loadCatalog` usa `CatalogDir` y `screens_settings.go` invalida `app.catalog` al guardar.
- Comandos `apps`: resuelven el directorio con la misma función.

Alternativas descartadas:

- `config.EffectiveCatalogDir`: mezcla la política de la CLI con el parseo de configuración.
- Duplicar la resolución en TUI y CLI: dos caminos que se desincronizan.
- Hacer que `clone`/`deploy` lean `config.toml` directamente: rompe la separación entre casos de uso y configuración.

`internal/clone` y `internal/deploy` conservan `CatalogDir` como inyección y no importan `config`. La dirección de dependencia nueva es TUI → `app`; no hay ciclo porque `app` no importa TUI ni CLI.

### 2. Silencio en runtime, aviso en autoría

- `clone`, `deploy` y `update` no informan de un `config.toml` inválido; usan el catálogo integrado y los valores por defecto.
- `apps list`, `apps validate` y `apps new` escriben un aviso a stderr cuando la configuración es inválida y usan el catálogo integrado. Además, `list` y `validate` muestran siempre la ruta efectiva en stdout.
- `apps validate <ruta>` con una ruta explícita no depende de la configuración y no muestra ese aviso.

Un error de lectura del directorio del catálogo no es un error de configuración: `LoadCatalog` sigue fallando de forma visible.

### 3. Núcleo de autoría en `internal/catalog` y flujo de escritura en `internal/app`

`internal/catalog` gana:

- `Parse(name string, data []byte) (*AppDefinition, error)`: exporta el parseo estricto actual.
- `InspectDir(dir string) []DefinitionResult`: enumera los `*.toml`, parsea cada uno y devuelve archivo, id, nombre, error y avisos. Detecta ids duplicados entre archivos. No aborta por un archivo malo.
- `RenderTemplate(TemplateOptions) (string, error)`: genera la plantilla comentada.
- `DefaultExcludes []string`: lista estándar usada por la plantilla.
- `CreateDefinition(dir string, data []byte, force bool) (CreateResult, error)`: parsea, valida, comprueba duplicados y escribe de forma atómica. Es el único punto de escritura.

`loadUser` y `InspectDir` comparten la enumeración ordenada de archivos para no divergir. `LoadCatalog` conserva su fail-fast: colapsa el primer error de `InspectDir`/`loadUser` y no cambia los códigos de salida existentes.

`internal/app` añade los flujos `apps` (`ExecuteAppsList`, `ExecuteAppsValidate`, `ExecuteAppsNew`) o funciones equivalentes; `internal/cli` solo traduce argumentos y salida. La TUI llama a `CreateDefinition` a través de la misma capa.

Alternativas descartadas:

- Escribir TOML desde CLI y TUI por separado: duplica validación, duplicados y atomicidad.
- Un paquete nuevo `internal/authoring`: la superficie es pequeña y el esquema ya vive en `catalog`.
- Cambiar `LoadCatalog` a tolerante: alteraría la semántica de `clone`/`deploy` y los códigos de salida.

### 4. `apps new` nace válido

Contrato:

```
kamino apps new <id> \
  (--command <cmd> | --bundle-id <id>) \
  (--path <ruta> ... | --optional-path <ruta> ...) \
  [--name <nombre>] [--search-path <dir> ...] \
  [--force] [--print]
```

- `--name` por defecto es el id.
- Las rutas y los `--search-path` son repetibles. El parser actual guarda un valor por flag; se amplía con un conjunto de flags multivalor o se parsea `apps new` con un acumulador local.
- `--print` escribe a stdout y no toca el disco. Sirve para generar definiciones del catálogo embebido.
- `--force` solo autoriza sobrescribir `<id>.toml`. Si otro archivo del usuario declara el mismo id, se rechaza siempre y se nombra el archivo.
- Override de una app integrada: se permite con aviso, igual que hoy en la carga.
- Antes de escribir: parsear, validar, comprobar duplicados y detectar si la app está instalada. Si la detección no la encuentra, se avisa y se crea igualmente.
- Escritura: crear `catalog.dir` con `0755` si falta, temporal en el mismo directorio, `0644`, `rename` atómico. Si algo falla, no queda archivo ni temporal.
- Códigos: 0 creada; 1 validación, duplicado, archivo existente sin `--force` o error de E/S; 2 uso incorrecto.

La plantilla incluye activos `id`, `name`, detección, rutas y `exclude` estándar. Los campos avanzados (`command_aliases`, `install_marker`, `optional_patterns`, `env_paths`, `xdg_paths`) van comentados. Los comentarios van en español.

Alternativas descartadas:

- Esqueleto inválido escrito en el catálogo: deja el catálogo roto hasta que el usuario edite.
- Wizard interactivo en la CLI: nueva UX de preguntas, dos caminos de entrada y más pruebas; la TUI ya cubre el flujo guiado.

### 5. `apps list` y `apps validate`

- `validate` sin ruta: catálogo de usuario efectivo, todos los archivos y duplicados.
- `validate <archivo>`: solo sintaxis, semántica y avisos de ese archivo.
- `validate <directorio>`: todos los archivos y duplicados.
- `list`: vista fusionada integradas + usuario, con origen (`integrado`/`usuario`) y estado. Muestra los archivos inválidos con su error, pero termina con 0 si pudo leer el directorio; `validate` es el comando que falla con 1.
- Avisos: `exclude` vacío, `id` distinto del nombre del archivo, override de una app integrada. No cambian el código de una validación correcta.
- Formato en español, con una fila por definición y la ruta efectiva en la cabecera.

### 6. Pantalla TUI «Añadir app»

```
[ Menú ] --Enter--> [ Configuración ] --Enter en "Añadir app"--> [ Añadir app ]
                                                                     |
                                                                     | Enter/e
                                                                     v
                                                             .kamino-draft-<rand>.toml.tmp
                                                             en el catalog.dir efectivo
                                                                     |
                                                                     v
                                                             $VISUAL -> $EDITOR -> vi
                                                                     |
                                                      +--------------+---------------+
                                                      |                              |
                                                 editor falla                   vuelve
                                                      |                              |
                                                      v                              v
                                               muestra la ruta                 valida el borrador
                                                                                     |
                                                                +--------------------+--------------------+
                                                                |                    |                    |
                                                             inválido              válido          id en otro archivo
                                                                |                    |                    |
                                                                v                    v                    v
                                                         errores + reeditar   confirma override/     error + reeditar
                                                         o descartar          sobrescritura
                                                                                     |
                                                                                    sí
                                                                                     |
                                                                                     v
                                                                           rename a <id>.toml
                                                                           recarga catálogo
                                                                           éxito -> Configuración
```

- El borrador vive en el mismo directorio que el destino para que el rename sea atómico, y termina en `.toml.tmp` para que `loadUser` no lo vea.
- El esqueleto de la TUI nace inválido: `id = ""`, `name = ""`, `command = ""`, `paths = []`. Un borrador sin tocar no se puede commitear.
- La pantalla es una sola `Screen` con estados internos (intro, error, confirmación, éxito). No se añade un formulario multilínea. La acción vive en Configuración; el menú conserva sus cinco opciones y la autoría se apila sobre Configuración con `push`, así que `Esc` o el éxito vuelven a Configuración.
- Confirmaciones: una sola pantalla enumera el override de la app integrada y la sobrescritura del archivo. Si el usuario rechaza, no se escribe.
- Editor ausente: se muestra la ruta del borrador y las instrucciones; no se pierde.
- Al guardar Configuración con otro `catalog.dir`, `app.catalog = nil`; el siguiente flujo recarga. Esto también aplica si el catálogo nuevo es inválido: se muestra el error y no se toca el archivo.
- `AppOptions` gana una función inyectable para abrir el editor. En producción usa `tea.ExecProcess` con `sh -c` y la ruta como argumento posicional, para admitir `$EDITOR` con flags. En pruebas se sustituye por un editor falso que escribe el borrador; nunca se abre un editor real.
- `View` no lee entorno, reloj ni disco: la resolución del editor y la validación ocurren en `Update` o en comandos.

### 7. Exclusiones por defecto sin cambiar el clon

La plantilla usa `catalog.DefaultExcludes`, una constante con la lista estándar que hoy repiten las definiciones embebidas. `apps validate` avisa si una definición no declara `exclude`, pero no se aplican exclusiones implícitas al cargar. Así no cambian los bytes que produciría un clon con definiciones existentes y el autor recibe feedback.

`plan.sensitivePatterns` sigue siendo una lista distinta para rutas automáticas (Ghostty, env, XDG); no se unifica para no alterar esa semántica.

Alternativa descartada: fusionar exclusiones por defecto en `AppDefinition` al parsear. Silenciaría archivos que un autor quiso incluir y cambiaría clones existentes.

### 8. Ayuda y autocompletado

`printMainHelp` lista `apps`, `apps list`, `apps validate` y `apps new`. `completionScript` incluye `apps` para bash, zsh y fish. Cada subcomando acepta `-h`/`--help` y sale con 0. `internal/cli` sigue sin importar `config`.

## Risks / Trade-offs

- **Un `catalog.dir` antes ignorado pasa a aplicarse** → `apps validate` como preflight, aviso en `apps`, documentación y CHANGELOG. Si el catálogo es inválido, `clone`/`deploy` fallan de forma visible, como ya ocurre con el directorio integrado.
- **La CLI deja de ser hermética respecto al catálogo** → el fallback silencioso evita que un archivo inválido rompa scripts; las pruebas usan HOME temporal. Un `KAMINO_CATALOG_DIR` o `--catalog-dir` queda como posible escape hatch futuro.
- **Id lógico distinto del nombre del archivo** → `CreateDefinition` comprueba ids, no nombres; `validate` avisa del desajuste. Un duplicado en otro archivo siempre se rechaza.
- **Escapado del TOML generado** → se valida antes de escribir y se prueban nombres y rutas con comillas y barras invertidas. Un valor imposible no produce archivo.
- **Editor con argumentos o rutas con espacios** → se ejecuta con `sh -c` y la ruta como `$1`; si el editor no existe, se muestra el borrador. En pruebas se inyecta un ejecutor falso.
- **Borradores huérfanos tras un cierre abrupto** → son `.toml.tmp` ignorados por el loader; se limpian al cancelar o salir. Una limpieza por antigüedad queda fuera de alcance.
- **Dos procesos creando el mismo id** → comprobación de duplicados y rename atómico; sin bloqueo entre procesos. El último `--force` gana, igual que en una edición manual.
- **Pantalla nueva y Configuración distinta** → goldens de Configuración y de la pantalla de autoría, contrato de altura/ancho/SGR, y pruebas de que el editor falso no deja archivos si se cancela.
- **Specs y documentación desactualizadas** → se actualizan el `Purpose` de `user-configuration`, el README (tabla de `catalog.dir`, sección de autoría, árbol del proyecto) y `docs/diseno-tecnico.md`.

## Migration Plan

1. No hay cambio de formato de archivo, manifiesto ni configuración: las cuatro claves y el esquema TOML siguen igual.
2. Usuarios sin `catalog.dir`: sin cambios de comportamiento.
3. Usuarios con `catalog.dir`: el valor empieza a aplicarse; conviene ejecutar `kamino apps validate` antes o después de actualizar. Si el directorio contiene TOMLs inválidos, se corrigen o se retiran.
4. Rollback: revertir el binario. `config.toml` y las definiciones siguen siendo válidos; no hay migración de datos.
5. Guardar Configuración invalida el catálogo cacheado en la misma sesión; no hace falta reiniciar.

## Open Questions

- Escape hatch para ejecuciones herméticas (`KAMINO_CATALOG_DIR` frente a `--catalog-dir`): se puede añadir después sin cambiar las specs.
- Limpieza automática de borradores antiguos: no es necesaria para la corrección; los archivos `.toml.tmp` son invisibles para el loader.
- Mostrar el estado de detección en `apps list`: C1 ya lo muestra; se puede añadir como columna opcional más adelante.
- Compartir definiciones de usuario entre máquinas o dentro del clon: fuera de alcance de este cambio.
