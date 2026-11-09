# Design

## Context

Ver `proposal.md` para la motivación. El estado actual relevante está en `src/kamino/cli.py` (comandos Typer y menú con `questionary`), `src/kamino/tui/` (solo revisión de despliegue), `src/kamino/clone.py`, `deploy.py`, `archive.py`, `plan.py`, `conflicts.py`, `snapshot.py` y `update.py`. Las specs vivas afectadas son `cli-options`, `clone-safety`, `deploy-conflicts`, `self-update` y `tui-ansi-theme`; se añaden `tui-navigation` y `clone-tui`.

Restricciones que se mantienen: el formato del archivo no cambia; el id sintético `snapshot` se conserva; el núcleo no imprime ni pregunta; el orden de seguridad es verificar -> extraer -> decidir -> holocrón -> aplicar; `dry_run` no escribe; los códigos 0/1/2/3 se conservan; Python 3.11+; `textual==8.2.8` en el lock actual.

## Goals / Non-Goals

**Goals:**

- Una sola app interactiva con menú, flujos y transversales, y una vía CLI equivalente.
- Progreso y cancelación reales en clonar, desplegar y actualizar, con puntos de no retorno claros.
- Un modelo de clasificación de despliegue que separe desplegado, híbrido, omitido, sin cambios y no instalado.
- Vocabulario visible coherente sin romper nombres internos ni el formato.
- Pruebas headless a 80x24 y menores, sin leer el HOME real.

**Non-Goals:**

- Cambiar el formato del archivo, el id `snapshot` o la compatibilidad con clones antiguos.
- Implementar detección de apps abiertas, cifrado, `undo`, perfiles o programación.
- Renombrar el comando `update`.
- Rediseñar el algoritmo de diff o la validación de mezclas más allá de la presentación y el rendimiento.

## Decisions

### 1. Una app, dos modos y navegación por pila

`KaminoApp` sustituye a `KaminoDeployApp` y arranca en modo `menu` o `direct`. En modo `menu`, `MenuScreen` es la raíz y cada flujo termina volviendo a ella; en modo `direct`, el flujo termina con `App.exit(código)` y no se muestra el menú. El estado de sesión (`SessionState`) guarda la frase elegida, `last_mission`, `last_code` y el modo de arranque. Cada flujo tiene su propio estado (`CloneState`, `DeployState`, `UpdateState`).

La navegación usa `push_screen`/`pop_screen`: la pantalla inferior permanece montada y conserva su estado. Los resultados de diálogos se recogen con callbacks; las pantallas no se reconstruyen al volver. `Atrás` desde C2 a C1 conserva la selección de apps; `Atrás` desde D2 a D1 pide confirmación y descarta decisiones; `Atrás` desde D3 o D4 a D2 conserva decisiones.

**Alternativa descartada:** una app por flujo. Evita el estado compartido, pero duplica el menú, la ayuda, el error y la política de salida, y no permite volver al menú sin reiniciar.

### 2. Flujos fuera de Typer y códigos de salida

`cli.py` sigue siendo una capa fina de argumentos y salida. Los comandos dejan de usar `ctx.invoke` y `typer.Exit` dentro de la lógica: un módulo de flujos/commands devuelve resultados tipados y un código, y `cli.py` lo traduce a `typer.Exit`. El menú llama a los mismos servicios, sin pasar por Typer. Así se cumple “lanzado desde el menú vuelve al menú” y “lanzado directamente no vuelve”.

La selección de TUI se decide en la capa CLI: `clone` abre TUI si hay TTY, no está `KAMINO_PLAIN` y no se pidió una salida puramente no interactiva; `deploy` abre TUI solo en el flujo con decisiones (sin `--yes` ni `--dry-run`); `update` abre U1 si hay TTY y no está `KAMINO_PLAIN`. Sin TTY, o con `KAMINO_PLAIN`, se usa la salida de texto.

**Decisión de compatibilidad:** `deploy --dry-run` y `deploy --yes` conservan la vía no interactiva de la spec `cli-options`; `clone --dry-run` con TTY sí muestra C3 porque la decisión C3 lo exige y no hay contrato previo que lo prohíba. `clone --apps --output` con TTY muestra C3/C4; `KAMINO_PLAIN` es la vía de scripting.

**Alternativa descartada:** hacer que todos los comandos pasen siempre por la TUI. Rompe scripts y la spec de simulación de despliegue.

### 3. Contrato de progreso y cancelación con el núcleo

