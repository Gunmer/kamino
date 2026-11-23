# Pantallas de la implantación (D0–D5)

### 4.6 D0 · Elegir el clon

```text
kamino · Implantar · Clon                                    Paso 1 de 4   * o o o
--------------------------------------------------------------------------------

  Elige el clon que quieres implantar

  ARCHIVO                          ORIGEN            TAMAÑO
  --------------------------------------------------------------------------
  > clone-20261113.tar.gz          por defecto        1.2 MB
    clone-20261112.tar.gz          este directorio    980 KB
    clone-20261001.tar.gz          por defecto        1.1 MB

  Ruta: ~/.config/kamino/vault/clone-20261113.tar.gz
  Fecha: 2026-11-13 10:30   Tamaño: 1.2 MB

  o Otro archivo...

  ! No se pudo leer ~/Downloads; se muestran los clones del vault.

--------------------------------------------------------------------------------
^v mover  / filtrar  o otro archivo  Enter continuar  Esc volver  ? ayuda
```

Variante: filtro con foco.

```text
  > Filtro    clone|
```

Variante: ruta manual.

```text
  > Ruta    /Users/test/Descargas/clone-20261113.tar.gz|

--------------------------------------------------------------------------------
Enter abrir  Esc volver  ←→ cursor  F1/? ayuda
```

Notas:

- La tabla combina el vault y el directorio actual, ordena por fecha y deduplica
  por ruta resuelta.
- El preview de la fila enfocada muestra ruta, fecha y tamaño.
- `/` abre el campo «Filtro»: `Enter` lo aplica y devuelve el foco a la tabla;
  `Esc` lo limpia y vuelve. Mientras tiene foco, el footer muestra
  `Enter aplicar`, `Esc volver`, `←→ cursor` y `F1/? ayuda`.
- `o` abre el campo «Ruta» con placeholder atenuado; `Enter` valida el manifiesto
  y `Esc` vuelve a la tabla sin cancelar el flujo. Una ruta inválida conserva el
  valor, marca el campo y muestra un `ERROR` en la zona de mensajes sin mover las
  filas. Las rutas largas se desplazan con `…` manteniendo visible el cursor.
- Si una fuente no existe o no se puede leer, se informa sin ocultar la otra.
- Variantes: filtro activo, sin clones (`No hay clones disponibles`), ruta manual
  con error de manifiesto, y scroll si hay más filas que espacio.

### 4.7 D1 · Apps de la implantación

```text
kamino · Implantar · Apps                                    Paso 2 de 4   * * o o
--------------------------------------------------------------------------------

  Elige las apps que quieres implantar

  SEL  APP                     ARCHIVOS
  --------------------------------------------------------------------------
  > [ ] herdr                       12
    [ ] ghostty                      8
    [ ] git                          4
    [ ] bat                          3

  ^ 0 arriba   v 2 abajo
  Sin base en este Mac: claude-code, yazi

AVISO  Cierra las apps antes de continuar: pueden sobrescribir los cambios al cerrarse.
--------------------------------------------------------------------------------
^v mover  Espacio marcar  a/n todas/ninguna  Enter continuar  Esc volver  ? ayuda
```

Variante: preparación.

```text
  Preparando la implantación...

  Verificando el archivo y extrayendo el clon...
  Esc cancelar   Ctrl+C cancelar
```

Notas:

- Ninguna app empieza marcada.
- El aviso de cerrar apps va en la zona de mensajes, no en el body.
- Las apps no instaladas son informativas y no cuentan como omitidas.
- La preparación es cancelable y limpia el staging si se cancela.
- Un checksum fallido vuelve a D0 sin escribir.

#### 4.7.1 D1 · Pregunta de instalación

Aparece al pulsar Enter en D1 solo si alguna app ausente tiene una receta aplicable a este Mac. Si no hay ninguna, D1 pasa directamente al plan.

