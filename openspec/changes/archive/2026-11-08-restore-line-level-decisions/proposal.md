# Proposal

## Why

La revisión de conflictos de texto decide bloques completos. Cuando un bloque contiene dos cambios separados, la persona no puede aceptar uno y conservar el otro: o restaura todo el bloque desde el respaldo, o conserva todo el bloque local. Eso obliga a elegir entre perder cambios locales y renunciar a líneas del respaldo aunque solo una línea sea la que importa.

## What Changes

- La unidad de decisión pasa de bloque a línea: cada línea cambiada se presenta como una fila con la versión local y la del respaldo. En un reemplazo se emparejan por índice; las líneas sobrantes de un lado se muestran como filas de inserción o eliminación.
- La vista de detalle muestra el contexto una sola vez y alinea cada par dentro del bloque. La cabecera del bloque y las líneas de contexto no son decidibles.
- Cada fila empieza en el respaldo y se alterna con el local. Se mantiene la regla de que no existe un estado sin decidir.
- Se conservan las acciones globales `A` (todo respaldo) y `L` (todo local), y se añaden `Shift+A`/`Shift+L` para el bloque actual y `n`/`p` para saltar entre bloques.
- La composición permite mezclar líneas dentro de un mismo bloque. Todo respaldo sigue produciendo exactamente el respaldo y todo local exactamente el local.
- Se corrige el salto de línea final en composiciones mixtas: una línea sin salto final solo puede quedar al final; si queda en medio, se le añade el EOL dominante local.
- El detalle de la tabla muestra el número de bloques y de líneas, y la decisión de revisión cuenta líneas del respaldo.
- **BREAKING** (solo TUI): la revisión por bloques se reemplaza por la revisión por líneas. No hay decisiones persistidas que migrar; el formato del archivo de respaldo, `--yes`, `--dry-run`, el snapshot y la validación no cambian.
- Los conflictos binarios, los textos de más de 5 MB y los enlaces con enlace siguen resolviéndose como archivo entero.

## Capabilities

### New Capabilities

<!-- Ninguna. El comportamiento encaja en la capability existente. -->

### Modified Capabilities

- `restore-conflicts`: la decisión por bloque pasa a decisión por línea dentro del bloque, la composición admite mezclar líneas del mismo bloque y se ajustan los escenarios de final de línea y salto final.

## Impact

- `src/kamino/conflicts.py`: modelo de filas, decisión por línea, composición y `review_all`.
- `src/kamino/restore.py`: validación de la longitud de la decisión y clasificación restaurado/mezclado.
- `src/kamino/tui/widgets.py`: render alineado por par, conteo de líneas y etiquetas de decisión.
- `src/kamino/tui/screens.py`: cursor por fila, cabeceras y contexto deshabilitados, acciones globales y por bloque.
- `tests/test_conflicts.py`, `tests/test_restore.py`, `tests/test_integration.py`, `tests/test_tui.py` y `tests/test_ui.py`.
- `openspec/specs/restore-conflicts/spec.md`, `README.md`, `docs/Diseño funcional.md`, `docs/Diseño técnico.md` y la ayuda de la TUI.
- Sin dependencias nuevas ni cambios en el formato del archivo de respaldo.
