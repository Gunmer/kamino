# Tareas: ampliar el catálogo de desarrollo agéntico

## 1. Confinamiento seguro al HOME

- [x] 1.1 Extender `src/kamino/paths.py` con acceso seguro anclado al HOME que rechace enlaces intermedios y conserve la validación de rutas y enlaces finales; verificar en `tests/test_paths.py` rutas normales, absolutas, con `..` y enlaces a un directorio externo dentro de `tmp_path`.
- [x] 1.2 Usar el acceso seguro en la planificación y en las lecturas de `archive.py`, incluidos el cálculo del checksum y la reapertura al empaquetar; probar en `tests/test_plan.py` y `tests/test_archive.py` que sustituir un directorio por un enlace después del plan no lee datos externos ni deja un respaldo válido inseguro.
- [x] 1.3 Usar el mismo confinamiento para clasificar destinos, crear instantáneas y hacer reemplazos atómicos en `restore.py` y `snapshot.py`; probar en `tests/test_restore.py` y `tests/test_snapshot.py` el rechazo de enlaces intermedios antes de escribir y la sustitución de un directorio durante la operación.
- [x] 1.4 Mantener la verificación íntegra del archivo antes de cualquier escritura de destino, la instantánea previa y `dry_run` sin escrituras; probar un archivo corrupto y un respaldo válido de formato 1 anterior, incluidos destinos ahora excluidos cuya instantánea debe quedar completa, con HOME y directorios de instantáneas temporales.
- [x] 1.5 Documentar en `docs/Diseño técnico.md` el límite del HOME, los enlaces finales admitidos y el tratamiento de cambios concurrentes; contrastar la explicación con las pruebas de rutas y restauración.

## 2. Modelo, planificación y detección

- [x] 2.1 Ampliar `catalog/models.py` con rutas y patrones opcionales acotados, y un marcador de instalación relativo al HOME; verificar en `tests/test_catalog.py` las definiciones con solo rutas opcionales, el rechazo de definiciones sin entradas y la compatibilidad de definiciones anteriores y del usuario.
- [x] 2.2 Expandir las rutas y patrones opcionales sin seguir enlaces ni modificar la semántica de `paths` heredados; probar en `tests/test_plan.py` que una variante opcional ausente no se omite, una ruta obligatoria ausente sí se omite y un patrón inseguro se rechaza.
- [x] 2.3 Deduplicar por aplicación y ruta relativa las operaciones solapadas; verificar en `tests/test_plan.py` y `tests/test_archive.py` que una ruta declarada dos veces produce un único miembro verificable y que dos aplicaciones pueden conservar su propia entrada.
- [x] 2.4 Propagar el HOME inyectado a la detección por marcador utilizada por CLI, respaldo y restauración; probar en `tests/test_detect.py`, `tests/test_backup.py` y `tests/test_restore.py` que el marcador no se consulta en el HOME real y que no se ejecutan las herramientas detectadas.
- [x] 2.5 Mantener la sustitución completa por id del catálogo del usuario y las reglas de selección de instaladas; verificar en `tests/test_catalog.py` y `tests/test_cli.py` que una definición personalizada no pierde precedencia ni el confinamiento de rutas.

## 3. Privacidad y aviso previo

- [x] 3.1 Aplicar exclusiones de credenciales conocidas, historiales, sesiones, cachés y registros a las nuevas definiciones y referencias automáticas antes de leer contenidos; probar `.env`, `.env.*`, `auth.json`, `secrets.yaml`, `.ssh` y `hosts.yml` de GitHub CLI con archivos señuelo en HOME temporales.
- [x] 3.2 Mantener intactos los archivos de configuración permitidos aunque contengan secretos incrustados; verificar en `tests/test_backup.py` y `tests/test_archive.py` los bytes originales y que una referencia explícita a una credencial excluida se informa sin leerla.
- [x] 3.3 Mostrar en `cli.py` y `ui.py` un aviso en español antes de respaldar, con y sin TTY y en simulación, sin pedir confirmación ni mostrar valores; verificar con dobles en `tests/test_cli.py` que alias, selección y códigos 1 y 3 se conservan.
- [x] 3.4 Documentar en `README.md` que los respaldos y las instantáneas no están cifrados, que los ajustes pueden contener secretos y qué archivos se excluyen de nuevos respaldos; integrar la documentación sin descartar cambios preexistentes y comprobar que los ejemplos coinciden con la CLI.

## 4. Catálogo de agentes y herramientas agénticas

