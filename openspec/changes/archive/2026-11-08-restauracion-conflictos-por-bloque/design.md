# Design

## Context

La restauración actual vive en `src/kamino/restore.py` como un caso de uso sin interfaz: lee el manifiesto, detecta las apps instaladas, clasifica cada archivo como `NEW`, `IDENTICAL`, `DIFFERENT` o `UNSUPPORTED`, verifica el archivo, extrae a un directorio temporal, crea un snapshot de los destinos que van a cambiar y aplica reemplazos atómicos. `src/kamino/cli.py` es una capa fina de Typer, pero hoy duplica la detección de apps para la selección interactiva; `src/kamino/ui.py` concentra la presentación con `questionary` y `rich`.

El archivo de respaldo es `manifest.json` más `apps/<id>/<ruta>`. El contenido completo ya está en el `tar.gz`, así que no hace falta cambiar el formato ni añadir metadatos de conflicto. `SafeHome` confina el acceso al home y `snapshot.py` reutiliza el formato de respaldo para el estado previo.

Ver `proposal.md` para la motivación y las specs `restore-conflicts` y `cli-options` para el contrato de comportamiento.

## Goals / Non-Goals

**Goals:**

- Detectar conflictos de contenido y resolverlos por bloques en una sola interfaz interactiva.
- Mantener el orden de seguridad: verificar el archivo antes de extraer, snapshot antes de escribir, re-verificar destinos antes de aplicar y no escribir nada al cancelar.
- Conservar las vías no interactivas (`--yes`, `--dry-run`) y la compatibilidad con respaldos existentes.
- Dejar el diff, la composición y la validación en el núcleo, probables sin TUI.
- Mantener `cli.py` como capa de argumentos y despacho; la lógica de restauración sigue en `restore.py`.

**Non-Goals:**

- Merge semántico por clave ni validación de YAML/shell.
- Transacción atómica para el conjunto completo de archivos; se mantiene la atomicidad por archivo y el snapshot como red de seguridad.
- Edición del archivo final en un editor externo o vista previa completa.
- Cambiar el formato del archivo de respaldo o persistir decisiones entre ejecuciones.
- Mover la selección de apps de `backup` a la TUI; solo cambia la de `restore`.
- Detección de apps abiertas, `cfprefsd` o permisos especiales de macOS.

## Decisions

### 1. Sesión de restauración en el núcleo

`restore.py` expone una sesión con tres pasos y un contexto que limpia el directorio temporal:

```text
prepare_restore(backup_file, *, home, catalog, application_dirs, path, snapshots_dir)
    -> RestoreSession
         .manifest
         .restorable_apps        apps del manifiesto instaladas (y la app snapshot)
         .plan(selected, dry_run=False) -> RestorePlan
         .apply(decisions)              -> RestoreResult
         .close()                       limpia el staging
```

- `plan` clasifica los archivos, verifica el `tar.gz`, extrae a un temporal y construye los conflictos. Con `dry_run=True` no verifica ni extrae: lee el contenido de los conflictos directamente del archivo para poder imprimir los diffs.
- `apply` exige que todos los conflictos tengan decisión, re-valida las mezclas, re-verifica los destinos, crea el snapshot y escribe.
- La TUI solo llama a `plan` y `apply` y muestra pantallas; no conoce la verificación ni el snapshot.

**Por qué:** mantiene la invariante de seguridad en un único lugar y permite probar la lógica sin TUI. **Alternativas descartadas:** callbacks dentro de `run_restore` (abre la TUI en dos momentos y separa la experiencia), funciones sueltas orquestadas por la TUI (la seguridad depende de que la TUI recuerde el orden), un `run_restore_tui` paralelo (duplica la lógica y puede divergir).

`run_restore` se conserva como envoltorio no interactivo para no romper llamadores existentes: prepara la sesión, planifica y aplica todas las decisiones como “aplicar respaldo”.

### 2. Modelo de conflictos y módulo puro de diff/mezcla

Un módulo nuevo (`kamino/conflicts.py`) contiene funciones puras y modelos:

