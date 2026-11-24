// Package texts centralizes every user-visible string.
package texts

import (
	"fmt"
	"strings"

	"github.com/Gunmer/kamino/internal/errors"
)

const (
	AppName  = "kamino"
	Subtitle = "Clona e implanta configuraciones de macOS en un solo archivo."
)

var Wordmark = []string{
	"██╗  ██╗ █████╗ ███╗   ███╗██╗███╗   ██╗ ██████╗ ",
	"██║ ██╔╝██╔══██╗████╗ ████║██║████╗  ██║██╔═══██╗",
	"█████╔╝ ███████║██╔████╔██║██║██╔██╗ ██║██║   ██║",
	"██╔═██╗ ██╔══██║██║╚██╔╝██║██║██║╚██╗██║██║   ██║",
	"██║  ██╗██║  ██║██║ ╚═╝ ██║██║██║ ╚████║╚██████╔╝",
	"╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝╚═╝╚═╝  ╚═══╝ ╚═════╝ ",
}

var Phrases = []string{
	"La Fuerza estará contigo.",
	"Este es el camino.",
	"Siempre hay un clon más.",
	"Que la configuración te acompañe.",
	"Nunca subestimes un clon bien hecho.",
}

const (
	CapsuleTotal         = 8
	CapsuleEmpty         = "○"
	CapsuleOriginal      = "◉"
	CapsuleASCIIEmpty    = "o"
	CapsuleASCIIOriginal = "o"
	MarkerChosen         = "●"
	MarkerNotChosen      = "○"
	MarkerChosenASCII    = "*"
	MarkerNotChosenASCII = "."
)

const (
	Original         = "original"
	Snapshot         = "holocrón de seguridad"
	Hybrid           = "híbrido"
	Hybrids          = "híbridos"
	PerfectClone     = "clon perfecto"
	PerfectClones    = "clones perfectos"
	NotInstalled     = "sin base en este Mac"
	ForceDisturbance = "perturbación en la Fuerza"
	IntegrityFailed  = "clon con mutaciones"
	Manifest         = "registro de los kaminoanos"
)

const (
	PrivacyWarning = "aviso: el clon no está cifrado y las configuraciones incluidas pueden " +
		"contener secretos. Los archivos de credenciales conocidos se excluyen, " +
		"pero los ajustes permitidos se conservan íntegros."
	DiffWarning = "Los diffs pueden contener datos sensibles (tokens, claves o rutas). " +
		"No se guardan ni se registran fuera de esta pantalla."
)

const (
	MenuClone   = "Clonar"
	MenuImplant = "Implantar"
	MenuUpdate  = "Actualizar"
	MenuConfig  = "Configuración"
	MenuExit    = "Abandonar Kamino"

	MenuCloneHelp   = "Crea un clon de las apps instaladas."
	MenuImplantHelp = "Implanta un clon desde un archivo."
	MenuUpdateHelp  = "Actualiza a la última versión publicada."
	MenuConfigHelp  = "Edita las rutas y añade apps."
	MenuExitHelp    = "Cierra kamino."

	LastMission   = "Última misión: {summary}"
	NoLastMission = ""
)

var (
	CloneSteps   = []string{"apps", "destino", "revisión"}
	ImplantSteps = []string{"clon", "apps", "conflictos", "confirmación"}
)

const (
	CloneStepLabel = "Paso {current} de {total} · {title}"
)

var StepTitles = map[string]string{
	"apps": "apps", "destino": "destino", "revisión": "revisión", "clon": "clon",
	"conflictos": "conflictos", "confirmación": "confirmación", "actualización": "actualización",
}

var StepDescriptions = map[string]string{
	"apps": "apps", "destino": "destino", "revisión": "revisión", "clon": "clon",
	"conflictos": "conflictos", "confirmación": "confirmación",
}

