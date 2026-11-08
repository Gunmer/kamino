# Tasks

## 1. Dependencia y esqueleto

- [x] 1.1 Añadir `textual` a `pyproject.toml` y actualizar `uv.lock`; verificar con `uv sync --all-extras` y `uv run python -c "import textual; print(textual.__version__)"`.
- [x] 1.2 Crear `src/kamino/conflicts.py` con los modelos `Conflict`, `Hunk`, `FileDecision` y `ConflictDecisions`, y las firmas de `build_conflict`, `compose` y `validate_merge`; verificar con `uv run python -c "from kamino.conflicts import Conflict"` y `uv run ruff check src/kamino/conflicts.py`.
- [x] 1.3 Crear el paquete `src/kamino/tui/` (`__init__.py`, `app.py`, `screens.py`, `widgets.py` y `styles.tcss`) con una app mínima; verificar que la app se importa con `uv run python -c "from kamino.tui.app import KaminoRestoreApp"`.
- [x] 1.4 Añadir una prueba de humo de la TUI a `tests/test_tui.py` que abra y cierre la app con `run_test`; verificar `uv run pytest tests/test_tui.py` sin TTY.

## 2. Diff, composición y validación

- [x] 2.1 Implementar `build_conflict` para texto, binario, grande y enlace con enlace; verificar con pruebas unitarias de cada tipo y de los casos que no son conflicto (nuevo, idéntico, solo modo y cambio de tipo).
- [x] 2.2 Implementar la generación de bloques con `difflib`, contexto de 3 líneas, normalización CRLF y guard de 20 000 líneas; verificar con pruebas de cambios, añadidos, borrados, diferencia solo de CRLF y archivos grandes.
- [x] 2.3 Implementar `compose` con decisiones por bloque, estilo de final de línea local y último salto de línea; verificar con pruebas de mezcla, todos-respaldo, todos-local y CRLF.
- [x] 2.4 Implementar `validate_merge` para TOML, JSON, INI y plist, con detección de claves duplicadas y la regla de bloquear solo si ambos originales parsean; verificar con pruebas de mezcla válida, inválida, original inválido y formato no validado.
- [x] 2.5 Documentar en docstrings (inglés) el modelo de conflicto y las reglas de diff, composición y validación; verificar con `uv run ruff check src/kamino/conflicts.py` y `uv run python -c "import kamino.conflicts"`.

## 3. Sesión de restauración

- [x] 3.1 Añadir `prepare_restore` y `RestoreSession` en `src/kamino/restore.py` con manifiesto, catálogo, apps restaurables y cierre del staging; conservar `run_restore` como envoltorio no interactivo; verificar que `uv run pytest tests/test_restore.py` sigue pasando.
- [x] 3.2 Implementar `session.plan(selected)` con clasificación, verificación del archivo, extracción a temporal y construcción de conflictos; implementar `plan(dry_run=True)` leyendo el contenido del archivo sin verificar ni extraer; verificar con pruebas de plan y dry-run.
- [x] 3.3 Implementar `session.apply(decisions)` con cobertura obligatoria de conflictos, re-validación de mezclas, re-verificación de destinos, snapshot y escritura atómica; verificar con pruebas de aplicar, mezclar, omitir y snapshot.
- [x] 3.4 Añadir `merged` a `RestoreResult`, `conflicts` a `RestorePlan` y los ayudantes de decisiones `apply_all` y `omit_all`; verificar que el resumen clasifica restaurado, mezclado y omitido.
- [x] 3.5 Implementar la detección de cambios concurrentes (hash de archivo regular, destino de enlace y aparición de un archivo nuevo) y abortar antes de escribir; verificar con una prueba que cambia el destino entre `plan` y `apply` y comprueba que no hay snapshot ni escrituras.
- [x] 3.6 Actualizar `tests/test_restore.py` para la nueva API y añadir casos de conflicto, modo, no-escritura, snapshot y undo; verificar `uv run pytest tests/test_restore.py`.
- [x] 3.7 Documentar el orden `verificar -> extraer -> decidir -> snapshot -> aplicar` en `docs/Diseño técnico.md` (§5.2 y §6); verificar que el documento lo describe sin contradicciones.

