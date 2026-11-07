#!/bin/sh
# Instalador de kamino (https://github.com/Gunmer/kamino)
#
# Uso:
#   sh scripts/install.sh
#   curl -fsSL <url-del-script> | sh
#
# Descarga la última release (kamino-arm64) y la instala en ~/.local/bin/kamino.

set -eu

REPO="Gunmer/kamino"
ASSET="kamino-arm64"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"
INSTALL_DIR="${HOME}/.local/bin"
TARGET="${INSTALL_DIR}/kamino"
TMP_FILE=""

info() { printf '==> %s\n' "$*"; }
warn() { printf 'AVISO: %s\n' "$*" >&2; }
fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

cleanup() {
  if [ -n "${TMP_FILE}" ] && [ -f "${TMP_FILE}" ]; then
    rm -f "${TMP_FILE}"
  fi
}
trap cleanup EXIT INT TERM

# ¿Hay terminal disponible para preguntar? (también funciona con `curl | sh`)
has_tty() {
  (: </dev/tty) 2>/dev/null
}

# Pregunta sí/no leyendo de /dev/tty. Devuelve 0 si la respuesta es afirmativa.
ask_yes_no() {
  printf '%s [s/N]: ' "$1" >/dev/tty
  answer=""
  read -r answer </dev/tty || answer=""
  case "${answer}" in
    s|S|si|SI|Si|sí|SÍ|Sí|y|Y|yes|YES|Yes) return 0 ;;
    *) return 1 ;;
  esac
}

# 1. Comprobar sistema y arquitectura
info "Comprobando sistema y arquitectura..."
OS="$(uname -s)"
ARCH="$(uname -m)"
if [ "${OS}" != "Darwin" ]; then
  fail "Sistema no soportado: ${OS}. kamino solo está disponible para macOS."
fi
if [ "${ARCH}" != "arm64" ]; then
  fail "Arquitectura no soportada: ${ARCH}. Solo hay binario para Apple Silicon (arm64)."
fi

command -v curl >/dev/null 2>&1 || fail "No se encuentra 'curl', que es necesario para descargar kamino."

# 2. Comprobar el directorio de instalación
info "Comprobando el directorio de instalación (${INSTALL_DIR})..."
if [ ! -d "${INSTALL_DIR}" ]; then
  if ! has_tty; then
    fail "El directorio ${INSTALL_DIR} no existe y no hay terminal para preguntar si crearlo."
  fi
  if ask_yes_no "El directorio ${INSTALL_DIR} no existe. ¿Quieres crearlo?"; then
    mkdir -p "${INSTALL_DIR}" || fail "No se pudo crear ${INSTALL_DIR}."
    info "Directorio ${INSTALL_DIR} creado."
  else
    fail "Instalación cancelada: no se creó ${INSTALL_DIR}."
  fi
fi

# 3. Comprobar si ya existe una instalación previa
if [ -e "${TARGET}" ]; then
  if ! has_tty; then
    fail "Ya existe ${TARGET} y no hay terminal para preguntar si sobrescribirlo."
  fi
  if ! ask_yes_no "Ya existe ${TARGET}. ¿Quieres sobrescribirlo?"; then
    fail "Instalación cancelada: no se sobrescribió ${TARGET}."
  fi
fi

# 4. Descargar la última versión a un archivo temporal
info "Descargando la última versión de kamino (${URL})..."
TMP_FILE="$(mktemp -t kamino)" || fail "No se pudo crear un archivo temporal."
if ! curl -fSL --progress-bar -o "${TMP_FILE}" "${URL}"; then
  fail "La descarga ha fallado. Comprueba tu conexión y que exista una release publicada con el binario ${ASSET}."
fi
if [ ! -s "${TMP_FILE}" ]; then
  fail "La descarga ha fallado: el archivo descargado está vacío."
fi

# 5. Copiar el binario al destino
info "Copiando el binario a ${TARGET}..."
cp "${TMP_FILE}" "${TARGET}" || fail "No se pudo copiar el binario a ${TARGET}."

# 6. Marcar como ejecutable si no lo está
if [ ! -x "${TARGET}" ]; then
  info "Marcando ${TARGET} como ejecutable..."
  chmod +x "${TARGET}" || fail "No se pudo marcar ${TARGET} como ejecutable."
else
  info "${TARGET} ya es ejecutable."
fi

# 7. Quitar la cuarentena de macOS
info "Quitando el atributo de cuarentena de macOS..."
if xattr -d com.apple.quarantine "${TARGET}" 2>/dev/null; then
  info "Atributo de cuarentena eliminado."
else
  warn "No se pudo ejecutar 'xattr -d com.apple.quarantine ${TARGET}' (puede que el archivo no tuviera cuarentena). Se continúa igualmente."
fi

info "kamino se ha instalado correctamente en ${TARGET}."
