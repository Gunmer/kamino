# Tasks

## 1. Núcleo de clonación

- [x] 1.1 Añadir motivos a las rutas omitidas del plan (`skipped` con ruta y causa) y adaptar los consumidores de `Plan`; verificar con `uv run pytest tests/test_plan.py tests/test_clone.py`.
- [x] 1.2 Implementar el recorrido de conteo de C1 compartiendo expansión, exclusiones y `SafeHome` con el plan, incluido Ghostty; verificar con una prueba de paridad entre conteo y plan y con `uv run pytest tests/test_plan.py`.
- [x] 1.3 Separar `plan_clone` y `write_clone` con progreso por archivos y cancelación cooperativa; verificar con pruebas de plan sin escritura, progreso y cancelación que dejen el clon anterior intacto.
- [x] 1.4 Cambiar el nombre por defecto a `clone-yyyyMMdd.tar.gz` con fecha local inyectable y añadir `force` con revalidación antes del renombrado; verificar nombre, conflicto sin `force`, reemplazo con `force` y error/cancelación con temporal limpio en `tests/test_clone.py` y `tests/test_archive.py`.
- [x] 1.5 Actualizar la documentación de clonación en README y `docs/Diseño funcional.md` con el nombre nuevo, `--force` y la cancelación; verificar que los comandos documentados coinciden con la CLI.

## 2. Núcleo de despliegue

- [x] 2.1 Endurecer `read_manifest`: envolver errores de `tarfile`/OS, exigir un único `manifest.json` regular, limitar el manifiesto a 4 MiB y acotar el número de miembros; verificar con pruebas de tar corrupto, duplicado, gigante y formato más nuevo.
- [x] 2.2 Añadir las categorías `unchanged`, `not_installed` y `missing_requested` a `DeployPlan`, `DeployResult` y `DeployClassification`, y recalcular el código 3 solo con omitidos y pedidos no cumplidos; verificar con pruebas de todas las combinaciones y `uv run pytest tests/test_deploy.py tests/test_cli.py`.
- [x] 2.3 Hacer que el holocrón se omita cuando no hay destinos existentes, informar del número de archivos protegidos y conservar el id sintético `snapshot`; verificar con despliegue solo-nuevos, despliegue con reemplazos y lectura de un holocrón antiguo.
- [x] 2.4 Añadir progreso y cancelación a `verify_archive`, `extract_archive`, `DeploySession.plan` y `DeploySession.apply`, con limpieza del staging y no-retorno tras el holocrón; verificar cancelación en cada fase y que no se escribe nada antes del snapshot.
- [x] 2.5 Registrar los archivos aplicados durante `apply` y lanzar `PartialApplyError` con aplicados, fallido y holocrón ante un fallo parcial; verificar la variante “Despliegue incompleto” y que el holocrón permite deshacer.
- [x] 2.6 Exponer avisos de arquitectura y versión de macOS distintos en `prepare_deploy` sin bloquear; verificar que se muestran en D0 y no alteran el código.

## 3. Núcleo de actualización

- [x] 3.1 Descomponer `run_update` en consulta, descarga/verificación y reemplazo, manteniendo la fachada para CLI; verificar actualización, ya al día e instalada más reciente en `tests/test_update.py`.
- [x] 3.2 Añadir progreso de descarga, cancelación por chunk y limpieza del temporal antes de `os.replace`; verificar cancelación antes del reemplazo con el binario anterior intacto.
- [x] 3.3 Clasificar los errores de actualización (no autocontenido, plataforma, red, integridad, reemplazo) para que la TUI elija la acción contextual; verificar cada categoría sin parsear mensajes.

## 4. Textos y vocabulario

- [x] 4.1 Crear `src/kamino/texts.py` como única fuente de textos de presentación y migrar `ui.py`, `cli.py` y las pantallas; verificar con pruebas de cadenas visibles y `uv run pytest tests/test_ui.py tests/test_cli.py`.
- [x] 4.2 Aplicar el mapa visible local -> original, snapshot -> holocrón de seguridad, mezcla -> híbrido, idéntico -> clon perfecto y no instalada -> sin base en este Mac, conservando los nombres internos y `SNAPSHOT_APP_ID`; verificar con una prueba de compatibilidad de holocrón antiguo.
- [x] 4.3 Actualizar README y `docs/Diseño funcional.md` con el glosario visible; verificar que no quedan textos de usuario con “local”, “snapshot”, “mezcla” o “idéntico” en las pantallas y resúmenes.

## 5. App, menú y transversales

- [x] 5.1 Crear `KaminoApp` con modos `menu` y `direct`, estado de sesión y navegación por pila; verificar que un flujo directo no vuelve al menú y que uno del menú sí vuelve, en `tests/test_tui.py`.
- [x] 5.2 Implementar el menú con wordmark, cápsulas, frase de rotación inyectable, opciones `c`/`d`/`u`/`q` y degradación por altura; verificar 24, 24-25, 26-29 y 30+ filas y que Escape no sale.
- [x] 5.3 Implementar ayuda contextual, pantalla de error de cuatro partes con detalle `t`/copia `c`, pantalla de terminal pequeña y política de `Ctrl+C`; verificar `?`, `F1`, redimensionado sin pérdida de estado y error en comando directo sin “Volver al menú”.
- [x] 5.4 Implementar última misión y código de salida del último flujo; verificar que una sesión nueva no muestra última misión y que salir con `q` devuelve el último código.

## 6. Flujo de clonación en la TUI