El núcleo acepta callbacks opcionales, nunca imprime:

- `progress(event)`: fase (`planning`, `verifying`, `extracting`, `classifying`, `writing`, `downloading`, `replacing`), actual, total y ruta opcional.
- `cancelled()`: devuelve `True` si la UI pidió cancelar.

`build_plan`, `verify_archive`, `extract_archive`, `write_archive`, `DeploySession.plan`, `DeploySession.apply` y `run_update` incorporan esos callbacks. La cancelación lanza `OperationCancelled`, una excepción de dominio que no se confunde con un error. Los workers de `textual` son de hilo (`@work(thread=True, exclusive=True)`); la UI lee `get_current_worker().is_cancelled` o el evento del worker y actualiza con `call_from_thread`. `Worker.cancel()` no mata un hilo: todos los bucles largos deben comprobar el callback.

Puntos de no retorno:

- Clonar: `_commit` de `write_archive` (`os.link` o `os.replace`).
- Desplegar: después de crear el holocrón, antes de la primera escritura de destino.
- Actualizar: `os.replace` del binario.

**Alternativa descartada:** workers de proceso para diff/CPU. Añade serialización y complejidad; el límite de 20k líneas y la fase de preparación en worker son suficientes.

### 4. Clonar: contar, planificar y escribir son pasos distintos

Se separa `plan_clone` (plan + manifiesto + tamaño + omitidos con motivo) de `write_clone` (temporal + progreso + cancelación + commit). `run_clone` queda como fachada para CLI y pruebas.

C1 no reutiliza el plan completo porque `build_plan` lee y hashea cada archivo. Se añade un recorrido de conteo que comparte `_expand`/`_walk`, exclusiones y `SafeHome`, pero cuenta entradas en lugar de leer bytes. Ghostty es el caso especial: su descubrimiento necesita leer configuraciones y seguir referencias; el conteo debe incluir esas referencias sin hashear todos los archivos, con una prueba de paridad contra el plan. El recuento corre en un worker y no bloquea C1.

El nombre por defecto pasa a `clone-yyyyMMdd.tar.gz` con fecha local y sin sufijo automático. `run_clone` acepta `force`; sin `force`, `write_archive` conserva el `os.link` atómico que no sobrescribe; con `force`, usa `os.replace` y revalida el destino justo antes del renombrado. La revalidación es best-effort: no existe un compare-and-swap portátil, así que queda una ventana mínima documentada. En la TUI, `--force` omite el diálogo de reemplazo; sin `--force`, el diálogo se muestra.

**Alternativa descartada:** mantener el sufijo automático. La decisión pide un nombre estable y un reemplazo explícito; el sufijo ocultaría el conflicto.

### 5. Desplegar: manifiesto, clasificación y holocrón

D0 endurece `read_manifest`: envuelve `tarfile.TarError`/`OSError` en `ArchiveError`, exige exactamente un `manifest.json`, comprueba que sea un archivo regular y limita su tamaño a 4 MiB (constante, no configuración). El escaneo de miembros tiene un tope para evitar tarros con millones de entradas. No se extrae ni se siguen enlaces. La integridad sigue verificándose completa en D1 antes de extraer.

La clasificación cambia:

- `NEW` -> desplegado nuevo.
- `DIFFERENT` resuelto con el clon -> desplegado que reemplaza.
- `DIFFERENT` resuelto con revisión mixta -> híbrido.
- `DIFFERENT` resuelto conservando el original -> omitido.
- `IDENTICAL` -> sin cambios (clon perfecto), no omitido.
- `UNSUPPORTED` -> omitido.
- Apps del manifiesto no instaladas -> `not_installed`, informadas, no omitidas.
- `--apps` pedidos no cumplidos -> `missing_requested`, omitidos y código 3.

`DeployResult`/`DeployClassification` incorporan esas categorías; `omitted` deja de incluir idénticos y no instalados. El código 3 se calcula solo con `omitted` y `missing_requested`.

El holocrón se crea solo si hay al menos un destino existente entre las escrituras. `create_snapshot` devuelve `None` si el plan de snapshots no tiene operaciones; un despliegue solo-nuevos no crea un archivo vacío. D4 informa del número de archivos protegidos y del directorio; D5 muestra la ruta y la orden de deshacer solo si existe.

