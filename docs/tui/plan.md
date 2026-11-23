# Plan de implementación de la TUI

## 7. Impacto en specs y documentación

| Artefacto | Cambio |
| --------- | ------ |
| `tui-navigation` | Frame, pila, contrato `Screen`, footer/ayuda desde `Keys()`, altura, menú, terminal pequeña |
| `tui-ansi-theme` | Solo ANSI, jerarquía, `theme.Styles`, `NO_COLOR`, ASCII, `ansi-light` |
| `clone-tui` | C1 como tabla y campo de filtro; C2 con tres opciones de carpeta, campos «Destino» y «Nombre» y diálogo Sobrescribir/Cambiar nombre/Volver; C3 absorbe C4: revisión, progreso y resultado en la misma tabla; el flujo sigue en tres pasos |
| `implant-conflicts` | D2 como tabla, D3 con archivo completo, D4 por app y protegidos |
| `docs/tecnico/arquitectura.md` | Sustituir «ANSI propio, sin framework» por Bubble Tea/Lip Gloss y el contrato de pantalla |
| `docs/producto/funcional.md` | Actualizar C1, C3/C4, D3 y D4 |
| `AGENTS.md` | Retirar el invariante «sin framework» y describir el nuevo runtime |

## 8. Plan de implementación

1. `theme` (tokens, estilos, símbolos) y `Frame`; `App` con pila y
   `tea.WindowSizeMsg`; harness de golden files y test de contrato.
2. C1, C3, D3 y D4 como pantallas de referencia, con goldens a 80×24, 100×26,
   140×40 y 200×50, más `NO_COLOR` y ASCII.
3. Resto por derivación: menú, C2, D0–D2, D5, U1, configuración, ayuda, error y
   aviso de terminal pequeña.
4. Comandos asíncronos para recuento, planificación, escritura, implantación y
   actualización; pruebas de cancelación y códigos de salida.
5. Actualizar specs, `docs/tecnico/` y `docs/producto/funcional.md` y `AGENTS.md`.

Cada paso debe compilar y pasar los golden files. Los commits siguen
Conventional Commits (`feat(tui):`, `refactor(tui):`, `test(tui):`).

## 9. Puntos a validar antes de generar goldens

1. **D3:** ¿se acepta mostrar el archivo completo en dos columnas y sustituir la
   restricción actual «sin previsualización completa»? El aviso de secretos y los
   límites de 5 MB/20 000 líneas se mantienen.
2. **D4:** ¿la tabla por app y la fórmula `protegidos = reemplazos + híbridos`
   representan lo que quieres validar? Los totales se calculan con `Preview` y la
   misma deduplicación por ruta que el núcleo.
3. **Layout:** `ColGap=4` corrige el 3 de las reglas para que `46 | 4 | 46` sume
   96. ¿Se acepta la corrección?
4. **Altura:** «30+ layout completo» se interpreta como más altura para el
   contenido, sin una zona adicional por altura. ¿Es suficiente o quieres reservar
   la zona de mensajes a partir de 30 filas?
5. **Menú:** las ocho cápsulas pasan al lado derecho del header y el wordmark
   queda en el body. ¿Se mantiene así?
6. **Acciones de D4:** orden `Abortar misión`, `Retirada`, `Implantar`, con
   Retirada preseleccionada. ¿Se mantiene el orden de la spec?
7. **`KAMINO_NO_LOGO`:** además de ocultar el wordmark, fuerza símbolos ASCII.
   ¿Es el comportamiento deseado?
8. **Asincronía:** Path A incluye mover a `tea.Cmd` las operaciones que hoy
   bloquean la UI. ¿Confirmas que entra en este cambio y no se deja para después?
9. **C3/C4:** la pantalla única conserva `Paso 3 de 3` en el estado final y cambia
   el header a `Resultado`. ¿Prefieres ocultar el paso al completar?
10. **Mensajes:** la zona muestra hasta tres líneas, una por categoría, y el body
    se reduce. ¿Prefieres una sola línea con prioridad `ERROR` > `AVISO` > `INFO`?
    Los errores bloqueantes siguen usando la pantalla de error común.
11. **C2 (resuelto):** las tres opciones de carpeta sustituyen a `F2`/`F3`/`F4`,
    que se retiran. La sugerencia del nombre incluye `.tar.gz`.
