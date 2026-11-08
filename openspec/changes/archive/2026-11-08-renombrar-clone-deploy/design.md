# Design

## Context

Ver `proposal.md` para la motivación. El estado actual relevante:

- La CLI sigue exponiendo `backup` y `restore` (`src/kamino/cli.py:98-214`), con los módulos `backup.py` y `restore.py`.
- La revisión de conflictos ya es por líneas: `RowKind`, `HunkRow`, `RowDecision` y `FileDecision.rows` viven en `conflicts.py`. Cualquier renombrado debe usar esos nombres, no el antiguo `HunkDecision`.
- Las carpetas `openspec/specs/clone-safety/` y `openspec/specs/deploy-conflicts/` ya se movieron en `04d398b` (el `git mv` preparatorio de `backup-safety` y `restore-conflicts`), pero sus títulos y propósitos siguen diciendo `backup-safety` y `restore-conflicts`. Este cambio no repite el `git mv`; verifica las rutas y completa el contenido.
- La versión actual es `0.6.0`; `major_on_zero = false`.
- El árbol está limpio y el cambio `restore-line-level-decisions` ya está archivado. No hay que coordinar el renombrado con un cambio activo.

## Goals / Non-Goals

**Goals:**

- Dejar `kamino clone` y `kamino deploy` como únicos comandos de clonación y despliegue.
- Eliminar por completo `backup` y `restore`, incluidas sus funciones, módulos, pruebas y textos.
- Usar un vocabulario coherente en código, CLI, TUI, documentación y specs vivas.
- Mantener intactos el formato del archivo, las opciones, los alias, los códigos de salida y la compatibilidad con clones existentes.

**Non-Goals:**

- Rediseñar la TUI, el menú, el logo o las frases temáticas.
- Renombrar `local` a `original`, `snapshot` a holocrón o `mezcla` a híbrido.
- Cambiar el nombre por defecto `kamino-<hostname>-<fecha>.tar.gz`.
- Añadir `--force`/`-f` o cualquier comportamiento nuevo.
- Reescribir `openspec/changes/archive/` ni `CHANGELOG.md`.

## Decisions

### 1. Comandos Typer: `clone` y `deploy`, sin alias

Las funciones de comando se llaman `clone` y `deploy`; Typer deriva de ahí los nombres. No se usa `@app.command(name=...)` ni se conserva un alias oculto. `kamino backup` y `kamino restore` pasan a ser comandos desconocidos y terminan con código 2, que ya es el código de uso incorrecto.

El menú sin argumentos y `_run_menu` cambian sus IDs a `clone` y `deploy`; `ctx.invoke` usa `clone` y `deploy`, y el argumento posicional de `deploy` pasa de `backup_file` a `clone_file`.

**Alternativa descartada:** mantener alias obsoletos ocultos. La decisión del cambio es eliminarlos del todo para que la ayuda y el menú no muestren dos generaciones de nombres.

### 2. Renombrado completo, no solo de la fachada

El mapa del proposal se amplía a toda la API y al dominio:

| Antes | Después |
|---|---|
| `BackupResult` | `CloneResult` |
| `run_backup` | `run_clone` |
| `print_backup_summary` | `print_clone_summary` |
| `RestoreStatus` / `RestoreEntry` / `RestorePlan` / `RestoreResult` / `RestoreClassification` / `RestoreSession` | `DeployStatus` / `DeployEntry` / `DeployPlan` / `DeployResult` / `DeployClassification` / `DeploySession` |
| `prepare_restore` / `run_restore` / `run_restore_tui` | `prepare_deploy` / `run_deploy` / `run_deploy_tui` |
| `KaminoRestoreApp` | `KaminoDeployApp` |
| `print_restore_summary` | `print_deploy_summary` |
| `RestoreResult.restored` / `RestoreClassification.restored` | `DeployResult.deployed` / `DeployClassification.deployed` |
| `ResultCategory.RESTORED` | `ResultCategory.DEPLOYED` |
| `restorable_apps` | `deployable_apps` |
| `RowDecision.BACKUP` / `RowKind.BACKUP_ONLY` | `RowDecision.CLONE` / `RowKind.CLONE_ONLY` |
| `backup_start/end/lines/size/mode/sha256/link/text` | `clone_start/end/lines/size/mode/sha256/link/text` |
| `_backup_content` / `_backup_output` | `_clone_content` / `_clone_output` |
| `action_all_backup` / `action_block_backup` / IDs `all-backup` / `block-backup` | `action_all_clone` / `action_block_clone` / IDs `all-clone` / `block-clone` |
| `kamino-restore-` (temporal) | `kamino-deploy-` |
| `restore session is closed` | `deploy session is closed` |

