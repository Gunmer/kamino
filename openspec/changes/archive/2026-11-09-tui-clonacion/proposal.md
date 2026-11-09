# Proposal

## Why

Kamino tiene hoy tres experiencias de usuario distintas: el menú y la selección de apps de `clone` usan `questionary`, la revisión de conflictos de `deploy` usa una TUI de `textual` y `update` imprime por consola. Esa división impide una navegación coherente, deja fuera la clonación y la actualización de la TUI y mantiene un vocabulario mezclado entre “local”, “snapshot”, “mezcla” e “idéntico”. Este cambio unifica la interfaz y completa los flujos de clonar, desplegar y actualizar con una sola app de `textual`, conservando el orden de seguridad y los códigos de salida de la CLI.

## What Changes

- **Una sola app `textual`**: menú principal, flujo de clonar (C1-C4), flujo de desplegar (D0-D5), flujo de actualizar (U1) y pantallas transversales (ayuda contextual, error común, terminal pequeña, diálogos).
- **Menú principal** con wordmark, separador de cápsulas, frase de rotación fija por sesión, opciones Clonar/Desplegar/Actualizar/Abandonar y línea de última misión. `q` solo sale desde el menú.
- **Flujo de clonar** con selección de apps instaladas y recuento en segundo plano, destino validado en vivo, revisión del plan, progreso por archivos y cancelación hasta el renombrado final.
- **Flujo de desplegar** con validación de manifiesto en D0, selección de apps instaladas en D1, preparación cancelable, conflictos D2, detalle por líneas D3, confirmación D4 y resumen D5. Los archivos idénticos pasan a ser “clones perfectos” y ni ellos ni las apps no instaladas cuentan como omitidos; solo los conflictos conservados en el original, los no soportados y los `--apps` no cumplidos producen código 3.
- **Flujo de actualizar** con estados de consulta, descarga y verificación, reemplazo y resultado, con cancelación hasta el reemplazo y petición de reinicio tras actualizar.
- **CLI**: nueva opción `--force`/`-f` en `clone`; nombre por defecto `clone-yyyyMMdd.tar.gz` con fecha local; `--dry-run`, `--yes`, `--apps` y `--output` omiten las pantallas que correspondan. Se conserva la vía no interactiva de `deploy --dry-run` y `deploy --yes`; `clone --dry-run` con TTY termina en la revisión C3 sin acción de clonar.
- **Vocabulario visible**: “original” en lugar de “local”, “holocrón de seguridad” en lugar de “snapshot”, “híbridos” en lugar de “mezclas”, “clones perfectos” en lugar de “idénticos” y “sin base en este Mac” para apps no instaladas. Los nombres internos y el formato del archivo no cambian.
- **Colores**: solo paleta ANSI del terminal, `NO_COLOR` en monocromo, wordmark en cian/azul, original en amarillo con `◉` e icono más palabra en todos los estados.
- **Dependencias**: se retira `questionary`; se conserva `rich` para la salida no interactiva.

**BREAKING**: el nombre por defecto del clon cambia de `kamino-<equipo>-<fecha>.tar.gz` a `clone-yyyyMMdd.tar.gz`; un clon existente sin `--force` pasa a ser error; los archivos idénticos y las apps no instaladas dejan de contar como omitidos y de producir código 3; el menú y la selección de apps de `clone` dejan de usar `questionary`.

## Capabilities

### New Capabilities

- `tui-navigation`: app única, apertura con TTY, menú principal, navegación y conservación de estado, numeración de pasos, ayuda contextual, error común, terminal pequeña, `Ctrl+C`, frases, logo y última misión.
- `clone-tui`: selección de apps para clonar, destino y reemplazo, revisión del plan, progreso y cancelación, resumen y copia de ruta, y dry-run en la TUI.

### Modified Capabilities

- `cli-options`: `--force`/`-f`, nombre por defecto, reglas de apertura de la TUI con argumentos, simulación no interactiva de `deploy`, y resumen/código 3 por categorías.
- `clone-safety`: aviso de secretos en la TUI, confinamiento de lectura frente a destinos fuera del HOME, reemplazo atómico con `--force` y cancelación sin efectos.
- `deploy-conflicts`: validación de manifiesto en D0, apps no instaladas informadas, conteos y acciones de D2, detalle D3, confirmación y resumen D4/D5, holocrón, cancelación y despliegue parcial.
- `self-update`: progreso y cancelación en U1, taxonomía de errores y resultados de la TUI.
- `tui-ansi-theme`: alcance a toda la TUI, `NO_COLOR`, semántica de color e iconos y convivencia con `TEXTUAL_THEME`.

## Impact

- **Código**: `src/kamino/cli.py` (flujos fuera de Typer), nuevo módulo de flujos y `texts.py`; `src/kamino/tui/` (app, pantallas, widgets, estilos); `src/kamino/clone.py`, `deploy.py`, `archive.py`, `plan.py`, `conflicts.py`, `snapshot.py`, `update.py` y `ui.py` para progreso, cancelación, categorías y textos.
- **Dependencias**: se elimina `questionary` de `pyproject.toml` y `uv.lock`; se mantienen `textual`, `rich`, `typer`, `pydantic` y `certifi`.
- **Empaquetado**: datos de `textual`, catálogo y estilos en PyInstaller; si los textos o el logo son módulos Python no se añaden datos nuevos.
- **Pruebas**: `tests/test_cli.py`, `tests/test_ui.py`, `tests/test_clone.py`, `tests/test_deploy.py`, `tests/test_tui.py` y nuevas pruebas de flujos, progreso, cancelación, tamaños 80x24 y menores, `NO_COLOR` y portapapeles.
- **Documentación**: README, `docs/Diseño funcional.md`, `docs/Diseño técnico.md` y las specs vivas afectadas.
- **Compatibilidad**: el formato `manifest.json`/`tar.gz` no cambia; los clones y holocrones antiguos siguen siendo legibles. El id sintético `snapshot` se conserva.
