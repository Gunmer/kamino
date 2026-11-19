# Proposal

## Why

Los campos de texto de la TUI no comparten un mismo lenguaje visual: C2 y D0
dibujan cadenas desnudas con `_`, mientras que Configuración usa una línea para
la etiqueta y otra para el valor. En todos ellos el foco es ambiguo, el cursor
no se distingue con precisión y las rutas largas no se desplazan. Esto obliga a
la persona a adivinar qué campo está editando y dónde se insertará el siguiente
carácter.

## What Changes

- Añadir un componente de campo reutilizable en `internal/tui` que concentre
  render y edición: una sola fila con marcador, etiqueta y valor, foco visible,
  cursor de edición, placeholder, scroll horizontal y estado de error.
- Migrar todos los inputs de la TUI al componente:
  - C1: filtro `/`.
  - C2: ruta manual y nombre del archivo.
  - D0: filtro `/` y ruta manual (`Otro archivo...`).
  - Configuración: `catalog.dir`, `clone.output_dir`, `deploy.snapshots_dir` y
    el límite de holocrones.
- Añadir estilos ANSI de foco, atenuado y error al tema, sin truecolor, con
  `ansi-dark` y `ansi-light`, degradación `NO_COLOR` por forma (`▌`, vídeo
  inverso y la palabra «editando») y equivalentes ASCII.
- Usar el cursor de edición por vídeo inverso como alternativa determinista al
  cursor real del terminal: Bubble Tea v1.3.10 no permite posicionarlo y su
  renderer lo recoloca al inicio de la última línea tras cada repintado.
- Ajustar el pie de atajos para que, con un campo enfocado, muestre las teclas
  de esa pantalla y `F1`/`?` para la ayuda, sin insertar caracteres.
- Mantener el scroll del valor dentro del ancho disponible con `…` al inicio
  (y al final cuando corresponda), sin partir caracteres UTF-8 ni anchos.
- Añadir pruebas de modelo y de render para foco, cursor en medio, ruta larga,
  placeholder, error, `NO_COLOR`, ASCII y 80x24; actualizar goldens.
- Actualizar `README.md`, `docs/diseno-funcional.md`, `docs/diseno-tecnico.md`
  y la referencia visual `docs/diseno-tui.md`.

## Capabilities

### New Capabilities

Ninguna.

### Modified Capabilities

- `tui-ansi-theme`: estilos de foco, atenuado y error de los campos; fondo ANSI
  resaltado, acento, negrita de etiqueta, vídeo inverso con `NO_COLOR` y
  equivalentes ASCII.
- `tui-navigation`: contrato transversal del campo de texto, cursor de edición,
  scroll horizontal, pie de atajos con campo enfocado, ayuda con `F1` y `?`, y
  repintado limpio al cambiar de foco o de pantalla.
- `clone-tui`: filtro de C1 y campos de destino y nombre de C2 con el componente
  compartido, incluidos placeholder, scroll y errores.
- `deploy-conflicts`: filtro de D0 y ruta manual de `Otro archivo...` con el
  componente compartido.
- `user-configuration`: los cuatro campos de Configuración usan el componente,
  con foco visible, scroll de rutas y error sin desplazar las filas.

## Impact

- `internal/tui`: nuevo `field.go` con el componente y su lógica de edición;
  cambios en `screens_clone.go`, `screens_deploy.go`, `screens_settings.go`,
  `keys.go`, `frame.go` y `layout.go` si hace falta medir texto.
- `internal/tui/theme`: nuevos estilos de campo y símbolos de cursor/elipsis;
  sin colores nuevos fuera de la paleta ANSI.
- `internal/texts`: etiquetas, placeholders, «editando» y leyendas de atajos.
- Pruebas: unitarias del campo, render por pantalla, goldens de C1, C2, D0 y
  Configuración en modo color, `NO_COLOR` y ASCII, incluido 80x24.
- `README.md`, `docs/diseno-funcional.md`, `docs/diseno-tecnico.md` y
  `docs/diseno-tui.md`.
- Sin cambios en la CLI, los códigos de salida, el formato del clon, el
  manifiesto, las rutas de seguridad ni la lógica de `clone`/`deploy`.
