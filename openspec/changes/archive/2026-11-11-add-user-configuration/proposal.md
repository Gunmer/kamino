# Proposal

## Why

Hoy las preferencias estables dependen de valores fijados en el código o de opciones repetidas en cada invocación. No existe un lugar único para que la persona usuaria ajuste el catálogo, el destino de los clones y el directorio de los holocrones desde la interfaz interactiva, y los valores por defecto actuales no agrupan esos datos bajo `~/.config/kamino`.

Centralizar la configuración en `~/.config/kamino/config.toml`, gestionada desde la TUI, permite ajustar las rutas sin editar archivos a mano y mantiene una separación clara: la TUI es la única vía que lee y escribe la configuración; la CLI conserva sus valores integrados.

## What Changes

- Añadir un archivo opcional `~/.config/kamino/config.toml` en formato TOML con tres claves:
  - `[catalog] dir`: directorio de definiciones de aplicaciones del usuario. Por defecto, `~/.config/kamino/apps`.
  - `[clone] output_dir`: directorio del clon cuando no se indica otro destino. Por defecto, `~/.config/kamino/vault/`.
  - `[deploy] snapshots_dir`: directorio de los holocrones. Por defecto, `~/.config/kamino/vault/snapshot/`.
- **BREAKING**: el destino por defecto de `kamino clone` deja de ser el directorio actual y pasa a ser `~/.config/kamino/vault/`. El destino por defecto de los holocrones deja de ser `~/Library/Application Support/kamino/snapshots/` y pasa a ser `~/.config/kamino/vault/snapshot/`.
- La TUI DEBE leer el archivo al arrancar y usar sus valores en todos sus flujos. La vía no interactiva de la CLI NO DEBE leer, crear, modificar ni guardar la configuración; sigue usando los valores integrados.
- Añadir a la TUI una pantalla de Configuración, accesible desde el menú principal después de Actualizar y sin atajo de teclado, que muestre y permita editar las tres rutas.
- La TUI DEBE crear el archivo solo al guardar desde esa pantalla. Un archivo ausente equivale a los valores por defecto y no se crea al arrancar ni al abrir la pantalla.
- Al guardar, la TUI DEBE escribir siempre las tres claves, de forma atómica, y aplicar los nuevos valores de inmediato a los flujos de la sesión actual.
- Si el archivo es inválido al arrancar la TUI, la pantalla de Configuración DEBE abrirse con el error y con los valores que se hayan podido leer; los campos no legibles muestran el valor por defecto. El archivo NO DEBE sobrescribirse hasta que se guarde.
- Crear automáticamente los directorios de salida y de holocrones cuando se usan, no al guardar la configuración. El catálogo ausente sigue equivaliendo a un catálogo de usuario vacío.
- Validar el archivo de forma estricta: TOML inválido, claves desconocidas, tipos incorrectos, valores vacíos o rutas relativas producen un error claro en la TUI. La CLI no se ve afectada por errores de configuración.
- Mantener fuera de la configuración `--force`, `--yes`, `--dry-run` y las políticas de conflicto. Tampoco se configura `ui.plain` ni `ui.theme`.
- Documentar el archivo, sus claves, los valores por defecto, la pantalla de configuración y la separación CLI/TUI en el README.

## Capabilities

### New Capabilities

- `user-configuration`: archivo TOML de la TUI, validación, valores por defecto, pantalla de configuración, guardado atómico y aplicación inmediata.

### Modified Capabilities

- `cli-options`: el destino por defecto del clon en la vía no interactiva pasa a `~/.config/kamino/vault/`.
- `clone-tui`: C2 propone el destino en `~/.config/kamino/vault/` o en `[clone] output_dir` cuando la TUI tenga configuración.
- `tui-navigation`: el menú principal incorpora Configuración después de Actualizar, sin atajo de teclado.

## Impact

- Código: nuevos `src/kamino/defaults.py`, `src/kamino/config.py` y `src/kamino/tui/config.py`; cambios en `catalog/loader.py`, `clone.py`, `snapshot.py`, `commands.py`, `deploy.py`, `tui/app.py`, `tui/menu.py`, `tui/clone.py`, `tui/deploy.py`, `texts.py` y `tui/styles.tcss`.
- Pruebas: carga y guardado del archivo, validación, pantalla de configuración, valores por defecto, creación de directorios al usarlos, separación CLI/TUI y navegación del menú.
- Documentación: README, textos visibles de la TUI y las secciones afectadas de `docs/Diseño funcional.md` y `docs/Diseño técnico.md`.
- Migración: los holocrones existentes en `~/Library/Application Support/kamino/snapshots/` no se mueven ni se copian; siguen pudiendo desplegarse indicando su ruta. No se modifica el formato del clon ni el manifiesto.
- Dependencias: ninguna nueva; se usa `tomllib` de la biblioteca estándar y la escritura atómica existente del proyecto.
