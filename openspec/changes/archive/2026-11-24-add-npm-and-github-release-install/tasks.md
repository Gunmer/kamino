# Tasks

## 1. Método npm

- [x] 1.1 Añadir `MethodNpm` y el campo `package` a `Recipe`, con validación (patrón de nombre, campos no admitidos) y mensaje de métodos permitidos actualizado; verificar con tests de `Validate` y que el error nombra los cinco métodos
- [x] 1.2 Generalizar en `Select` la comprobación de herramienta (`brew`→`brew`, `npm`→`npm`) y ejecutar `npm install --global <package>` en `Installer`; verificar con tests de selección (brew+npm, solo npm, ninguna) y de ejecución con `Runner` falso, más un test de que `brew` mantiene su comportamiento anterior
- [x] 1.3 Documentar el método `npm` en `docs/guia/catalogo.md` y `docs/tecnico/`, y actualizar la ayuda de `apps validate`; verificar con `go run ./cmd/kamino apps validate --help`

## 2. Método github-release

- [x] 2.1 Añadir `MethodGithubRelease` y los campos (`repo`, `asset`, `format`, `extract`, `checksums`) a `Recipe` con validación; verificar con tests de repo inválido, destino inválido, campos de otros métodos y `format` desconocido
- [x] 2.2 Resolver el último tag por redirección con `curl` y construir la URL de descarga con `{version}`; verificar con tests que usan un `Runner` falso (tag `v2.105.0` → `2.105.0`, sin redirección → error)
- [x] 2.3 Descargar, verificar `checksums` opcionales y extraer zip, tar.gz y binario suelto con escritura por `SafeHome` y comprobación previa de todos los miembros; verificar con tests de suma correcta, suma incorrecta, suma ausente, miembro ausente, ruta con `..` y dos ejecutables, comprobando que no se escribe nada en los fallos
- [x] 2.4 Mostrar en la confirmación y en `--dry-run` la URL base y el aviso «sin verificación de suma» cuando falte `checksums`; verificar con tests de `internal/implant` y actualización de goldens de la TUI (`go test ./internal/tui -run Golden -update`)
- [x] 2.5 Documentar el método `github-release` y la rebaja de verificación en `docs/guia/catalogo.md` y `docs/tecnico/`; verificar que los ejemplos del doc pasan `apps validate`

## 3. Catálogo integrado

- [x] 3.1 Añadir recetas `npm` a claude-code, codex, gemini-cli, copilot-cli, opencode, openspec y pi, con paquetes verificados con `npm view`; verificar con `go test ./internal/catalog`
- [x] 3.2 Añadir recetas `github-release` (una por arquitectura) a github-cli, lazygit, glow, bat, micro, yazi, claude-code, copilot-cli, openspec, pi, opencode, gemini-cli, herdr y codex, con `checksums` donde el release lo publique; verificar cada nombre de asset contra el release `latest` real
- [x] 3.3 Ordenar las recetas de cada app como brew, npm, installer, github-release y añadir un test del catálogo integrado que valide todas las recetas y ese orden; verificar con `go test ./internal/catalog ./internal/install`
- [x] 3.4 Actualizar la tabla del catálogo en `docs/guia/catalogo.md` con los métodos de cada app; verificar con `gofmt -l .`, `go vet ./...` y `go test ./... -race`

## 4. Integración

- [x] 4.1 Probar el despliegue completo con HOME temporal y runner falso: app con brew ausente que cae a npm, y otra que cae a github-release; verificar que `--dry-run` no escribe nada y que un fallo deja salida 3

## Workflow follow-up

- Archivar el change tras revisar el diff y ejecutar `openspec validate`.
