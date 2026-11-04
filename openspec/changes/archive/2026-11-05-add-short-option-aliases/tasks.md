# Tareas: alias cortos para las opciones de la CLI

## 1. Ayuda y versión

- [x] 1.1 Añadir `-V` a `--version` en `src/kamino/cli.py` y probar en `tests/test_cli.py` que ambas formas producen la misma versión, código 0 y ningún despacho del menú ni de los casos de uso.
- [x] 1.2 Configurar `-h` junto a `--help` y probar ambas formas en el nivel principal, `backup` y `restore`, incluso sin el argumento posicional; comprobar sus alias visibles y que no se ejecutan los casos de uso. Verificar en la ayuda que las opciones de autocompletado mantienen sus nombres sin invocar su instalación.
- [x] 1.3 Documentar en `README.md` los alias de ayuda y versión y comprobar que los ejemplos concuerdan con los comandos cubiertos por las pruebas de CLI.

## 2. Opciones de respaldo

- [x] 2.1 Añadir `-a` y `-o` a las declaraciones de `backup` y probar que las formas cortas y largas seleccionan las mismas aplicaciones y transmiten la misma ruta al caso de uso simulado.
- [x] 2.2 Añadir `-n` y `-v` a `backup`; probar equivalencia con `--dry-run` y `--verbose`, la transmisión de `dry_run=True` y del detalle, y el código 3 ante aplicaciones omitidas. Conservar la prueba existente de simulación sin archivo de salida.
- [x] 2.3 Documentar los alias de respaldo y un ejemplo de simulación en `README.md`; comprobar que los nombres y valores coinciden con los cubiertos por las pruebas aisladas.

## 3. Opciones de restauración

- [x] 3.1 Añadir `-n`, `-y` y `-v` a `restore`; probar que las formas cortas y largas transmiten el mismo archivo posicional, selección, simulación y nivel de detalle a los dobles correspondientes.
- [x] 3.2 Probar que sin terminal la restauración sin confirmación ni simulación termina con código 1 sin llamar al caso de uso, que `-y` permite continuar y que `-n` permite simular sin confirmación. Probar también la selección interactiva con `-n` y el código 3 ante elementos omitidos; conservar la prueba existente de simulación sin escrituras ni instantánea.
- [x] 3.3 Documentar los alias de restauración y ejemplos con `-n` y `-y` en `README.md`; comprobar que reflejan las mismas reglas que las pruebas de CLI.

## 4. Corrección de la prueba de publicación y verificación integrada

- [x] 4.1 Ejecutar `uv run pytest tests/test_cli.py` y comprobar que las pruebas de CLI pasan usando dobles, sin acceder al HOME real ni a `/Applications`.
- [x] 4.2 En `tests/test_release.py`, inicializar el repositorio de prueba con un historial sintético que termine en `v0.1.0` y versiones iniciales coherentes en la etiqueta, `pyproject.toml`, `src/kamino/__init__.py` y `uv.lock`, sin copiar el historial ni la versión reales. Conservar las plantillas y la configuración de publicación, el HOME temporal y el aislamiento de los remotos.
- [x] 4.3 Añadir una regresión que compruebe la coherencia de la base y su independencia cuando el historial real simulado ya contiene `v0.2.0` o la versión real difiere. Mantener las comprobaciones existentes de versión, historial, notas, dependencias y repetición; verificar que la nota incompatible pertenece a la nueva sección y que al repetir no se crean etiquetas nuevas.
- [x] 4.4 Ejecutar `uv run pytest tests/test_release.py` y comprobar que pasan las pruebas de publicación, incluida la regresión, sin modificar los archivos reales ni contactar remotos.
- [x] 4.5 Ejecutar en este orden `uv run ruff check .`, `uv run ruff format --check .` y `uv run pytest`; comprobar que las tres verificaciones finalizan correctamente.
