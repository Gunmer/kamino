# Proposal

## Why

Hoy, al implantar un clon en un Mac que no tiene instalada una app del clon, Kamino solo informa «Sin base en este Mac» y salta su configuración. El usuario debe instalar la app a mano y repetir la implantación. Ofrecer instalarla desde Kamino cierra el flujo de restauración, siempre que sea seguro: sin ejecutar nada que dicte un clon ajeno y sin que el clon lleve nunca binarios.

## What Changes

- Las definiciones del catálogo (embebido y de usuario) pueden declarar recetas de instalación con un esquema cerrado de tres métodos: `brew`, `installer` (instalador oficial descargado con curl) y `binary` (binario descargado con curl).
- La receta vive **solo** en el catálogo. El manifiesto del clon nunca aporta URLs, comandos ni métodos de instalación.
- Al implantar, las apps ausentes con receta aplicable se ofrecen para instalar: la TUI pregunta por app; la CLI exige `--install-missing` además de `--yes`. `--dry-run` solo muestra qué se instalaría y no ejecuta nada.
- Las recetas son una lista ordenada filtrable por `os` y `arch`. Si `brew` no está disponible se prueba el siguiente método; Kamino nunca instala Homebrew.
- `binary` exige HTTPS y SHA-256. `installer` exige HTTPS y confirmación mostrando la URL exacta; el script se descarga a un temporal y se ejecuta con `sh`, sin tubería directa.
- La instalación ocurre tras verificar el archivo completo y antes de la instantánea. Una instalación fallida deja la app como omitida (código de salida 3) y el resto continúa.
- La clonación rechaza ejecutables y bundles `.app`, aunque una definición de usuario los declare en `paths`, para que el clon nunca guarde el binario de una app.
- `kamino apps validate` valida el esquema de recetas.
- Alcance de la primera versión: solo el mecanismo. Las apps integradas no reciben recetas todavía.

## Capabilities

### New Capabilities
- `app-install`: recetas de instalación del catálogo, los tres métodos permitidos, sus garantías de seguridad, filtro por plataforma, orden respecto al despliegue y tratamiento de fallos.

### Modified Capabilities
- `clone-safety`: la clonación nunca guarda binarios ejecutables ni bundles de aplicación.
- `cli-options`: nuevo flag `--install-missing` del despliegue.
- `tui-navigation`: paso de la TUI que pregunta qué apps ausentes instalar.
- `app-authoring`: la validación del catálogo cubre las recetas de instalación.

## Impact

- `internal/catalog`: nuevo bloque `[[install]]` en `AppDefinition`, su validación y los avisos de `apps validate`.
- `internal/plan`: guardia que rechaza binarios y bundles al clonar.
- `internal/deploy`: fase de instalación entre la verificación del archivo y la instantánea; resultado y código de salida 3 por fallos.
- `internal/cli`, `internal/tui`, `internal/texts`: flag, pantalla de confirmación y textos en español.
- Seguridad: nueva superficie de descarga y ejecución remota, acotada a HTTPS, catálogo local y confirmación explícita.
- Sin cambios en el formato del manifiesto ni en la versión del archivo del clon.
