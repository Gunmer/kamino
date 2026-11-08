# Propuesta: ampliar el catálogo de desarrollo agéntico

## Why

**Motivación:**

Kamino solo incluye cuatro definiciones integradas y no cubre las personalizaciones globales de los agentes y herramientas de terminal usados para desarrollar. Además, Ghostty admite archivos y temas que el catálogo actual no contempla, mientras que las referencias adicionales y los directorios de configuración de plugins introducen riesgos de privacidad y de escape del HOME.

## What Changes

**Cambios propuestos:**

- Ampliar el catálogo con configuraciones globales de agentes CLI/TUI y herramientas de desarrollo, detectadas por sus ejecutables sin ejecutarlos. El lote inicial propuesto incluye OpenCode, Claude Code, Codex, Gemini CLI, Copilot CLI, Pi, Aider, Goose, Gentle Shell, Gentle AI, GGA, OpenSpec, Caveman, LazyGit, Micro, Navi, Glow, bat, eza, fd y GitHub CLI. Las configuraciones compartidas con Git, como las de delta, se cubren mediante Git cuando corresponda.
- Incorporar Oh My Zsh como configuración global, detectando el framework y no solo el ejecutable `zsh`. Incluir sus ajustes, atajos y temas personales, pero no las carpetas de plugins, que deberán reinstalarse por separado.
- Completar Ghostty con las variantes actuales y antiguas de su archivo de configuración, sus temas personalizados y los archivos de configuración o temas globales referenciados dentro del HOME. Conservar los archivos sin reescribirlos y advertir si una ruta absoluta puede requerir ajuste tras la restauración.
- Comprobar que se preservan los temas y atajos de Herdr presentes en su configuración global. Excluir credenciales independientes y sesiones, incluidos archivos `.env` en la configuración de plugins, sin copiar sus programas instalados.
- Excluir credenciales separadas, historiales y cachés de las nuevas definiciones y de las rutas descubiertas; avisar antes del respaldo de que el `tar.gz` no está cifrado y los archivos de ajustes todavía pueden contener secretos incrustados.
- Proteger la planificación, la instantánea y la restauración frente a rutas que atraviesen enlaces simbólicos fuera del HOME; evitar duplicados y distinguir las variantes opcionales ausentes de las omisiones reales, sin debilitar los códigos de salida existentes.

La lista es una propuesta concreta para revisar, no una afirmación de que todas las herramientas estén instaladas. Quedan fuera los archivos propios de proyectos, la instalación de ejecutables o plugins, el cifrado, la reescritura de configuraciones y el respaldo de bases de datos de sesiones o memoria como la de Engram. Las referencias explícitas desde la configuración global de Ghostty se incluyen solo como archivos individuales dentro del HOME, no como un descubrimiento de repositorios.

## Capacidades

### Capacidades nuevas

- `global-app-config`: detección y respaldo de la configuración global, temas y atajos de las aplicaciones y herramientas cubiertas, incluidos Oh My Zsh y Herdr.
- `ghostty-config-references`: descubrimiento seguro y conservación de configuraciones y temas referenciados por Ghostty, con advertencias de portabilidad.
- `backup-safety`: exclusión de credenciales conocidas, aviso de privacidad y confinamiento seguro de rutas en respaldo y restauración.

### Capacidades modificadas

- Ninguna. `cli-options` conserva el significado de sus opciones, alias, simulaciones y códigos de salida; el aviso de privacidad es información adicional.

## Impacto

- `src/kamino/catalog/apps/`, `catalog/models.py`, `catalog/loader.py` y `detect.py`: definiciones, rutas opcionales y detección del framework, respetando la prioridad del catálogo del usuario.
- `plan.py`, `paths.py`, `backup.py`, `archive.py`, `snapshot.py` y `restore.py`: referencias de Ghostty, exclusiones, deduplicación y protección del HOME a lo largo del ciclo completo.
- `cli.py` y `ui.py`: avisos visibles en español sin incorporar lógica de negocio a la interfaz.
- Pruebas de catálogo, detección, planificación, archivo, respaldo, restauración y CLI con HOME y ejecutables simulados; documentación de cobertura y límites. No se prevé instalar herramientas ni agregar dependencias de ejecución.
