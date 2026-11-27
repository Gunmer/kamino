# Clonar e implantar

Referencia detallada de `kamino clone` y `kamino implant`.

## Clonar

En `clone`, `-a` equivale a `--apps`, `-o` a `--output`, `-n` a `--dry-run`, `-v` a `--verbose`
y `-f` a `--force`:

```bash
kamino clone -a opencode,ghostty,git -o clon.tar.gz -v
kamino clone -a opencode,ghostty,git -o clon.tar.gz -n   # simulación: no crea el archivo
kamino clone -f                                      # reemplaza clone-<aaaammdd>.tar.gz
```

Sin `--output`, el nombre por defecto es `clone-<aaaammdd>.tar.gz` con la fecha local dentro de
`~/.config/kamino/vault/`. Kamino crea ese directorio al escribir el clon; `--dry-run` no lo crea.
Sin `--force`, un clon existente es un error y no se sobrescribe; con `--force` se reemplaza de
forma atómica. `--dry-run --force` no modifica nada.

El cambio de destino es incompatible con scripts que esperaban el archivo en el directorio actual:
usa `--output` para conservar la ruta anterior. Los holocrones nuevos se guardan en
`~/.config/kamino/vault/snapshot/`; los antiguos en
`~/Library/Application Support/kamino/snapshots/` no se mueven ni se copian y siguen pudiéndose
implantar indicando su ruta.

Con terminal interactiva, `clone` abre la TUI si no se definió `KAMINO_PLAIN`. `--apps` omite C1,
`--output` omite C2 y ambos juntos pasan directamente a la revisión C3. Un valor inválido abre la
pantalla correspondiente precargada con el error. `--dry-run` termina en C3 sin acción de clonar.

En C1 cada app sale colapsada: `→` muestra sus archivos con ruta y tamaño y `←` los oculta.
`Espacio` marca o desmarca la app entera o un archivo concreto, y `a`/`n` actúan sobre las apps
visibles o, con el cursor en un archivo, sobre los de esa app. Una app con selección parcial se
muestra como `[~]` y `9/12`; el clon incluye solo los archivos marcados. Las apps instaladas sin
archivos para clonar no se listan y se anotan como «Sin archivos». `--apps` clona todos los
archivos de esas apps.

## Implantar

En `implant`, `-a` equivale a `--apps`, `-n` a `--dry-run`, `-y` a `--yes`
y `-v` a `--verbose`; el archivo del clon sigue siendo un argumento posicional:

```bash
kamino implant clone-20261109.tar.gz            # TUI: apps, conflictos, confirmación
kamino implant clone-20261109.tar.gz -a git,bat # sin selección de apps
kamino implant clone-20261109.tar.gz -y -v      # aplica todo sin preguntar
kamino implant clone-20261109.tar.gz -n         # simulación: imprime diffs, no escribe
```

Desde el menú, **Implantar** abre D0 con una sola lista que combina los clones de
`clone.output_dir` (por defecto `~/.config/kamino/vault/`) y los `.tar.gz` del directorio actual.
La lista se ordena por fecha de modificación, deduplica los archivos que aparecen en ambas rutas y
muestra el origen de cada fila, además de la ruta completa, la fecha y el tamaño del clon
resaltado. La tabla recibe el foco y deja seleccionado el clon más reciente; `/` filtra por nombre
u origen y `o` salta a `Otro archivo...`, que abre el campo de ruta manual. Si `kamino implant`
recibe una ruta inexistente o inválida, D0 se abre con esa ruta precargada, el error visible y la
lista de clones disponible.

Sin terminal interactiva, una implantación real requiere `-y` o `--yes`; la simulación con `-n`
no lo requiere. `implant -n` y `implant --yes` no abren la TUI aunque haya terminal. `-a` valida los
ids contra las apps implantables: los desconocidos o no instalados se avisan y cuentan como
omitidos (código de salida 3). Las apps del manifiesto no instaladas se informan como “sin base en
este Mac” y no alteran el código.

### Instalar apps ausentes

Si una app del clon no está instalada y su definición del catálogo declara una receta
`[[install]]` aplicable a este Mac, Kamino puede instalarla antes de aplicar su configuración.

- En la TUI, D1 lista todas las apps del clon como `Instalada` o `No instalada`. Marcar una no
  instalada con receta pide confirmar su instalación, listando todas las vías posibles (método y,
  para `installer` y `github-release`, el origen; sin suma de verificación lo avisa) para que
  elijas una. Nada viene marcado; desmarcar
  no cancela la implantación.
- En la CLI, `--install-missing` activa la instalación. Sin terminal interactiva exige `--yes`:
  `kamino implant clon.tar.gz --yes --install-missing`. Sin el flag no se instala nada.
- `--dry-run --install-missing` solo muestra qué se instalaría, sin descargar ni ejecutar nada.
- La instalación ocurre después de verificar el archivo completo y antes de crear la instantánea.
  Si falla, la app queda como omitida, el resto continúa y el código de salida es 3.
