# Cobertura de specs

Generado a partir de `openspec/specs/*/spec.md`. Cada requisito tiene una fila; la columna de pruebas indica el paquete o las pruebas que lo cubren.

Total: 99 requisitos.

| Capacidad | Requisito | Pruebas / verificación |
|---|---|---|
| `cli-options` | Alias de las opciones de clonación | `internal/cli/cli_test.go` (TestVersionAndHelp, TestCloneCommandParsesFlagsAndAliases, TestDeployRequiresYes, TestHelpAliases); `internal/integration/golden_test.go` (TestGoldenCodes) |
| `cli-options` | Alias de las opciones de despliegue | `internal/cli/cli_test.go` (TestVersionAndHelp, TestCloneCommandParsesFlagsAndAliases, TestDeployRequiresYes, TestHelpAliases); `internal/integration/golden_test.go` (TestGoldenCodes) |
| `cli-options` | Selección de apps en el despliegue | `internal/cli/cli_test.go` (TestVersionAndHelp, TestCloneCommandParsesFlagsAndAliases, TestDeployRequiresYes, TestHelpAliases); `internal/integration/golden_test.go` (TestGoldenCodes) |
| `cli-options` | Simulación sin escrituras | `internal/cli/cli_test.go` (TestVersionAndHelp, TestCloneCommandParsesFlagsAndAliases, TestDeployRequiresYes, TestHelpAliases); `internal/integration/golden_test.go` (TestGoldenCodes) |
| `cli-options` | Confirmación de despliegues no interactivos | `internal/cli/cli_test.go` (TestVersionAndHelp, TestCloneCommandParsesFlagsAndAliases, TestDeployRequiresYes, TestHelpAliases); `internal/integration/golden_test.go` (TestGoldenCodes) |
| `cli-options` | Resumen de despliegue con mezclas | `internal/cli/cli_test.go` (TestVersionAndHelp, TestCloneCommandParsesFlagsAndAliases, TestDeployRequiresYes, TestHelpAliases); `internal/integration/golden_test.go` (TestGoldenCodes) |
| `cli-options` | Ayuda y versión con alias | `internal/cli/cli_test.go` (TestVersionAndHelp, TestCloneCommandParsesFlagsAndAliases, TestDeployRequiresYes, TestHelpAliases); `internal/integration/golden_test.go` (TestGoldenCodes) |
| `cli-options` | Compatibilidad de las opciones existentes | `internal/cli/cli_test.go` (TestVersionAndHelp, TestCloneCommandParsesFlagsAndAliases, TestDeployRequiresYes, TestHelpAliases); `internal/integration/golden_test.go` (TestGoldenCodes) |
| `cli-options` | Nombre por defecto y reemplazo del clon | `internal/cli/cli_test.go` (TestVersionAndHelp, TestCloneCommandParsesFlagsAndAliases, TestDeployRequiresYes, TestHelpAliases); `internal/integration/golden_test.go` (TestGoldenCodes) |
| `cli-options` | Apertura de la TUI y omisión de pantallas | `internal/cli/cli_test.go` (TestVersionAndHelp, TestCloneCommandParsesFlagsAndAliases, TestDeployRequiresYes, TestHelpAliases); `internal/integration/golden_test.go` (TestGoldenCodes) |
| `clone-safety` | Exclusión de credenciales independientes conocidas | `internal/paths/paths_test.go`; `internal/archive/archive_test.go`; `internal/plan/plan_test.go`; `internal/clone/clone_test.go`; `internal/deploy/deploy_test.go` |
| `clone-safety` | Exclusión de historiales y cachés | `internal/paths/paths_test.go`; `internal/archive/archive_test.go`; `internal/plan/plan_test.go`; `internal/clone/clone_test.go`; `internal/deploy/deploy_test.go` |
| `clone-safety` | Aviso de contenido sensible | `internal/paths/paths_test.go`; `internal/archive/archive_test.go`; `internal/plan/plan_test.go`; `internal/clone/clone_test.go`; `internal/deploy/deploy_test.go` |
| `clone-safety` | Confinamiento del clon al HOME | `internal/paths/paths_test.go`; `internal/archive/archive_test.go`; `internal/plan/plan_test.go`; `internal/clone/clone_test.go`; `internal/deploy/deploy_test.go` |
| `clone-safety` | Confinamiento del despliegue al HOME | `internal/paths/paths_test.go`; `internal/archive/archive_test.go`; `internal/plan/plan_test.go`; `internal/clone/clone_test.go`; `internal/deploy/deploy_test.go` |
| `clone-safety` | Orden y compatibilidad del despliegue | `internal/paths/paths_test.go`; `internal/archive/archive_test.go`; `internal/plan/plan_test.go`; `internal/clone/clone_test.go`; `internal/deploy/deploy_test.go` |
| `clone-safety` | Reemplazo explícito y atómico del clon | `internal/paths/paths_test.go`; `internal/archive/archive_test.go`; `internal/plan/plan_test.go`; `internal/clone/clone_test.go`; `internal/deploy/deploy_test.go` |
| `clone-safety` | Cancelación sin efectos del clon | `internal/paths/paths_test.go`; `internal/archive/archive_test.go`; `internal/plan/plan_test.go`; `internal/clone/clone_test.go`; `internal/deploy/deploy_test.go` |
| `clone-tui` | Selección de apps para clonar | `internal/tui/app_test.go` (TestCloneFlowDryRun, TestCloneCancelBeforeWrite, TestSmallTerminalKeepsState) |
| `clone-tui` | Destino del clon | `internal/tui/app_test.go` (TestCloneFlowDryRun, TestCloneCancelBeforeWrite, TestSmallTerminalKeepsState) |
| `clone-tui` | Revisión del plan de clonación | `internal/tui/app_test.go` (TestCloneFlowDryRun, TestCloneCancelBeforeWrite, TestSmallTerminalKeepsState) |
| `clone-tui` | Progreso y cancelación de la clonación | `internal/tui/app_test.go` (TestCloneFlowDryRun, TestCloneCancelBeforeWrite, TestSmallTerminalKeepsState) |
| `clone-tui` | Resumen de la clonación | `internal/tui/app_test.go` (TestCloneFlowDryRun, TestCloneCancelBeforeWrite, TestSmallTerminalKeepsState) |
| `deploy-conflicts` | Detección de conflictos de contenido | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Selección de apps en el despliegue interactivo | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Tabla de conflictos | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Decisión por línea dentro del bloque | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Acciones globales y por bloque en el detalle | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Decisión obligatoria y acciones globales | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Confirmación final | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Composición del archivo por líneas | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Modo y clasificación del resultado | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Validación del resultado | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Conflictos sin diff de líneas | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Seguridad y cancelación | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Resumen del resultado | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Validación del manifiesto en D0 | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Categorías del resultado y código 3 | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Holocrón de seguridad | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Cancelación y aplicación parcial | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Fuentes y orden de los clones en D0 | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Navegación y selección por teclado en D0 | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Ruta manual como última opción en D0 | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `deploy-conflicts` | Conservación de la selección al volver a D0 | `internal/conflicts/conflicts_test.go`; `internal/deploy/deploy_test.go`; `internal/tui/app_test.go` (TestDeployFlowEndToEnd); `internal/tui/screens_test.go` (TestD3BlockAndFileActions, TestD4AggregationTotalsAndDedup) |
| `ghostty-config-references` | Archivos y temas globales de Ghostty | `internal/plan/plan_test.go` (TestGhosttyReferencesAndCycles); `internal/ghostty` |
| `ghostty-config-references` | Referencias de archivos de configuración | `internal/plan/plan_test.go` (TestGhosttyReferencesAndCycles); `internal/ghostty` |
| `ghostty-config-references` | Referencias de temas personalizados | `internal/plan/plan_test.go` (TestGhosttyReferencesAndCycles); `internal/ghostty` |
| `ghostty-config-references` | Referencias sin duplicados | `internal/plan/plan_test.go` (TestGhosttyReferencesAndCycles); `internal/ghostty` |
| `ghostty-config-references` | Advertencia por rutas absolutas no portables | `internal/plan/plan_test.go` (TestGhosttyReferencesAndCycles); `internal/ghostty` |
| `global-app-config` | Catálogo integrado de herramientas de desarrollo | `internal/catalog/catalog_test.go`; `internal/detect/detect_test.go`; `internal/plan/plan_test.go` |
| `global-app-config` | Detección sin ejecutar herramientas | `internal/catalog/catalog_test.go`; `internal/detect/detect_test.go`; `internal/plan/plan_test.go` |
| `global-app-config` | Conservación de personalizaciones globales | `internal/catalog/catalog_test.go`; `internal/detect/detect_test.go`; `internal/plan/plan_test.go` |
| `global-app-config` | Configuración global de Oh My Zsh | `internal/catalog/catalog_test.go`; `internal/detect/detect_test.go`; `internal/plan/plan_test.go` |
| `global-app-config` | Tema y atajos globales de Herdr | `internal/catalog/catalog_test.go`; `internal/detect/detect_test.go`; `internal/plan/plan_test.go` |
| `global-app-config` | Rutas opcionales y omisiones reales | `internal/catalog/catalog_test.go`; `internal/detect/detect_test.go`; `internal/plan/plan_test.go` |
| `global-app-config` | Prioridad del catálogo del usuario | `internal/catalog/catalog_test.go`; `internal/detect/detect_test.go`; `internal/plan/plan_test.go` |
| `release-verification` | Pruebas de publicación aisladas | `internal/integration/golden_test.go` (TestGoldenManifestAndArchive, TestGoldenCodes); `internal/performance/startup_test.go`; `scripts/gen_golden.py` |
| `release-verification` | Estado inicial coherente de las pruebas | `internal/integration/golden_test.go` (TestGoldenManifestAndArchive, TestGoldenCodes); `internal/performance/startup_test.go`; `scripts/gen_golden.py` |
| `release-verification` | Integridad de la publicación verificada | `internal/integration/golden_test.go` (TestGoldenManifestAndArchive, TestGoldenCodes); `internal/performance/startup_test.go`; `scripts/gen_golden.py` |
| `release-verification` | Repetición sin cambios publicables | `internal/integration/golden_test.go` (TestGoldenManifestAndArchive, TestGoldenCodes); `internal/performance/startup_test.go`; `scripts/gen_golden.py` |
| `self-update` | Actualización exclusiva del binario instalado | `internal/update/update_test.go`; `internal/cli/cli_test.go` (TestUpdateCommand) |
| `self-update` | Destino del reemplazo | `internal/update/update_test.go`; `internal/cli/cli_test.go` (TestUpdateCommand) |
| `self-update` | Consulta de la última release y comparación de versiones | `internal/update/update_test.go`; `internal/cli/cli_test.go` (TestUpdateCommand) |
| `self-update` | Verificación de integridad | `internal/update/update_test.go`; `internal/cli/cli_test.go` (TestUpdateCommand) |
| `self-update` | Reemplazo atómico y recuperación | `internal/update/update_test.go`; `internal/cli/cli_test.go` (TestUpdateCommand) |
| `self-update` | Compatibilidad de plataforma | `internal/update/update_test.go`; `internal/cli/cli_test.go` (TestUpdateCommand) |
| `self-update` | Sin privilegios elevados | `internal/update/update_test.go`; `internal/cli/cli_test.go` (TestUpdateCommand) |
| `self-update` | Salida y códigos de salida | `internal/update/update_test.go`; `internal/cli/cli_test.go` (TestUpdateCommand) |
| `self-update` | Descubrimiento en el menú | `internal/update/update_test.go`; `internal/cli/cli_test.go` (TestUpdateCommand) |
| `self-update` | Progreso y cancelación en la TUI | `internal/update/update_test.go`; `internal/cli/cli_test.go` (TestUpdateCommand) |
| `self-update` | Errores de actualización | `internal/update/update_test.go`; `internal/cli/cli_test.go` (TestUpdateCommand) |
| `tui-ansi-theme` | Paleta ANSI del terminal | `internal/tui/app_test.go` (TestHelpAndNoColor); `internal/tui/screens_*.go` |
| `tui-ansi-theme` | Tema sobrescribible | `internal/tui/app_test.go` (TestHelpAndNoColor); `internal/tui/screens_*.go` |
| `tui-ansi-theme` | NO_COLOR y monocromo | `internal/tui/app_test.go` (TestHelpAndNoColor); `internal/tui/screens_*.go` |
| `tui-navigation` | Aplicación única y apertura con terminal | `internal/tui/app_test.go` (TestSizeGuardAndMenuNavigation, TestHelpAndNoColor, TestDeployFlow) |
| `tui-navigation` | Menú principal | `internal/tui/app_test.go` (TestSizeGuardAndMenuNavigation, TestHelpAndNoColor, TestDeployFlow) |
| `tui-navigation` | Navegación por teclado y pie de atajos | `internal/tui/app_test.go` (TestSizeGuardAndMenuNavigation, TestHelpAndNoColor, TestDeployFlow) |
| `tui-navigation` | Navegación y conservación de estado | `internal/tui/app_test.go` (TestSizeGuardAndMenuNavigation, TestHelpAndNoColor, TestDeployFlow) |
| `tui-navigation` | Numeración de pasos adaptable | `internal/tui/app_test.go` (TestSizeGuardAndMenuNavigation, TestHelpAndNoColor, TestDeployFlow) |
| `tui-navigation` | Ayuda contextual | `internal/tui/app_test.go` (TestSizeGuardAndMenuNavigation, TestHelpAndNoColor, TestDeployFlow) |
| `tui-navigation` | Error común | `internal/tui/app_test.go` (TestSizeGuardAndMenuNavigation, TestHelpAndNoColor, TestDeployFlow) |
| `tui-navigation` | Terminal pequeña y redimensionado | `internal/tui/app_test.go` (TestSizeGuardAndMenuNavigation, TestHelpAndNoColor, TestDeployFlow) |
| `tui-navigation` | Ctrl+C, salida y códigos | `internal/tui/app_test.go` (TestSizeGuardAndMenuNavigation, TestHelpAndNoColor, TestDeployFlow) |
| `tui-navigation` | Frases, logo y degradación por altura | `internal/tui/app_test.go` (TestSizeGuardAndMenuNavigation, TestHelpAndNoColor, TestDeployFlow) |
| `tui-navigation` | Última misión | `internal/tui/app_test.go` (TestSizeGuardAndMenuNavigation, TestHelpAndNoColor, TestDeployFlow) |
| `user-configuration` | Archivo de configuración opcional de la TUI | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | La vía no interactiva no usa la configuración | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | Creación del archivo solo al guardar | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | Esquema estricto de configuración | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | Valores por defecto | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | Creación de directorios al usarlos | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | Precedencia de rutas en la TUI | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | Pantalla de configuración | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | Guardado atómico y aplicación inmediata | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | Carga inválida al arrancar la TUI | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | Recuperación tras corregir la configuración | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | Sin automatización de confirmaciones destructivas | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | Invariantes de seguridad conservados | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
| `user-configuration` | Holocrones anteriores sin migración | `internal/config/config_test.go`; `internal/tui/app_test.go` (TestSettingsSave) |
