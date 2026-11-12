# Proposal

## Why

D0 solo lista los `.tar.gz` del directorio desde el que se lanzó la TUI. Los clones que `kamino clone` guarda en `clone.output_dir` (por defecto `~/.config/kamino/vault/`) no aparecen, así que la persona tiene que escribir la ruta a mano o copiarla. Además, el foco arranca en el campo de texto, la tabla no se recorre con las flechas hasta pulsar `Tab`, el orden alfabético deja el clon más reciente al final y no hay forma de distinguir el origen de cada archivo.

## What Changes

- D0 pasa a ser un selector de clon con una lista plana que combina:
  - los `.tar.gz` de `clone.output_dir`, la carpeta de clones configurada;
  - los `.tar.gz` del directorio actual, donde se lanzó la TUI.
- La lista se ordena por fecha de modificación descendente, muestra el origen de cada fila y deduplica los archivos que aparecen en ambas rutas.
- La última fila es `Otro archivo...`: al elegirla aparece el campo de ruta. `o` salta directo a ella.
- El foco inicial está en la tabla; `Enter` sobre una fila valida el manifiesto y continúa. `/` filtra por nombre u origen.
- Se muestra la ruta completa, la fecha y el tamaño del clon resaltado, y hay estados vacíos explícitos cuando una ruta no existe o no tiene clones.
- `DeployState` incorpora `clone_output_dir`; el menú y el flujo directo lo rellenan desde la configuración.
- Al volver de D1 se preselecciona el clon que ya estaba elegido. El flujo directo con un archivo inválido conserva el campo de ruta precargado y el error visible.
- Se actualizan los textos de ayuda y atajos, la documentación de D0 y las pruebas de la TUI.

Fuera de alcance: `kamino deploy` sin argumento, listar holocrones, leer manifiestos al resaltar y cambiar la verificación de D1.

## Capabilities

### New Capabilities

Ninguna.

### Modified Capabilities

- `deploy-conflicts`: D0 incorpora la selección del clon entre la carpeta configurada, el directorio actual y una ruta manual, con orden por fecha, deduplicación y filtro.

## Impact

- Código: `src/kamino/tui/deploy.py` (D0, `DeployState`, `start_deploy_from_menu`, `build_state`), `src/kamino/texts.py` (títulos, atajos y ayuda) y `tests/test_tui.py`.
- Documentación: `README.md`, `docs/Diseño funcional.md` y `docs/Diseño técnico.md` en las secciones de D0 y despliegue.
- Configuración: se reutiliza `clone.output_dir` de `~/.config/kamino/config.toml`; no cambia el formato ni se añaden claves.
- Compatibilidad: no cambia el formato del clon, el manifiesto, los holocrones ni los códigos de salida. La CLI no cambia.
