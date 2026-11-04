# Propuesta: alias cortos para las opciones de la CLI

## Motivación

Las opciones actuales de Kamino solo disponen de nombres largos, lo que hace más laboriosa su escritura en la terminal. Añadir alias breves permite usar los comandos con menos pulsaciones sin romper los scripts existentes.

La verificación completa del cambio también está bloqueada por una prueba de publicación preexistente: su repositorio de prueba mezcla una versión inicial simulada con el historial y la versión reales del proyecto.

## Cambios propuestos

- Añadir `-a` para `--apps` y `-o` para `--output` en `backup`.
- Añadir `-n` para `--dry-run` y `-v` para `--verbose` en `backup` y `restore`.
- Añadir `-y` para `--yes` en `restore`, `-V` para `--version` en el nivel principal y `-h` para `--help` en el nivel principal y ambos comandos.
- Mostrar los alias junto con sus formas largas en la ayuda y documentar ejemplos de uso.
- Conservar los comportamientos actuales de selección, simulación, confirmación y códigos de salida. Las opciones automáticas `--install-completion` y `--show-completion`, los nombres de los comandos y el argumento posicional de restauración quedan fuera del alcance.
- Aislar los datos del repositorio de prueba de publicación para corregir el fallo existente y evitar que reaparezca al cambiar las versiones reales, sin debilitar las verificaciones de publicación.

## Capacidades

### Capacidades nuevas

- `cli-options`: reconocimiento y presentación de las opciones propias de la CLI y sus alias cortos.
- `release-verification`: verificación aislada y coherente de la generación de publicaciones, historial y notas.

### Capacidades modificadas

- Ninguna. El proyecto no tiene especificaciones principales existentes para estas capacidades.

## Impacto

- `src/kamino/cli.py`: declaraciones de opciones y configuración de ayuda de Typer.
- `tests/test_cli.py`: pruebas de equivalencia, ayuda, seguridad y compatibilidad mediante dobles aislados.
- `tests/test_release.py`: datos sintéticos coherentes y pruebas de regresión para la verificación de publicaciones.
- `README.md`: referencia de alias y ejemplos. No se prevén nuevas dependencias ni cambios en los casos de uso, el formato de respaldo o la configuración de publicación real.
