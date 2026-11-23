# Pruebas y distribución

## 7. Pruebas

- **Unitarias:** `internal/catalog` valida definiciones válidas e inválidas; `internal/paths` cubre expansión, normalización y rechazo de rutas fuera del HOME; `internal/archive` cubre el manifiesto y los checksums.
- **Conflictos:** `internal/conflicts` cubre detección de texto, binario, grande y enlace; archivos nuevos, idénticos, solo modo, CRLF y cambios de tipo; generación de filas con contexto, emparejamiento e inserciones/eliminaciones; composición con híbridos dentro de un bloque, todo clon, todo original, CRLF, salvaguarda de salto final y longitud de decisiones; y validación de TOML, JSON, INI, plist, original inválido y formatos sin validar.
- **Integración:** `internal/clone`, `internal/implant` y `internal/snapshot` cubren el ciclo completo entre dos HOME falsos distintos, con apps instaladas simuladas mediante un `Info.plist` falso, comprobando que solo se implantan las instaladas. Las pruebas de conflicto cubren híbridos, holocrón, undo, `--yes` y `--dry-run` con diffs.
- **TUI:** `internal/tui` prueba el `Frame` a 80/100/140 columnas y 24/26/30 filas, la pila, la zona de mensajes, C1–C3, D0–D5, U1, ayuda, error de cuatro partes, copia, cancelación y implantación parcial. `field_test.go` cubre inserción, retroceso, límites, UTF-8, runas de doble ancho, ventana horizontal con `…` a ambos lados, placeholder, error, alineación de etiquetas, `NO_COLOR` y ASCII; las pruebas de pantalla cubren el filtro y los campos de C1, C2, D0 y Configuración, el foco único, la ayuda sin inserción y un solo cursor de edición. Los goldens cubren 80×24, 100×26, 140×40 y 200×50, más `NO_COLOR` y ASCII; C1 con filtro, C2 con foco en ruta y nombre, D0 con filtro y ruta manual, y Configuración vacía tienen casos propios. El test de contrato exige exactamente `h` líneas, ninguna línea más ancha que `w`, solo SGR ANSI y, en `NO_COLOR`, ni colores ni fondos ANSI. D3 usa un modelo de filas completo con benchmark de 20 000 líneas y solo renderiza la ventana visible. `View` no toca disco, reloj ni entorno.
- **Actualización:** `internal/update` cubre el parseo y la comparación de versiones, la consulta de la release, el asset ausente, el digest ausente o incorrecto, la descarga y el reemplazo atómico, los symlinks y la plataforma no soportada. Ninguna prueba accede a la red ni al HOME real.
- **Releases:** `internal/release` cubre la política de versiones, el parseo de Conventional Commits, el renderizado e inserción del changelog, las notas únicas, la validación de tags y el build con smoke test en macOS ARM64.
- **Seguridad:** archivos tar con rutas `..`, rutas absolutas y enlaces simbólicos maliciosos, y un checksum alterado. `internal/paths` cubre el rechazo de enlaces intermedios y la conservación de enlaces finales; `internal/implant` cubre el rechazo antes de la instantánea, la sustitución de un directorio durante la aplicación y un destino cambiado entre plan y apply.
- **Aislamiento:** las pruebas usan directorios HOME temporales e inyectan las dependencias para no tocar nunca el home real.
- **CI:** GitHub Actions con Go en macOS y Ubuntu (`go vet`, `gofmt`, `golangci-lint`, `go test ./... -race`).

## 8. Empaquetado y distribución

| Vía | Cómo | Cuándo |
| --- | --- | --- |
| Binario autocontenido | `cmd/release` con `CGO_ENABLED=0`, `-trimpath` y `-s -w`; asset `kamino-arm64` en GitHub Releases | en cada merge a `main` |
| Instalador | `scripts/install.sh` descarga `kamino-arm64` en `~/.local/bin/kamino` sin `sudo` | v1 |
| Homebrew tap | Fórmula que instala el asset `kamino-arm64` | tras la primera versión etiquetada |

El binario sin firmar recibe la cuarentena de macOS: hay que documentar cómo quitarla (`xattr -d com.apple.quarantine`) o firmarlo y notarizarlo con una cuenta de desarrollador.

El binario Go usa las CAs del sistema para que `kamino update` verifique TLS. `internal/release` construye y comprueba la versión, el catálogo y la ausencia de escrituras en un HOME temporal.

El versionado y la publicación se automatizan con `cmd/release`: los commits integrados en `main` siguen Conventional Commits y, tras el CI, se calcula el bump semver, se actualiza `CHANGELOG.md`, se crea el tag `vX.Y.Z`, se construye el binario y se publica la GitHub Release (workflow `.github/workflows/release.yml`). La recuperación de un tag existente reutiliza el mismo comando con `--tag`.

El ejecutable y el módulo se llaman `kamino`; desde código se compila con `go build ./...`.

