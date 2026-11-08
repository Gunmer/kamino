# Tasks

## 1. Modelo de filas y composición

- [x] 1.1 Añadir `RowKind` y `HunkRow` en `src/kamino/conflicts.py`, sustituir las líneas planas de `Hunk` por `rows` y retirar `local_lines`, `backup_lines`, `context_before`, `context_after` y `decision`. Verificar con `uv run python -c "from kamino.conflicts import RowKind, HunkRow, Hunk; print(RowKind.EQUAL)"`.
- [x] 1.2 Implementar `_rows_for_opcode` y reescribir `build_hunks` para producir filas `EQUAL`, `REPLACE`, `LOCAL_ONLY` y `BACKUP_ONLY`, emparejando por índice y dejando las sobrantes de un solo lado. Añadir pruebas de filas, contexto, inserción y eliminación en `tests/test_conflicts.py` y verificar con `uv run pytest tests/test_conflicts.py -k "hunks or rows or pairing"`.
- [x] 1.3 Renombrar `HunkDecision` a `RowDecision`, cambiar `FileDecision.hunks` por `FileDecision.rows`, añadir `row_count(conflict)` y adaptar `review_all` para construir una decisión por fila. Actualizar las pruebas de decisiones y verificar con `uv run pytest tests/test_conflicts.py -k "decisions or review_all"`.
- [x] 1.4 Reescribir `compose` para consumir una decisión por fila, conservar los finales locales, adaptar las líneas del respaldo al EOL dominante y añadir la salvaguarda de salto final; retirar `_map_grouped_opcodes`. Añadir pruebas de todo-respaldo, todo-local, mezcla dentro de un bloque, CRLF y salto final en `tests/test_conflicts.py` y verificar con `uv run pytest tests/test_conflicts.py -k compose`.

## 2. Integración con la restauración

- [x] 2.1 Actualizar `_resolve_review` en `src/kamino/restore.py` para validar `len(decision.rows) == row_count(conflict)`, componer con `decision.rows` y clasificar con `RowDecision`; ajustar los mensajes de error a “revisión por líneas”. Verificar con `uv run pytest tests/test_restore.py -k review`.
- [x] 2.2 Actualizar `tests/test_restore.py` y `tests/test_integration.py` para construir decisiones por fila, cubrir una mezcla dentro de un mismo bloque, una longitud incorrecta y las rutas todo-local y todo-respaldo. Verificar con `uv run pytest tests/test_restore.py tests/test_integration.py`.

## 3. Render de la TUI

- [x] 3.1 Sustituir `hunk_text` por helpers de render de cabecera, contexto y fila (par o un solo lado), con el lado elegido brillante y etiquetado `[local]`/`[respaldo]`/`[omitir]`, el otro atenuado, `rstrip("\r\n")` y `local -`/`respaldo -` cuando falte un lado. Añadir pruebas directas de render en `tests/test_tui.py` y verificar con `uv run pytest tests/test_tui.py -k "render or row"`.
- [x] 3.2 Actualizar `conflict_detail` para mostrar bloques y líneas y ajustar el conteo de la decisión de revisión a filas del respaldo. Verificar con `uv run pytest tests/test_tui.py -k "detail or decision"`.

## 4. Pantalla de detalle

- [x] 4.1 Construir una sola vez el `ListView` plano con cabeceras y contexto como `ListItem(disabled=True)` y cada fila decidible como ítem habilitado; guardar los mapas ítem→fila, fila→bloque y primera fila de cada bloque, y fijar el cursor inicial en la primera fila decidible. Verificar con `uv run pytest tests/test_tui.py -k detail`.
- [x] 4.2 Implementar el toggle por fila con actualización en sitio, las acciones `A`/`L` globales, `Shift+A`/`Shift+L` por bloque, los saltos `n`/`p`, los botones por bloque, la guarda de clics en ítems deshabilitados y la ayuda actualizada. Verificar con `uv run pytest tests/test_tui.py -k "detail or actions or block"`.
- [x] 4.3 Ajustar `_initial_decisions` para reutilizar una decisión solo si cubre `row_count(conflict)` y actualizar el texto de la tabla para las decisiones de revisión. Verificar con `uv run pytest tests/test_tui.py -k "detail or table"`.
- [x] 4.4 Ampliar `tests/test_tui.py` para cubrir el cursor que salta cabeceras y contexto, el toggle de una fila, las acciones globales y por bloque, los saltos de bloque, la reapertura con decisiones y la mezcla inválida. Verificar con `uv run pytest tests/test_tui.py`.

## 5. Contrato no interactivo

- [x] 5.1 Confirmar que `--yes` sigue aplicando el respaldo completo y que `--dry-run` sigue volcando el diff unificado sin abrir la TUI; ajustar `tests/test_ui.py` o `tests/test_cli.py` solo si la refactorización cambia la construcción de conflictos. Verificar con `uv run pytest tests/test_ui.py tests/test_cli.py`.

## 6. Documentación

- [x] 6.1 Actualizar la sección de conflictos de `README.md`, `docs/Diseño funcional.md` y `docs/Diseño técnico.md` para describir la decisión por línea, las acciones por bloque, la composición mixta y el salto final. Verificar con `rg -n "decisi[oó]n por bloque|sin mezclar l[ií]neas|bloque a bloque" README.md docs` y revisar que no queden afirmaciones obsoletas.
- [x] 6.2 Revisar que el delta de `restore-conflicts` conserva los renombrados y los escenarios existentes, y verificar con `openspec validate "restore-line-level-decisions" --strict`.

## 7. Verificación final

- [x] 7.1 Ejecutar las comprobaciones de CI en orden: `uv run ruff check .`, `uv run ruff format --check .` y `uv run pytest`; las tres deben terminar sin errores.

## Workflow follow-up

- Revisar el cambio antes de implementarlo.
- Archivar el cambio cuando la implementación y la revisión estén completas.
- Verificar el spec archivado: los requisitos renombrados deben aparecer una sola vez, sin restos de “Decisión por bloque de cambio” ni “Composición del archivo por bloques”.