const (
	CloneTitle            = "Clonar apps"
	CloneAppsTitle        = "Elige las apps que quieres clonar"
	CloneDestinationTitle = "Elige el destino del clon"
	CloneReviewTitle      = "Revisa el plan del clon"
	CloneSummaryTitle     = "Clonación completada"
	CloneDryRunTitle      = "Simulación del clon"
	ClonePlanning         = "Planificando el clon…"
	CloneWriting          = "Escribiendo el clon…"
	CloneCancelling       = "Cancelando; el clon anterior queda intacto…"
	CloneNoSelection      = "Selecciona al menos una app para continuar."
	CloneNothingToDo      = "No hay archivos para incluir en el clon."
	CloneAll              = "Todas"
	CloneNone             = "Ninguna"
	CloneContinue         = "Continuar"
	CloneBack             = "Atrás"
	CloneFilter           = "Filtro"
	CloneDetected         = "Detectadas"
	CloneNotDetected      = "Sin detectar en este Mac: {apps}"
	CloneFiles            = "{count} archivos"
	CloneZeroFiles        = "sin archivos"
	CloneAppsSummary      = "Apps: {count}"
	CloneFileCount        = "Archivos: {count}"
	CloneSize             = "Tamaño: {size}"
	CloneDestination      = "Destino: {path}"
	CloneSecrets          = PrivacyWarning
	CloneSkippedTitle     = "Rutas omitidas"
	CloneIncludedTitle    = "Archivos incluidos"
	CloneDryRunNote       = "Simulación: no se escribirá ningún archivo."
	CloneReplaceTitle     = "El clon ya existe"
	CloneReplaceBody      = "Ya existe un clon en {path}.\nCreado: {created}\nTamaño: {size}\n\n¿Quieres volver, elegir otro nombre o reemplazarlo al terminar?"
	CloneReplaceBack      = "Volver"
	CloneReplaceOther     = "Otro nombre"
	CloneReplaceForce     = "Reemplazar clon"
	CloneForceNotice      = "El clon existente se reemplazará al final."
	CloneExitLine         = "Salida 3: se omitieron rutas o apps pedidas."
	CloneCopyPath         = "Ruta copiada."
	CloneCopyUnsupported  = "Este terminal no admite copia OSC 52; copia la ruta a mano."
	CloneCancelled        = "Clonación cancelada; no se escribió nada."
	CloneComplete         = "Clon creado en {path}."
	CloneAppsLabel        = "Apps"
	CloneFilesLabel       = "Archivos"
	CloneSizeLabel        = "Tamaño"
	ClonePathLabel        = "Ruta"
)

