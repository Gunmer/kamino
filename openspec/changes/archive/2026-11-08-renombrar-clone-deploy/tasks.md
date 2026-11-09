# Tasks

## 1. Preparación y renombrado estructural

- [x] 1.1 Confirmar que el árbol está limpio, que `restore-line-level-decisions` ya está archivado y que las capacidades ya están en `clone-safety` y `deploy-conflicts`; verificar con `git status --porcelain`, `openspec list --json` y `openspec list --specs --json`.
- [x] 1.2 Renombrar con `git mv` los módulos y las pruebas: `backup.py` → `clone.py`, `restore.py` → `deploy.py`, `test_backup.py` → `test_clone.py`, `test_restore.py` → `test_deploy.py`; actualizar imports y verificar con `uv run pytest tests/test_smoke.py`.
- [x] 1.3 Actualizar `tests/test_smoke.py`, `scripts/build_release.py` y `tests/test_release.py` (incluidos el fixture del docstring y el ejemplo de commit) para usar `clone`; verificar con `uv run pytest tests/test_release.py`.

## 2. API y dominio

- [x] 2.1 Renombrar la API de clonación (`BackupResult` → `CloneResult`, `run_backup` → `run_clone`, `print_backup_summary` → `print_clone_summary`) y la función de comando `backup` → `clone`; verificar con `uv run pytest tests/test_cli.py tests/test_clone.py tests/test_ui.py`.
- [x] 2.2 Renombrar la API de despliegue (`RestoreStatus` → `DeployStatus`, `RestoreEntry` → `DeployEntry`, `RestorePlan` → `DeployPlan`, `RestoreResult` → `DeployResult`, `RestoreClassification` → `DeployClassification`, `RestoreSession` → `DeploySession`, `prepare_restore` → `prepare_deploy`, `run_restore` → `run_deploy`, `run_restore_tui` → `run_deploy_tui`) y la función de comando `restore` → `deploy`; verificar con `uv run pytest tests/test_cli.py tests/test_deploy.py tests/test_tui.py`.
- [x] 2.3 Renombrar campos y enums del dominio: `restored` → `deployed`, `ResultCategory.RESTORED` → `DEPLOYED`, `restorable_apps` → `deployable_apps`, `RowDecision.BACKUP` → `CLONE`, `RowKind.BACKUP_ONLY` → `CLONE_ONLY`, y los campos `backup_*` → `clone_*`; verificar con `uv run pytest tests/test_conflicts.py tests/test_deploy.py tests/test_tui.py`.
- [x] 2.4 Renombrar helpers y detalles internos: `_backup_content`/`_backup_output` → `_clone_*`, prefijo temporal `kamino-restore-` → `kamino-deploy-` y mensaje `restore session is closed` → `deploy session is closed`; verificar con `uv run pytest tests/test_deploy.py`.
- [x] 2.5 Revisar docstrings y comentarios: conservar su idioma y sustituir solo los términos del dominio, sin traducir la prosa; verificar con `git diff --word-diff` y revisión manual de `src/`.

## 3. TUI y textos visibles

- [x] 3.1 Actualizar la ayuda de Typer, el menú sin argumentos y los mensajes de `cli.py`; revisar el cambio de género `restauración` → `despliegue`; verificar con `uv run pytest tests/test_cli.py`.
- [x] 3.2 Actualizar los resúmenes y encabezados de diff de `ui.py` (`Plan de clonación`, `Clonación completada`, `Archivos clonados`, `Plan de despliegue`, `Desplegadas`, `clon/…`); verificar con `uv run pytest tests/test_ui.py`.
- [x] 3.3 Actualizar títulos, botones, bindings, etiquetas e IDs de `tui/app.py`, `tui/screens.py` y `tui/widgets.py` (`KaminoDeployApp`, `Todo clon`, `all-clone`, `block-clone`, `clon`); verificar con `uv run pytest tests/test_tui.py`.
- [x] 3.4 Revisar manualmente las cadenas que cambian de género o número (`de la restauración` → `del despliegue`, `Restaurados` → `Desplegados`, `Restauradas` → `Desplegadas`, `clonación cancelada`) y verificar que no queden “la despliegue” ni “el restauración” con `rg -n "la despliegue|el restauración|Despliegue completada|despliegue cancelada" src tests`.

