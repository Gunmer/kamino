# Tasks

## 1. Esquema y validación de recetas en el catálogo

- [x] 1.1 Añadir `[[install]]` a `AppDefinition` (`method`, `formula`/`cask`, `url`, `sha256`, `dest`, `os`, `arch`) y validarlo al cargar: método entre `brew`/`installer`/`binary`, `https` obligatorio, `sha256` obligatorio en `binary`, `os`/`arch` reconocidos. Verificar con tests de `internal/catalog` para cada error (`go test ./internal/catalog`).
- [x] 1.2 Hacer que `kamino apps validate` informe las recetas con errores en español y código 1, y que `apps list` muestre los mismos errores por archivo sin cambiar su código de salida (0). Verificar con tests de `internal/app` y `internal/cli`, y actualizar la ayuda y los docs de autoría (`docs/`, Markdown en español).
- [x] 1.3 Probar que una definición sin `[[install]]` y las integradas se cargan exactamente igual que antes (test de regresión del catálogo embebido).

## 2. Guardia: el clon nunca guarda binarios

- [x] 2.1 En `internal/plan`, rechazar directorios `.app` y archivos ejecutables (bit de ejecución o Mach-O) al recorrer rutas, y devolverlos como no clonados sin leerlos. Verificar con tests de plan y de escritura del clon con HOME temporal (binario en `.local/bin`, bundle `.app`, ajustes junto a binario).
- [x] 2.2 Mostrar los rechazos en la salida de la CLI y en el resumen de la TUI. Verificar con tests de `internal/ui` y un golden de la TUI regenerado (`go test ./internal/tui -run Golden -update`).

## 3. Paquete `internal/install`

- [x] 3.1 Crear `Recipe`, selección por plataforma y `Runner` inyectable (procesos y búsqueda en `PATH`). Verificar con tests de filtrado `os`/`arch` y de elección del primer método viable, incluido `brew` ausente con y sin alternativa.
- [x] 3.2 Implementar `brew` (fórmula y cask) sin instalar Homebrew nunca. Verificar con el fake de `Runner` que se invoca el comando esperado y que sin `brew` no se ejecuta nada.
- [x] 3.3 Implementar `binary`: descarga con curl a temporal, SHA-256 en streaming, `rename` atómico con permisos 0755 mediante `paths.SafeHome`. Verificar con servidor HTTPS local: éxito, checksum incorrecto (temporal borrado y destino intacto) y fallo de descarga.
- [x] 3.4 Implementar `installer`: descarga a temporal y ejecución con `sh`, sin tubería. Verificar con fake de `Runner` que el script se ejecuta desde archivo y que un fallo de descarga no ejecuta nada.

## 4. Fase de instalación en el despliegue

- [x] 4.1 Añadir a `DeploySession` la etapa de instalación entre verificar el archivo y la instantánea, reclasificando las apps instaladas con los detectores. Verificar con tests de `internal/deploy`: archivo inválido sin instalaciones, orden respecto a la instantánea, app instalada que pasa a desplegable.
- [x] 4.2 Tratar los fallos como omitidas con código de salida 3 sin abortar el resto. Verificar con un test de dos apps donde una instalación falla.
- [x] 4.3 Garantizar que `--dry-run` calcula y devuelve el plan de instalación sin llamar al `Runner` ni escribir. Verificar con un test que el fake no recibe llamadas.
- [x] 4.4 Comprobar que el manifiesto y el contenido del clon no influyen en la receta usada. Verificar con un test que fija el comportamiento con un manifiesto que incluye campos de instalación.

## 5. CLI y TUI

- [x] 5.1 Añadir `--install-missing` a `implant`/`deploy`: exige `--yes` en modo no interactivo (código 1 si falta) y no cambia nada sin el flag. Verificar con tests de `internal/cli` (con y sin flag, con `--dry-run`) y que la ayuda y el autocompletado lo describen en español.
- [x] 5.2 Añadir la pantalla de TUI que pregunta por app, muestra método y URL de `installer`, y permite rechazar sin cancelar. Verificar con tests de pantallas, el test de contrato de altura/ancho/SGR y goldens regenerados.
- [x] 5.3 Añadir los textos en `internal/texts` y actualizar `docs/diseno-tui.md` y el README con el flujo de instalación y sus garantías. Verificar con `gofmt -l .` vacío y revisión de los textos en español.

## 6. Integración

- [x] 6.1 Ejecutar `gofmt -l .`, `go vet ./...`, `golangci-lint run ./...` (si está instalado) y `go test ./... -race`, y confirmar que ninguna prueba toca el HOME real ni `/Applications`.
- [x] 6.2 Ejecutar `openspec validate install-missing-apps --strict` y recorrer cada escenario de los specs contra los tests añadidos.