const (
	ImplantTitle             = "Implantar un clon"
	ImplantFileTitle         = "Elige el clon que quieres implantar"
	ImplantAppsTitle         = "Elige las apps que quieres implantar"
	ImplantConflictsTitle    = "Conflictos"
	ImplantDetailTitle       = "Revisión por bloques"
	ImplantConfirmTitle      = "Confirma la implantación"
	ImplantSummaryTitle      = "Implantación completada"
	ImplantIncompleteTitle   = "Implantación incompleta"
	ImplantNoFile            = "Escribe o elige un archivo .tar.gz."
	ImplantFileInvalid       = "El archivo no es un clon válido: {message}"
	ImplantFileMissing       = "no existe o no es un archivo"
	ImplantOtherFile         = "Otro archivo..."
	ImplantOriginDefault     = "por defecto"
	ImplantOriginCWD         = "este directorio"
	ImplantOriginManual      = "ruta manual"
	ImplantSourceMissing     = "{origin}: no existe {path}"
	ImplantSourceUnreadable  = "{origin}: no se puede leer {path}: {message}"
	ImplantSourceEmpty       = "{origin}: sin clones en {path}"
	ImplantPreview           = "{path} · {date} · {size}"
	ImplantPreviewManual     = "Ruta manual: escribe la ruta completa de un clon .tar.gz."
	ImplantDateFormat        = "%Y-%m-%d %H:%M"
	ImplantManifestWarnings  = "Avisos del manifiesto"
	ImplantArchWarning       = "Arquitectura distinta: {message}"
	ImplantMacOSWarning      = "Versión de macOS distinta: {message}"
	ImplantPreparing         = "Preparando la implantación…"
	ImplantVerifying         = "Verificando la integridad del clon…"
	ImplantExtracting        = "Extrayendo en un directorio temporal…"
	ImplantClassifying       = "Clasificando los archivos…"
	ImplantApplying          = "Implantando el clon…"
	ImplantAfterSnapshot     = "El holocrón ya está creado: no se puede cancelar."
	ImplantNoneSelected      = "Selecciona al menos una app para continuar."
	ImplantNotInstalledTitle = "Sin base en este Mac"
	ImplantNotInstalledNote  = "Estas apps se informan, no se implantan y no cuentan como omitidas."
	ImplantNew               = "Nuevos: {count}"
	ImplantPerfect           = "Clones perfectos: {count}"
	ImplantChanged           = "Conflictos: {count}"
	ImplantUnsupported       = "No soportados: {count}"
	ImplantNotices           = "Avisos: {count}"
	ImplantActions           = "r revisar · c aplicar clon · o conservar original · C todo clon · O todo original"
	ImplantConfirm           = "Confirmar"
	ImplantAllClone          = "Todo al clon"
	ImplantAllOriginal       = "Todo al original"
	ImplantReview            = "Revisar"
	ImplantApply             = "Aplicar clon"
	ImplantOmit              = "Conservar original"
	ImplantImplanted         = "Implantados: {count}"
	ImplantImplantedDetail   = "{new} nuevos y {replaced} reemplazos"
	ImplantHybrids           = "Híbridos: {count}"
	ImplantOmitted           = "Omitidos: {count}"
	ImplantInstallPlanned    = "Se instalaría: {app} con {method}{url}"
	ImplantInstallUnverified = " (sin verificación de suma: solo HTTPS y origen GitHub)"
	ImplantInstallDone       = "Instalada: {app} con {method}"
	ImplantInstallFailed     = "No instalada: {app} ({reason})"
	ImplantInstallTitle      = "Instalaciones"
	ImplantNotImplanted      = "Sin implantar: {count}"
	ImplantUnchanged         = "Sin cambios: {count}"
	ImplantProtected         = "Archivos protegidos: {count}"
	ImplantSnapshotDir       = "Holocrón de seguridad en: {path}"
	ImplantSnapshotPath      = "Holocrón: {path}"
	ImplantNoSnapshot        = "No había originales que proteger; no se creó holocrón."
	ImplantOpenApps          = "Si vas a implantar sobre apps abiertas, ciérralas antes de continuar."
	ImplantBack              = "Retirada"
	ImplantAbort             = "Abortar misión"
	ImplantApplyButton       = "Implantar"
	ImplantUndo              = "Para deshacer: kamino implant {path}"
	ImplantExitLine          = "Salida {code}: se conservaron originales, no soportados o apps pedidas no cumplidas."
	ImplantPartial           = "Implantación incompleta: {applied} aplicados; falló {failed}."
	ImplantPartialSnapshot   = "Holocrón disponible para deshacer: {path}\nPara restaurarlo: kamino implant {path} --yes"
	ImplantCopyPath          = "Ruta del holocrón copiada."
	ImplantBlock             = "Bloque {index} de {total}"
	ImplantBlockCounts       = "Clon: {clone} líneas · Original: {original} líneas"
	ImplantRowClone          = "● clon"
	ImplantRowOriginal       = "○ original"
	ImplantRowCloneOnly      = "solo clon"
	ImplantRowOriginalOnly   = "solo original"
	ImplantDecisionHint      = "Espacio/Enter alterna · c/o bloque · C/O archivo · Tab/Mayús+Tab bloque"
	ImplantValidationInvalid = "Híbrido inválido: {message}"
)

const (
	UpdateTitle          = "Actualizar kamino"
	UpdateChecking       = "Consultando la última versión publicada…"
	UpdateDownloading    = "Descargando y verificando tamaño y SHA-256…"
	UpdateReplacing      = "Reemplazando el binario…"
	UpdateUpdated        = "kamino {current} → {latest}. Actualizado: {path}"
	UpdateUpToDate       = "kamino {current} ya es la última versión disponible."
	UpdateNewer          = "kamino {current} es más reciente que la última publicada ({latest}); no se modifica."
	UpdateRestart        = "Reinicia kamino para usar la versión nueva."
	UpdateCancelled      = "Actualización cancelada; el binario actual sigue intacto."
	UpdateCannotCancel   = "El reemplazo ya empezó: no se puede cancelar."
	UpdateResultUpdated  = "✔ Actualizado"
	UpdateResultUpToDate = "✔ Ya al día"
	UpdateResultNewer    = "▲ Instalada más reciente"
)