```text
kamino · Implantar · Apps                                    Paso 2 de 4   * * o o
--------------------------------------------------------------------------------

  Elige las apps que quieres implantar

  ¿Instalar las apps ausentes que tienen receta para este Mac?
  Marca las que quieras instalar. Las no marcadas quedan como «Sin base en este Mac».

  > [x] iTerm2  brew
    [ ] yazi  installer · desde https://example.com/yazi.sh

AVISO  ¿Instalar las apps ausentes que tienen receta para este Mac?
--------------------------------------------------------------------------------
^v mover  Espacio marcar  a/n todas/ninguna  Enter continuar  Esc volver  ? ayuda
```

Notas:

- Ninguna receta se marca por defecto. Rechazar una o todas no cancela la implantación: las no marcadas quedan como «Sin base en este Mac».
- El método se muestra siempre. Para `installer`, la URL exacta que se descargará.
- Esc vuelve a la elección de apps sin perder la selección.
- Al confirmar, la instalación se ejecuta durante la planificación, después de verificar el archivo y antes de clasificar los archivos y crear la instantánea.

### 4.8 D2 · Conflictos

```text
kamino · Implantar · Conflictos                              Paso 3 de 4   * * * o
--------------------------------------------------------------------------------

  Conflictos
  Nuevos 4   Clones perfectos 10   No soportados 1   Avisos 1

  ARCHIVO                          BLOQUES  LÍNEAS  ACCIÓN
  --------------------------------------------------------------------------
  > ~/.config/herdr/config.toml         2      14  Revisar
    ~/.config/ghostty/config            1       3  Aplicar clon
    ~/.config/git/config                0       0  Conservar original
    ~/.config/bat/config                3       7  Revisar

  ^ 0 arriba   v 0 abajo

--------------------------------------------------------------------------------
^v mover  r revisar  c/o clon/original  C/O todo  Enter confirmar  Esc volver  ? ayuda
```

Notas:

- Solo se listan los conflictos. Los conteos de nuevos, clones perfectos, no
  soportados y avisos van arriba.
- El cero se muestra como `.`.
- `r` abre D3; `c` aplica el clon; `o` conserva el original. `C`/`O` actúan
  sobre todos.
- Enter solo avanza si todos los conflictos tienen decisión. Si falta alguno, la
  zona de mensajes muestra un `AVISO` con el archivo pendiente.
- Sin conflictos se muestra «Sin conflictos: todo se puede aplicar» y Enter pasa
  a D4.
- Variantes: scroll, todas las filas decididas, y aviso de secretos en la zona de
  mensajes al abrir D3.

### 4.9 D3 · Revisión por líneas

Vista completa del archivo, dos columnas. El ejemplo muestra un bloque; la vista
recorre el archivo entero con scroll.

```text
kamino · Implantar · Conflictos                              Paso 3 de 4   * * * o
--------------------------------------------------------------------------------

  ~/.config/herdr/config.toml

  Original                                      Clon
  12 líneas elegidas                            13 líneas elegidas

  bloque 1 de 2
  >  12 * const a = 1                             12 o const a = 2
     13 o -- sin línea --                          13 * const b = 3
     14   contexto igual                           14   contexto igual
     15 * const some_long_key = "value_that..."    15 o const some_long_key = "value_that..."
     16 * const other = true                       16 o const other = false

  ^ 120 arriba   v 19850 abajo

AVISO  Los diffs pueden contener secretos.
--------------------------------------------------------------------------------
^v línea  <> / Espacio elegir  c/o bloque  C/O archivo  Tab bloque  Esc volver  ? ayuda
```

Notas:

- Cada fila cambiada tiene exactamente una versión elegida (`*`); la otra queda
  en `Faint`. No existe un estado sin decidir.
- Las líneas iguales no llevan casilla y se muestran como contexto.
- El lado sin línea muestra `-- sin línea --`; elegirlo omite la línea.
- Los contadores bajo cada título cuentan solo líneas cambiadas elegidas de ese
  lado.
- `↑↓` mueve la fila; `←→` o `Espacio` eligen versión; `c`/`o` actúan sobre el
  bloque; `C`/`O` sobre todo el archivo; `Tab`/`Shift+Tab` recorren bloques;
  `Esc` vuelve y valida el híbrido.
