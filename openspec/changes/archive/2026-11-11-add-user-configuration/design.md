# Design

## Context

Ver `proposal.md` para la motivación y `specs/` para el contrato de comportamiento.

El estado actual reparte los valores por defecto entre constantes del dominio: el catálogo del usuario en `~/.config/kamino/apps`, los holocrones en `~/Library/Application Support/kamino/snapshots` y el clon en el directorio actual. La CLI, los flujos de texto y la TUI comparten los casos de uso, pero la TUI construye sus estados con valores fijos al iniciar flujos desde el menú.

Este cambio introduce una configuración que solo usa la TUI. La vía no interactiva de la CLI conserva valores integrados y no lee el archivo. Las pruebas deben seguir usando HOMEs temporales e inyectar rutas; ninguna debe leer el HOME real.

Restricciones:

- El núcleo (`clone.py`, `deploy.py`, `snapshot.py`, `catalog/loader.py`) no imprime, no pregunta y no depende de la interfaz.
- No se añaden dependencias; se usa `tomllib` y la escritura atómica con `os.replace`.
- La configuración no puede debilitar confirmaciones, confinamiento, checksums ni reemplazos atómicos.
- Los textos visibles y la documentación van en español; el código y los identificadores, en inglés.

## Goals / Non-Goals

**Goals:**

- Archivo opcional `~/.config/kamino/config.toml` con tres rutas, leído y escrito solo por la TUI.
- Pantalla de Configuración en el menú, con valores efectivos, validación y guardado atómico.
- Valores por defecto bajo `~/.config/kamino/vault/`.
- Crear los directorios de salida y holocrones solo cuando se usan.
- Recuperar una TUI con configuración inválida sin sobrescribir el archivo.
- Mantener la CLI no interactiva independiente de la configuración.

**Non-Goals:**

- Que la CLI lea, cree o modifique `config.toml`.
- Configurar `ui.plain`, `ui.theme`, `--force`, `--yes`, `--dry-run` o políticas de conflicto.
- Migrar, mover o copiar holocrones existentes.
- Soportar `XDG_CONFIG_HOME` u otra ubicación para el archivo.
- Añadir un comando `config` o flags de configuración a la CLI.
- Retención automática de holocrones.

## Decisions

### 1. La configuración es exclusiva de la TUI; los defaults son compartidos

`src/kamino/config.py` solo lo importan módulos de la TUI (`tui/app.py`, `tui/config.py`, `tui/clone.py`, `tui/deploy.py` y `tui/update.py`). `cli.py`, `commands.py`, `clone.py`, `deploy.py` y `snapshot.py` no leen `config.toml`. La vía no interactiva usa siempre los valores integrados.

Para evitar que CLI y TUI diverjan en los defaults, se crea `src/kamino/defaults.py`:

```python
def catalog_dir(home: Path) -> Path:
    return home / ".config" / "kamino" / "apps"


def vault_dir(home: Path) -> Path:
    return home / ".config" / "kamino" / "vault"


def snapshots_dir(home: Path) -> Path:
    return vault_dir(home) / "snapshot"
```

`catalog/loader.py`, `clone.py` y `snapshot.py` usan estas funciones. `config.py` las usa para completar claves ausentes. La TUI guarda los valores efectivos en `AppConfig`; la CLI nunca construye un `AppConfig`.

Alternativa considerada: que la CLI lea el archivo. Se descarta por decisión de producto: la configuración no debe alterar scripts ni comandos no interactivos, y un archivo inválido no debe romper la CLI.

### 2. Formato, carga y guardado

El archivo es `~/.config/kamino/config.toml` y solo admite:

```toml
[catalog]
dir = "~/.config/kamino/apps"

[clone]
output_dir = "~/.config/kamino/vault"

[deploy]
snapshots_dir = "~/.config/kamino/vault/snapshot"
```

`AppConfig` contiene tres `Path` efectivos, no valores opcionales:

```python
@dataclass(frozen=True)
class AppConfig:
    catalog_dir: Path
    clone_output_dir: Path
    snapshots_dir: Path
```

En una carga válida los tres valores están resueltos. En una carga inválida, los valores legibles se conservan para mostrarlos en la pantalla de corrección —incluido un valor relativo que se debe corregir— y el resto usa el default.

`load_config(*, home=None, path=None) -> ConfigLoadResult` devuelve:

```python
@dataclass(frozen=True)
class ConfigLoadResult:
    config: AppConfig
    error: ConfigError | None = None
    exists: bool = False
    raw: dict[str, str] = field(default_factory=dict)
```

