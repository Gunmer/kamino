"""Textos de presentación de Kamino en un único lugar.

Solo contiene constantes y plantillas visibles. El núcleo sigue lanzando
mensajes de dominio; esta capa traduce el vocabulario de la interfaz sin
renombrar identificadores internos ni el formato del archivo.
"""

from __future__ import annotations

from kamino.errors import UpdateErrorKind

APP_NAME = "kamino"

WORDMARK = (
    "██╗  ██╗ █████╗ ███╗   ███╗██╗███╗   ██╗ ██████╗ ",
    "██║ ██╔╝██╔══██╗████╗ ████║██║████╗  ██║██╔═══██╗",
    "█████╔╝ ███████║██╔████╔██║██║██╔██╗ ██║██║   ██║",
    "██╔═██╗ ██╔══██║██║╚██╔╝██║██║██║╚██╗██║██║   ██║",
    "██║  ██╗██║  ██║██║ ╚═╝ ██║██║██║ ╚████║╚██████╔╝",
    "╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝╚═╝╚═╝  ╚═══╝ ╚═════╝ ",
)

SUBTITLE = "Clona y despliega configuraciones de macOS en un solo archivo."

PHRASES = (
    "La Fuerza estará contigo.",
    "Este es el camino.",
    "Siempre hay un clon más.",
    "Que la configuración te acompañe.",
    "Nunca subestimes un clon bien hecho.",
)

CAPSULE_TOTAL = 8
CAPSULE_EMPTY = "○"
CAPSULE_ORIGINAL = "◉"
CAPSULE_ASCII_EMPTY = "o"
CAPSULE_ASCII_ORIGINAL = "o"
MARKER_CHOSEN = "●"
MARKER_NOT_CHOSEN = "○"
MARKER_CHOSEN_ASCII = "*"
MARKER_NOT_CHOSEN_ASCII = "."
FLOW_MARKER = "◉"
FLOW_MARKER_ASCII = "o"
HELP_STEP_MARKER = "○"
HELP_STEP_MARKER_ASCII = "."

#: Glosario visible. Los nombres internos no cambian.
ORIGINAL = "original"
SNAPSHOT = "holocrón de seguridad"
HYBRID = "híbrido"
HYBRIDS = "híbridos"
PERFECT_CLONE = "clon perfecto"
PERFECT_CLONES = "clones perfectos"
NOT_INSTALLED = "sin base en este Mac"
FORCE_DISTURBANCE = "perturbación en la Fuerza"
INTEGRITY_FAILED = "clon con mutaciones"
MANIFEST = "registro de los kaminoanos"

PRIVACY_WARNING = (
    "aviso: el clon no está cifrado y las configuraciones incluidas pueden "
    "contener secretos. Los archivos de credenciales conocidos se excluyen, "
    "pero los ajustes permitidos se conservan íntegros."
)
DIFF_WARNING = (
    "Los diffs pueden contener datos sensibles (tokens, claves o rutas). "
    "No se guardan ni se registran fuera de esta pantalla."
)

MENU_CLONE = "Clonar"
MENU_DEPLOY = "Desplegar"
MENU_UPDATE = "Actualizar"
MENU_EXIT = "Abandonar Kamino"

MENU_CLONE_HELP = "Crea un clon de las apps instaladas."
MENU_DEPLOY_HELP = "Despliega desde un archivo del clon."
MENU_UPDATE_HELP = "Actualiza a la última versión publicada."
MENU_EXIT_HELP = "Cierra kamino."

LAST_MISSION = "Última misión: {summary}"
NO_LAST_MISSION = ""

CLONE_STEPS = ("apps", "destino", "revisión")
DEPLOY_STEPS = ("clon", "apps", "conflictos", "confirmación")
CLONE_STEP_LABEL = "Paso {current} de {total} · {title}"
STEP_TITLES = {
    "apps": "apps",
    "destino": "destino",
    "revisión": "revisión",
    "clon": "clon",
    "conflictos": "conflictos",
    "confirmación": "confirmación",
    "actualización": "actualización",
}
STEP_DESCRIPTIONS = {
    "apps": "apps",
    "destino": "destino",
    "revisión": "revisión",
    "clon": "clon",
    "conflictos": "conflictos",
    "confirmación": "confirmación",
}

