# AGENTS.md

## Configuración y verificación

- Usa Python 3.11.4 o superior (`tarfile` usa `filter="data"` al extraer). El producto está dirigido a macOS, aunque CI también ejecuta las pruebas en Ubuntu.
- Instala las dependencias con `uv sync --all-extras` y ejecuta la CLI con `uv run kamino --help`.
- Antes de terminar, ejecuta las mismas comprobaciones que CI, en este orden: `uv run ruff check .`, `uv run ruff format --check .` y `uv run pytest`.
- Ejecuta una prueba concreta con `uv run pytest tests/test_deploy.py::test_name` (o indica otro archivo/nodo).

## Arquitectura

- La distribución se llama `kamino-cli`, pero tanto el paquete de importación como el ejecutable se llaman `kamino`. Los puntos de entrada de Typer son `src/kamino/cli.py` y `src/kamino/__main__.py`.
- Mantén `cli.py` como una capa fina: solo argumentos, interacción, salida y códigos de terminación. La lógica de negocio corresponde a `clone.py` y `deploy.py`; la presentación en terminal, a `ui.py`. Los casos de uso no deben imprimir ni solicitar información.
- Las definiciones de aplicaciones integradas están en `src/kamino/catalog/apps/*.toml`. `catalog/loader.py` las carga mediante `importlib.resources` y las combina con los TOML del usuario en `~/.config/kamino/apps/`; las definiciones del usuario reemplazan las integradas con el mismo id.
- `archive.py` es responsable del manifiesto versionado y del formato del archivo; `plan.py` gestiona la planificación de la clonación y `deploy.py` define su propio plan de despliegue clasificado.

## Invariantes de seguridad y compatibilidad

- Las pruebas deben usar directorios HOME temporales e inyectar `home`, `catalog`, `application_dirs` y `snapshots_dir` cuando corresponda. Las pruebas de clon/despliegue nunca deben leer ni modificar el HOME real del desarrollador ni `/Applications`; en las pruebas de CLI, reemplaza detección y casos de uso con `monkeypatch`.
- Conserva el orden del despliegue: validar el manifiesto, verificar el archivo completo antes de escribir en cualquier destino, crear una instantánea de los destinos que cambiarán y, por último, aplicar reemplazos atómicos. `dry_run` no debe escribir nada.
- Las rutas del archivo son rutas POSIX relativas al HOME. Rechaza rutas absolutas y `..`; solo se admiten archivos regulares y enlaces simbólicos relativos. Si modificas el manifiesto, considera la compatibilidad del formato y añade pruebas del archivo y del ciclo de serialización/despliegue.
- El despliegue no interactivo requiere `--yes`, salvo con `--dry-run`. El código de salida 1 de la CLI indica error/cancelación; el 3 indica éxito parcial con elementos omitidos.
- Escribe en español todo el contenido Markdown (`.md`) del repositorio y todo el texto visible para el usuario en la CLI. Escribe en inglés todo el código, incluidos nombres, identificadores y comentarios.
- Si cambias la versión manualmente, mantén sincronizados `pyproject.toml:project.version` y `src/kamino/__init__.py:__version__`; semantic-release está configurado para actualizar ambos.