`apply` lleva la cuenta de archivos escritos. Si falla una escritura, lanza `PartialApplyError` con aplicados, fallido y holocrón; D5 muestra “Despliegue incompleto” y el código es 1. No hay rollback automático: el holocrón es la vía de recuperación.

**Alternativa descartada:** mantener el snapshot vacío. Es uniforme, pero “Deshacer” apuntaría a un archivo sin contenido y D4 mostraría 0 archivos protegidos sin explicación.

### 6. Desplegar: D2 y D3

D2 conserva la tabla de solo conflictos y las acciones globales, pero cambia los conteos y las etiquetas. `r` revisa, `c` aplica el clon, `o` conserva el original; `C`/`O` actúan sobre todos. Sin conflictos, el flujo pasa directo a D4. `Atrás` desde D2 pide confirmación y descarta decisiones.

D3 muestra **un bloque a la vez** para escalar a 20k líneas sin montar miles de widgets. La cabecera del bloque y el contexto se muestran una vez; cada línea cambiada es una fila con la versión del original y la del clon; `●` marca la que irá al archivo final y `○` la que no. Las líneas de un solo lado se etiquetan “solo clon” o “solo original”. `↑↓` recorre líneas decidibles, `Espacio`/`Enter` alterna, `c`/`o` aplican el bloque, `C`/`O` todo el archivo y `Tab`/`Mayús+Tab` cambian de bloque. El pie fijo muestra el recuento de líneas del clon y del original. `Esc` vuelve y valida.

**Alternativa descartada:** el `ListView` con todos los bloques montados. Es la implementación actual, pero no escala y complica la navegación por bloques; la vista por bloque mantiene los mismos atajos y el contexto único.

### 7. Actualizar: pasos y cancelación

`run_update` se descompone en consulta, descarga/verificación y reemplazo, manteniendo una fachada para CLI. La descarga reporta `(descargado, total)` y comprueba cancelación por chunk; la verificación de tamaño/SHA ocurre antes de `os.replace`. `UpdateResult` distingue actualizado, ya al día e instalada más reciente. Los errores se clasifican (no autocontenido, plataforma, red, integridad, reemplazo) para que U1 elija la acción contextual sin parsear mensajes.

### 8. Textos y vocabulario

Se añade `src/kamino/texts.py` como única fuente de textos de presentación, sin lógica de negocio. Contiene constantes y plantillas de formato para CLI, TUI, ayuda y error. El núcleo sigue lanzando mensajes literales; la pantalla común los envuelve en las cuatro partes. El mapa visible es: local -> original, snapshot -> holocrón de seguridad, mezcla -> híbrido, idéntico -> clon perfecto, no instalada -> sin base en este Mac, cambio concurrente -> perturbación en la Fuerza, integridad fallida -> clon con mutaciones, manifiesto -> registro de los kaminoanos.

No se renombran `LocalFile`, `local_text`, `RowDecision.LOCAL`, `snapshot.py`, `create_snapshot`, `SNAPSHOT_APP_ID` ni `DeployResult.snapshot`: el id `snapshot` es formato de archivo y debe seguir leyendo holocrones antiguos.

**Alternativa descartada:** mover todos los mensajes del núcleo a `texts.py`. Acopla el dominio a la presentación y amplía el cambio; se hará solo si se decide que `str(exc)` también sea temático.

### 9. ANSI, iconos y layout

La app mantiene `theme = "ansi-dark"` y `TEXTUAL_THEME` como override explícito. Las pantallas nuevas usan solo tokens `$...` y nombres ANSI; nada de hex ni truecolor. `NO_COLOR` se lee antes de crear la app. El wordmark usa bloques en cian/azul; las cápsulas y el original usan amarillo con `◉`. Todos los estados llevan icono y palabra.

La codificación se detecta con `sys.stdout.encoding`/locale. Sin UTF-8 o con `KAMINO_NO_LOGO`, se oculta el wordmark. `◉`, `●`, `○` y `↪` tienen equivalentes ASCII (`o`, `*`, `.`, `>`), porque la decisión de logo no cubre todos los glifos.

El layout del menú degrada por altura: 30+ todo; 26-29 cabecera, separador y menú; 24-25 cabecera y menú; menos de 24 aviso. La pantalla de aviso se apila y se retira al redimensionar; el estado inferior no se destruye. `q` se declara solo en `MenuScreen`, así no existe durante una operación. `ctrl+c` se define por pantalla y se ignora cuando el flujo está en no-retorno.

### 10. Widgets y rendimiento

