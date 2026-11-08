# Proposal

## Why

Restaurar un respaldo sobrescribe sin aviso cualquier archivo de configuración que exista y difiera. En una máquina con ajustes propios, eso puede borrar cambios locales valiosos y dejar una configuración mezclada o rota sin que la persona lo haya visto. El respaldo ya conserva el contenido completo y el snapshot previo protege el estado anterior, pero falta el paso de revisión: mostrar los conflictos y decidir qué se restaura y qué se conserva.

## What Changes

- Detectar conflictos de contenido entre el destino y el respaldo, incluidos valores cambiados, líneas añadidas y líneas borradas.
- **BREAKING** La restauración interactiva pasa a una TUI de `textual` con selección de apps, tabla de conflictos, detalle por bloques y confirmación final. `--yes` conserva el comportamiento actual de aplicar todo sin preguntar.
- Permitir decidir por bloque de cambio (hunk): cada bloque se toma entero del respaldo o se conserva entero del local; no se mezclan líneas sueltas.
- Componer el archivo resultante, conservar el estilo de final de línea local y aplicar el modo del respaldo solo si el contenido cambia.
- Validar el resultado de TOML, JSON, INI y plist con la biblioteca estándar; bloquear si ambos originales eran válidos y la mezcla no lo es.
- Tratar binarios, archivos de más de 5 MB y enlaces con enlace como conflictos de archivo entero (Aplicar/Omitir), sin diff de líneas.
- Abortar toda la restauración antes de escribir si un destino cambió mientras la TUI estaba abierta.
- Añadir `restore --apps` (con `-a` por consistencia con `backup`) para seleccionar apps sin TUI, y hacer que `--dry-run` vuelque los diffs a stdout en lugar de abrir la TUI.
- Separar en el resumen los archivos restaurados, mezclados y omitidos; el código 3 sigue reservado a los omitidos.

## Capabilities

### New Capabilities

- `restore-conflicts`: detección y resolución interactiva de conflictos durante la restauración, con selección de apps en la misma TUI, decisiones por bloque, composición del archivo, validación, protecciones de concurrencia y resumen.

### Modified Capabilities

- `cli-options`: se añade `restore --apps` (con alias `-a`), se ajusta la simulación para que no abra la TUI de selección y se documenta que `--yes` sigue aplicando todo sin preguntar.

## Impact

- Código: `src/kamino/restore.py` (sesión de restauración y modelos de conflicto), un nuevo módulo de diff/mezcla/validación, una nueva TUI en `src/kamino/tui/`, `src/kamino/cli.py` y `src/kamino/ui.py`.
- Dependencias: `textual` como dependencia directa; hay que actualizar `pyproject.toml` y `uv.lock` y verificar el empaquetado con PyInstaller.
- Pruebas: `tests/test_restore.py`, nuevas pruebas de diff/mezcla/validación y de la TUI, `tests/test_cli.py` y `tests/test_integration.py`.
- Documentación: `docs/Diseño funcional.md` (§3.7), `docs/Diseño técnico.md` (§5.2 y §7) y `README.md`.
- Compatibilidad: el formato del archivo de respaldo no cambia; los respaldos existentes se pueden restaurar. El comportamiento interactivo cambia; `--yes` y `--dry-run` conservan una vía no interactiva.