- Las líneas largas se truncan por la derecha con `...`. Las rutas conservan el
  final y abrevian el home con `~`.
- El aviso de secretos va en la zona de mensajes, no en el body.
- Solo se renderiza la ventana visible. El modelo de filas puede tener 20 000
  entradas; no se montan 20 000 widgets.

Variante: decisión de archivo entero (binario, texto de más de 5 MB o enlace).

```text
  Conflicto de archivo entero

  Ruta: ~/.config/herdr/state.bin
  Motivo: binario (contiene NUL)

  > Aplicar clon
    Conservar original

--------------------------------------------------------------------------------
^v mover  Enter aceptar  Esc volver  ? ayuda
```

### 4.10 D4 · Confirmación

```text
kamino · Implantar · Confirmación                            Paso 4 de 4   * * * *
--------------------------------------------------------------------------------

  Confirma la implantación

  APP               NUEVOS  REEMPLAZOS  HÍBRIDOS  SIN DESPLEGAR  SIN CAMBIOS
  --------------------------------------------------------------------------
  herdr                  2           3         1              1            4
  ghostty                1           2         .              .            3
  git                    .           1         1              .            2
  bat                    .           .         .              1            5
  --------------------------------------------------------------------------
  TOTAL                  3           6         2              2           14

  Protegidos: 8    Holocrón: ~/.config/kamino/vault/snapshot/20261113.tar.gz

  Abortar misión        > Retirada <        Implantar

AVISO  Cierra las apps antes de implantar: pueden sobrescribir los cambios.
--------------------------------------------------------------------------------
<> mover  Enter aceptar  Esc retroceder  ? ayuda
```

Notas:

- Filas por app; columnas Nuevos, Reemplazos, Híbridos, Sin implantar y Sin
  cambios. El cero es `.`.
- La fila de totales va separada por una línea. No hay líneas entre filas.
- `Protegidos = reemplazos + híbridos` contando solo archivos existentes que se
  escribirán. Un híbrido que compone idéntico al local cae en «Sin cambios» y no
  se cuenta como protegido.
- El aviso de cerrar apps va en la zona de mensajes, no en el body.
- La tabla se calcula con `ImplantSession.Preview(decisions)` y
  `ImplantPlan.Entries`, deduplicando por `Path` igual que `resolve`: la primera
  app gana. Así los totales coinciden con el núcleo.
- Las acciones son horizontales, con Retirada preseleccionada. No hay atajo de
  una tecla para implantar: hay que mover el foco y pulsar Enter.
- Si no hay originales que proteger, se muestra «No hay originales que proteger;
  no se creará holocrón» en lugar de la ruta.
- Variantes: muchas apps con scroll y tabla abreviada a 80 columnas.

### 4.11 D5 · Resumen de la implantación

```text
kamino · Implantar · Resumen
--------------------------------------------------------------------------------

  Implantación completada

  RESUMEN
  --------------------------------------------------------------------------
  Implantados                                                            9
  Híbridos                                                               2
  Sin implantar                                                          2
  Sin cambios                                                           14
  Sin base en este Mac                                                   2
  Archivos protegidos                                                    8

  Holocrón: ~/.config/kamino/vault/snapshot/20261113.tar.gz
  Para deshacer: kamino implant ~/.config/kamino/vault/snapshot/20261113.tar.gz

  c Copiar holocrón

--------------------------------------------------------------------------------
c copiar holocrón  Enter/Esc continuar  ? ayuda
```

Notas:

- «Implantados» incluye nuevos y reemplazos; el detalle puede mostrarse como
  `9 (3 nuevos, 6 reemplazos)`.
- Un híbrido cuenta como éxito. El código 3 se reserva a omitidos, no soportados
  y `--apps` no cumplidos.
- La variante «Implantación incompleta» usa el título en rojo, un `ERROR` en la
  zona de mensajes, muestra aplicados, fallido y holocrón, y termina con código 1.
  No hay rollback automático.
- Si no se creó holocrón, se muestra «No había originales que proteger».
- `c` copia la ruta del holocrón; el aviso `INFO` va en la zona de mensajes.

