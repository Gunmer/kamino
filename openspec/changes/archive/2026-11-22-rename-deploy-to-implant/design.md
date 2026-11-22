# Design

## Context

La CLI despacha `clone`, `deploy`, `update` y `apps` desde `internal/cli/cli.go`, y concentra los textos visibles en `internal/texts/texts.go`. La TUI consume esos textos y mantiene goldens en `internal/tui/testdata/golden/`. El paquete `internal/deploy`, las claves `deploy.snapshots_dir` y `deploy.snapshots_limit`, los flags y los códigos de salida ya son estables.

La propuesta fija `implant` como nombre visible principal y conserva `deploy` como alias. El reto es cambiar la superficie sin romper scripts, sin renombrar la lógica interna y sin alterar el formato del clon.

## Goals / Non-Goals

**Goals:**

- Añadir `kamino implant` con el mismo contrato que `kamino deploy`.
- Mantener `kamino deploy` como alias exacto y silencioso.
- Cambiar el menú, títulos, botones, mensajes y resumen de la TUI a «Implantar»/«implantación».
- Actualizar ayuda, autocompletado, README y documentación vigente.
- Corregir los textos ingleses residuales de la ayuda.

**Non-Goals:**

- Renombrar `internal/deploy`, `ExecuteDeploy`, `DeployPlan`, `DeployFlow` ni las claves `deploy.snapshots_*`.
- Cambiar el formato del clon, el manifiesto, los flags, los códigos de salida o la lógica de conflictos.
- Renombrar `clone`, `update`, `apps` o sus subcomandos.
- Introducir `install`, `restore` o el aprovisionamiento de herramientas en este cambio.
- Retirar `deploy`; la posible deprecación queda para una versión posterior.

## Decisions

### 1. `implant` como comando principal y `deploy` como alias

En `internal/cli/cli.go`, el despacho pasa a:

```go
case "implant", "deploy":
    return runImplantCommand(ctx, args[1:], options, interactive)
```

`runDeployCommand` puede conservar su nombre o renombrarse a `runImplantCommand`; la lógica interna no cambia. El alias es silencioso: no imprime avisos de deprecación, para no romper scripts que capturen stderr.

**Alternativa descartada:** hacer `implant` un alias de `deploy`. Mantendría `deploy` como forma principal y no resolvería la incoherencia temática.

### 2. Cambiar solo la capa visible, no los identificadores internos

Se modifican los valores de `internal/texts/texts.go` y las etiquetas de la TUI, pero se conservan los nombres internos `Deploy*`, `deploy` y `snapshot`. Ya existe el precedente de `snapshot` interno frente a «holocrón» visible.

**Alternativa descartada:** renombrar el paquete `internal/deploy` y todos sus tipos. Sería un refactor grande, con riesgo en pruebas y sin beneficio para la persona usuaria.

### 3. Mapeo de terminología visible

| Actual | Nuevo |
|---|---|
| `Desplegar` | `Implantar` |
| `Desplegar un clon` | `Implantar un clon` |
| `despliegue` | `implantación` |
| `Desplegados` | `Implantados` |
| `Sin desplegar` | `Sin implantar` |
| `Despliegue completado` | `Implantación completada` |
| `Despliegue incompleto` | `Implantación incompleta` |
| `kamino deploy <ruta>` (deshacer) | `kamino implant <ruta>` |

Las claves `deploy.snapshots_dir` y `deploy.snapshots_limit` no se renombran: son esquema de configuración y romperían archivos existentes.

### 4. Ayuda y autocompletado con dos nombres

La ayuda principal muestra `implant` como comando y una línea `deploy` como alias. `kamino implant -h` y `kamino deploy -h` muestran el mismo contrato de opciones. El autocompletado incluye ambos nombres.

La ayuda deja de mezclar inglés y español: se sustituyen `Show this message and exit.` y `No such command` por textos en español.

### 5. Sin cambios en el flujo ni en la seguridad

`implant` y `deploy` comparten el mismo `app.ExecuteDeploy`, el mismo orden de verificación, extracción, holocrón y aplicación, y los mismos códigos 0/1/2/3. No se añaden escrituras, descargas ni permisos nuevos.

## Risks / Trade-offs

- **Confusión entre `implant` y una futura fase de instalación** → La ayuda define `implant` como «aplica las configuraciones»; la instalación de herramientas queda para un comando `install` separado.
- **Scripts que dependen de la ayuda o del autocompletado** → El alias mantiene `deploy`; se añaden pruebas de equivalencia y de completion.
- **Goldens y textos duplicados** → Los textos viven en `internal/texts`; se actualizan una vez y se regeneran los goldens con el comando del proyecto.
- **Confusión por conservar `deploy` en el código** → Se documenta la frontera «interno `deploy`, visible `implant`» y se mantiene el alias en la ayuda.
- **El nombre `implant` puede sonar clínico** → La ayuda y los títulos añaden «aplica las configuraciones» para desambiguar.

## Migration Plan

1. Añadir el despacho `implant` y el alias `deploy`; conservar `runDeployCommand` o renombrarlo solo si no arrastra cambios de comportamiento.
2. Actualizar `internal/texts/texts.go` y las etiquetas de la TUI; regenerar goldens.
3. Actualizar ayuda principal, ayuda de `implant`/`deploy`, completion y mensajes de error.
4. Actualizar `README.md` y los documentos vigentes de `docs/`; dejar intactos los archivos históricos de `openspec/changes/archive/`.
5. Añadir pruebas de equivalencia `implant`/`deploy`, ayuda, completion, alias y etiquetas de la TUI.
6. Publicar como cambio compatible de la serie `0.x`; no hay migración de datos ni de archivos.

**Rollback:** revertir el cambio no afecta a clones, holocrones ni configuración. `deploy` sigue existiendo durante todo el cambio, así que una reversión no deja comandos rotos.

## Open Questions

Ninguna que bloquea la implementación. La eventual retirada de `deploy` se decidirá en una versión posterior y requerirá su propia propuesta.
