# Design

## Context

Ver `proposal.md` para la motivación y `specs/restore-conflicts/spec.md` para el contrato de comportamiento.

La revisión actual vive en `src/kamino/conflicts.py`: `build_hunks` agrupa `SequenceMatcher.get_grouped_opcodes(3)` en `Hunk`, cada uno con las líneas locales y del respaldo completas y una `HunkDecision`. `compose` recalcula el matcher, mapea los opcodes completos a los bloques con `_map_grouped_opcodes` y emite un bloque entero desde un lado. La TUI (`src/kamino/tui/screens.py`) muestra un `ListItem` por bloque, renderiza todas las líneas locales y después todas las del respaldo (`hunk_text`) y reconstruye la lista completa en cada toggle. `src/kamino/restore.py` valida la longitud de las decisiones, compone y clasifica el resultado.

Restricciones que se mantienen:

- El formato del archivo de respaldo no cambia; los respaldos existentes se restauran igual.
- La composición y la validación viven en el núcleo; la TUI solo presenta y recoge decisiones, nunca inyecta bytes.
- Solo biblioteca estándar para diff/composición/validación; Python 3.11; Textual 8.2.8.
- Se conservan los guards de 5 MB y 20 000 líneas y el orden de seguridad (verificar, snapshot, aplicar).
- Las pruebas de TUI usan `App.run_test()` sin TTY.

## Goals / Non-Goals

**Goals:**

- Modelo de decisión por fila (par local/respaldo o línea de un solo lado), determinista y probable sin TUI.
- Composición por líneas que permita mezclar dentro de un bloque y conserve los invariantes de todo-respaldo y todo-local.
- Navegación plana por filas en la TUI, con cabeceras y contexto visibles pero no decidibles.
- Mantener la clasificación restaurado/mezclado/omitido, la validación y el comportamiento no interactivo.

**Non-Goals:**

- Merge semántico por clave, validación de YAML/shell o edición manual del archivo final.
- Persistir decisiones entre ejecuciones o cambiar el formato del respaldo.
- Cambiar `--yes`, `--dry-run`, `--apps`, el snapshot o el código de salida.
- Virtualizar la lista o rediseñar la pantalla como un visor por bloques; el límite de 20 000 líneas sigue siendo el techo.
- Rediseñar la tabla de conflictos o la selección de apps.

## Decisions

### 1. Modelo de filas en `conflicts.py`

Se sustituye el modelo plano de bloques por filas explícitas:

```text
RowKind = EQUAL | REPLACE | LOCAL_ONLY | BACKUP_ONLY

HunkRow
  kind: RowKind
  local: str | None      # línea original, con su final
  backup: str | None     # línea original, con su final

Hunk
  index
  local_start, local_end
  backup_start, backup_end
  rows: tuple[HunkRow, ...]
```

- `EQUAL`: contexto; `local` y `backup` presentes, no decidible.
- `REPLACE`: par `local`/`backup`; decidible.
- `LOCAL_ONLY`: línea que solo existe en el local; decidible.
- `BACKUP_ONLY`: línea que solo existe en el respaldo; decidible.

En un `REPLACE` con `n` líneas locales y `m` del respaldo se generan `max(n, m)` filas emparejando por índice; las sobrantes quedan como `LOCAL_ONLY` o `BACKUP_ONLY`. `SequenceMatcher` no garantiza correspondencia semántica dentro de un reemplazo, pero el emparejamiento por índice es determinista y la interfaz muestra ambos lados para que la persona decida.

`Hunk` deja de guardar `local_lines`, `backup_lines`, `context_before`, `context_after` y `decision`: los tres primeros se derivan de `rows` y el último no se usaba. `local_start/end` y `backup_start/end` se conservan para la cabecera. Se factoriza `_rows_for_opcode(...)` para que `build_hunks` y `compose` generen exactamente las mismas filas.

`HunkDecision` pasa a `RowDecision` (`LOCAL`/`BACKUP`). `FileDecision.hunks` pasa a `FileDecision.rows: tuple[RowDecision, ...]`, plano y en orden de documento. La lista plana evita anidar decisiones por bloque y hace trivial la validación de longitud; el bloque sigue siendo una agrupación de presentación. `row_count(conflict)` suma las filas decidibles de todos los `Hunk`.

**Alternativas descartadas:**

