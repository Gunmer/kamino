# Proposal

## Why

`deploy` es el único verbo principal de Kamino que no pertenece a su relato: `clone` crea el clon, pero `deploy` aplica configuraciones con el vocabulario estándar de DevOps. La metáfora de Kamino gana coherencia si el clon se **implanta** en la máquina destino, como se implantan memorias en un clon, sin perder la compatibilidad de los scripts que ya usan `deploy`.

## What Changes

- Añadir `kamino implant` como comando principal para aplicar un clon en este Mac.
- Mantener `kamino deploy` como alias de compatibilidad, con los mismos argumentos, opciones, resultados y códigos de salida.
- Renombrar en la TUI y en los textos visibles «Desplegar» por «Implantar», y «despliegue» por «implantación», sin cambiar el comportamiento del flujo.
- Actualizar la ayuda principal, la ayuda de `implant`/`deploy`, el autocompletado y los ejemplos de la documentación para mostrar `implant` como forma principal y `deploy` como alias.
- Mantener sin cambios `clone`, `update`, `apps`, los flags, las claves `deploy.snapshots_dir` y `deploy.snapshots_limit`, el formato del paquete y los identificadores internos del paquete `deploy`.
- Corregir los textos visibles que aún están en inglés en la ayuda (`Show this message and exit.`, `No such command`) para mantener el contrato de idioma del proyecto.

## Capabilities

### New Capabilities

Ninguna.

### Modified Capabilities

- `cli-options`: el comando principal pasa a ser `implant`; `deploy` queda como alias, y la ayuda, el autocompletado y la compatibilidad reflejan ambos nombres.
- `tui-navigation`: el menú y las etiquetas visibles del flujo pasan de «Desplegar»/«despliegue» a «Implantar»/«implantación», conservando la misma navegación, pasos y comportamiento.

## Impact

- Afecta a `internal/cli/cli.go` (despacho del comando y ayudas), `internal/texts/texts.go` (textos visibles), `internal/tui` (etiquetas y goldens), `README.md` y la documentación vigente de `docs/`.
- No cambia el formato del clon, las claves de configuración, los flags, los códigos de salida ni la lógica de `internal/deploy`.
- `kamino deploy` sigue funcionando; no hay migración de datos ni cambios en los archivos existentes.
