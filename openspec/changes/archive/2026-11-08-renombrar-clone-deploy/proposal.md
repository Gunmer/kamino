# Proposal

## Why

La CLI todavía habla de `backup` y `restore`, mientras el producto ya usa la temática de Kamino: el archivo es el clon, respaldar es clonar y restaurar es desplegar. Este cambio prepara la TUI de clonación sobre nombres coherentes y elimina la deuda de vocabulario antes de construir nuevas pantallas.

Las carpetas de las capacidades ya se renombraron en `04d398b` (`backup-safety` → `clone-safety`, `restore-conflicts` → `deploy-conflicts`), pero sus títulos, propósitos y requisitos siguen usando el vocabulario antiguo. Este cambio completa ese renombrado y alinea código, pruebas, textos y especificaciones.

## What Changes

- **BREAKING**: `kamino clone` y `kamino deploy` sustituyen a `kamino backup` y `kamino restore`. Los comandos antiguos se eliminan del todo, sin alias visibles ni ocultos.
- Renombrar los módulos `backup.py` → `clone.py` y `restore.py` → `deploy.py`, sus funciones, clases, campos y pruebas.
- Renombrar el vocabulario visible en español: respaldo/respaldar → clon/clonar (o clonación para la acción) y restauración/restaurar → despliegue/desplegar. `local` se conserva; `snapshot`, `mezcla` y el resto del glosario temático quedan fuera.
- Conservar el idioma actual de docstrings y comentarios; sustituir solo los términos del dominio, sin traducir ni reescribir su prosa.
- Actualizar el menú sin argumentos, la ayuda de Typer, los resúmenes de `ui.py`, las etiquetas de la TUI y los mensajes de error.
- Actualizar `README.md`, `docs/Diseño funcional.md`, `docs/Diseño técnico.md`, `AGENTS.md` y los metadatos de `pyproject.toml`.
- Corregir los títulos y propósitos de `openspec/specs/clone-safety/spec.md` y `openspec/specs/deploy-conflicts/spec.md`, y sustituir el vocabulario antiguo en las specs vivas.
- Actualizar el smoke test de release (`scripts/build_release.py`) y su prueba (`tests/test_release.py`) para que usen `clone`.
- Conservar sin cambios las opciones (`--apps`, `--output`, `--dry-run`, `--verbose`, `--yes`), sus alias cortos, los códigos 0/1/2/3, `format_version = 1`, el manifiesto, las instantáneas, las rutas de usuario y el nombre por defecto `kamino-<hostname>-<fecha>.tar.gz`.

## Capabilities

### New Capabilities

- Ninguna.

### Modified Capabilities

- Ninguna mediante delta. El cambio es un renombrado mecánico: no altera el comportamiento de clonar, desplegar, resolver conflictos, validar mezclas, crear instantáneas ni actualizar el binario. Por eso el cambio declara `skip_specs: true` y actualiza directamente las specs vivas.

Las specs `cli-options`, `clone-safety`, `deploy-conflicts`, `global-app-config`, `ghostty-config-references`, `self-update` y `tui-ansi-theme` se editan directamente para sustituir comandos, títulos, propósitos, requisitos y escenarios. No se usa `REMOVED + ADDED` ni `RENAMED Requirements` para capacidades: el `git mv` preparatorio de `backup-safety` y `restore-conflicts` ya se ejecutó en `04d398b`, y este cambio solo verifica las rutas nuevas y completa su contenido. OpenSpec no permite renombrar escenarios dentro de un bloque `MODIFIED`; forzar un delta obligaría a conservar títulos como “Salida detallada del respaldo” o a retirar y recrear requisitos completos. La edición directa evita esa pérdida y mantiene el cambio estrictamente mecánico.

## Impact

- `src/kamino/backup.py` → `src/kamino/clone.py`
- `src/kamino/restore.py` → `src/kamino/deploy.py`
- `src/kamino/{cli,ui,conflicts,archive,snapshot,plan,errors}.py`
- `src/kamino/tui/{app,screens,widgets,__init__}.py`
- `tests/test_backup.py` → `tests/test_clone.py`
- `tests/test_restore.py` → `tests/test_deploy.py`
- `tests/{test_cli,test_tui,test_ui,test_integration,test_conflicts,test_smoke,test_release}.py`
- `scripts/build_release.py`
- `README.md`, `docs/*.md`, `AGENTS.md`, `pyproject.toml`
- `openspec/specs/{cli-options,clone-safety,deploy-conflicts,global-app-config,ghostty-config-references,self-update,tui-ansi-theme}/spec.md`

## Out of Scope

- La TUI nueva, el menú rediseñado, el logo y las frases temáticas.
- `snapshot` → holocrón, `mezcla` → híbrido y el resto del glosario.
- Renombrar `local` → `original` en código o textos.
- `--force`/`-f`, el nombre `clone-yyyyMMdd` y cualquier cambio de comportamiento.
- `openspec/changes/archive/` y `CHANGELOG.md`: son historia y no se reescriben.