- Mantener decisiones por bloque y cambiar solo el render: no permite el caso que motiva el cambio.
- Decidir cada lado de forma independiente (incluir u omitir líneas locales y del respaldo por separado): permite duplicar contenido y deja ambiguo el orden cuando se incluyen ambos lados de un par.
- Guardar las decisiones anidadas por bloque: más estructura y validación sin beneficio; la UI ya conoce el bloque por la posición de la fila.

### 2. Composición por líneas

`compose(local_text, backup_text, decisions: Sequence[RowDecision])`:

1. Normaliza CRLF solo para comparar y calcula `SequenceMatcher.get_opcodes()`.
2. Recorre los opcodes en orden:
   - `equal`: emite las líneas locales con su final original.
   - no igual: genera las filas con `_rows_for_opcode`, consume una decisión por fila y emite el lado elegido. `LOCAL` en una fila `BACKUP_ONLY` omite la línea; `BACKUP` en una fila `LOCAL_ONLY` también la omite.
3. Las líneas del respaldo se adaptan al EOL dominante local; las locales conservan su final.
4. Salvaguarda de salto final: si una línea emitida no termina en `\n` y no es la última, se le añade el EOL dominante local. Solo la última línea puede conservar la ausencia de salto final.

El paso 4 corrige un caso que no existía con decisiones por bloque:

```text
local:   L1\nL2\nL3        (L3 sin salto final)
respaldo: B1\nB2          (B2 sin salto final)
filas: (L1|B1) (L2|B2) (L3|-)

B1, B2, L3  ->  B1\nB2L3      incorrecto
                B1\nB2\nL3    correcto (EOL local entre B2 y L3)
```

Invariantes que se prueban:

- Todas las filas en `BACKUP` producen el respaldo (con el EOL local en las líneas iguales).
- Todas las filas en `LOCAL` producen exactamente el local.
- El orden de las líneas no cambia.
- `len(decisions) == row_count(conflict)`; si no, `compose` falla y `restore.py` lanza `ConflictError`.

**Alternativas descartadas:**

- `difflib.ndiff` o `unified_diff` para componer: muestran el diff, pero no exponen una decisión por fila ni conservan los finales originales.
- Componer en la TUI: rompería la invariante de que la interfaz no inyecta bytes.

### 3. TUI: lista plana con filas decidibles

Se mantiene un único `ListView` y se construye una vez:

```text
+-- ListView ----------------------------------------------+
| Bloque 1 - local 1-5 - respaldo 1-5 - 2 lineas (disabled) |
|   alpha                                      (disabled)   |
|   one                                        (disabled)   |
| * + ONE                        [respaldo]    (par)        |
|   two                                        (disabled)   |
|   three                                      (disabled)   |
|   omega                                      (disabled)   |
| * + OMEGA                      [respaldo]    (par)        |
| Bloque 2 - ...                               (disabled)   |
+-----------------------------------------------------------+
```

- Cabeceras y contexto usan `ListItem(disabled=True)`. Textual 8.2.8 salta los ítems deshabilitados en `action_cursor_down/up`; el cursor inicial se fija explícitamente a la primera fila decidible porque el `_on_mount` del `ListView` ocurre antes de añadir los hijos.
- Cada par es un `ListItem` con un `Static` de dos líneas; cada fila de un solo lado es un `ListItem` de una línea. El lado elegido va brillante, con `*` y etiqueta `[local]`/`[respaldo]`; el otro queda atenuado. En filas de un solo lado, el lado ausente se etiqueta `[omitir]`.
- El texto se muestra con `rstrip("\r\n")`, no `rstrip()`, para conservar espacios finales significativos.
- La cabecera muestra `local -` o `respaldo -` cuando ese lado no tiene líneas.

Estado de la pantalla:

```text
decisions: list[RowDecision]
item_to_decision: list[int | None]   # índice del ListView -> índice de fila
row_widgets: list[Static]            # por índice de fila
row_block: list[int]                 # índice de fila -> índice de bloque
block_first_item: list[int]          # índice del ListView de la primera fila del bloque
```

- `Espacio`/`Enter` alterna la fila resaltada y actualiza solo su `Static`.
- `A`/`L` aplican a todas las filas; `Shift+A`/`Shift+L`, solo al bloque de la fila actual. Se añade una fila de botones para las acciones por bloque, además de las bindings, para que sean descubribles.
- `n`/`p` saltan al primer ítem decidible del bloque siguiente/anterior; en los extremos se quedan, sin wrap.
- Un clic sobre una cabecera o contexto se ignora y reubica el cursor en la fila decidible más cercana.
- `action_save` compone con `self.decisions`, valida y devuelve `FileDecision(REVIEW, tuple(self.decisions))`; si la decisión existente no cubre `row_count(conflict)`, se reinicia a todo respaldo.
- No se reconstruye la lista al alternar: se actualiza el `Static` afectado. Las acciones globales y por bloque actualizan en sitio las filas afectadas.