const (
	ErrorWhat            = "Qué pasó"
	ErrorSafety          = "Estado de seguridad"
	ErrorAction          = "Qué hacer"
	ErrorDetail          = "Detalle técnico"
	ErrorNothingWritten  = "No se escribió nada."
	ErrorCloneWritten    = "El clon anterior sigue intacto."
	ErrorSnapshotWritten = "El holocrón de seguridad está disponible para deshacer."
	ErrorBackToMenu      = "Volver al menú"
	ErrorRetry           = "Reintentar"
	ErrorExit            = "Salir"
	ErrorShowDetail      = "t detalle · c copiar"
	ErrorCopied          = "Detalle copiado."
	ErrorCopyUnsupported = "Este terminal no admite copia OSC 52; copia el detalle a mano."
	ErrorSmallTitle      = "La terminal es demasiado pequeña"
	ErrorSmallBody       = "Amplía la terminal a 80×24 o más para continuar.\nEl estado del flujo se conserva mientras redimensionas."
)

// ShortcutLine composes a key legend joined by middle dots.
func ShortcutLine(entries ...[2]string) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, entry[0]+" "+entry[1])
	}
	return strings.Join(parts, " · ")
}

var ShortcutsMenu = ShortcutLine([2]string{"↑↓", "mover"}, [2]string{"Enter", "aceptar"}, [2]string{"?", "ayuda"})

const (
	StatusOK     = "✔ listo"
	StatusWarn   = "▲ aviso"
	StatusError  = "✖ error"
	IconSuccess  = "✔"
	IconWarning  = "▲"
	IconError    = "✖"
	IconOriginal = "◉"

	Cancelled         = "cancelado"
	RequiresYes       = "se requiere --yes para implantar sin terminal interactiva"
	CloneForceHelp    = "Reemplaza un clon existente de forma atómica. Sin esta opción, un clon existente es un error."
	CloneExistsHint   = "El clon ya existe: usa --force (-f) para reemplazarlo."
	ImplantNoDecision = "Faltan decisiones de conflicto: {paths}"
	ImplantInvalidMix = "La composición de {path} no es válida: {message}"
)

// UpdateErrorAdvice maps an update failure kind to a visible action.
var UpdateErrorAdvice = map[errors.UpdateErrorKind]string{
	errors.UpdateNotSelfContained: "Esta instalación no es autocontenida. Instala el binario autocontenido con `scripts/install.sh` o desde una release; no se descargó nada.",
	errors.UpdatePlatform:         "kamino update solo admite macOS en Apple Silicon (arm64). No se descargó nada.",
	errors.UpdateNetwork:          "No se pudo contactar con la release. Revisa la red y vuelve a intentarlo; el binario actual sigue intacto.",
	errors.UpdateIntegrity:        "La descarga no coincide con el tamaño o el SHA-256 publicado. El binario actual sigue intacto; vuelve a intentarlo.",
	errors.UpdateReplace:          "No se pudo reemplazar el binario. El temporal se eliminó y la versión actual sigue disponible.",
	errors.UpdateOther:            "La actualización no se pudo completar. No se modificó el binario.",
}

// SizeLabel renders a compact human-readable size.
func SizeLabel(size int64) string {
	value := float64(size)
	units := []string{"B", "KB", "MB", "GB"}
	for _, unit := range units {
		if value < 1024 || unit == "GB" {
			if unit == "B" {
				return fmt.Sprintf("%d %s", int64(value), unit)
			}
			return fmt.Sprintf("%.1f %s", value, unit)
		}
		value /= 1024
	}
	return fmt.Sprintf("%d B", size)
}

