# Tasks

## 1. Valores por defecto y núcleo

- [x] 1.1 Crear `src/kamino/defaults.py` con `catalog_dir`, `vault_dir` y `snapshots_dir`; actualizar los puntos que hoy fijan esas rutas (`catalog/loader.py`, `clone.py`, `snapshot.py`, `deploy.py` y `commands.py`) para usarlos y verificar con pruebas de las tres rutas por defecto.
- [x] 1.2 Cambiar el destino por defecto de `clone` a `~/.config/kamino/vault/clone-yyyyMMdd.tar.gz`, permitir que `write_archive` cree el directorio al escribir y conservar `--dry-run` sin creación; verificar con pruebas de clon por defecto, directorio ausente y simulación.
- [x] 1.3 Cambiar el directorio por defecto de holocrones a `~/.config/kamino/vault/snapshot/` en `snapshot.py`, `deploy.py` y `commands.py`; verificar con pruebas de holocrón por defecto, directorio ausente y despliegue con ruta explícita.
- [x] 1.4 Actualizar las pruebas existentes que esperaban el directorio actual como destino por defecto o la ruta antigua de holocrones; ejecutar `uv run pytest tests/test_clone.py tests/test_deploy.py tests/test_snapshot.py`.

## 2. Módulo de configuración de la TUI

- [x] 2.1 Crear `src/kamino/config.py` y `tests/test_config.py` con `AppConfig`, `ConfigLoadResult` (incluido `exists`), `ConfigError` y `load_config(home=None, path=None)`; verificar con pruebas de archivo ausente, válido, con claves omitidas y con secciones o claves desconocidas.
- [x] 2.2 Implementar la validación estricta de TOML, tipos, cadenas no vacías y rutas absolutas o con `~`; verificar con pruebas de TOML inválido, `[ui]`, `clone.force`, tipo incorrecto, cadena vacía, ruta relativa y ruta existente que no es directorio.
- [x] 2.3 Conservar valores parciales legibles cuando el TOML es válido pero alguna clave es inválida, y usar defaults en los campos no legibles; verificar con pruebas de `catalog.dir` válido junto a `clone.output_dir` relativo y de TOML sintácticamente inválido.
- [x] 2.4 Implementar `save_config` con las tres claves, serialización TOML escapada, creación de `~/.config/kamino`, temporal en el mismo directorio y `os.replace`; verificar con pruebas de archivo nuevo, reemplazo, fallo de escritura sin pérdida del anterior y ausencia de creación de los directorios configurados.

## 3. Pantalla de Configuración

- [x] 3.1 Crear `src/kamino/tui/config.py` con `ConfigScreen` y sus estilos siguiendo `settings-screen.md`: tres campos, valores efectivos, pista contextual, estado de error, `Ctrl+S` para guardar y `Esc` para volver; verificar con pruebas headless de apertura, edición y cancelación.
- [x] 3.2 Añadir la opción Configuración al menú después de Actualizar, sin atajo de teclado, y actualizar textos y barra de atajos; verificar que las flechas y `Enter` abren la pantalla, que `c`, `d`, `u` y `q` conservan su comportamiento y que el menú de cinco opciones sigue siendo legible a 80x24.
- [x] 3.3 Validar al guardar que las tres rutas no estén vacías, sean absolutas o empiecen por `~` y que una ruta existente sea un directorio; verificar con pruebas de error de validación sin escritura.
- [x] 3.4 Tras guardar correctamente, actualizar `KaminoApp.config` y aplicar los valores de inmediato; verificar con una prueba de guardado seguida de un flujo que use el nuevo directorio de holocrones sin reiniciar.

## 4. Arranque de la TUI y flujos

- [x] 4.1 Añadir `config` y `config_error` a `KaminoApp` y cargar la configuración una sola vez desde `run_app` o desde el punto de entrada de cada flujo directo; verificar con pruebas de modo menú y modo directo.
- [x] 4.2 Si la configuración es inválida, abrir `ConfigScreen` con el error y los valores parciales; tras guardar, volver al menú o reconstruir el flujo directo con los valores nuevos; tras cancelar, salir con código 1 sin escribir. Verificar con pruebas de menú, `clone` directo y `deploy` directo.
- [x] 4.3 Añadir `catalog_dir` y `default_output_dir` a `CloneState`, usar el catálogo configurado y proponer en C2 el destino configurado o `vault/`; permitir un padre ausente solo para esa ruta propuesta y crearlo al iniciar la clonación. Verificar con pruebas de C2, catálogo configurado y directorio ausente creado al clonar.
- [x] 4.4 Añadir `catalog_dir` a `DeployState`, pasar `snapshots_dir` y hacer que `start_clone_from_menu` y `start_deploy_from_menu` usen `app.config`; verificar con pruebas de menú con catálogo, salida y holocrón configurados.
- [x] 4.5 Verificar que el esquema solo admite las tres claves y que `[ui]` se rechaza; ejecutar `uv run pytest tests/test_tui.py`.

## 5. CLI no interactiva

- [x] 5.1 Confirmar que `cli.py` y `commands.py` no importan `config.py` y que usan los defaults de `defaults.py`; verificar con una prueba que crea un `config.toml` distinto y comprueba que `kamino clone` y `kamino deploy` no lo aplican.
- [x] 5.2 Verificar que la CLI no crea ni modifica `config.toml` y que un archivo inválido no altera `clone`, `deploy` ni `update`; ejecutar `uv run pytest tests/test_cli.py tests/test_integration.py`.

## 6. Documentación

- [x] 6.1 Documentar en el README `~/.config/kamino/config.toml`, las tres claves, los defaults bajo `vault/`, la pantalla de Configuración, `Ctrl+S` y la separación CLI/TUI; verificar que el ejemplo es TOML válido y coincide con `AppConfig`.
- [x] 6.2 Documentar los cambios incompatibles: destino por defecto del clon, nuevo directorio de holocrones y holocrones anteriores sin migración; verificar que las instrucciones indican cómo usar `--output` y cómo desplegar un holocrón antiguo.
- [x] 6.3 Documentar que `--force`, `--yes`, `--dry-run`, `ui.plain` y `ui.theme` no se configuran; verificar que la documentación no promete claves no admitidas.
- [x] 6.4 Actualizar las secciones de menú, estructura del paquete, valores por defecto y holocrones de `docs/Diseño funcional.md` y `docs/Diseño técnico.md`; verificar que las rutas por defecto son las nuevas y que cualquier mención a la ruta antigua se presenta como holocrones existentes sin migrar.

## 7. Verificación final

- [x] 7.1 Ejecutar `uv run ruff check .` y `uv run ruff format --check .`; corregir cualquier incidencia sin cambiar el alcance.
- [x] 7.2 Ejecutar la suite completa con `uv run pytest`; verificar que todas las pruebas usan HOMEs temporales y que no leen el HOME real.
- [x] 7.3 Probar manualmente con un HOME temporal: TUI sin archivo, guardado desde la pantalla, configuración inválida y CLI con un archivo distinto; comprobar que el comportamiento coincide con las specs.
- [x] 7.4 Ejecutar `openspec validate add-user-configuration --strict` y confirmar que el cambio sigue siendo válido.

## Workflow follow-up

- Archivar el cambio cuando la revisión y las comprobaciones del proyecto estén satisfechas.
- Verificar el resultado archivado y actualizar las specs principales.