CLONE_TITLE = "Clonar apps"
CLONE_APPS_TITLE = "Elige las apps que quieres clonar"
CLONE_DESTINATION_TITLE = "Elige el destino del clon"
CLONE_REVIEW_TITLE = "Revisa el plan del clon"
CLONE_SUMMARY_TITLE = "Clonación completada"
CLONE_DRY_RUN_TITLE = "Simulación del clon"
CLONE_PLANNING = "Planificando el clon…"
CLONE_WRITING = "Escribiendo el clon…"
CLONE_CANCELLING = "Cancelando; el clon anterior queda intacto…"
CLONE_NO_SELECTION = "Selecciona al menos una app para continuar."
CLONE_NOTHING_TO_DO = "No hay archivos para incluir en el clon."
CLONE_ALL = "Todas"
CLONE_NONE = "Ninguna"
CLONE_CONTINUE = "Continuar"
CLONE_BACK = "Atrás"
CLONE_FILTER = "Filtro"
CLONE_DETECTED = "Detectadas"
CLONE_NOT_DETECTED = "Sin detectar en este Mac: {apps}"
CLONE_FILES = "{count} archivos"
CLONE_ZERO_FILES = "sin archivos"
CLONE_APPS_SUMMARY = "Apps: {count}"
CLONE_FILE_COUNT = "Archivos: {count}"
CLONE_SIZE = "Tamaño: {size}"
CLONE_DESTINATION = "Destino: {path}"
CLONE_SECRETS = PRIVACY_WARNING
CLONE_SKIPPED_TITLE = "Rutas omitidas"
CLONE_INCLUDED_TITLE = "Archivos incluidos"
CLONE_DRY_RUN_NOTE = "Simulación: no se escribirá ningún archivo."
CLONE_REPLACE_TITLE = "El clon ya existe"
CLONE_REPLACE_BODY = (
    "Ya existe un clon en {path}.\n"
    "Creado: {created}\n"
    "Tamaño: {size}\n\n"
    "¿Quieres volver, elegir otro nombre o reemplazarlo al terminar?"
)
CLONE_REPLACE_BACK = "Volver"
CLONE_REPLACE_OTHER = "Otro nombre"
CLONE_REPLACE_FORCE = "Reemplazar clon"
CLONE_FORCE_NOTICE = "El clon existente se reemplazará al final."
CLONE_EXIT_LINE = "Salida 3: se omitieron rutas o apps pedidas."
CLONE_COPY_PATH = "Ruta copiada."
CLONE_COPY_UNSUPPORTED = "Este terminal no admite copia OSC 52; copia la ruta a mano."
CLONE_CANCELLED = "Clonación cancelada; no se escribió nada."
CLONE_COMPLETE = "Clon creado en {path}."
CLONE_APPS_LABEL = "Apps"
CLONE_FILES_LABEL = "Archivos"
CLONE_SIZE_LABEL = "Tamaño"
CLONE_PATH_LABEL = "Ruta"