No se renombran `local`, `LocalFile`, `local_*`, `Hunk`, `snapshot`, `mezcla` ni `Archive`/`Manifest`: no forman parte del vocabulario backup/restore y cambiarlos ampliaría el alcance sin necesidad.

### 3. Vocabulario visible: artefacto y acción

- `clon` es el artefacto (`el archivo del clon`, `los clones`).
- `clonación` es la acción (`Plan de clonación`, `Clonación completada`, `clonación cancelada`).
- `clonar` es el verbo.
- `despliegue` es la acción y el resultado (`Plan de despliegue`, `Despliegue completado`, `despliegue cancelado`).
- `desplegar` es el verbo; `desplegable` el adjetivo.

Esto evita “Clon completado” y “clon cancelado”, que suenan a objeto terminado o abandonado, y mantiene la concordancia: `restauración` es femenino y `despliegue` masculino, así que “de la restauración” pasa a “del despliegue” y “restauración completada” a “despliegue completado”.

`local` se conserva en los textos y en el código. “Original” queda para `tui-clonacion`; introducirlo ahora obligaría a renombrar `LocalFile`, `HunkRow.local`, `RowDecision.LOCAL` y todos los textos de la TUI.

Se conserva la concordancia de género actual en los resúmenes: la CLI cuenta configuraciones en femenino (`Desplegadas`) y la TUI cuenta archivos en masculino (`Desplegados`). No se unifican en este cambio.

### 3.1. Docstrings y comentarios

Los docstrings y comentarios conservan su idioma actual. El renombrado sustituye solo los términos del dominio (`backup` → `clone`, `restore` → `deploy`, `respaldo` → `clon`, `restauración` → `despliegue`) y no traduce ni reescribe la prosa. Esto evita convertir un cambio mecánico en una traducción masiva y respeta el estado real del código, que ya mezcla docstrings en español e inglés.

### 4. Specs vivas: `skip_specs: true` y edición directa

El cambio declara `skip_specs: true` porque no cambia el comportamiento de clonar, desplegar, resolver conflictos, validar mezclas, crear instantáneas ni actualizar el binario. Sí cambia el nombre de los comandos, pero el contrato funcional de `clone`/`deploy` es el mismo que el de `backup`/`restore`.

La alternativa era un delta `MODIFIED` para `cli-options`. Al validarlo, OpenSpec exige conservar los nombres de los escenarios existentes; renombrar “Salida detallada del respaldo” a “Salida detallada de la clonación” se interpreta como eliminar un escenario y añadir otro. Para evitar títulos antiguos o una retirada/recreación de requisitos, se editan directamente las specs vivas:

- `cli-options`: comandos, ayuda, simulación, confirmación y resumen.
- `clone-safety`: título, propósito y los 6 requisitos.
- `deploy-conflicts`: título, propósito y los 13 requisitos.
- `global-app-config`: 6 requisitos y propósito.
- `ghostty-config-references`: 5 requisitos y propósito.
- `self-update`: escenario de “Destino del reemplazo”.
- `tui-ansi-theme`: propósito y requisito de paleta ANSI.
- `release-verification`: sin cambios.

