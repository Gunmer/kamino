# Tasks

## 1. Estado y descubrimiento de clones

- [x] 1.1 Añadir `clone_output_dir: Path | None` a `DeployState` y rellenarlo en `start_deploy_from_menu`, `build_state` y el estado que reconstruye `after_config`; verificar con las pruebas de menú y configuración de `tests/test_tui.py`.
- [x] 1.2 Implementar el descubrimiento de candidatos desde `state.clone_output_dir` y `Path.cwd()`: solo `.tar.gz` hijos directos, deduplicación por ruta resuelta, origen, tamaño y `mtime`, tolerancia a rutas ausentes o ilegibles; verificar con pruebas de ambas fuentes, deduplicación, carpeta compartida, orden por fecha y ruta ausente.
- [x] 1.3 Verificar que `clone_output_dir=None` lista solo el directorio actual y que el flujo directo sigue funcionando sin leer el HOME real; ejecutar las pruebas de D0 y de `run_deploy_flow` afectadas.

## 2. Lista, foco y preview de D0

- [x] 2.1 Rehacer `DeployFileScreen` para que la tabla sea la fuente de verdad: columnas `Archivo`, `Origen` y `Tamaño`, fila final `Otro archivo...`, foco inicial en la tabla y clon más reciente seleccionado; verificar con pruebas de filas, flechas y `Enter`.
- [x] 2.2 Implementar el preview de la fila resaltada con ruta completa abreviada con `~`, fecha `%Y-%m-%d %H:%M` y tamaño, y los estados vacíos por fuente; verificar con pruebas del contenido de `#preview` y `#status`.
- [x] 2.3 Actualizar el título, los textos de origen, los atajos y `HELP_D0` para el estado de tabla; verificar que las pruebas de textos y ayuda pasan y que la pantalla se mantiene usable a 80x24.
- [x] 2.4 Ajustar las pruebas existentes de D0 por la fila adicional y el nuevo foco, y añadir casos de ambas fuentes, orden, deduplicación y ruta ausente; ejecutar `uv run pytest tests/test_tui.py`.

## 3. Filtro y ruta manual

- [x] 3.1 Añadir el filtro `/` con campo oculto, coincidencia por nombre u origen y `Esc` que limpia y devuelve el foco a la tabla, manteniendo `Otro archivo...` visible sin coincidencias; verificar con pruebas de filtro y de lista vacía filtrada.
- [x] 3.2 Implementar la ruta manual: `o` salta a la última fila, `Enter` revela y enfoca el campo, `Esc` vuelve a la tabla, una ruta válida avanza y una inválida muestra el error sin perder el valor; verificar con pruebas de cada caso.
- [x] 3.3 Conservar el flujo directo con archivo inválido: campo precargado, error visible y lista disponible; verificar con una prueba de `DeployFileScreen` abierta con `file_error` y con la prueba existente de `run_deploy_flow`.
- [x] 3.4 Actualizar la leyenda de atajos y la ayuda para los estados de filtro y ruta manual; verificar que `HELP_D0` describe las teclas y que las pruebas de ayuda pasan.

## 4. Conservación de la selección

- [x] 4.1 Preseleccionar `state.clone_file` al montar o reanudar D0 cuando coincide con un candidato, y abrir el campo de ruta precargado cuando es una ruta manual fuera de la lista; verificar con pruebas de ida y vuelta desde D1.
- [x] 4.2 Conservar en `#status` el error de D1 al volver a D0 y mantener la selección del clon; verificar con una prueba de error que devuelve a D0.
- [x] 4.3 Añadir una prueba de que la tabla recupera el foco y la fila elegida al volver desde D1; ejecutar `uv run pytest tests/test_tui.py`.

## 5. Documentación

- [x] 5.1 Actualizar la sección de despliegue del `README.md` para describir el selector de D0, las dos fuentes y la ruta manual; verificar que los comandos documentados siguen coincidiendo con la CLI actual.
- [x] 5.2 Actualizar las secciones de D0 de `docs/Diseño funcional.md` y `docs/Diseño técnico.md` con el orden por fecha, la deduplicación, el filtro y la ruta manual; verificar que no contradicen el delta de `deploy-conflicts`.

## 6. Verificación integrada

- [x] 6.1 Ejecutar `uv run ruff check .` y `uv run ruff format --check .`; corregir lo que aparezca y verificar que ambos terminan sin errores.
- [x] 6.2 Ejecutar `uv run pytest`; verificar que toda la suite pasa, incluidas las pruebas nuevas de D0.
- [x] 6.3 Probar manualmente la pantalla a 80x24 con clones en ambas rutas: flechas, `Enter`, `/`, `o`, ruta manual inválida y retorno desde D1; verificar que no hay recortes ni pérdida de foco.
- [x] 6.4 Ejecutar `openspec validate d0-clone-picker --strict`; verificar que el cambio es válido antes de archivar.

## Workflow follow-up

- Revisar y aprobar el proposal, el delta de spec y el diseño antes de implementar.
- Implementar con `/opsx-apply` cuando las tareas estén acordadas.
- Archivar con `/opsx-archive` después de completar la verificación.