DEPLOY_TITLE = "Desplegar un clon"
DEPLOY_FILE_TITLE = "Elige el archivo del clon"
DEPLOY_APPS_TITLE = "Elige las apps que quieres desplegar"
DEPLOY_CONFLICTS_TITLE = "Conflictos"
DEPLOY_DETAIL_TITLE = "Revisión por bloques"
DEPLOY_CONFIRM_TITLE = "Confirma el despliegue"
DEPLOY_SUMMARY_TITLE = "Despliegue completado"
DEPLOY_INCOMPLETE_TITLE = "Despliegue incompleto"
DEPLOY_NO_FILE = "Escribe o elige un archivo .tar.gz."
DEPLOY_FILE_INVALID = "El archivo no es un clon válido: {message}"
DEPLOY_MANIFEST_WARNINGS = "Avisos del manifiesto"
DEPLOY_ARCH_WARNING = "Arquitectura distinta: {message}"
DEPLOY_MACOS_WARNING = "Versión de macOS distinta: {message}"
DEPLOY_PREPARING = "Preparando el despliegue…"
DEPLOY_VERIFYING = "Verificando la integridad del clon…"
DEPLOY_EXTRACTING = "Extrayendo en un directorio temporal…"
DEPLOY_CLASSIFYING = "Clasificando los archivos…"
DEPLOY_APPLYING = "Aplicando el despliegue…"
DEPLOY_AFTER_SNAPSHOT = "El holocrón ya está creado: no se puede cancelar."
DEPLOY_NONE_SELECTED = "Selecciona al menos una app para continuar."
DEPLOY_NOT_INSTALLED_TITLE = "Sin base en este Mac"
DEPLOY_NOT_INSTALLED_NOTE = "Estas apps se informan, no se despliegan y no cuentan como omitidas."
DEPLOY_NEW = "Nuevos: {count}"
DEPLOY_PERFECT = "Clones perfectos: {count}"
DEPLOY_CHANGED = "Conflictos: {count}"
DEPLOY_UNSUPPORTED = "No soportados: {count}"
DEPLOY_NOTICES = "Avisos: {count}"
DEPLOY_ACTIONS = "r revisar · c aplicar clon · o conservar original · C todo clon · O todo original"
DEPLOY_CONFIRM = "Confirmar"
DEPLOY_ALL_CLONE = "Todo al clon"
DEPLOY_ALL_ORIGINAL = "Todo al original"
DEPLOY_REVIEW = "Revisar"
DEPLOY_APPLY = "Aplicar clon"
DEPLOY_OMIT = "Conservar original"
DEPLOY_DEPLOYED = "Desplegados: {count}"
DEPLOY_DEPLOYED_DETAIL = "{new} nuevos y {replaced} reemplazos"
DEPLOY_HYBRIDS = "Híbridos: {count}"
DEPLOY_OMITTED = "Omitidos: {count}"
DEPLOY_NOT_DEPLOYED = "Sin desplegar: {count}"
DEPLOY_UNCHANGED = "Sin cambios: {count}"
DEPLOY_PROTECTED = "Archivos protegidos: {count}"
DEPLOY_SNAPSHOT_DIR = "Holocrón de seguridad en: {path}"
DEPLOY_SNAPSHOT_PATH = "Holocrón: {path}"
DEPLOY_NO_SNAPSHOT = "No había originales que proteger; no se creó holocrón."
DEPLOY_OPEN_APPS = "Si desplegarás sobre apps abiertas, ciérralas antes de continuar."
DEPLOY_BACK = "Retirada"
DEPLOY_ABORT = "Abortar misión"
DEPLOY_APPLY_BUTTON = "Desplegar"
DEPLOY_UNDO = "Para deshacer: kamino deploy {path}"
DEPLOY_EXIT_LINE = (
    "Salida {code}: se conservaron originales, no soportados o apps pedidas no cumplidas."
)
DEPLOY_PARTIAL = "Despliegue incompleto: {applied} aplicados; falló {failed}."
DEPLOY_PARTIAL_SNAPSHOT = "Holocrón disponible para deshacer: {path}"
DEPLOY_COPY_PATH = "Ruta del holocrón copiada."
DEPLOY_BLOCK = "Bloque {index} de {total}"
DEPLOY_BLOCK_COUNTS = "Clon: {clone} líneas · Original: {original} líneas"
DEPLOY_ROW_CLONE = "● clon"
DEPLOY_ROW_ORIGINAL = "○ original"
DEPLOY_ROW_CLONE_ONLY = "solo clon"
DEPLOY_ROW_ORIGINAL_ONLY = "solo original"
DEPLOY_DECISION_HINT = "Espacio/Enter alterna · c/o bloque · C/O archivo · Tab/Mayús+Tab bloque"
DEPLOY_VALIDATION_INVALID = "Híbrido inválido: {message}"

UPDATE_TITLE = "Actualizar kamino"
UPDATE_CHECKING = "Consultando la última versión publicada…"
UPDATE_DOWNLOADING = "Descargando y verificando tamaño y SHA-256…"
UPDATE_REPLACING = "Reemplazando el binario…"
UPDATE_UPDATED = "kamino {current} → {latest}. Actualizado: {path}"
UPDATE_UP_TO_DATE = "kamino {current} ya es la última versión disponible."
UPDATE_NEWER = (
    "kamino {current} es más reciente que la última publicada ({latest}); no se modifica."
)
UPDATE_RESTART = "Reinicia kamino para usar la versión nueva."
UPDATE_CANCELLED = "Actualización cancelada; el binario actual sigue intacto."
UPDATE_CANNOT_CANCEL = "El reemplazo ya empezó: no se puede cancelar."
UPDATE_RESULT_UPDATED = "✔ Actualizado"
UPDATE_RESULT_UP_TO_DATE = "✔ Ya al día"
UPDATE_RESULT_NEWER = "▲ Instalada más reciente"

