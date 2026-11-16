# Límite de holocrones

Feature: retención configurable de holocrones. Rama `feature/inprove_config`.

## Specs

- **S1** — «quiero limitar el numero de holocrones, por defecto a 5, pero esto se puede cambiar desde una nueva opcion en la configuración». El límite de retención de holocrones vale 5 por defecto y es configurable.
- **S2** — La opción vive en `~/.config/kamino/config.toml` como `deploy.snapshots_limit` (entero) y en la pantalla Configuración de la TUI como cuarto campo editable de tipo numérico.
- **S3** — Al crear un holocrón nuevo, si el directorio de holocrones contiene más de N, se eliminan los más antiguos hasta que queden N. `0` desactiva la poda (ilimitado). Un valor negativo o no numérico es un error de configuración, no un valor implícito.
- **S4** — Solo son elegibles para borrado los archivos regulares del directorio de holocrones cuyo nombre sea `<fecha ISO>[-N].tar.gz`. Cualquier otro archivo, directorio o enlace del directorio nunca se toca.
- **S5** — El CLI aplica el mismo límite leyendo `config.toml` únicamente para ese valor; las rutas siguen usando los valores integrados, como hoy. Un archivo de configuración inválido no bloquea el despliegue por CLI: se usa el valor por defecto.
- **S6** — `--dry-run` no escribe ni borra ningún holocrón.
- **S7** — Si el holocrón nuevo no se pudo escribir, no se borra ninguno. Si el excedente no se puede borrar, el despliegue se aborta antes de escribir en los destinos.

## Tasks

- T1 — config: campo `SnapshotsLimit`, clave `deploy.snapshots_limit`, default 5, parseo numérico, serialización y validación (S2, S3) — commit e576a7c `feat(config)`.
- T2 — snapshot: poda de holocrones y su test (S3, S4, S7) — commit e11ebd1 `feat(snapshot)`.
- T3 — wiring: `deploy.Options`/`DeploySession`, `app.ExecuteDeploy`, pantalla de despliegue de la TUI (S5, S6) — commit e11ebd1.
- T4 — TUI: cuarto campo numérico en Configuración, textos y goldens (S2) — commit 7af9e8d `feat(tui)`.
- T5 — docs y specs: README, `docs/diseno-funcional.md`, `docs/diseno-tecnico.md`, `docs/diseno-tui.md`, `openspec/specs/user-configuration/spec.md` (S1-S7) — commit `docs`.

## Log

- L1 — «quiero limitar el numero de holocrones, por defecto a 5, pero esto se puede cambiar desde una nueva opcion en la configuración»
- L2 — Decisiones del humano: editable en la TUI y en `config.toml`; al superar el límite se borran los más antiguos con `0` = ilimitado; el CLI también aplica el límite leyendo `config.toml`.
- L3 — Corte de revisión: e576a7c, e11ebd1 y 7af9e8d forman la unidad de comportamiento (config + poda + wiring + TUI); el commit `docs` cierra la feature. Los commits anteriores de `feature/inprove_config` no pertenecen a esta feature.
- L4 — Verificación: `gofmt -l .` vacío, `go vet ./...` y `golangci-lint run ./...` sin hallazgos, `go test ./... -race` en verde. El verificador independiente `gentle-ai-verify` quedó no disponible (timeout); la comprobación crítica se hizo en el padre: un solo llamador de `snapshot.CreateSnapshot`, poda posterior a la escritura del archivo y ninguna ruta de `--dry-run` que llegue a `Apply`.