**Alternativas descartadas:**

- Una vista de un bloque a la vez: acota los widgets, pero pierde la vista completa y añade estado de navegación.
- Una `DataTable` con columnas local/respaldo: es más literal, pero trunca líneas largas de configuración y obliga a gestionar filas no decidibles.
- Un cursor propio sobre un `Static` por bloque: evita muchos widgets, pero es más código y menos idiomático que `ListItem(disabled=True)`.

### 4. Integración con la restauración

- `restore.py::_resolve_review` valida que `len(decision.rows) == row_count(conflict)`, compone con `decision.rows` y mantiene la clasificación actual: si el resultado es igual al local, omitido; si todas las filas son `BACKUP`, restaurado; en caso contrario, mezclado.
- `review_all(conflict, decision)` construye `row_count(conflict)` decisiones.
- Los conflictos no textuales siguen rechazando `REVIEW` y resolviéndose como archivo entero.
- `ConflictSummaryScreen` muestra `N bloques - M líneas` en el detalle y `revisión (X/M respaldo)` en la decisión.
- `--yes`, `--dry-run`, `--apps`, la selección de apps y `ui.py` no cambian. El dry-run sigue usando `unified_diff` y no decide nada.

La clasificación se mantiene basada en “todas las filas en respaldo” y no en comparar el texto con el respaldo: evita recalcular el diff y conserva el significado actual de restaurado/mezclado. Una mezcla que casualmente produzca el texto del respaldo se sigue contando como mezclada, igual que hoy con los bloques.

## Risks / Trade-offs

- [El emparejamiento por índice dentro de un `replace` es arbitrario] → La interfaz muestra ambos lados y la persona decide; los invariantes todo-respaldo/todo-local no dependen del emparejamiento. Se documenta en el diseño y se cubre con pruebas.
- [Una línea sin salto final puede quedar en medio de una mezcla] → Salvaguarda de composición y escenario dedicado con CRLF/LF y sin salto final.
- [Muchos widgets en archivos enormes] → Se mantiene el guard de 20 000 líneas; el toggle actualiza solo una fila. Si aparecen casos reales, una vista por bloque o virtualización queda para otro cambio.
- [La refactorización de `Hunk` toca pruebas existentes] → Se actualizan `test_conflicts.py` y `test_tui.py`; no hay API pública afectada.
- [Clic en cabecera o contexto deja el cursor en un ítem deshabilitado] → Se ignora la acción y se reubica en la fila decidible más cercana.
- [Decisiones desalineadas si cambia el diff] → `build_hunks` y `compose` comparten `_rows_for_opcode` y el mismo orden de documento; `restore.py` valida la longitud antes de componer.
- [Más botones y atajos saturan la pantalla] → Las acciones por bloque van en una fila propia y también en la ayuda; si la terminal es estrecha, los botones pueden envolver sin cambiar el comportamiento.

## Migration Plan

- No hay migración de datos ni de formato: los respaldos existentes se restauran igual y las decisiones no se persisten.
- El cambio de comportamiento es solo en la TUI; `--yes` y `--dry-run` conservan su contrato.
- Actualizar el delta de `restore-conflicts` (ya incluye los renombrados y las nuevas exigencias), `README.md`, `docs/Diseño funcional.md`, `docs/Diseño técnico.md` y la ayuda de la TUI.
- Actualizar pruebas de conflicto, restauración, integración, TUI y dry-run.
- Rollback: revertir los cambios de código y la documentación; no hay datos convertidos ni snapshots que migrar.
- Verificación: `uv run ruff check .`, `uv run ruff format --check .` y `uv run pytest`, en ese orden.

## Open Questions

- El texto exacto de las etiquetas de filas de un solo lado y la disposición final de los botones por bloque se pueden pulir durante la implementación sin cambiar el comportamiento acordado.
- Si las pruebas de rendimiento muestran que el límite de 20 000 líneas es demasiado alto para una lista plana, se puede añadir un límite específico de filas decidibles o una vista por bloque en un cambio posterior; no afecta al contrato actual.