const (
	ButtonCancel              = "Cancelar"
	ButtonContinue            = "Continuar"
	ButtonBack                = "Atrás"
	ButtonClose               = "Cerrar"
	ButtonHelp                = "Ayuda"
	ButtonCopyClone           = "Copiar ruta (c)"
	ButtonCopySnapshot        = "Copiar holocrón (c)"
	ButtonAllClone            = "Todo al clon"
	ButtonAllOriginal         = "Todo al original"
	ButtonDiscard             = "Descartar y volver"
	ButtonRetry               = "Reintentar"
	ButtonTerminateSimulation = "Terminar simulación"
	ButtonCurrentDir          = "Directorio actual"
	ButtonDownloads           = "Descargas"
	ButtonDesktop             = "Escritorio"
	ButtonStartClone          = "Clonar"
	ButtonApplyAnyway         = "Aplicar de todas formas"
	ButtonBackToDetail        = "Volver al detalle"

	ColumnApp      = "App"
	ColumnFiles    = "Archivos"
	ColumnFile     = "Archivo"
	ColumnOrigin   = "Origen"
	ColumnSize     = "Tamaño"
	ColumnType     = "Tipo"
	ColumnDetail   = "Detalle"
	ColumnDecision = "Decisión"

	LabelFilter               = "Filtro"
	LabelDefaultName          = "Nombre por defecto: {name}"
	LabelPathPlaceholder      = "Ruta absoluta del clon"
	LabelCloneFilePlaceholder = "Ruta del clon (.tar.gz)"
)

const (
	SettingsTitle                = "Configuración"
	SettingsSubtitle             = "Rutas de Kamino · ~/.config/kamino/config.toml"
	SettingsCLINote              = "La CLI lee catalog.dir y implant.snapshots_limit; las rutas de salida son de la TUI."
	SettingsCatalogLabel         = "Catálogo de aplicaciones"
	SettingsCloneLabel           = "Destino de los clones"
	SettingsSnapshotsLabel       = "Directorio de holocrones"
	SettingsWarnCatalogMissing   = "no existe; se usarán solo las definiciones integradas"
	SettingsWarnCloneMissing     = "no existe; se creará al clonar"
	SettingsWarnSnapshotsMissing = "no existe; se creará al implantar"
	SettingsWarnNoFile           = "no existe config.toml; se creará al guardar"
	SettingsSaved                = "configuración guardada"
	SettingsSaveError            = "no se pudo guardar. El archivo anterior no cambió"

	StatusRenameStarted      = "El renombrado ya había empezado: no se puede cancelar."
	StatusCancelling         = "Cancelando…"
	StatusPreparingCancelled = "Preparación cancelada; no se escribió nada."
	StatusConflictWholeFile  = "Este conflicto se resuelve como archivo entero: aplica el clon o conserva el original."
	StatusNoDetail           = "sin detalle técnico adicional"
	StatusReviewDefault      = "Revisa el estado y vuelve a intentarlo."

	TitleCancelClone      = "Clonación cancelada"
	TitleSimulation       = "Simulación de clonación"
	TitleCloneCreated     = "Clon creado en {path}"
	TitleImplantCancelled = "Implantación cancelada"
	TitleImplantAborted   = "Implantación abortada"
	TitleBackToApps       = "Volver a las apps"
	TitleBackToAppsBody   = "Volver descarta las decisiones de conflicto tomadas hasta ahora."
	TitlePreparing        = "Preparando la implantación"
	TitleRetryAdvice      = "Vuelve a D0 y elige un clon válido."
	TitleCancelPrepare    = "Cancelando…"
)