ERROR_WHAT = "Qué pasó"
ERROR_SAFETY = "Estado de seguridad"
ERROR_ACTION = "Qué hacer"
ERROR_DETAIL = "Detalle técnico"
ERROR_NOTHING_WRITTEN = "No se escribió nada."
ERROR_CLONE_WRITTEN = "El clon anterior sigue intacto."
ERROR_SNAPSHOT_WRITTEN = "El holocrón de seguridad está disponible para deshacer."
ERROR_BACK_TO_MENU = "Volver al menú"
ERROR_RETRY = "Reintentar"
ERROR_EXIT = "Salir"
ERROR_SHOW_DETAIL = "t detalle · c copiar"
ERROR_COPIED = "Detalle copiado."
ERROR_COPY_UNSUPPORTED = "Este terminal no admite copia OSC 52; copia el detalle a mano."
ERROR_SMALL_TITLE = "La terminal es demasiado pequeña"
ERROR_SMALL_BODY = (
    "Amplía la terminal a 80×24 o más para continuar.\n"
    "El estado del flujo se conserva mientras redimensionas."
)
HELP_TITLE = "Ayuda"
HELP_GLOBAL = "Global: ? ayuda · F1 ayuda en campos · Esc volver · Ctrl+C cancelar si se permite"
HELP_FLOWS = "Flujo"
HELP_SECRETS = DIFF_WARNING
HELP_CLOSE = "Cerrar"


def shortcut_line(*entries: tuple[str, str]) -> str:
    """Compone una leyenda de teclas separada por puntos medios."""
    return " · ".join(f"{key} {label}" for key, label in entries)


SHORTCUTS_MENU = shortcut_line(
    ("↑↓", "mover"),
    ("Enter", "aceptar"),
    ("c/d/u/q", "atajos"),
    ("?", "ayuda"),
)
SHORTCUTS_C1 = shortcut_line(
    ("↑↓", "mover"),
    ("Espacio", "marcar"),
    ("a/n", "todas/ninguna"),
    ("/", "filtrar"),
    ("Enter", "continuar"),
    ("Esc", "volver o cerrar filtro"),
    ("?", "ayuda"),
)
SHORTCUTS_C2 = shortcut_line(
    ("Enter", "continuar"),
    ("Esc", "volver"),
    ("F2", "directorio actual"),
    ("F3", "descargas"),
    ("F4", "escritorio"),
    ("?", "ayuda"),
)
SHORTCUTS_C3 = shortcut_line(
    ("d", "omitidos"),
    ("f", "archivos"),
    ("Enter", "clonar"),
    ("Esc", "volver"),
    ("Ctrl+C", "cancelar"),
    ("?", "ayuda"),
)
SHORTCUTS_C3_DRY_RUN = shortcut_line(
    ("d", "omitidos"),
    ("f", "archivos"),
    ("Enter", "terminar simulación"),
    ("Esc", "volver"),
    ("?", "ayuda"),
)
SHORTCUTS_C4 = shortcut_line(
    ("d", "omitidos"),
    ("c", "copiar ruta"),
    ("Enter/Esc", "continuar"),
    ("?", "ayuda"),
)
SHORTCUTS_D0 = shortcut_line(
    ("↑↓", "archivo"),
    ("Tab", "foco"),
    ("Enter", "continuar"),
    ("Esc", "cancelar"),
    ("?", "ayuda"),
)
SHORTCUTS_D1 = shortcut_line(
    ("↑↓", "mover"),
    ("Espacio", "marcar"),
    ("a/n", "todas/ninguna"),
    ("Enter", "continuar"),
    ("Esc", "volver"),
    ("?", "ayuda"),
)
SHORTCUTS_D2 = shortcut_line(
    ("↑↓", "mover"),
    ("r", "revisar"),
    ("c/o", "clon/original"),
    ("C/O", "todo"),
    ("Enter", "confirmar"),
    ("Esc", "volver"),
    ("?", "ayuda"),
)
SHORTCUTS_D3 = shortcut_line(
    ("↑↓", "línea"),
    ("Espacio/Enter", "alternar"),
    ("c/o", "bloque"),
    ("C/O", "archivo"),
    ("Tab/Mayús+Tab", "bloque"),
    ("Esc", "volver"),
    ("?", "ayuda"),
)
SHORTCUTS_D4 = shortcut_line(
    ("↑↓", "mover"),
    ("Enter", "aceptar"),
    ("Esc", "retroceder"),
    ("?", "ayuda"),
)
SHORTCUTS_D5 = shortcut_line(
    ("c", "copiar holocrón"),
    ("Enter/Esc", "continuar"),
    ("?", "ayuda"),
)
SHORTCUTS_U1 = shortcut_line(
    ("Enter", "continuar"),
    ("Esc", "cancelar/salir"),
    ("?", "ayuda"),
)
SHORTCUTS_HELP = shortcut_line(("Enter/Esc/?", "cerrar"))
SHORTCUTS_DIALOG = shortcut_line(
    ("↑↓", "mover"),
    ("Enter", "aceptar"),
    ("Esc", "volver"),
)
SHORTCUTS_PREPARE = shortcut_line(
    ("Esc", "cancelar"),
    ("Ctrl+C", "cancelar"),
    ("?", "ayuda"),
)
SHORTCUTS_SMALL = shortcut_line(("?", "ayuda"))