- [x] 4.1 Añadir las definiciones globales de OpenCode, Claude Code, Codex y Gemini CLI según la matriz de `design.md`; verificar con HOME y PATH simulados configuración, temas y atajos, perfiles de Codex y exclusión de autenticación.
- [x] 4.2 Añadir Copilot CLI, Pi, Aider y Goose según la matriz; probar con datos sintéticos sus ajustes, temas o atajos disponibles y que `auth.json`, `secrets.yaml` y el estado de autenticación no se copian.
- [x] 4.3 Añadir Gentle Shell, Gentle AI, GGA, OpenSpec y Caveman; probar sus ubicaciones globales permitidas, la distinción de Gentle Shell aislado o enlazado y una sola entrada para `cave`/`caveman`, sin copiar instalaciones ni sesiones.
- [x] 4.4 Verificar en `tests/test_catalog.py` y `tests/test_backup.py`, mediante casos parametrizados, que cada definición nueva se carga, detecta su ejecutable simulado sin ejecutarlo e incluye sus archivos globales representativos solo cuando está instalada.
- [x] 4.5 Documentar en `README.md` el lote de agentes, sus ids y las limitaciones de archivos mixtos, configuraciones locales y ubicaciones alternativas; comprobar que los ids y ejemplos coinciden con el catálogo.

## 5. Terminales, Oh My Zsh y Herdr

- [x] 5.1 Añadir las definiciones de LazyGit, Micro, Navi y Glow con sus ubicaciones globales documentadas; probar con HOME temporales los ajustes, temas y atajos disponibles, sin incluir cachés ni respaldos del editor.
- [x] 5.2 Añadir bat, eza, fd y GitHub CLI y ampliar Git con `.config/git/config` para configuraciones compartidas con delta; probar temas, archivos de preferencias y exclusión de `hosts.yml` sin duplicar una entrada para delta.
- [x] 5.3 Añadir Oh My Zsh con `zsh` y el marcador del framework, sus archivos globales y temas; probar que `zsh` sin framework no lo detecta y que se excluye `custom/plugins`, incluso cuando contiene repositorios `.git`.
- [x] 5.4 Mantener la configuración global de Herdr, admitir `HERDR_CONFIG_PATH` y XDG dentro del HOME y excluir credenciales de plugins; probar la restauración íntegra de `[theme]`, `[theme.custom]`, `[keys]` y `[[keys.command]]` y el rechazo de ubicaciones externas.
- [x] 5.5 Documentar en `README.md` los ids de estas herramientas, la reinstalación de plugins de Oh My Zsh y los límites de ubicaciones personalizadas; verificar los ejemplos con las definiciones y pruebas del grupo.

## 6. Ghostty y sus referencias

- [x] 6.1 Ampliar la definición de Ghostty con `config.ghostty` y temas personalizados, conservando como obligatorias las rutas históricas `config`; probar las cuatro variantes globales, el XDG dentro del HOME y la semántica previa de omisiones.
- [x] 6.2 Implementar el descubrimiento específico de `config-file` y temas, con rutas relativas o absolutas dentro del HOME, archivos opcionales con `?`, referencias anidadas, ciclos y límites; verificar cada caso con configuraciones sintéticas sin ejecutar Ghostty ni un shell.
- [x] 6.3 Aplicar el confinamiento y las exclusiones antes de abrir referencias, e informar de archivos requeridos externos o sensibles; probar enlaces intermedios, rutas fuera del HOME, `.env`, temas claro/oscuro y referencias repetidas con `tests/test_plan.py` y `tests/test_archive.py`.
- [x] 6.4 Advertir al planificar y restaurar cuando un archivo de Ghostty conserve una ruta absoluta de otro HOME, sin reescribirlo; probar restauración real y `dry_run`, bytes idénticos, ausencia de valores secretos en el aviso y compatibilidad con respaldos de formato 1.
- [x] 6.5 Documentar en `README.md` los temas y atajos de Ghostty, el alcance de las referencias y la posible necesidad de ajustar rutas absolutas o permisos de accesibilidad; comprobar los ejemplos frente a las pruebas del grupo.

## 7. Verificación integrada

- [x] 7.1 Ampliar `tests/test_integration.py` con dos HOME falsos, PATH y aplicaciones simuladas para cubrir agentes, Herdr, Oh My Zsh y Ghostty; verificar el ciclo completo, las exclusiones, los avisos y la omisión de aplicaciones no instaladas.
- [x] 7.2 Verificar que el paquete y el binario autocontenido incluyen todas las nuevas definiciones TOML y que `uv run kamino --help` conserva los comandos y alias existentes, sin depender de configuraciones personales.
- [x] 7.3 Ejecutar, en este orden, `uv run ruff check .`, `uv run ruff format --check .` y `uv run pytest`; comprobar que las tres verificaciones terminan correctamente.
