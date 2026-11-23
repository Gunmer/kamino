# Pantallas del clon (C1–C4)

### 4.2 C1 · Apps del clon

```text
kamino · Clonar · Apps                                       Paso 1 de 3   * o o
--------------------------------------------------------------------------------

  Elige las apps que quieres clonar                    2 de 9 marcadas

  SEL  APP                     ARCHIVOS
  --------------------------------------------------------------------------
  > [x] herdr                       12
    [x] ghostty                      8
    [ ] git                          4
    [ ] bat                          3
    [-] glow                         0
    [ ] codex                        6

  ^ 0 arriba   v 3 abajo
  No detectadas: claude-code, yazi

--------------------------------------------------------------------------------
^v mover  Espacio marcar  a/n todas/ninguna  / filtrar  Enter continuar  Esc volver  ? ayuda
```

Notas:

- Solo aparecen apps instaladas. Las no detectadas van en una línea aparte y no
  son seleccionables.
- `glow` está instalada pero no tiene archivos: casilla `[-]` deshabilitada.
- El contador es de selección total, no de filas visibles.
- `Espacio` marca sin mover el cursor ni sacar el foco.
- `a`/`n` actúan solo sobre las filas visibles del filtro.
- Enter sin selección muestra un `AVISO` en la zona de mensajes y no avanza.
- Variantes: filtro activo, lista vacía (`No hay apps instaladas con archivos`) y
  lista desplazada con indicador. El filtro es una fila de campo, no texto suelto.

Variante: filtro con foco.

```text
  > Filtro    gh|
```

- El campo «Filtro» solo aparece mientras hay valor o foco. `Enter` aplica y
  devuelve el foco a la tabla; `Esc` lo limpia y lo cierra.

### 4.3 C2 · Destino del clon

```text
kamino · Clonar · Destino                                    Paso 2 de 3   * * o
--------------------------------------------------------------------------------

  Elige el destino del clon

  Carpeta de destino
  > * Vault por defecto        ~/.config/kamino/vault/
    o Carpeta actual           /Users/test/Projects/kamino
    o Indicar carpeta

    Nombre    clone-20261113.tar.gz

  Ruta: ~/.config/kamino/vault/clone-20261113.tar.gz

AVISO  El clon no está cifrado y puede contener secretos.
--------------------------------------------------------------------------------
^v mover  Enter elegir  Esc volver  ? ayuda
```

Variante: «Indicar carpeta» seleccionada. La opción abre el campo «Destino».

```text
  Carpeta de destino
    o Vault por defecto        ~/.config/kamino/vault/
    o Carpeta actual           /Users/test/Projects/kamino
    * Indicar carpeta

  > Destino   /Users/test/Descargas|
    Nombre    clone-20261113.tar.gz

  Ruta: /Users/test/Descargas/clone-20261113.tar.gz

--------------------------------------------------------------------------------
Enter fijar  Esc volver  ←→ cursor  Tab campo  F1/? ayuda
```

Variante: el clon ya existe.

```text
  El clon ya existe

  Ruta: ~/.config/kamino/vault/clone-20261113.tar.gz
  Fecha: 2026-11-12 10:30   Tamaño: 1.2 MB

    Sobrescribir
    Cambiar nombre
  > Volver
```

Notas:

- La primera opción muestra la ruta efectiva: `clone.output_dir` si está
  configurado, o `~/.config/kamino/vault/`.
- La segunda opción usa el directorio de trabajo actual (`pwd`), capturado al
  arrancar e inyectado; `View` no llama a `os.Getwd`.
- «Indicar carpeta» abre el campo «Destino». La carpeta puede no existir; se crea
  recién al empezar a escribir, nunca al abrir C2.
- El campo «Nombre» se rellena con `clone-20261113.tar.gz` y es editable. La
  sugerencia conserva el prefijo `clone-`, la fecha local y la extensión
  `.tar.gz`.
- Los dos campos comparten una fila: marcador `▌`, etiqueta alineada y valor. El
  foco añade fondo ANSI, etiqueta en negrita y el cursor en vídeo inverso; sin
  foco, el valor queda atenuado. Una ruta vacía muestra un placeholder atenuado.
- `Enter` en la ruta pasa al nombre; `Enter` en el nombre fija el destino. Si el
  archivo existe, abre el diálogo; si no, continúa a C3.
- El diálogo muestra ruta, fecha y tamaño. `Volver` está preseleccionada;
  `Sobrescribir` confirma el reemplazo y continúa a C3; `Cambiar nombre` enfoca
  el campo «Nombre» y propone el primer sufijo libre.
- `--force` omite el diálogo y muestra el aviso de reemplazo como `AVISO` en la
  zona de mensajes. `--output` omite C2; si el valor es inválido, C2 se abre con
  «Indicar carpeta» y los campos precargados, y un `ERROR` en la zona de
  mensajes.
- La validación en vivo muestra un `ERROR` en la zona de mensajes y marca el
  campo; no abre la pantalla de error común ni añade filas.
- Las tres opciones sustituyen a `F2`, `F3` y `F4`. Esos atajos se retiran de C2
  y de su footer; el cambio se documenta en `clone-tui` y en la leyenda.