STATUS_OK = "✔ listo"
STATUS_WARN = "▲ aviso"
STATUS_ERROR = "✖ error"
ICON_SUCCESS = "✔"
ICON_WARNING = "▲"
ICON_ERROR = "✖"
ICON_ORIGINAL = "◉"

CANCELLED = "cancelado"
REQUIRES_YES = "se requiere --yes para desplegar sin terminal interactiva"
CLONE_FORCE_HELP = (
    "Reemplaza un clon existente de forma atómica. Sin esta opción, un clon existente es un error."
)
CLONE_EXISTS_HINT = "El clon ya existe: usa --force (-f) para reemplazarlo."
DEPLOY_NO_DECISION = "Faltan decisiones de conflicto: {paths}"
DEPLOY_INVALID_MIX = "La composición de {path} no es válida: {message}"

UPDATE_ERROR_ADVICE = {
    UpdateErrorKind.NOT_SELF_CONTAINED: (
        "Esta instalación no es autocontenida. Actualiza con `uv tool upgrade "
        "kamino-cli`, con pip o desde el código; no se descargó nada."
    ),
    UpdateErrorKind.PLATFORM: (
        "kamino update solo admite macOS en Apple Silicon (arm64). No se descargó nada."
    ),
    UpdateErrorKind.NETWORK: (
        "No se pudo contactar con la release. Revisa la red y vuelve a intentarlo; "
        "el binario actual sigue intacto."
    ),
    UpdateErrorKind.INTEGRITY: (
        "La descarga no coincide con el tamaño o el SHA-256 publicado. "
        "El binario actual sigue intacto; vuelve a intentarlo."
    ),
    UpdateErrorKind.REPLACE: (
        "No se pudo reemplazar el binario. El temporal se eliminó y la versión "
        "actual sigue disponible."
    ),
    UpdateErrorKind.OTHER: "La actualización no se pudo completar. No se modificó el binario.",
}


def size_label(size: int) -> str:
    """Tamaño compacto y legible (B, KB, MB, GB)."""
    value = float(size)
    for unit in ("B", "KB", "MB", "GB"):
        if value < 1024 or unit == "GB":
            if unit == "B":
                return f"{int(value)} {unit}"
            return f"{value:.1f} {unit}"
        value /= 1024
    return f"{size} B"


def count_lines(paths: list[str]) -> int:
    """Número de entradas para las plantillas de conteo."""
    return len(paths)


#: Etiquetas de acción reutilizadas por las pantallas.
BUTTON_CANCEL = "Cancelar"
BUTTON_CONTINUE = "Continuar"
BUTTON_BACK = "Atrás"
BUTTON_CLOSE = "Cerrar"
BUTTON_HELP = "Ayuda"
BUTTON_COPY_CLONE = "Copiar ruta (c)"
BUTTON_COPY_SNAPSHOT = "Copiar holocrón (c)"
BUTTON_ALL_CLONE = "Todo al clon"
BUTTON_ALL_ORIGINAL = "Todo al original"
BUTTON_DISCARD = "Descartar y volver"
BUTTON_RETRY = "Reintentar"
BUTTON_TERMINATE_SIMULATION = "Terminar simulación"
BUTTON_CURRENT_DIR = "Directorio actual"
BUTTON_DOWNLOADS = "Descargas"
BUTTON_DESKTOP = "Escritorio"
BUTTON_START_CLONE = "Clonar"
BUTTON_APPLY_ANYWAY = "Aplicar de todas formas"
BUTTON_BACK_TO_DETAIL = "Volver al detalle"