- Los métodos son `brew` y `npm` (Kamino no instala Homebrew ni Node: si la herramienta falta, se
  prueba la siguiente receta; `npm` instala siempre la última versión con
  `npm install --global`), `installer` (script HTTPS descargado a un temporal y ejecutado con
  `sh`), `binary` (HTTPS con SHA-256 verificado antes de escribir el destino con modo 0755) y
  `github-release` (último release de un repositorio de GitHub, instalado bajo el HOME). Si
  una receta `github-release` no declara `checksums`, la confirmación y `--dry-run` avisan «sin
  verificación de suma». El clon nunca aporta recetas ni URLs.
- El clon nunca guarda ejecutables ni bundles `.app`: se informan como no clonados.

## Holocrones de seguridad

Cada implantación crea su holocrón y después borra los más antiguos hasta dejar
`implant.snapshots_limit` (5 por defecto; `0` conserva todos). Solo se borran archivos regulares con
nombre `<fecha>.tar.gz` o `<fecha>-N.tar.gz` del directorio de holocrones: cualquier otro archivo,
directorio o enlace queda intacto. Si el holocrón nuevo no se pudo escribir no se borra ninguno, y
si el excedente no se puede borrar la implantación se detiene antes de escribir en los destinos.
`--dry-run` no borra holocrones.

Dos implantaciones simultáneas que comparten el directorio de holocrones se serializan con un bloqueo
exclusivo (`<directorio>/.snapshot.lock`), de modo que el que espera no puede borrar el holocrón de
una implantación en curso. El archivo de bloqueo nunca se poda. Mientras espera el bloqueo, `Esc` o
Ctrl+C cancelan la implantación.

Un holocrón tiene el mismo formato que un clon, así que se restaura implantándolo:
`kamino implant ~/.config/kamino/vault/snapshot/<fecha>.tar.gz --yes`. Restaura el contenido que
tenían los archivos antes de la implantación; no borra los archivos que esa implantación creó nuevos,
porque el holocrón solo guarda lo que ya existía. Si una implantación se interrumpe a medias, el error
muestra la ruta del holocrón y este mismo comando para deshacerlo. Si un destino cambia mientras la
implantación espera el bloqueo o entre dos escrituras, la implantación se detiene sin pisar ese cambio.

## Conflictos en la implantación

- D0 valida solo el manifiesto: rechaza un `tar.gz` dañado, con más de un `manifest.json`, con un
  manifiesto de más de 4 MiB o con un formato más nuevo. Los avisos de arquitectura o macOS no
  bloquean. La integridad completa se comprueba en D1 antes de extraer.
- D1 lista solo las apps instaladas con su número de archivos; las no instaladas aparecen aparte
  como “sin base en este Mac”. Ninguna está marcada por defecto y `--apps` omite esta pantalla.
- D2 lista solo los conflictos y muestra los conteos de nuevos, clones perfectos, no soportados y
  avisos. `r` revisa, `c` aplica el clon y `o` conserva el original; `C`/`O` actúan sobre todos.
  Confirmar queda bloqueado hasta que todos tengan decisión. Sin conflictos se pasa directo a D4.
- D3 muestra un bloque a la vez: el contexto se ve una vez y cada línea cambiada ofrece la versión
  del original y la del clon. `●` marca lo que irá al archivo final y `○` lo que no. `c`/`o` actúan
  sobre el bloque, `C`/`O` sobre todo el archivo y `Tab`/`Mayús+Tab` cambian de bloque. `Esc` vuelve
  y valida el híbrido.
- Antes de aplicar un híbrido se validan TOML, JSON, INI y plist con la biblioteca estándar. Si
  ambos originales eran válidos y el híbrido no, se bloquea; si alguno ya era inválido, se avisa y
  se permite aplicar. YAML y shell no se validan.
- D4 muestra los conteos antes de escribir, el número de archivos protegidos y el directorio del
  holocrón. Retirada está preseleccionada; las flechas recorren las acciones, `Enter` acepta la
  elegida y no hay atajo de una tecla para implantar. Volver o cancelar no escribe ni crea holocrón.
  Si no hay destinos existentes, no se crea un holocrón vacío.
- D5 distingue implantados, híbridos, omitidos y sin cambios, muestra la ruta del holocrón y la
  orden `kamino implant <ruta>` para deshacer. Solo los conservados en el original, los no soportados
  y los `--apps` no cumplidos producen código 3. Un fallo después del holocrón se presenta como
  “Implantación incompleta” con los aplicados, el fallido y el holocrón, y termina con código 1.
- `--yes` no abre la TUI y resuelve todos los conflictos aplicando el clon. `--dry-run` tampoco
  abre la TUI: usa `--apps` o todas las apps instaladas y vuelca los diffs a la salida estándar.
