# Tasks

## 1. Preparación

- [x] 1.1 Añadir `certifi` a las dependencias de `pyproject.toml` y regenerar `uv.lock` con `uv lock`; verificar con `uv sync --all-extras` y `uv run python -c "import certifi; print(certifi.where())"`.
- [x] 1.2 Añadir `UpdateError(KaminoError)` en `src/kamino/errors.py`; verificar con `uv run python -c "from kamino.errors import UpdateError, KaminoError; assert issubclass(UpdateError, KaminoError)"`.
- [x] 1.3 Crear `src/kamino/update.py` con las constantes del repositorio y del asset, y los dataclasses `ReleaseInfo` y `UpdateResult`; verificar con `uv run python -c "from kamino.update import ReleaseInfo, UpdateResult"`.

## 2. Versión y consulta de la release

- [x] 2.1 Implementar el parseo `vX.Y.Z` y la comparación SemVer; añadir pruebas en `tests/test_update.py` para `0.5.0`, `v0.5.0`, `0.10.0` frente a `0.9.0` y formatos inválidos; verificar con `uv run pytest tests/test_update.py -k version`.
- [x] 2.2 Implementar la consulta de `releases/latest`, la selección del asset `kamino-arm64` y la validación de `tag_name`, `size` y `digest`; probar payload válido, asset ausente, digest ausente, JSON inválido y error HTTP; verificar con `uv run pytest tests/test_update.py -k release`.
- [x] 2.3 Documentar en `docs/Diseño técnico.md` la consulta a GitHub, la comparación SemVer y la política de no degradar; verificar que la sección describe el flujo y que no afirma que la actualización sea automática.

## 3. Descarga, integridad y reemplazo

- [x] 3.1 Implementar la descarga por bloques a un temporal en el mismo directorio, el cálculo de SHA-256 y la comprobación de tamaño y digest; probar éxito, tamaño incorrecto, digest incorrecto y limpieza del temporal; verificar con `uv run pytest tests/test_update.py -k download`.
- [x] 3.2 Implementar `installed_binary_path()` con la comprobación de `sys.frozen` y `Path(sys.executable).resolve()`, y la comprobación de macOS arm64; probar instalación no frozen, plataforma no soportada y resolución de symlink; verificar con `uv run pytest tests/test_update.py -k installed`.
- [x] 3.3 Implementar el reemplazo atómico con `os.replace`, conservando el modo ejecutable y el binario anterior ante errores; probar destino regular y destino detrás de un symlink; verificar con `uv run pytest tests/test_update.py -k replace`.
- [x] 3.4 Implementar `run_update` para orquestar consulta, comparación, descarga, verificación y reemplazo, devolviendo `UpdateResult`; probar actualización, ya al día y versión local superior; verificar con `uv run pytest tests/test_update.py -k run_update`.
- [x] 3.5 Añadir una prueba que compare `REPOSITORY` y `ASSET_NAME` de `update.py` con `REPO` y `ASSET` de `scripts/install.sh`; verificar con `uv run pytest tests/test_update.py -k install_script`.
- [x] 3.6 Documentar en `docs/Diseño técnico.md` la verificación de integridad, el reemplazo atómico y la limpieza, y añadir la cobertura de `tests/test_update.py` a la sección de pruebas; verificar que la sección enumera las garantías y que no promete rollback.

## 4. CLI, UI y menú

- [x] 4.1 Implementar el subcomando `update` en `src/kamino/cli.py`, con captura de `KaminoError` y salida 1; probar en `tests/test_cli.py` con `sys.frozen` simulado; verificar con `uv run pytest tests/test_cli.py -k update`.
- [x] 4.2 Implementar en `src/kamino/ui.py` el mensaje de inicio y el resumen de actualización con versión actual, versión nueva y ruta; probar en `tests/test_ui.py`; verificar con `uv run pytest tests/test_ui.py -k update`.
- [x] 4.3 Añadir la opción de actualización al menú de `_run_menu` y su despacho con `ctx.invoke(update)`; probar en `tests/test_cli.py`; verificar con `uv run pytest tests/test_cli.py -k menu`.
- [x] 4.4 Documentar el comando y la opción del menú en `README.md` y `docs/Diseño funcional.md`; verificar que ambos describen `kamino update`, el requisito de binario y los códigos 0/1.

## 5. Tema ANSI de la TUI

- [x] 5.1 Implementar `_default_theme()` y asignar `theme` en `KaminoRestoreApp`; añadir pruebas en `tests/test_tui.py` para el valor por defecto, `TEXTUAL_THEME=ansi-light` y `native_ansi_color`; verificar con `uv run pytest tests/test_tui.py -k theme`.
- [x] 5.2 Documentar en `README.md` y `docs/Diseño técnico.md` que la TUI usa la paleta ANSI y que `TEXTUAL_THEME` la sobrescribe; verificar que ambos documentos mencionan `ansi-dark`, `ansi-light` y la variable de entorno.

## 6. Verificación de integración

- [x] 6.1 Ejecutar `uv run ruff check .` y `uv run ruff format --check .`; verificar que ambos terminan sin errores.
- [x] 6.2 Ejecutar `uv run pytest` completo; verificar que la suite pasa y que las pruebas de actualización no acceden a la red ni al HOME real.
- [x] 6.3 Construir el binario con `uv run --no-sync python scripts/build_release.py` y verificar que incluye `certifi/cacert.pem`, que `--help`/`--version` funcionan y que `kamino update` desde una instalación de código muestra el error esperado; si falta certifi, añadir `--collect-data certifi` a `scripts/build_release.py` y repetir.

## Workflow follow-up

- Archivar el cambio después de la revisión y de que la implementación pase las comprobaciones.
