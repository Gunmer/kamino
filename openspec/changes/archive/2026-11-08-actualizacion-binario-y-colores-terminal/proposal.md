# Propuesta

## Por qué

El binario autocontenido se instala con `scripts/install.sh`, pero después no tiene una forma propia de actualizarse: la persona tiene que volver a descargar y ejecutar el instalador. Además, la TUI de restauración usa el tema truecolor por defecto de Textual, así que sus botones, bordes y estados ignoran la paleta del terminal, mientras que los diffs ya usan colores ANSI.

## Qué cambia

- Añadir el comando `kamino update`, disponible solo cuando kamino se ejecuta como binario autocontenido. Consulta la última release publicada en GitHub, compara la versión con `__version__`, descarga `kamino-arm64`, verifica tamaño y SHA-256, y reemplaza el binario en ejecución de forma atómica.
- Si la versión instalada es igual o superior a la última publicada, el comando no descarga ni escribe nada e informa de que ya está al día. Nunca degrada.
- Si la consulta, la descarga o la verificación fallan, el binario actual queda intacto y el temporal se elimina. La actualización no requiere `sudo` y no modifica configuraciones, respaldos ni instantáneas.
- Las instalaciones desde código, `uv` o `pip` no se actualizan con este comando: reciben un error que indica la vía correspondiente.
- Añadir la opción de actualización al menú que aparece al ejecutar `kamino` sin argumentos.
- La TUI de restauración usa el tema `ansi-dark` de Textual por defecto, de modo que fondo, texto y colores de énfasis provienen de la paleta ANSI del terminal. `TEXTUAL_THEME` puede seguir sobrescribiendo el tema.

## Capacidades

### Capacidades nuevas

- `self-update`: actualización segura del binario instalado desde la última release oficial de GitHub, sin reinstalar a mano ni usar `sudo`.
- `tui-ansi-theme`: uso de la paleta ANSI del terminal en la TUI de restauración, sin imponer colores truecolor propios.

### Capacidades modificadas

- Ninguna.

## Impacto

- Nuevo módulo `src/kamino/update.py` con la consulta de la release, la descarga, la verificación y el reemplazo atómico.
- Nuevo error de dominio en `src/kamino/errors.py`.
- `src/kamino/cli.py`: subcomando `update`, integración con el menú sin argumentos y manejo de códigos de salida.
- `src/kamino/ui.py`: mensajes de progreso y resumen de la actualización.
- `src/kamino/tui/app.py`: tema ANSI por defecto.
- `pyproject.toml` y `uv.lock`: dependencia `certifi` para verificar TLS sin depender de las rutas de certificados de la máquina de build.
- Pruebas nuevas o ampliadas en `tests/test_update.py`, `tests/test_cli.py`, `tests/test_tui.py` y `tests/test_ui.py`.
- Documentación en `README.md`, `docs/Diseño funcional.md` y `docs/Diseño técnico.md`.
- No cambia el formato del archivo de respaldo, el catálogo de apps, la restauración, las instantáneas ni el flujo de publicación de releases.
- Quedan fuera de alcance: actualizar instalaciones de `uv`/`pip`/Homebrew, comprobar actualizaciones automáticamente en cada arranque, conservar un binario anterior para rollback, distribuir un binario Intel, firmar o notarizar el binario y añadir una barra de progreso de descarga.