- `Conflict`: app, ruta, destino, tipo (`text`, `binary`, `large`, `symlink`), tamaño, modo local y del respaldo, hashes locales y del respaldo, contenido decodificado de ambos lados y lista de bloques.
- `Hunk`: índice, rangos de líneas locales y del respaldo, líneas de ambos lados, contexto y decisión (`backup` o `local`).
- `FileDecision`: `APPLY`, `OMIT` o `REVIEW` con la lista de decisiones por bloque.
- `ConflictDecisions`: mapa ruta -> `FileDecision`.
- `build_conflict(...)`, `compose(...)`, `validate_merge(...)`.

La TUI recibe los mismos `Conflict` y `Hunk` que luego usa el núcleo para componer. La composición se hace en el núcleo, no en la TUI: la interfaz no debe poder inyectar bytes arbitrarios en el destino.

**Por qué:** el diff y la composición son la parte más delicada y se prueban con casos límite sin levantar una TUI. La TUI queda como presentación y recogida de decisiones.

### 3. Diff por bloques y composición

- Se decodifica UTF-8. Un archivo con bytes NUL o que no decodifica es binario. Un texto de más de 5 MB o de más de 20 000 líneas se trata como “grande” para proteger `difflib` y la TUI.
- Para comparar se normaliza CRLF a LF. Un archivo que solo difiere en CRLF es idéntico.
- Se usa `difflib.SequenceMatcher` sobre `splitlines(keepends=True)` normalizado y `get_grouped_opcodes(3)` para agrupar bloques con 3 líneas de contexto.
- Cada bloque agrupa los opcodes no iguales contiguos. Todos empiezan en `backup`; el usuario alterna a `local`.
- La composición recorre los opcodes: las líneas iguales salen del local para conservar sus finales; un bloque `backup` aporta las líneas del respaldo; un bloque `local` conserva las locales. Las líneas del respaldo se adaptan al estilo de final de línea local (CRLF o LF dominante). Las líneas locales conservan su final original, incluso si el archivo local mezcla estilos. El último salto de línea se conserva tal como venga del lado elegido.

**Por qué:** es determinista, usa solo la biblioteca estándar y respeta la decisión del usuario sin reinterpretar el formato. **Alternativas descartadas:** merge por clave (requiere parsers por formato y no cubre comentarios ni orden), diff por línea (el usuario eligió bloques), biblioteca externa de diff (dependencia innecesaria).

### 4. Validación del resultado

Se valida por extensión, solo cuando el archivo es texto y la mezcla existe:

- `.toml` -> `tomllib.loads`.
- `.json` -> `json.loads` con detección de claves duplicadas.
- `.ini`, `.cfg` y `.gitconfig` -> `configparser.ConfigParser(strict=True)`.
- `.plist` -> `plistlib.loads` (XML; los plists binarios no llegan a merge de texto).

Se valida el original local, el original del respaldo y la mezcla. Si ambos originales parsean y la mezcla no, se bloquea y se vuelve al detalle. Si alguno de los originales ya era inválido, solo se avisa y se permite aplicar. Un formato desconocido, YAML o shell no se valida y se indica en la interfaz.

La validación se repite en `apply` como defensa del núcleo, aunque la TUI ya la haya mostrado.

**Por qué:** la stdlib cubre los formatos estructurados más comunes sin dependencias nuevas. La regla “solo bloquear si ambos originales parsean” evita falsos positivos con archivos que el parser no entiende pero la app sí. **Alternativas descartadas:** PyYAML y un parser de shell (más dependencias y más falsos positivos), no validar (el usuario pidió bloquear las mezclas rotas).

### 5. TUI con textual

Se añade un paquete `kamino/tui/`:

- `app.py`: `KaminoRestoreApp`, estado compartido y pila de pantallas.
- `screens.py`: selección de apps, tabla de conflictos, detalle, confirmación y resumen.
- `widgets.py`: tabla, visor de bloques, barra de estado y ayuda.
- `styles.tcss`: estilos de la aplicación.