## 4. CLI no interactiva y dry-run

- [x] 4.1 Añadir `--apps/-a` a `restore`, validar los ids contra las apps restaurables y avisar de los desconocidos o no instalados; verificar con pruebas de CLI de `--apps` y `-a`.
- [x] 4.2 Implementar el despacho no interactivo (`--yes`, sin TTY, error sin `--yes`) y las decisiones de aplicar todo; verificar que `--yes` sobrescribe conflictos y que sin TTY y sin `--yes` se sale con código 1.
- [x] 4.3 Implementar la salida de `--dry-run` con diffs a stdout y notas para binarios, grandes y enlaces; verificar que no abre TUI, no verifica, no extrae, no escribe y sale con el código esperado.
- [x] 4.4 Añadir la categoría “mezclados” a `ui.print_restore_summary` y conservar el detalle de `--verbose`; verificar con pruebas de `tests/test_ui.py`.
- [x] 4.5 Actualizar `README.md` y `docs/Diseño funcional.md` §3.7 con `--apps`, el comportamiento de `--yes`, los diffs de `--dry-run` y el resumen; verificar que `uv run kamino restore --help` muestra las opciones documentadas.
- [x] 4.6 Actualizar `tests/test_cli.py` y `tests/test_ui.py` para los nuevos caminos; verificar `uv run pytest tests/test_cli.py tests/test_ui.py`.

## 5. TUI

- [x] 5.1 Implementar la pantalla de selección de apps (ninguna marcada, seleccionar todas/ninguna, continuar/cancelar) y verificarla con `Pilot`.
- [x] 5.2 Implementar la tabla de conflictos (solo conflictos, conteos, acciones por archivo, aplicar todo/omitir todo y confirmación bloqueada) y verificarla con `Pilot`.
- [x] 5.3 Implementar el detalle por bloques (inicial en respaldo, alternar, atajos `A`/`L`, error de validación y volver) y verificarlo con `Pilot`.
- [x] 5.4 Implementar la confirmación final y el resumen (conteos, directorio y ruta del snapshot, aplicar/volver/cancelar) y verificarlos con `Pilot`.
- [x] 5.5 Conectar `run_restore_tui(session, selected)` con el despacho TTY de la CLI y manejar cancelación y cambio concurrente; verificar un flujo completo con HOME temporal y un respaldo de prueba.
- [x] 5.6 Añadir la ayuda en español, los atajos y el aviso de secretos en los diffs; verificar que la ayuda se abre y que los textos visibles están en español.
- [x] 5.7 Documentar las pantallas, atajos, dependencia de `textual` y pruebas headless en `docs/Diseño técnico.md` §5.2 y §7; verificar que el documento coincide con la TUI implementada.
- [x] 5.8 Añadir las pruebas de la TUI a `tests/test_tui.py` y verificar `uv run pytest tests/test_tui.py` en CI sin TTY.

## 6. Integración, empaquetado y comprobaciones

- [x] 6.1 Añadir pruebas de integración de ciclo completo con conflicto, mezcla, snapshot y undo, `--yes`, `--dry-run` y compatibilidad con respaldos antiguos; verificar `uv run pytest tests/test_integration.py`.
- [x] 6.2 Actualizar `scripts/build_release.py` para incluir los datos de `textual` en PyInstaller y verificar el build con `uv run python scripts/build_release.py` en macOS ARM64 (o documentar el comando equivalente si el entorno no es ARM64).
- [x] 6.3 Ejecutar las comprobaciones de CI en orden: `uv run ruff check .`, `uv run ruff format --check .` y `uv run pytest`; corregir cualquier fallo.
- [x] 6.4 Verificar que el cambio OpenSpec valida con `openspec validate "restauracion-conflictos-por-bloque" --strict`.

## Workflow follow-up

- Archivar el cambio después de que la revisión del proyecto esté satisfecha.
- Verificar el resultado archivado y que las specs principales reflejan `restore-conflicts` y `cli-options`.
