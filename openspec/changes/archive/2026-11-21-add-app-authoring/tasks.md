# Tasks

## 1. Núcleo de catálogo y autoría

- [x] 1.1 Exportar el parseo estricto y añadir `InspectDir` con resultados por archivo y detección de ids duplicados; verificar con pruebas de archivo válido, TOML inválido, campo desconocido, duplicado y catálogo inexistente, y comprobar que `LoadCatalog` sigue siendo fail-fast.
- [x] 1.2 Añadir `DefaultExcludes` y `RenderTemplate` con comentarios en español, campos avanzados comentados y exclusión estándar activa; verificar con pruebas de contenido, de escapado de comillas y barras invertidas, y de que el TOML generado se parsea y valida.
- [x] 1.3 Añadir `CreateDefinition` con validación previa, comprobación de duplicados y override, creación de `catalog.dir` y escritura atómica; verificar con pruebas de creación, `--force`, duplicado en otro archivo, sin archivo parcial y directorio inexistente.

## 2. Resolución de `catalog.dir` y flujos compartidos

- [x] 2.1 Añadir `app.CatalogDir(home)` con fallback al directorio integrado ante archivo ausente o inválido; verificar con pruebas de configuración ausente, válida, con `catalog.dir` ausente e inválida, comprobando que un error en cualquier clave usa el valor por defecto.
- [x] 2.2 Conectar `ExecuteClone` y `ExecuteDeploy` a la resolución compartida, pasando `CatalogDir` a `deploy.Options`; verificar con pruebas de catálogo personalizado que incluye una app de usuario en el clon y que `deploy` la reconoce, sin cambiar las rutas integradas de salida y holocrones.
- [x] 2.3 Implementar los flujos `apps` de `internal/app` sobre `InspectDir`, `CreateDefinition` y la detección existente, con resultados estructurados y sin imprimir; verificar con pruebas de creación válida, duplicado, aviso de app no detectada y código de validación.
- [x] 2.4 Actualizar la tabla de `catalog.dir` y el árbol del proyecto en `README.md`, y el párrafo de configuración de `docs/diseno-tecnico.md`; verificar que ambos describen `catalog.dir` como compartido y el aviso de `apps` ante configuración inválida.

## 3. TUI: catálogo efectivo y recarga

- [x] 3.1 Hacer que `loadCatalog` cargue desde el `catalog.dir` efectivo; verificar con una prueba de TUI que una definición de usuario en un directorio personalizado aparece en C1/D1 y que el directorio por defecto sigue funcionando.
- [x] 3.2 Invalidar el catálogo en memoria al guardar Configuración con un `catalog.dir` distinto; verificar con pruebas de que el siguiente flujo usa el directorio nuevo en la misma sesión y de que un catálogo nuevo inválido muestra el error sin modificar archivos.
- [x] 3.3 Actualizar los textos `SettingsSubtitleTUI` y `SettingsCLINote` para reflejar que el catálogo también lo usa la CLI; verificar con las aserciones de textos y los goldens de Configuración.

## 4. CLI: comando `apps`

- [x] 4.1 Añadir el despacho de `apps` y los ayudas de `list`, `validate` y `new`; verificar con pruebas de `-h` en cada nivel y de comando desconocido con código 2.
- [x] 4.2 Añadir los flags de `apps new` (`--command`, `--bundle-id`, `--path`, `--optional-path`, `--name`, `--search-path`, `--force`, `--print`), incluido el soporte de flags repetibles; verificar con pruebas de creación válida, falta de detección o rutas con código 2, `--print` sin escritura, `--force` y aviso de app no detectada.
- [x] 4.3 Implementar la salida y los códigos de `apps list` y `apps validate`; verificar con pruebas de orden, origen integrado/usuario, archivo inválido mostrado por `list` con código 0, `validate` con código 1, duplicados entre archivos y avisos de `exclude`, id distinto del archivo y override.
- [x] 4.4 Actualizar la ayuda principal y `completionScript` para bash, zsh y fish; verificar que `kamino --show-completion` incluye `apps` en los tres shells y que la ayuda principal lista los subcomandos.
- [x] 4.5 Documentar la autoría por CLI en `README.md`, `docs/diseno-tecnico.md` y `docs/diseno-funcional.md`; verificar que los ejemplos usan flags reales y que la sección explica `catalog.dir`, `--print`, `--force` y los códigos de salida.

## 5. TUI: pantalla «Añadir app»

- [x] 5.1 Añadir «Añadir app» a Configuración y la pantalla con sus estados (intro, error, confirmación, éxito); verificar con pruebas de navegación, apertura y cancelación sin crear archivos.
- [x] 5.2 Implementar el borrador `.toml.tmp`, la resolución de editor inyectable (`$VISUAL`, `$EDITOR`, `vi`) y la validación al volver; verificar con pruebas de borrador sin tocar que no se guarda, TOML inválido que muestra errores, edición válida que crea `<id>.toml`, editor ausente que muestra la ruta y borrador invisible para `LoadCatalog`.
- [x] 5.3 Implementar las confirmaciones de override y sobrescritura, el rename atómico y la recarga del catálogo; verificar con pruebas de aceptar y rechazar, de que rechazar no modifica archivos y de que el catálogo se recarga sin reiniciar.
- [x] 5.4 Añadir textos, ayuda y estilos de la pantalla, y regenerar los goldens del menú y de la pantalla; verificar con `go test ./internal/tui -run Golden -update`, revisar los goldens y ejecutar el test de contrato de altura, ancho y SGR.
- [x] 5.5 Actualizar `docs/diseno-tui.md` y la sección de autoría por TUI de `README.md`; verificar que la pantalla documentada respeta 80x24, el foco, el borrador y las confirmaciones descritas.

## 6. Integración y verificación

- [x] 6.1 Añadir una prueba de integración con HOME temporal: crear una app por CLI, verla en el catálogo de la TUI y clonarla con `--apps`; crear otra con editor falso desde la TUI, validarla con `apps validate` y comprobar que `clone` la incluye. Verificar que no se lee ni modifica el HOME real ni `/Applications`.
- [x] 6.2 Ejecutar la suite completa: `gofmt -l .` vacío, `go vet ./...`, `golangci-lint run ./...` si está instalado, `go test ./... -race` y `openspec validate "add-app-authoring"`; verificar que todo pasa y que no hay errores de carrera.

## 7. Reubicación de «Añadir app» en Configuración

- [x] 7.1 Devolver el menú a cinco opciones y actualizar su golden; verificar navegación, salida y que «Añadir app» no está en el menú.
- [x] 7.2 Añadir la acción «Añadir app» a Configuración con foco, `Enter`, retorno a Configuración y guardado independiente; verificar pruebas y golden a 80x24.
- [x] 7.3 Actualizar proposal, design, specs, README y docs para la nueva ubicación; verificar `openspec validate --strict`.

## Workflow follow-up

- Archivar el cambio con OpenSpec cuando la revisión y la suite estén completas.
- Publicar mediante `cmd/release` para que el CHANGELOG y la versión se generen sin editarlos a mano.