La edición directa se revisa con `git diff openspec/specs/` y se valida con `openspec validate --specs --strict`. No se usa `REMOVED + ADDED` ni `RENAMED Requirements` para capacidades: el `git mv` preparatorio ya está hecho y esas operaciones borrarían o recrearían requisitos completos. No se toca `openspec/changes/archive/`.

### 5. `git mv` separado del contenido, pero con commits verdes

Se renombran los archivos con `git mv` para conservar el historial:

```text
src/kamino/backup.py        -> src/kamino/clone.py
src/kamino/restore.py       -> src/kamino/deploy.py
tests/test_backup.py        -> tests/test_clone.py
tests/test_restore.py       -> tests/test_deploy.py
```

El primer commit incluye el `git mv` y los imports mínimos para que `pytest` siga pasando. Después se separan commits de dominio, TUI, documentación y specs. Un `git mv` sin actualizar imports dejaría el árbol roto entre commits; no se hace.

### 6. Falsos positivos que no se tocan

- `**/backups` en `micro.toml` y `gentle-ai.toml`: son directorios de datos de esas apps, no clones de Kamino.
- `test_micro_does_not_include_editor_backups_or_buffers`: se refiere a los backups del editor Micro.
- “queda como respaldo” en `docs/Diseño técnico.md:216`: significa fallback.
- `Topic :: System :: Archiving :: Backup` en `pyproject.toml`: es una categoría PyPI, no una cadena de producto.
- `CHANGELOG.md` y `openspec/changes/archive/`: historia.

### 7. Release y smoke test

`scripts/build_release.py:32` ejecuta `kamino backup --dry-run`; `tests/test_release.py:448` lo verifica. Ambos deben usar `clone`. Si se olvidan, `semantic-release` falla al construir el binario y la release se bloquea después del bump.

El commit será `feat!: rename backup/restore to clone/deploy` (o `feat(cli)!`) con footer `BREAKING CHANGE:`. Con `major_on_zero = false`, la versión pasa de `0.6.0` a `0.7.0`, no a 1.0.0.

## Risks / Trade-offs

- [El renombrado de campos internos genera un diff grande] → Separar en commits por capa y usar `rg` para verificar que no quedan referencias cruzadas.
- [Los textos cambian de género y una sustitución ciega produce “la despliegue”] → Revisión manual de la tabla de textos y pruebas de `ui.py`/`tui`.
- [El módulo `clone` puede sombrear una variable o función `clone`] → Importar siempre `from kamino.clone import run_clone`; en pruebas, `clone_module`/`deploy_module`.
- [Los tests esperan cadenas exactas como “Restauradas: 1” o “+++ respaldo/”] → Actualizar las aserciones junto con los textos.
- [El smoke test de release queda desactualizado] → Incluirlo en la primera tarea estructural y en la verificación final.
- [Las specs vivas se editan sin delta] → Revisar el diff completo de `openspec/specs/` y validar con `openspec validate --specs --strict`; el cambio es mecánico y no altera comportamiento.
- [Quedan falsos positivos] → Allowlist explícita y `rg` con revisión manual.

## Migration Plan

1. Confirmar árbol limpio y que `restore-line-level-decisions` está archivado.
2. `git mv` de módulos y pruebas; actualizar imports; `pytest`.
3. Renombrar API pública y dominio; `pytest`.
4. Renombrar textos de CLI/TUI; `pytest`.
5. Actualizar README, docs, AGENTS y `pyproject.toml`.
6. Editar las specs vivas y validarlas.
7. Ejecutar CI y el smoke test de release.
8. Publicar con `feat!` y verificar la versión `0.7.0`.

Rollback: revertir los commits del cambio. No hay migración de datos ni de formato; los clones existentes siguen siendo legibles y desplegables con `kamino deploy`.

## Open Questions

Ninguna que cambie el enfoque. Las decisiones de copy (`clon`/`clonación`, `local`/`original`) y el classifier de PyPI quedan resueltas arriba.