- El footer cambia de contexto: en las opciones, `^v mover`/`Enter elegir`; en un
  campo, `Enter fijar` o `Enter continuar`, `Esc volver`, `←→ cursor`,
  `Tab campo` cuando hay dos y `F1/? ayuda`; en el diálogo,
  `^v mover`/`Enter aceptar`.
- Las rutas largas se desplazan dentro del valor con `…` al inicio y al final
  cuando corresponde, sin partir UTF-8 ni caracteres de doble ancho. El home se
  abrevia con `~` en el preview de la ruta completa.

### 4.4 C3 · Revisión, progreso y resultado

C3 absorbe la antigua C4. La pantalla es la misma tabla en todos los estados;
solo cambia la fila `Estado` y las acciones disponibles. El flujo de clonación
sigue teniendo tres pasos: apps, destino y revisión. El estado final no añade un
cuarto paso.

Estado de revisión:

```text
kamino · Clonar · Revisión                                   Paso 3 de 3   * * *
--------------------------------------------------------------------------------

  Clonación
  Revisa el plan antes de continuar.

  PLAN
  --------------------------------------------------------------------------
  Apps                                                                   2
  Archivos                                                              20
  Tamaño                                                             1.2 MB
  Ruta                              ~/.config/kamino/vault/clone-20261113.tar.gz
  Estado   Listo para clonar

  + Rutas omitidas (3)      + Archivos incluidos (20)

AVISO  El clon no está cifrado y puede contener secretos.
--------------------------------------------------------------------------------
d omitidos  f archivos  Enter clonar  Esc volver  Ctrl+C cancelar  ? ayuda
```

Estado de escritura:

```text
kamino · Clonar · Revisión                                   Paso 3 de 3   * * *
--------------------------------------------------------------------------------

  Clonación
  No cierres la terminal; la operación continúa.

  PLAN
  --------------------------------------------------------------------------
  Apps                                                                   2
  Archivos                                                              20
  Tamaño                                                             1.2 MB
  Ruta                              ~/.config/kamino/vault/clone-20261113.tar.gz
  Estado   Escribiendo · 12 de 20 archivos   [=========>          ]  60%

AVISO  Ctrl+C cancela antes del renombrado; el clon anterior queda intacto.
--------------------------------------------------------------------------------
Ctrl+C cancelar   ? ayuda
```

Estado final:

```text
kamino · Clonar · Resultado                                  Paso 3 de 3   * * *
--------------------------------------------------------------------------------

  Clonación
  El clon está listo.

  PLAN
  --------------------------------------------------------------------------
  Apps                                                                   2
  Archivos                                                              20
  Tamaño                                                             1.2 MB
  Ruta                              ~/.config/kamino/vault/clone-20261113.tar.gz
  Estado   Completado

  + Rutas omitidas (3)      c Copiar ruta

AVISO  Completado con 3 omitidos · código 3; pulsa d para verlos.
--------------------------------------------------------------------------------
d omitidos  c copiar ruta  Enter/Esc continuar  ? ayuda
```

Notas:

- La fila `Estado` es el único lugar donde vive la fase: `Planificando`,
  `Listo para clonar`, `Escribiendo` con barra de progreso, `Completado`,
  `Simulación completada` o `Cancelado antes de escribir`.
- El código 3 y los omitidos se muestran como `AVISO` en la zona de mensajes; la
  fila `Estado` no los duplica.
- El header cambia la última sección de `Revisión` a `Resultado` al terminar; el
  paso sigue siendo `Paso 3 de 3`.
- Acciones por estado:
  - revisión: `d`, `f`, `Enter` clonar, `Esc` volver, `Ctrl+C` cancelar;
  - escritura: solo `Ctrl+C` antes del renombrado; después se ignora con aviso;
  - final: `d` omitidos si hay, `c` copiar ruta si se creó el clon, `Enter`/`Esc`
    continuar.
- `f` solo está disponible en revisión. `d` está en revisión y en el estado
  final. `c` no aparece en simulación porque no se creó ningún archivo.
- La fila `Estado` se alinea a la izquierda y puede ocupar todo el ancho del
  valor; las filas numéricas siguen a la derecha.
- La barra de progreso se calcula con el ancho disponible y se actualiza por
  mensajes; a 80 columnas se acorta y el texto se trunca con `...`. `View` no usa
  `time.Now()`.
- `--dry-run` termina en `Resultado` con `Estado Simulación completada · no se
  escribió nada`, sin acción de copiar.
- Un error de planificación o escritura abre la pantalla de error común; la tabla
  queda en la pila para volver o reintentar.

### 4.5 C4 · absorbido por C3

La antigua C4 ya no es una pantalla separada. Su resumen es el estado final de
C3: la misma tabla, la misma ruta y una fila `Estado` que pasa de «Listo para
clonar» a la barra de progreso y al resultado. El flujo sigue teniendo tres
pasos; C3 es el tercero y su estado final no añade un paso. El enum `ScreenID`
no incluye `C4`; el estado final es `C3` con `phase = done`.