- C1 usa una lista con estado de selección propio y reconstrucción al filtrar; conserva seleccionadas las filas ocultas y `a`/`n` actúan solo sobre las visibles.
- C3 usa una barra de progreso por archivos y actualiza la UI desde el worker.
- D1 muestra las apps instaladas con su número de archivos y una lista informativa separada de no instaladas.
- D3 usa la vista por bloque descrita; el diff se calcula en la fase de preparación, no al abrir el detalle.
- D2 usa `DataTable` con atajos y D4 usa una lista de acciones navegable con flechas y `Enter`, preseleccionada en Retirada y sin atajo directo de aplicación. Todas las pantallas muestran una leyenda de teclas al pie en lugar de una botonera.

### 11. Pruebas

Se mantiene `run_test` headless. Los nuevos tests inyectan `home`, `catalog`, `application_dirs`, `path`, `snapshots_dir`, `cwd`, reloj, selector de frase, codificación y portapapeles. Se prueban 80x24, 79x23, 24-25, 26-29, 30+ y el redimensionado de vuelta. Los workers se prueban con `pilot.pause()`, cancelación y comprobación de que no se escribió nada. `NO_COLOR` se fija antes de crear la app. El portapapeles se envuelve en un adaptador para no depender de `_clipboard`.

### 12. Empaquetado y dependencias

Se elimina `questionary` de `pyproject.toml`, `uv.lock`, `ui.py` y las pruebas de selección. `rich` se mantiene para la CLI. `texts.py` y el logo son módulos Python, así que no añaden datos a PyInstaller; si se mueven a archivos, hay que actualizar `scripts/build_release.py` y `build/kamino-arm64.spec`. El smoke test del binario sigue usando `clone --dry-run` sin TTY.

## Risks / Trade-offs

- **Clasificación y código 3** -> un modelo único y pruebas de todas las combinaciones; el riesgo es cambiar scripts que dependían de 3 por idénticos.
- **Holocrón vacío** -> se omite; el riesgo es que un usuario espere un archivo siempre. D4/D5 lo explican.
- **Cancelación cooperativa** -> si un bucle no comprueba el callback, el worker sigue; pruebas de cancelación por fase y limpieza de temporales.
- **Revalidación de `--force`** -> `os.replace` no es compare-and-swap; la ventana residual se documenta y se minimiza re-stat justo antes.
- **D3 por bloque** -> cambia la vista actual de todos los bloques; se gana escalabilidad y se mantienen los atajos. Si se quiere la vista completa, será otro cambio.
- **C1 vs plan** -> el conteo sin hash puede divergir en Ghostty; prueba de paridad y revisión de referencias.
- **Diff grande** -> `SequenceMatcher` es CPU y el worker de hilo compite por el GIL; el límite de 20k líneas y la fase de preparación acotan el impacto. Medir antes de optimizar.
- **80x24** -> la cabecera, el pie y la leyenda de teclas compiten por filas; pruebas de layout y mensajes compactos.
- **Portapapeles** -> OSC 52 no se puede detectar; aviso permanente y heurística para Terminal.app/tmux.
- **PyInstaller** -> nuevos datos o cambios en `questionary` pueden romper el binario; smoke test en la release.
- **Textos** -> el vocabulario se dispersa; `texts.py` y pruebas de cadenas visibles.

## Migration Plan

1. Núcleo: conteo, plan/write separados, progreso/cancelación, `--force`, manifiesto endurecido, categorías, holocrón opcional, parcial, update por pasos.
2. Textos y vocabulario; actualizar CLI y specs.
3. App shell, menú, navegación, transversales.
4. Clone TUI.
5. Deploy TUI sobre las pantallas actuales.
6. Update TUI.
7. CLI, `KAMINO_PLAIN`, códigos.
8. Pruebas, PyInstaller, documentación.
9. Verificación: `uv run ruff check .`, `uv run ruff format --check .`, `uv run pytest`, smoke del binario y prueba manual 80x24/79x23.

Rollback: revertir los commits del cambio. No hay migración de datos; los clones y holocrones antiguos siguen siendo legibles y el id `snapshot` no cambia.

## Open Questions

Ninguna que cambie las specs, el enfoque o el desglose de tareas. Quedan como propuesta de copy los textos de pantallas no listados, la confirmación de los atajos `c`/`d`/`u`/`q` y el nombre “bloque” frente a “secuencia”; se resolverán al revisar los artefactos, sin alterar la arquitectura.