- Archivo ausente: `config` con los defaults, `error=None` y `exists=False`.
- Archivo válido: valores del archivo, `error=None` y `exists=True`.
- Archivo inválido: `config` con los defaults en los valores no legibles, `raw` con los valores de texto legibles para que la pantalla los muestre y permita corregirlos, `error` con el mensaje y `exists=True`. Un TOML sintácticamente inválido no aporta valores legibles, así que `config` queda con todos los defaults y `raw` vacío.
- `ConfigError` lleva un mensaje en español con la sección o clave; no se lanza en la CLI porque la CLI no carga configuración.

El parser es manual sobre `tomllib` para poder conservar valores parciales, nombrar claves exactas y producir mensajes en español sin depender del formato de errores de otra librería. Reglas:

- Solo secciones `catalog`, `clone` y `deploy`; solo `dir`, `output_dir` y `snapshots_dir` respectivamente. `[ui]` y cualquier otra sección o clave se rechazan.
- Todos los valores deben ser cadenas no vacías.
- Las rutas deben ser absolutas o empezar por `~` o `~/`. `~` se expande contra el `home` inyectado, no contra el HOME real del proceso, para que las pruebas sean deterministas; `~usuario` se rechaza. Una ruta relativa también se rechaza.
- La existencia no se exige al cargar: una ruta ausente es válida. Los directorios de salida y holocrones se crean al usarse; un catálogo ausente equivale a un catálogo vacío y no se crea. Si la ruta existe y no es un directorio, se considera inválida y la TUI abre la pantalla de corrección. La validación se repite al guardar y al usar la ruta.

`save_config(config, *, home=None, path=None)` escribe siempre las tres claves. Serializa TOML con cadenas básicas escapadas (comillas, barras y controles) y sin añadir dependencias. Crea `~/.config/kamino` si falta, escribe un temporal en el mismo directorio y lo reemplaza con `os.replace`; si falla, elimina el temporal y el archivo anterior queda intacto. No crea los directorios configurados.

### 3. Pantalla de Configuración

Se añade `src/kamino/tui/config.py` con `ConfigScreen`. El detalle visual, los wireframes y los estados están en `settings-screen.md`:

- Tres campos de texto para catálogo, salida y holocrones, con sus valores efectivos.
- `Ctrl+S` guarda; `Esc` vuelve sin guardar. La barra de atajos muestra ambas acciones.
- Un área de estado muestra el error de carga o de validación.
- Usa la ayuda contextual y el error común existentes en la TUI para no crear patrones nuevos.
- Al guardar: se recortan espacios, se valida que no estén vacíos, que sean absolutos o empiecen por `~` y que, si la ruta existe, sea un directorio. No se crean directorios. Si la validación falla, la pantalla permanece abierta con el error y no se escribe.
- Si `save_config` falla, la pantalla muestra el error y el archivo anterior queda intacto.

`MenuScreen` añade la opción Configuración después de Actualizar, sin tecla directa. Se selecciona con flechas y `Enter`. El menú mantiene `c`, `d`, `u` y `q`; la opción nueva no intercepta ninguna tecla.

### 4. Arranque de la TUI y recuperación de configuración inválida

La carga ocurre una sola vez en el punto de entrada de la TUI. `run_app` acepta `config_result: ConfigLoadResult | None`; si es `None`, llama a `load_config()`. `KaminoApp` guarda `config` y, si corresponde, `config_error`.

- Sin error: se abre el menú (modo menú) o la pantalla inicial del flujo directo.
- Con error: se abre `ConfigScreen` en lugar del menú o de la pantalla inicial. El error y los valores parciales ya vienen en `ConfigLoadResult`.
- Al guardar correctamente desde una TUI abierta por error:
  - si el modo es menú, se vuelve al menú;
  - si es un flujo directo, se reconstruye su estado con el nuevo `app.config` y se abre la primera pantalla del flujo solicitado.
- Si se cancela desde la pantalla abierta por error, la TUI sale con código 1 y no escribe el archivo. No se continúa con defaults silenciosamente.

Para reconstruir un flujo directo, `run_clone_flow`, `run_deploy_flow` y `run_update_flow` reciben la función o pantalla que debe abrirse después del guardado. En clone y deploy, esa función vuelve a construir el estado con `app.config.catalog_dir`, `app.config.clone_output_dir` y `app.config.snapshots_dir`; así el flujo continúa con los valores corregidos y no con los defaults usados para dibujar la pantalla de error.

### 5. Integración con los flujos de la TUI

