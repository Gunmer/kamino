# Configuración

La opción **Configuración** del menú, situada después de Actualizar, permite
ver y editar las tres rutas y el límite de holocrones en una pantalla. Cada valor ocupa una fila con
etiqueta alineada: `Tab`/`Mayús+Tab` y `↑`/`↓` mueven el foco, `←`/`→` el cursor dentro del valor y
el límite solo admite dígitos. La pantalla se guarda con `Ctrl+S`; `Esc` vuelve sin escribir. Un
valor inválido marca su campo y muestra el `ERROR` en la zona de mensajes sin desplazar las filas.
`catalog.dir` y `implant.snapshots_limit` los comparten la TUI y la CLI de texto; `clone.output_dir`
y `implant.snapshots_dir` solo los usa la TUI. Si el archivo falta o es inválido, la CLI usa el
catálogo integrado y los valores por defecto sin bloquear; los comandos `apps` muestran el aviso y
la ruta efectiva.

```toml
[catalog]
dir = "~/.config/kamino/apps"

[clone]
output_dir = "~/.config/kamino/vault"

[implant]
snapshots_dir = "~/.config/kamino/vault/snapshot"
snapshots_limit = 5
```

| Clave | Valor por defecto | Uso |
| --- | --- | --- |
| `catalog.dir` | `~/.config/kamino/apps` | Definiciones TOML de apps del usuario; las usan la TUI y la CLI |
| `clone.output_dir` | `~/.config/kamino/vault` | Directorio propuesto en C2 y destino del clon en la TUI |
| `implant.snapshots_dir` | `~/.config/kamino/vault/snapshot` | Directorio de los holocrones en la TUI |
| `implant.snapshots_limit` | `5` | Holocrones que se conservan; `0` no borra ninguno. Lo aplican la TUI y la CLI |

El archivo es opcional y se crea solo al guardar desde la pantalla. Admite únicamente esas cuatro
claves: las tres rutas deben ser cadenas y `implant.snapshots_limit` un entero mayor o igual que 0.
Un TOML inválido, una sección o clave desconocida, un tipo incorrecto, una ruta relativa,
`~usuario` o una ruta existente que no sea un directorio se rechazan. Si el archivo es inválido al
arrancar, la TUI abre Configuración con el error y los valores legibles; no lo sobrescribe hasta
que se guarde. Los directorios configurados se crean al clonar o implantar, no al guardar; el
catálogo se crea al guardar la primera definición con `apps new` o «Añadir app». Los cambios
guardados se aplican de inmediato a los flujos de la misma sesión. `--force`, `--yes` y
`--dry-run` no se configuran: son decisiones por operación. `ui.plain` y `ui.theme` tampoco existen
en este esquema.