COLUMN_APP = "App"
COLUMN_ARCHIVOS = "Archivos"
COLUMN_ARCHIVO = "Archivo"
COLUMN_TAMANO = "Tamaño"
COLUMN_TIPO = "Tipo"
COLUMN_DETALLE = "Detalle"
COLUMN_DECISION = "Decisión"

LABEL_FILTER = "Filtro"
LABEL_DEFAULT_NAME = "Nombre por defecto: {name}"
LABEL_PATH_PLACEHOLDER = "Ruta absoluta del clon"
LABEL_CLONE_FILE_PLACEHOLDER = "Ruta del clon (.tar.gz)"
LABEL_SCREEN = "Pantalla: {screen}"
LABEL_SHORTCUTS = "Atajos de esta pantalla:"
LABEL_FLOW = "Flujo:"

STATUS_RENAME_STARTED = "El renombrado ya había empezado: no se puede cancelar."
STATUS_CANCELLING = "Cancelando…"
STATUS_PREPARING_CANCELLED = "Preparación cancelada; no se escribió nada."
STATUS_CONFLICT_WHOLE_FILE = (
    "Este conflicto se resuelve como archivo entero: aplica el clon o conserva el original."
)
STATUS_NO_DETAIL = "sin detalle técnico adicional"
STATUS_REVIEW_DEFAULT = "Revisa el estado y vuelve a intentarlo."

TITLE_CANCEL_CLONE = "Clonación cancelada"
TITLE_SIMULATION = "Simulación de clonación"
TITLE_CLONE_CREATED = "Clon creado en {path}"
TITLE_DEPLOY_CANCELLED = "Despliegue cancelado"
TITLE_DEPLOY_ABORTED = "Despliegue abortado"
TITLE_BACK_TO_APPS = "Volver a las apps"
TITLE_BACK_TO_APPS_BODY = "Volver descarta las decisiones de conflicto tomadas hasta ahora."
TITLE_PREPARING = "Preparando el despliegue"
TITLE_RETRY_ADVICE = "Vuelve a D0 y elige un clon válido."
TITLE_CANCEL_PREPARE = "Cancelando…"

HELP_C1 = (
    ("↑↓", "recorrer apps"),
    ("Espacio", "marcar o desmarcar sin perder el foco"),
    ("a", "seleccionar todas las visibles"),
    ("n", "quitar todas las visibles"),
    ("/", "filtrar"),
    ("Enter", "continuar con la selección"),
    ("Esc", "volver o cancelar"),
)
HELP_C2 = (
    ("Enter", "continuar con el destino"),
    ("F2", "usar el directorio actual"),
    ("F3", "usar Descargas"),
    ("F4", "usar Escritorio"),
    ("F1", "abrir esta ayuda"),
    ("Esc", "volver a C1 sin perder la selección"),
)
HELP_C3 = (
    ("d", "rutas omitidas y su motivo"),
    ("f", "archivos incluidos por app"),
    ("Enter", "clonar cuando el plan esté listo"),
    ("Ctrl+C", "cancelar antes del renombrado"),
    ("Esc", "volver"),
)
HELP_C4 = (
    ("d", "rutas omitidas"),
    ("c", "copiar la ruta con OSC 52"),
    ("Enter/Esc", "continuar"),
)
HELP_D0 = (
    ("↑↓", "recorrer los clones del directorio"),
    ("Tab", "alternar entre la tabla y la ruta"),
    ("Enter", "continuar con el archivo elegido"),
    ("F1", "abrir esta ayuda"),
    ("Esc", "cancelar el flujo"),
)
HELP_D1 = (
    ("↑↓", "recorrer apps"),
    ("Espacio", "marcar o desmarcar sin perder el foco"),
    ("a", "seleccionar todas"),
    ("n", "quitar todas"),
    ("Enter", "continuar"),
    ("Esc", "volver sin desplegar"),
)
HELP_PREPARE = (
    ("Esc", "detener antes de escribir"),
    ("Ctrl+C", "detener antes de escribir"),
)
HELP_D2 = (
    ("↑↓", "recorrer conflictos"),
    ("r", "revisar el conflicto seleccionado"),
    ("c", "aplicar el clon completo"),
    ("o", "conservar el original"),
    ("C", "aplicar el clon a todos"),
    ("O", "conservar el original en todos"),
    ("Enter", "confirmar cuando todo esté decidido"),
    ("Esc", "retirada: descarta decisiones tras confirmar"),
)
HELP_D3 = (
    ("↑↓", "recorrer líneas decidibles"),
    ("Espacio/Enter", "alternar original/clon"),
    ("c/o", "bloque actual al clon u original"),
    ("C/O", "todo el archivo al clon u original"),
    ("Tab/Mayús+Tab", "cambiar de bloque"),
    ("Esc", "volver y validar"),
)
HELP_D4 = (
    ("↑↓", "elegir una acción"),
    ("Enter", "aceptar la acción seleccionada"),
    ("Retirada", "volver sin escribir ni crear holocrón"),
    ("Abortar misión", "cancelar el despliegue"),
    ("Desplegar", "crear el holocrón y escribir"),
    ("Esc", "retroceder"),
)
HELP_D5 = (
    ("c", "copiar la ruta del holocrón"),
    ("Enter/Esc", "continuar y cerrar el flujo"),
)
HELP_U1 = (
    ("Enter", "continuar al terminar"),
    ("Esc", "cancelar antes del reemplazo o salir al terminar"),
)
HELP_MENU = (
    ("↑↓", "mover la selección"),
    ("Enter", "aceptar la opción seleccionada"),
    ("c", "clonar"),
    ("d", "desplegar"),
    ("u", "actualizar"),
    ("q", "salir con el último código"),
)