Flujo de pantallas:

```text
AppSelection -> ConflictSummary -> ConflictDetail (por archivo)
             -> Confirm -> (apply) -> Summary
```

- **Apps:** lista con casillas, ninguna marcada por defecto; seleccionar todas/ninguna; continuar/cancelar.
- **Conflictos:** solo conflictos; conteos de nuevos, idénticos y no soportados; acciones revisar/aplicar/omitir; aplicar todo/omitir todo; confirmar deshabilitado hasta que todos tengan decisión.
- **Detalle:** bloques con contexto; espacio alterna; `A` toma todos los restantes del respaldo; `L` deja todos los restantes locales; validación al volver; sin vista previa.
- **Confirmación:** conteos de restaurados/mezclados/omitidos y directorio del snapshot; aplicar/volver/cancelar.
- **Resumen:** conteos finales y ruta del snapshot; salir.

Los textos y la ayuda van en español. La TUI no imprime secretos fuera de la pantalla ni los persiste; muestra un aviso discreto de que los diffs pueden contener datos sensibles.

**Por qué:** `textual` permite tabla navegable, detalle con scroll y pruebas con `run_test` sin TTY. **Alternativas descartadas:** `questionary` encadenado (sin navegación real), `rich` + `input()` (sin repintado ni teclas).

### 6. Despacho de la CLI

`restore` recibe `--apps/-a`, `--dry-run/-n`, `--yes/-y` y `--verbose/-v`. El despacho queda así:

```text
prepare_restore(...)
  |
  +-- --dry-run: plan(dry_run=True) -> imprimir diffs -> salir
  |
  +-- --yes o sin TTY: plan -> decisiones = aplicar todo -> apply -> resumen
  |     (sin TTY y sin --yes: error 1)
  |
  +-- TTY sin --yes: TUI -> plan -> decisiones -> apply -> resumen
```

- `--apps` valida los ids contra las apps restaurables; las desconocidas o no instaladas se avisan y cuentan como omitidas.
- `--yes` no abre la TUI: resuelve todos los conflictos aplicando el respaldo.
- `--dry-run` no abre la TUI: usa `--apps` o todas las instaladas y vuelca los diffs a stdout. No verifica, extrae, snapshot ni escribe.
- El código 3 se reserva a los omitidos; un archivo mezclado cuenta como éxito.
- `ui.print_restore_summary` añade la categoría “mezclados” y conserva el detalle con `--verbose`.

**Por qué:** preserva el contrato no interactivo y permite scripts con `--apps`, al tiempo que la experiencia interactiva queda completa en la TUI. **Alternativas descartadas:** un flag `--merge` separado (el usuario quiere el flujo por defecto en TTY), mantener la selección de apps con `questionary` (el usuario pidió una sola TUI).

### 7. Seguridad y concurrencia

- `plan` verifica el archivo con `verify_archive` antes de extraerlo y extrae solo en un `TemporaryDirectory`.
- `plan` guarda el hash, el destino del enlace o la ausencia del destino para cada archivo que va a cambiar.
- `apply` re-verifica todos esos destinos antes del snapshot. Si alguno cambió, lanza un error de cambio concurrente y no escribe ni crea snapshot.
- El snapshot incluye solo los destinos que se van a escribir (aplicar, mezclar o crear); los omitidos no se tocan.
- La escritura reutiliza `SafeHome.write_file` y `SafeHome.replace_with_symlink`, con reemplazo atómico y sin seguir enlaces intermedios.
- Cancelar en cualquier pantalla lanza una cancelación antes de `apply`; no hay snapshot ni escrituras.

Queda una ventana pequeña entre la re-verificación y la escritura, igual que en el diseño actual. La mitigación es re-verificar lo más tarde posible y documentar que la atomicidad es por archivo, no transaccional para todo el conjunto.

**Por qué:** mantiene las invariantes existentes y añade la protección que el usuario pidió para una TUI que puede estar abierta varios minutos. **Alternativas descartadas:** bloqueo de archivos (fuera de alcance), transacción multiarchivo (complejidad alta y no solicitada).