- [x] 6.1 Implementar C1 con apps instaladas, contador, filtro `/`, `a`/`n` sobre visibles, casillas deshabilitadas y línea de no detectadas; verificar con recuento en worker y `--apps` omitiendo la pantalla.
- [x] 6.2 Implementar C2 con nombre por defecto, ruta absoluta, validación en vivo, atajos y diálogo de reemplazo con Volver, Otro nombre y Reemplazar navegables por teclado y Volver preseleccionada; verificar que no se crea nada hasta C3 y que `--force` omite el diálogo.
- [x] 6.3 Implementar C3 en estados planificando, revisión y en curso, con `d`, `f`, aviso de secretos y dry-run sin acción de clonar; verificar que `--dry-run` no escribe.
- [x] 6.4 Implementar progreso por archivos y cancelación de C3 hasta el renombrado; verificar que cancelar borra el temporal y deja intacto el clon anterior.
- [x] 6.5 Implementar C4 con apps, archivos, tamaño, ruta, omitidos, línea de código 3 y copia con OSC 52; verificar el aviso cuando el terminal no soporta el portapapeles.

## 7. Flujo de despliegue en la TUI

- [x] 7.1 Implementar D0 con lista de `.tar.gz` del directorio actual, campo de ruta, validación de manifiesto y errores precargados; verificar archivo inválido, formato nuevo y avisos de arquitectura/macOS.
- [x] 7.2 Implementar D1 con apps instaladas y número de archivos, no instaladas informativas, `--apps` y estado “preparando” cancelable; verificar que un checksum fallido vuelve a D0 sin escribir.
- [x] 7.3 Adaptar D2 a los conteos nuevos, acciones `r`/`c`/`o`, globales `C`/`O`, confirmación bloqueada y paso directo a D4 sin conflictos; verificar que Atrás descarta decisiones tras confirmación.
- [x] 7.4 Reescribir D3 como vista por bloque con contexto único, marcadores ●/○, “solo clon”/“solo original”, `c`/`o`/`C`/`O`, `Tab`/`Mayús+Tab`, pie de recuento y validación al volver; verificar líneas largas, salto final y 20k líneas.
- [x] 7.5 Implementar D4 con conteos de nuevos/reemplazos, híbridos, sin desplegar y sin cambios, número de protegidos y directorio del holocrón, línea de apps abiertas y Retirada preseleccionada; verificar que cancelar no escribe ni crea holocrón.
- [x] 7.6 Implementar D5 con categorías, ruta del holocrón, `c`, orden de deshacer y variante “Despliegue incompleto”; verificar código 3 solo por omitidos/pedidos no cumplidos y código 1 por fallo parcial.

## 8. Flujo de actualización en la TUI

- [x] 8.1 Implementar U1 con estados consultando, descargando/verificando, reemplazando y resultado, con progreso y cancelación; verificar actualizado, ya al día e instalada más reciente.
- [x] 8.2 Implementar los errores de U1 y la petición de reinicio tras actualizar; verificar no autocontenido, plataforma, red, integridad y cancelación antes del reemplazo.

## 9. CLI, gating y códigos

- [x] 9.1 Extraer los flujos de `cli.py` para que el menú y los comandos compartan servicios sin `ctx.invoke` ni `typer.Exit` internos; verificar que el menú vuelve a sí mismo y que el comando directo sale con su código.
- [x] 9.2 Implementar la apertura de TUI por TTY, `KAMINO_PLAIN`, `--apps`, `--output`, `--yes` y `--dry-run` según el diseño; verificar con stdin/stdout TTY y no TTY y que `deploy -n`/`--yes` siguen sin TUI.
- [x] 9.3 Añadir `--force`/`-f` a `clone`, el nombre por defecto nuevo y los códigos 0/1/3 por categorías; verificar ayuda, aliases, clon existente, reemplazo y simulación sin escritura.
- [x] 9.4 Actualizar `tests/test_cli.py` y `tests/test_ui.py` para los flujos nuevos y retirar las pruebas de `questionary`; verificar `uv run pytest tests/test_cli.py tests/test_ui.py`.

## 10. Empaquetado y documentación

- [x] 10.1 Retirar `questionary` de `pyproject.toml`, `ui.py` y `uv.lock`; verificar `uv sync --all-extras` y que `uv run kamino --help` sigue funcionando.
- [x] 10.2 Actualizar `scripts/build_release.py` y `build/kamino-arm64.spec` si hay datos nuevos, y ejecutar el smoke test del binario; verificar que `clone --dry-run` sin TTY sigue funcionando.
- [x] 10.3 Actualizar `docs/Diseño técnico.md` con la app única, el contrato de progreso/cancelación, la clasificación y el empaquetado; verificar que las referencias a archivos y comandos existen.
- [x] 10.4 Revisar README y las ayudas de CLI para que no queden referencias a `questionary`, sufijos automáticos ni códigos 3 por idénticos; verificar con una búsqueda de textos obsoletos.

## 11. Verificación de integración

- [x] 11.1 Ejecutar `uv run ruff check .` y `uv run ruff format --check .`; corregir cualquier fallo sin cambiar el alcance.
- [x] 11.2 Ejecutar `uv run pytest` completo y verificar que las pruebas no leen el HOME real ni `/Applications`.
- [x] 11.3 Ejecutar una prueba manual headless a 80x24 y 79x23 de menú, clonar, desplegar y actualizar; verificar que el estado se conserva al redimensionar.
- [x] 11.4 Ejecutar `openspec validate tui-clonacion --type change --strict` y corregir los avisos de escenarios o requisitos.

## Workflow follow-up

- Archivar el cambio después de que la revisión y las comprobaciones de CI estén satisfechas.
- Verificar que las specs archivadas conservan el id `snapshot`, los códigos de salida y la compatibilidad con clones antiguos.
