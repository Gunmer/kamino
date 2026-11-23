# Pantallas generales (menú, actualización, ajustes, ayuda, errores, A1)

### 4.1 Menú

Tamaño de la maqueta: 100×30 (layout completo). A 26–29 se ocultan subtítulo,
frase y última misión; a 24–25 se oculta además el aire.

```text
kamino · Menú                                    o o o o o o o *
--------------------------------------------------------------------------------

   [wordmark KAMINO]
   v0.10.0

   Clona e implanta configuraciones de macOS en un solo archivo.
   La Fuerza estará contigo.

   > Clonar             Crea un clon de las apps instaladas.
     Implantar          Implanta un clon desde un archivo.
     Actualizar         Actualiza a la última versión publicada.
     Configuración      Edita las rutas y añade apps.
     Abandonar Kamino   Cierra kamino.

   Última misión: clonación completada · 20 archivos
   aviso: el clon no está cifrado y puede contener secretos.

--------------------------------------------------------------------------------
^v mover   Enter aceptar   ? ayuda
```

Notas:

- Las ocho cápsulas viven en el lado derecho del header; la última es la original
  (`*`). El wordmark sigue en el body.
- La versión de la compilación se muestra en dim justo debajo del wordmark, alineada
  con su borde izquierdo. Acompaña al cuerpo del menú: aparece desde 24 filas y sigue
  visible cuando `KAMINO_NO_LOGO` oculta el wordmark; por debajo de 24 filas el aviso de
  terminal pequeña reemplaza el cuerpo.
- No hay atajos de una tecla para las opciones. Escape no cierra.
- La frase se elige al arrancar y no cambia durante la sesión.
- `KAMINO_NO_LOGO` oculta el wordmark y fuerza los símbolos ASCII.

### 4.12 U1 · Actualización

```text
kamino · Actualización
--------------------------------------------------------------------------------

  Actualizar kamino
  Versión actual: 0.9.0
  Última versión: 0.9.1

  Descargando y verificando...

  [=========>          ]  60%

  Ctrl+C cancela antes de reemplazar el binario.

--------------------------------------------------------------------------------
Enter continuar  Esc cancelar/salir  ? ayuda
```

Notas:

- Estados: consultando, descargando, verificando, reemplazando y resultado.
- Resultados: actualizado, ya al día, instalada más reciente, error de red,
  integridad, plataforma no soportada o instalación no autocontenida.
- La barra se calcula con el ancho del body; `View` no usa `time.Now()`.
- Un error de actualización abre la pantalla de error común con la acción
  contextual que corresponda.
- La consulta y la descarga son comandos asíncronos; la cancelación se comprueba
  por chunk.

### 4.13 Configuración

```text
kamino · Configuración
--------------------------------------------------------------------------------

  Configuración
  El catálogo y el límite de holocrones también los aplica la CLI.

  > Catálogo    ~/.config/kamino/apps|
    Clones      ~/.config/kamino/vault
    Holocrones  ~/.config/kamino/vault/snapshot
    Límite      5

    Añadir app  Crea una definición TOML en el catálogo del usuario.

--------------------------------------------------------------------------------
Tab/Mayús+Tab mover  Ctrl+S guardar  Esc volver  ←→ cursor  F1/? ayuda
```

Notas:

- Las tres rutas y el límite son campos editables de una fila con etiqueta
  alineada, no una lista de solo lectura. El foco añade marcador `▌`, fondo ANSI,
  etiqueta en negrita y el cursor en vídeo inverso; los demás valores quedan
  atenuados. El cursor solo aparece en el campo enfocado.
- `Tab`/`Mayús+Tab` y `↑`/`↓` mueven el foco entre los cuatro campos; `←`/`→`
  mueven el cursor dentro del valor y no cambian de campo. El foco empieza en
  Catálogo.
- El campo del límite solo admite dígitos; un valor vacío, negativo o fuera de
  rango se rechaza con `ERROR` al guardar.
- `Ctrl+S` valida y guarda de forma atómica; `Esc` vuelve sin escribir.
- Los valores vacíos muestran un placeholder atenuado. Las rutas largas se
  desplazan dentro del campo con `…` para mantener visible el cursor, sin partir
  caracteres UTF-8 ni de doble ancho; sin foco conservan el estilo atenuado.
- Si el archivo es inválido, la pantalla se abre con los valores legibles y los
  defaults, muestra un `ERROR` en la zona de mensajes y marca el campo afectado
  sin desplazar las filas.
- El aviso «Configuración guardada» va en la zona de mensajes como `INFO`.
- Después de los cuatro campos hay una acción «Añadir app». `Tab`/`Mayús+Tab` y
  las flechas verticales la alcanzan; `Enter` abre la pantalla de autoría y `Esc`
  vuelve al menú. La acción no se edita como un campo y `Enter` no guarda.