### 8. Modelo de resultado

`RestoreResult` añade `merged: list[str]`. Las categorías quedan:

- `restored`: archivos escritos con el contenido del respaldo (aplicar completo o revisión con todos los bloques en respaldo).
- `merged`: archivos escritos con una mezcla real de bloques.
- `omitted`: idénticos, no soportados, apps no instaladas, omitidos por decisión y revisiones con todos los bloques en local.

`RestorePlan` añade `conflicts: list[Conflict]`. El núcleo exige que `ConflictDecisions` cubra todos los conflictos; `--yes` construye decisiones `APPLY` para todos.

**Por qué:** el resumen refleja el efecto real y el código 3 conserva su significado. **Alternativas descartadas:** contar los mezclados como restaurados (pierde información), contar los mezclados como omitidos (código 3 falso).

### 9. Dependencia y empaquetado

Se añade `textual` a `pyproject.toml` y se actualiza `uv.lock`. El build con PyInstaller debe incluir los datos de `textual` (CSS y recursos) con `--collect-data textual` o el mecanismo equivalente de `scripts/build_release.py`. Las pruebas de la TUI usan `App.run_test()` y no necesitan TTY; la lógica de diff/mezcla se prueba sin `textual`.

**Por qué:** el usuario eligió `textual`; el coste es una dependencia nueva y trabajo de empaquetado, no lógica de negocio. **Alternativas descartadas:** `prompt_toolkit` (ya instalado, pero más código propio), `questionary` (no cumple la TUI).

## Risks / Trade-offs

- [Dependencia y empaquetado de `textual`] -> Añadir los datos al build de PyInstaller y probar el binario `onefile` en macOS antes de publicar.
- [`difflib` lento con archivos grandes o muchas líneas] -> Límite de 5 MB y guard de 20 000 líneas; por encima, conflicto de archivo entero. Ajustar el guard tras medir.
- [Falsos positivos o negativos de validación] -> Solo bloquear cuando ambos originales parsean; avisar cuando alguno falla; documentar que YAML y shell no se validan. La detección de claves duplicadas en JSON/plist reduce el riesgo de mezclas semánticamente ambiguas.
- [Aplicación no transaccional entre archivos] -> Atomicidad por archivo y snapshot previo; documentar que un fallo a mitad deja una restauración parcial recuperable desde el snapshot.
- [Carrera residual entre re-verificación y escritura] -> Re-verificar lo más tarde posible; documentar la ventana; un futuro bloqueo o transacción queda fuera de alcance.
- [TUI difícil de probar en CI] -> Probar la lógica en módulos puros y la TUI con `run_test`; no depender de una terminal real.
- [Cambio de UX en restauración interactiva] -> Documentar en README y ayuda; `--yes` conserva el flujo automatizable y `--apps` da control sin TUI.
- [Secretos visibles en los diffs] -> Aviso en la interfaz; no persistir ni registrar contenido; los diffs viven solo en memoria.
- [La simulación deja de ser interactiva] -> `--apps` cubre la selección; documentar el cambio y el nuevo escenario en `cli-options`.

## Migration Plan

- No hay migración de datos ni cambio de formato: los respaldos existentes se leen como hoy.
- Añadir `textual`, actualizar `uv.lock`, ejecutar la suite y verificar el build de release.
- Cambio de comportamiento: la restauración interactiva usa la TUI y `--dry-run` ya no abre la selección interactiva. `--yes` conserva el comportamiento anterior para automatización.
- Rollback: revertir el despacho de la CLI y la dependencia; el formato y los respaldos no se ven afectados. No hay migración irreversible.

## Open Questions

- Ajuste fino del guard de líneas (20 000) y del contexto del diff (3 líneas) tras las pruebas de rendimiento; no cambia el contrato de las specs.
- Atajos definitivos y textos de ayuda de la TUI; se pueden pulir durante la implementación sin cambiar el comportamiento acordado.