var (
	HelpSettings = [][2]string{{"Tab", "siguiente campo"}, {"Mayús+Tab", "campo anterior"}, {"Enter", "siguiente campo"}, {"Ctrl+S", "guardar la configuración"}, {"F1", "abrir esta ayuda"}, {"Esc", "volver o salir sin guardar"}}
	HelpC1       = [][2]string{{"↑↓", "recorrer apps"}, {"Espacio", "marcar o desmarcar sin perder el foco"}, {"a", "seleccionar todas las visibles"}, {"n", "quitar todas las visibles"}, {"/", "filtrar"}, {"Enter", "continuar con la selección"}, {"Esc", "volver o cancelar"}}
	HelpC2       = [][2]string{{"Enter", "continuar con el destino"}, {"F2", "usar el directorio actual"}, {"F3", "usar Descargas"}, {"F4", "usar Escritorio"}, {"F1", "abrir esta ayuda"}, {"Esc", "volver a C1 sin perder la selección"}}
	HelpC3       = [][2]string{{"d", "rutas omitidas y su motivo"}, {"f", "archivos incluidos por app"}, {"Enter", "clonar cuando el plan esté listo"}, {"Ctrl+C", "cancelar antes del renombrado"}, {"Esc", "volver"}}
	HelpC4       = [][2]string{{"d", "rutas omitidas"}, {"c", "copiar la ruta con OSC 52"}, {"Enter/Esc", "continuar"}}
	HelpD0       = [][2]string{{"↑↓", "recorrer los clones de la lista"}, {"/", "filtrar por nombre u origen"}, {"o", "ir a Otro archivo... y escribir una ruta"}, {"Enter", "continuar con la fila resaltada o abrir la ruta manual"}, {"Esc", "volver a la tabla, cerrar el filtro o cancelar"}, {"F1", "abrir esta ayuda"}}
	HelpD1       = [][2]string{{"↑↓", "recorrer apps"}, {"Espacio", "marcar o desmarcar sin perder el foco"}, {"a", "seleccionar todas"}, {"n", "quitar todas"}, {"Enter", "continuar"}, {"Esc", "volver sin implantar"}}
	HelpPrepare  = [][2]string{{"Esc", "detener antes de escribir"}, {"Ctrl+C", "detener antes de escribir"}}
	HelpD2       = [][2]string{{"↑↓", "recorrer conflictos"}, {"r", "revisar el conflicto seleccionado"}, {"c", "aplicar el clon completo"}, {"o", "conservar el original"}, {"C", "aplicar el clon a todos"}, {"O", "conservar el original en todos"}, {"Enter", "confirmar cuando todo esté decidido"}, {"Esc", "retirada: descarta decisiones tras confirmar"}}
	HelpD3       = [][2]string{{"↑↓", "recorrer líneas decidibles"}, {"Espacio/Enter", "alternar original/clon"}, {"c/o", "bloque actual al clon u original"}, {"C/O", "todo el archivo al clon u original"}, {"Tab/Mayús+Tab", "cambiar de bloque"}, {"Esc", "volver y validar"}}
	HelpD4       = [][2]string{{"↑↓", "elegir una acción"}, {"Enter", "aceptar la acción seleccionada"}, {"Retirada", "volver sin escribir ni crear holocrón"}, {"Abortar misión", "cancelar la implantación"}, {"Implantar", "crear el holocrón y escribir"}, {"Esc", "retroceder"}}
	HelpD5       = [][2]string{{"c", "copiar la ruta del holocrón"}, {"Enter/Esc", "continuar y cerrar el flujo"}}
	HelpU1       = [][2]string{{"Enter", "continuar al terminar"}, {"Esc", "cancelar antes del reemplazo o salir al terminar"}}
	HelpMenu     = [][2]string{{"↑↓", "mover la selección, incluida Configuración"}, {"Enter", "aceptar la opción seleccionada"}}
)

const (
	ScreenD0       = "D0 · clon"
	ScreenD1       = "D1 · apps"
	ScreenD2       = "D2 · conflictos"
	ScreenD3       = "D3 · detalle por bloques"
	ScreenD4       = "D4 · confirmación"
	ScreenD5       = "D5 · resultado"
	ScreenC1       = "C1 · apps"
	ScreenC2       = "C2 · destino"
	ScreenC3       = "C3 · revisión"
	ScreenC4       = "C4 · resumen"
	ScreenU1       = "U1 · actualización"
	ScreenA1       = "A1 · añadir app"
	ScreenSettings = "Configuración"
	ScreenMenu     = "menú principal"
	ScreenError    = "error"
	ScreenSmall    = "terminal pequeña"
	ScreenPrepare  = "Preparando la implantación"

	AdviceBackToD0        = "Vuelve a D0 y elige un clon válido."
	AdviceRetryConflict   = "Revisa el conflicto y vuelve a intentarlo."
	StatusAllDecided      = "Todos los conflictos tienen decisión. Puedes confirmar."
	SummaryImplant        = "Implantación: {implanted} implantados, {merged} híbridos"
	SummaryImplantGeneric = "Implantación"

	TitleImplantDryRun = "Plan de implantación (simulación)"
	TitleDetail        = "Detalle"
	ConflictWholeFile  = "{path}: conflicto {reason}; se resuelve como archivo entero"
	ConflictBinary     = "binario"
	ConflictLarge      = "grande (>5 MB)"
	ConflictSymlink    = "{path}: enlace {original} -> {local}, clon -> {clone}"
	CloneDryRunHeader  = "Plan de clonación (simulación)"
)