## 4. Documentación y metadatos

- [x] 4.1 Actualizar `README.md`, `docs/Diseño funcional.md`, `docs/Diseño técnico.md` y `AGENTS.md`: comandos, módulos, estructura, ejemplos y vocabulario; verificar con `rg -n "kamino (backup|restore)|backup\\.py|restore\\.py|test_(backup|restore)\\.py" README.md docs AGENTS.md` y revisión manual.
- [x] 4.2 Actualizar la descripción y las keywords de `pyproject.toml`; conservar el classifier `Topic :: System :: Archiving :: Backup`; verificar con `uv run python -c "import tomllib; print(tomllib.load(open('pyproject.toml','rb'))['project']['description'])"`.
- [x] 4.3 Actualizar el comentario de `.gitignore` (“Respaldos generados localmente” → “Clones generados localmente”) si se quiere coherencia completa; verificar con `rg -n "Respaldos generados" .gitignore`.

## 5. Especificaciones vivas

- [x] 5.1 Editar `openspec/specs/cli-options/spec.md`: comandos `clone`/`deploy`, ayuda, simulación, confirmación, resumen y escenarios; verificar con `openspec validate cli-options --type spec --strict`.
- [x] 5.2 Editar `openspec/specs/clone-safety/spec.md` y `openspec/specs/deploy-conflicts/spec.md`: título (`# clone-safety Specification`, `# deploy-conflicts Specification`), propósito y los 6 + 13 requisitos; verificar con `openspec validate clone-safety --type spec --strict` y `openspec validate deploy-conflicts --type spec --strict`.
- [x] 5.3 Editar `global-app-config`, `ghostty-config-references`, `self-update` y `tui-ansi-theme` para sustituir respaldo/restauración por clon/despliegue; verificar con `openspec validate --specs --strict`.
- [x] 5.4 Revisar que no queden ocurrencias antiguas fuera de la allowlist (`archive/`, `CHANGELOG.md`, `**/backups` de Micro/Gentle AI, “editor backups”, “queda como respaldo” y el classifier de PyPI); verificar con `rg -n -i "backup|restore|respald|restaur" src tests docs README.md AGENTS.md pyproject.toml openspec/specs`.
- [x] 5.5 Confirmar que el cambio no creó deltas ni movió capacidades: `find openspec/changes/renombrar-clone-deploy/specs -type f` no debe devolver archivos y `openspec list --specs --json` debe seguir mostrando `clone-safety` y `deploy-conflicts`.

## 6. Integración y release

- [x] 6.1 Ejecutar las comprobaciones de CI en orden: `uv run ruff check .`, `uv run ruff format --check .` y `uv run pytest`.
- [x] 6.2 Probar la CLI: `uv run kamino --help`, `uv run kamino clone --help` y `uv run kamino deploy --help`; comprobar que `uv run kamino backup` y `uv run kamino restore` fallan con código 2 y no aparecen en la ayuda.
- [x] 6.3 Verificar el ciclo completo con un clon existente: crear un clon con `clone --dry-run`, desplegarlo con `deploy --dry-run` y comprobar que el formato, las opciones y los códigos de salida no cambian; cubrirlo en `tests/test_integration.py`.
- [x] 6.4 Ejecutar el smoke test de release (`uv run python scripts/build_release.py` en macOS arm64, o al menos `uv run pytest tests/test_release.py`) y confirmar que el binario usa `clone`.
- [x] 6.5 Preparar el commit `feat!: rename backup/restore to clone/deploy` con footer `BREAKING CHANGE:` y verificar que semantic-release calcula `0.7.0` sin publicar.

## Workflow follow-up

- Revisar el diff completo y archivar el cambio cuando la implementación esté validada.
- Verificar que las specs archivadas conservan los títulos `clone-safety` y `deploy-conflicts` y que no queda vocabulario antiguo fuera de la allowlist.