def help_texts(
    screen: str,
    shortcuts,
    *,
    flow_steps=(),
    current_step=None,
    secrets=False,
    unicode: bool = True,
) -> str:
    """Compone el cuerpo de ayuda a partir de la pantalla y sus atajos."""
    lines = [LABEL_SCREEN.format(screen=screen), ""]
    if shortcuts:
        lines.append(LABEL_SHORTCUTS)
        lines.extend(f"  {key}  {label}" for key, label in shortcuts)
        lines.append("")
    lines.append(HELP_GLOBAL)
    if flow_steps:
        lines.extend(("", LABEL_FLOW))
        current = FLOW_MARKER if unicode else FLOW_MARKER_ASCII
        pending = HELP_STEP_MARKER if unicode else HELP_STEP_MARKER_ASCII
        for step in flow_steps:
            marker = current if step == current_step else pending
            lines.append(f"  {marker} {STEP_TITLES.get(step, step)}")
    if secrets:
        lines.extend(("", HELP_SECRETS))
    return "\n".join(lines)


SCREEN_D0 = "D0 · archivo del clon"
SCREEN_D1 = "D1 · apps"
SCREEN_D2 = "D2 · conflictos"
SCREEN_D3 = "D3 · detalle por bloques"
SCREEN_D4 = "D4 · confirmación"
SCREEN_D5 = "D5 · resultado"
SCREEN_C1 = "C1 · apps"
SCREEN_C2 = "C2 · destino"
SCREEN_C3 = "C3 · revisión"
SCREEN_C4 = "C4 · resumen"
SCREEN_U1 = "U1 · actualización"
SCREEN_MENU = "menú principal"
SCREEN_ERROR = "error"
SCREEN_SMALL = "terminal pequeña"
SCREEN_PREPARE = "Preparando el despliegue"

ADVICE_BACK_TO_D0 = "Vuelve a D0 y elige un clon válido."
ADVICE_RETRY_CONFLICT = "Revisa el conflicto y vuelve a intentarlo."
STATUS_ALL_DECIDED = "Todos los conflictos tienen decisión. Puedes confirmar."
SUMMARY_DEPLOY = "Despliegue: {deployed} desplegados, {merged} híbridos"
SUMMARY_DEPLOY_GENERIC = "Despliegue"


TITLE_DEPLOY_DRY_RUN = "Plan de despliegue (simulación)"
TITLE_DETAIL = "Detalle"
CONFLICT_WHOLE_FILE = "{path}: conflicto {reason}; se resuelve como archivo entero"
CONFLICT_BINARY = "binario"
CONFLICT_LARGE = "grande (>5 MB)"
CONFLICT_SYMLINK = "{path}: enlace {original} -> {local}, clon -> {clone}"
CLONE_DRY_RUN_HEADER = "Plan de clonación (simulación)"