var (
	ShortcutsC1            = ShortcutLine([2]string{"↑↓", "mover"}, [2]string{"Espacio", "marcar"}, [2]string{"a/n", "todas/ninguna"}, [2]string{"/", "filtrar"}, [2]string{"Enter", "continuar"}, [2]string{"Esc", "volver o cerrar filtro"}, [2]string{"?", "ayuda"})
	ShortcutsC2            = ShortcutLine([2]string{"Enter", "continuar"}, [2]string{"Esc", "volver"}, [2]string{"F2", "directorio actual"}, [2]string{"F3", "descargas"}, [2]string{"F4", "escritorio"}, [2]string{"?", "ayuda"})
	ShortcutsC3            = ShortcutLine([2]string{"d", "omitidos"}, [2]string{"f", "archivos"}, [2]string{"Enter", "clonar"}, [2]string{"Esc", "volver"}, [2]string{"Ctrl+C", "cancelar"}, [2]string{"?", "ayuda"})
	ShortcutsC4            = ShortcutLine([2]string{"d", "omitidos"}, [2]string{"c", "copiar ruta"}, [2]string{"Enter/Esc", "continuar"}, [2]string{"?", "ayuda"})
	ShortcutsD0            = ShortcutLine([2]string{"↑↓", "mover"}, [2]string{"/", "filtrar"}, [2]string{"o", "otro archivo"}, [2]string{"Enter", "continuar"}, [2]string{"Esc", "volver"}, [2]string{"?", "ayuda"})
	ShortcutsD1            = ShortcutLine([2]string{"↑↓", "mover"}, [2]string{"Espacio", "marcar"}, [2]string{"a/n", "todas/ninguna"}, [2]string{"Enter", "continuar"}, [2]string{"Esc", "volver"}, [2]string{"?", "ayuda"})
	ShortcutsD2            = ShortcutLine([2]string{"↑↓", "mover"}, [2]string{"r", "revisar"}, [2]string{"c/o", "clon/original"}, [2]string{"C/O", "todo"}, [2]string{"Enter", "confirmar"}, [2]string{"Esc", "volver"}, [2]string{"?", "ayuda"})
	ShortcutsD3            = ShortcutLine([2]string{"↑↓", "línea"}, [2]string{"Espacio/Enter", "alternar"}, [2]string{"c/o", "bloque"}, [2]string{"C/O", "archivo"}, [2]string{"Tab/Mayús+Tab", "bloque"}, [2]string{"Esc", "volver"}, [2]string{"?", "ayuda"})
	ShortcutsD4            = ShortcutLine([2]string{"↑↓", "mover"}, [2]string{"Enter", "aceptar"}, [2]string{"Esc", "retroceder"}, [2]string{"?", "ayuda"})
	ShortcutsD5            = ShortcutLine([2]string{"c", "copiar holocrón"}, [2]string{"Enter/Esc", "continuar"}, [2]string{"?", "ayuda"})
	ShortcutsU1            = ShortcutLine([2]string{"Enter", "continuar"}, [2]string{"Esc", "cancelar/salir"}, [2]string{"?", "ayuda"})
	ShortcutsHelp          = ShortcutLine([2]string{"Enter/Esc/?", "cerrar"})
	ShortcutsSettings      = ShortcutLine([2]string{"Tab/Mayús+Tab", "mover"}, [2]string{"Ctrl+S", "guardar"}, [2]string{"Esc", "volver"}, [2]string{"?", "ayuda"})
	ShortcutsSettingsError = ShortcutLine([2]string{"Tab/Mayús+Tab", "mover"}, [2]string{"Ctrl+S", "guardar"}, [2]string{"Esc", "salir sin guardar"}, [2]string{"?", "ayuda"})
	ShortcutsDialog        = ShortcutLine([2]string{"↑↓", "mover"}, [2]string{"Enter", "aceptar"}, [2]string{"Esc", "volver"})
	ShortcutsPrepare       = ShortcutLine([2]string{"Esc", "cancelar"}, [2]string{"Ctrl+C", "cancelar"}, [2]string{"?", "ayuda"})
	ShortcutsSmall         = ShortcutLine([2]string{"?", "ayuda"})
)