- Guardar un `catalog.dir` distinto descarta el catálogo en memoria; el siguiente
  flujo lo carga desde la ruta nueva sin reiniciar.
- `?` y `F1` abren la ayuda desde un campo sin insertar caracteres.

### 4.14 Ayuda contextual

```text
kamino · Implantar · Ayuda                                   Paso 3 de 4   * * * o
--------------------------------------------------------------------------------

  Ayuda · Conflictos

  Atajos de esta pantalla
  ^v mover          r revisar          c/o clon/original
  C/O todo          Enter confirmar    Esc volver

  Atajos globales
  ? ayuda           Ctrl+C cancelar

  Flujo de implantación
  * clon   * apps   * conflictos   o confirmación

  ! Los diffs pueden contener secretos.

--------------------------------------------------------------------------------
Enter/Esc/? cerrar
```

Notas:

- Los atajos salen de `Keys()`, no de una lista escrita a mano.
- El flujo resalta el paso actual con `*` y acento; los pendientes van con `o`.
- El aviso de secretos solo aparece en D2 y D3.
- En campos de texto, `F1` y `?` abren esta pantalla sin insertar caracteres; al
  cerrarla, el campo conserva su valor, su foco y su cursor.
- La ayuda se apila sobre la pantalla actual; al cerrar se recupera su estado.

### 4.15 Error común

```text
kamino · Error
--------------------------------------------------------------------------------

  No se pudo planificar el clon

  No se escribió nada.

  Qué hacer: Revisa el destino y vuelve a intentarlo.

  [t Ver detalle]      [> Volver al menú]      [Salir]

--------------------------------------------------------------------------------
t detalle  c copiar  Enter aceptar  Esc volver  ? ayuda
```

Variante: detalle abierto.

```text
  Detalle técnico

  open /Users/test/.config/kamino/vault: permission denied
```

Notas:

- Cuatro partes: qué pasó, estado de seguridad, qué hacer y acciones.
- El detalle está plegado, se abre con `t` y se copia con `c`. Nunca incluye
  contenido de archivos.
- En modo directo no se ofrece «Volver al menú»; solo la acción contextual y
  «Salir».
- Las acciones se recorren con flechas y Enter; Esc vuelve a la pantalla anterior.
- Los textos de error siguen siendo literales y viven en `internal/texts`.

### 4.16 Aviso de terminal pequeña

Tamaño: 79×23. La guardia vive en el `Frame`; la pila y el estado no se tocan.

```text
kamino · Aviso
-------------------------------------------------------------------------------

  La terminal es demasiado pequeña
  Mínimo: 80 x 24   Actual: 79 x 23

  El flujo sigue en segundo plano. Amplía la ventana para continuar.
  Operación en curso: clonación, 12 de 20 archivos.

  ? ayuda
```

Notas:

- Se actualiza al redimensionar y recupera la pantalla anterior al volver a
  80×24.
- La operación en curso no se cancela ni se cierra la aplicación.
- No se pierde el estado de la pantalla inferior.

### 4.17 A1 · Añadir app

La pantalla crea un borrador `.toml.tmp` en el catálogo efectivo, abre el editor
del usuario y valida el archivo al volver. Un borrador sin editar no se guarda.

```text
kamino · Configuración · Añadir app
--------------------------------------------------------------------------------

  Nueva definición
  Se creará un borrador y se abrirá tu editor.

  Catálogo: ~/.config/kamino/apps
  Borrador: ~/.config/kamino/apps/.kamino-draft-123.toml.tmp

  Usa rutas relativas al HOME, sin ".." ni rutas absolutas.
  El borrador no se guardará hasta que sea válido.

--------------------------------------------------------------------------------
Enter/e editar   Esc descartar                                      ? ayuda
```

Notas:

- Se abre desde la acción «Añadir app» de Configuración; al terminar o cancelar
  se vuelve a Configuración.
- `Enter` o `e` abre `$VISUAL`, `$EDITOR` o `vi` con `tea.ExecProcess`; al volver,
  Kamino parsea y valida el borrador con las mismas reglas que `LoadCatalog`.
- Si el TOML es inválido o el editor falla, la pantalla sigue abierta, muestra el
  `ERROR` en la zona de mensajes y la ruta del borrador, y permite reeditar o
  descartar con `Esc`.
- Si el borrador es válido, el archivo se renombra de forma atómica a
  `<id>.toml` y el catálogo se recarga. Si el id reemplaza una app integrada o el
  archivo destino ya existe, se muestra una confirmación antes de escribir.
- La pantalla de éxito muestra la ruta creada y si la app se detectó. Si no se
  detectó, se avisa de que aparecerá en Clonar cuando esté instalada.
- `Esc` antes de editar no crea archivos; `Esc` con un borrador lo elimina.

