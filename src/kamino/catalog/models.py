"""Modelos del catálogo: validación de las definiciones TOML.

Esquema de una definición (diseño técnico §4.1):

- `id`: único, en minúsculas y guiones.
- `bundle_id`: detecta apps con `.app` por su `CFBundleIdentifier`.
- `command`: detecta herramientas de terminal sin `.app` (ej. `git`).
- `search_paths`: rutas adicionales donde buscar el binario de `command`.
- `paths`: rutas relativas al home, archivos o carpetas.
- `exclude`: patrones glob descartados al expandir `paths`.

Pendiente: paso 2 del plan de implementación (definir `AppDefinition` con pydantic).
"""

from __future__ import annotations
