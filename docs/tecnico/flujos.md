# Flujos

## 5. Flujos

### 5.1 Clonación

1. Cargar el catálogo (embebido y directorio del usuario).
2. Detectar las apps instaladas.
3. C1 cuenta archivos en segundo plano con el mismo recorrido y exclusiones que el plan, sin hashear. `--apps` omite C1.
4. C2 ofrece vault por defecto, carpeta actual e indicar carpeta, con los campos «Destino» y
   «Nombre»; `--output` omite C2. La ruta se valida en vivo y el destino se crea al escribir.
5. `plan_clone` expande `paths`, aplica `exclude` y construye el plan con tamaño y checksum.
   Las rutas ausentes se registran con su motivo. C3 puede cancelar durante la planificación.
6. C3 muestra la revisión y `write_clone` escribe un temporal con progreso por archivos. La
   cancelación comprueba el contexto antes de cada archivo y antes del renombrado.
7. Sin `--force` el commit usa `os.link` y falla si el destino existe; con `--force` usa
   `os.replace` y revalida la huella del destino justo antes. Cualquier error o cancelación elimina
   el temporal. El estado final de C3 muestra apps, archivos, tamaño, ruta y omitidos; no hay C4.

### 5.2 Implantación

1. D0 descubre los `.tar.gz` hijos directos de `clone.output_dir` y del directorio actual,
   deduplica por ruta resuelta, ordena por `mtime` descendente y construye la tabla
   `Archivo`/`Origen`/`Tamaño` con preview de ruta, fecha y tamaño, filtro `/` y la fila
   `Otro archivo...` para la ruta manual. Abre el archivo y valida solo el manifiesto: un único
   `manifest.json` regular, tamaño máximo de 4 MiB, tope de miembros y `format_version`. Envuelve
   errores de `tarfile`/OS. Los avisos de arquitectura o macOS no bloquean.
2. D1 detecta las apps instaladas y cruza el manifiesto. Las no instaladas se informan como “sin
   base en este Mac”. La verificación completa ocurre antes de extraer; la preparación es
   cancelable y limpia el staging si falla.
3. D1 elige qué apps aplicar: ninguna marcada por defecto, todas/ninguna, o `--apps` omite la
   pantalla.
4. Construir el plan: el destino de cada archivo es `Path.home()` más su ruta relativa, clasificado
   como nuevo, sin cambios (clon perfecto, incluida una diferencia solo de CRLF), distinto o no
   soportado. Para cada archivo distinto se construye un conflicto: filas de texto para revisar, o
   archivo entero si es binario, grande o un enlace con enlace.
5. D2 lista solo conflictos y permite `r` revisar, `c` aplicar el clon, `o` conservar el original,
   `C`/`O` sobre todos. Confirmar queda bloqueado hasta cubrir todos. Sin conflictos se pasa a D4.
6. D3 muestra el archivo completo en dos columnas con contexto alineado, números de línea,
   marcadores `●`/`○`, `c`/`o` por bloque, `C`/`O` por archivo, `Tab`/`Mayús+Tab` entre bloques y
   validación al volver. Solo renderiza la ventana visible.
7. D4 revalida los destinos, cuenta protegidos y exige una acción explícita con foco en Retirada.
8. Aplicar en este orden: cobertura de decisiones, revalidación de híbridos, huellas de destinos,
   holocrón si hay destinos existentes y escrituras atómicas con progreso. La cancelación se
   comprueba hasta crear el holocrón; después se ignora. Un fallo posterior lanza
   `PartialApplyError` con aplicados, fallido y holocrón.
9. D5 muestra implantados, híbridos, omitidos, sin cambios, protegidos y la ruta del holocrón. El
   código 3 se calcula solo con omitidos y `--apps` no cumplidos; una implantación incompleta usa 1.

El orden de seguridad es `verificar -> extraer -> decidir -> holocrón -> aplicar`. `--yes` y
`--dry-run` conservan la vía de texto y no abren la TUI.

Pantallas y atajos:

```text
Menú -> C1 apps -> C2 destino -> C3 revisión/resultado
Menú -> D0 clon -> D1 apps -> D2 conflictos -> D3 archivo -> D4 confirmación -> D5 resumen
Menú -> U1 actualización
```

- **C1:** tabla `SEL`/`APP`/`ARCHIVOS`, contador de selección, filtro `/` como campo de una fila,
  `a`/`n` según el ámbito (apps visibles o archivos de la app), árbol de archivos con `→`/`←`,
  casillas `[x]`/`[~]`/`[ ]`, `Espacio` sin perder el foco, `Enter`, notas de no detectadas y sin
  archivos, y recuento en segundo plano. La selección por archivo viaja en `PlanOptions.Files` y
  `clone.PlanClone` filtra las operaciones antes de construir el manifiesto.
- **C2:** tres opciones de carpeta, campos «Destino» y «Nombre», placeholder y scroll con `…`,
  validación en vivo y diálogo Sobrescribir/Cambiar nombre/Volver con Volver preseleccionada. Los antiguos atajos de directorio no existen.
- **C3:** tabla única con Apps, Archivos, Tamaño, Ruta y Estado; planificando, revisión, escritura,
  completado, simulación o cancelado. `d` omitidos, `f` archivos incluidos, `c` copiar ruta al
  terminar y aviso de secretos en mensajes. C4 es el estado final de C3, no una pantalla.
- **D0:** tabla `Archivo`/`Origen`/`Tamaño` con los clones de `clone.output_dir` y del directorio
  actual, orden por fecha, deduplicación por ruta resuelta, preview con ruta/fecha/tamaño, campos
  «Filtro» y «Ruta», y validación del manifiesto.
- **D1:** tabla `SEL`/`APP`/`ARCHIVOS`, no instaladas informativas y preparación cancelable.
- **D2:** tabla de conflictos `ARCHIVO`/`BLOQUES`/`LÍNEAS`/`ACCIÓN`, cero `·`, conteos, scroll y
  acciones `r`/`c`/`o`/`C`/`O`.
- **D3:** archivo completo en dos columnas, números de línea, `●`/`○`, `— sin línea —`, contadores,
  ventana visible y acciones por línea/bloque/archivo.
- **D4:** tabla por app, cinco categorías, cero `·`, totales, protegidos, holocrón, aviso de apps
  abiertas y acciones horizontales con Retirada preseleccionada.
- **D5:** categorías, holocrón, `c`, orden de deshacer y variante “Implantación incompleta”.
- **Ayuda:** `?` en todas las pantallas y `F1` en campos de texto; pie y ayuda salen de `Keys()`.

### 5.3 Actualización

1. `query_update` comprueba que el proceso es un binario autocontenido, resuelve el ejecutable real
   y valida macOS arm64. Si no procede, lanza `UpdateError` con una categoría estable.
2. Consulta `https://api.github.com/repos/Gunmer/kamino/releases/latest` y lee `tag_name`, el asset
   `kamino-arm64`, su `size` y su `digest` SHA-256.
3. Compara la versión publicada con `__version__` como `X.Y.Z`. Devuelve actualizado, ya al día o
   instalada más reciente sin descargar en los dos últimos casos.
4. `download_update` descarga a un temporal del mismo directorio, emite progreso por chunk y
   comprueba la cancelación. Verifica tamaño y digest antes de devolver la ruta.
5. `replace_binary` marca el temporal como ejecutable y lo reemplaza con `os.replace`. La
   cancelación se comprueba justo antes; una vez iniciado el reemplazo no se interrumpe.
6. Cualquier error o cancelación elimina el temporal y deja intacto el binario anterior.
7. U1 muestra consultando, descargando/verificando, reemplazando y resultado, pide reiniciar tras
   actualizar y usa la ayuda contextual de `texts.UPDATE_ERROR_ADVICE` según la categoría. El código
   es 0 si actualiza o ya está al día; 1 ante error o cancelación.

No se ejecuta el binario descargado antes de reemplazarlo. La actualización no usa `sudo` ni
modifica datos del usuario. La consulta usa las CAs del sistema y el binario de release no incluye
certificados propios.

