# Notación e inventario de pantallas

## 2. Notación de las maquetas

| ASCII       | Real       | Uso                                      |
| ----------- | ---------- | ---------------------------------------- |
| `>`         | `▌`        | cursor o foco                            |
| `*`         | `●` / `◉`  | elegido, cápsula original                |
| `o`         | `○`        | no elegido, cápsula vacía                |
| `[x]` `[ ]` | `[x]` `[ ]`| casilla marcada / vacía                  |
| `[~]`       | `[~]`      | casilla parcial (algunos archivos)       |
| `[-]`       | `[–]`      | casilla deshabilitada                    |
| `-`         | `─`        | separador                                |
| `...`       | `…`        | truncado                                 |
| `.`         | `·`        | cero                                     |
| `^` `v`     | `▲` `▼`    | scroll arriba / abajo                    |
| `<` `>`   | `◀` `▶`    | elegir versión original / clon           |
| `|`         | vídeo inverso | cursor de edición sobre la celda     |
| `i`         | `ℹ`        | mensaje informativo                      |
| `!`         | `⚠`        | aviso                                    |
| `x`         | `✕`        | error                                    |
| `+` `-`     | `▸` `▾`    | sección colapsada / expandida            |
| `~`         | `~`        | abreviatura del home                     |

Las maquetas muestran el body de 96 columnas. En una terminal de 100 columnas el
bloque se centra con 2 columnas de gutter a cada lado. En 80 columnas el body baja
a 76; en 140+ sube a 120.

## 3. Inventario de pantallas

| ID       | Header                         | Paso    | Mensajes | Spec             |
| -------- | ------------------------------ | ------- | -------- | ---------------- |
| Menu     | `kamino · Menú`                | no      | no     | tui-navigation   |
| C1       | `kamino · Clonar · Apps`       | 1 de 3  | sí     | clone-tui        |
| C2       | `kamino · Clonar · Destino`    | 2 de 3  | sí     | clone-tui        |
| C3       | `kamino · Clonar · Revisión` / `Resultado` | 3 de 3 | sí | clone-tui |
| D0       | `kamino · Implantar · Clon`    | 1 de 4  | sí     | implant-conflicts |
| D1       | `kamino · Implantar · Apps`    | 2 de 4  | sí     | implant-conflicts |
| D2       | `kamino · Implantar · Conflictos` | 3 de 4 | sí    | implant-conflicts |
| D3       | `kamino · Implantar · Conflictos` | 3 de 4 | sí    | implant-conflicts |
| D4       | `kamino · Implantar · Confirmación` | 4 de 4 | sí  | implant-conflicts |
| D5       | `kamino · Implantar · Resumen` | no      | sí     | implant-conflicts |
| U1       | `kamino · Actualización`       | no      | sí     | self-update      |
| Settings | `kamino · Configuración`       | no      | sí     | user-configuration |
| A1       | `kamino · Configuración · Añadir app` | no | sí    | app-authoring    |
| A1b      | `kamino · Configuración · Personalizar app` | no | sí | app-authoring |
| Help     | `kamino · <Flujo> · Ayuda`     | sí      | no     | tui-navigation   |
| Error    | `kamino · Error`               | no      | no     | tui-navigation   |
| Small    | `kamino · Aviso`               | no      | sí     | tui-navigation   |

## 4. Pantallas