- `CloneState` añade `catalog_dir` y `default_output_dir`. `_catalog(state)` usa `catalog_dir`; C2 propone `default_output_dir / default_name(now)`.
- `DeployState` añade `catalog_dir`; `snapshots_dir` ya existe. `_prepare_session` los pasa a `prepare_deploy`.
- `start_clone_from_menu` y `start_deploy_from_menu` usan `app.config` en lugar de valores fijos.
- Tras guardar, `KaminoApp.config` se actualiza antes de volver al menú o reconstruir el flujo directo, por lo que los siguientes flujos de la sesión usan los valores nuevos.

### 6. Defaults y creación de directorios al usarlos

- `clone` sin `--output` usa `~/.config/kamino/vault/clone-yyyyMMdd.tar.gz`.
- `snapshot` usa `~/.config/kamino/vault/snapshot/` por defecto.
- `catalog/loader.py` usa `~/.config/kamino/apps` por defecto.
- `write_archive` ya crea el directorio padre; por eso el clon y el holocrón crean su directorio al escribir. No se crea nada en `--dry-run`, al guardar la configuración ni al abrir C2.
- En C2, `validate_destination` debe aceptar un padre ausente cuando la ruta propuesta es el default o el `clone.output_dir` configurado. Una ruta escrita manualmente sigue exigiendo un directorio existente. La creación efectiva ocurre al iniciar la escritura del clon.
- Un `catalog.dir` inexistente no se crea: equivale a un catálogo de usuario vacío, como hoy.

### 7. CLI no interactiva sin configuración

`cli.py` no importa `config.py`. `commands.py` sigue cargando el catálogo con `defaults.catalog_dir` y pasando los defaults de holocrones. `clone.py` calcula el destino con `defaults.vault_dir`. La CLI no crea, lee ni modifica `config.toml`; un archivo inválido o con rutas distintas no altera sus resultados.

Esto implica una divergencia deliberada: la CLI no aplica `catalog.dir`, `clone.output_dir` ni `deploy.snapshots_dir` configurados en la TUI. Se documenta en el README y se cubre con pruebas de CLI que crean un archivo distinto y verifican que los comandos usan los defaults.

### 8. Seguridad, migración y compatibilidad

- El esquema no admite `force`, `yes`, `dry_run` ni políticas de conflicto; son claves desconocidas.
- El guardado de configuración solo modifica `config.toml`; no toca clones, holocrones ni configuraciones de apps.
- Los holocrones existentes en `~/Library/Application Support/kamino/snapshots/` no se mueven ni se copian. Siguen siendo desplegables indicando su ruta.
- El cambio de destino por defecto del clon es observable para scripts que esperaban el directorio actual. Se marca como **BREAKING** en el proposal; `--output` sigue permitiendo la ruta anterior.
- El formato del clon y el manifiesto no cambian.

## Risks / Trade-offs

- **[CLI y TUI pueden mostrar valores distintos]** → es una decisión de producto; se documenta y se prueba que la CLI ignora el archivo. La TUI muestra sus valores efectivos en la pantalla de Configuración.
- **[El cambio del destino por defecto rompe scripts que usaban el directorio actual]** → se marca **BREAKING** y se documenta `--output` como vía de compatibilidad.
- **[Los holocrones antiguos quedan fuera de la nueva ruta]** → no se migran; se documenta que se despliegan indicando su ruta. No se pierde ningún archivo.
- **[La configuración inválida deja la TUI en una pantalla de corrección]** → se muestran valores legibles y defaults, no se sobrescribe hasta guardar y cancelar sale con código 1. Se prueba con TOML inválido y con una ruta relativa.
- **[Crear directorios configurados al usarlos puede sorprender]** → solo ocurre al iniciar una escritura real, nunca en simulación, al guardar ni al abrir pantallas.
- **[La pantalla de configuración añade una quinta opción al menú]** → se mantienen los atajos existentes y la opción se recorre con flechas y `Enter`; se actualizan textos y pruebas de navegación.
- **[Cambios de firma en estados y flujos de la TUI]** → los parámetros nuevos son opcionales o se construyen desde `app.config`; las pruebas existentes se adaptan al nuevo destino por defecto.

## Migration Plan

No hay migración de configuración porque el archivo es nuevo. Los valores por defecto cambian: los clones nuevos se escriben en `~/.config/kamino/vault/` y los holocrones nuevos en `~/.config/kamino/vault/snapshot/`. Los holocrones anteriores permanecen en su ruta y se pueden desplegar explícitamente. Para volver al comportamiento anterior, se usa `--output` o se revierte el cambio de código; el archivo de configuración se puede eliminar sin afectar a clones ni holocrones.

## Open Questions

Ninguna. Las decisiones de alcance y de interacción quedan cerradas en las specs y en este diseño.
