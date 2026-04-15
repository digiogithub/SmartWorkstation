#!/usr/bin/env bash
# install-service.sh — instala SmartWorkstation como servicio systemd.
#
# Uso (desde la raíz del repo):
#   sudo ./scripts/install-service.sh [opciones]
#
# Opciones:
#   -u USER     Usuario bajo el que corre el servicio  (default: usuario actual o "pi")
#   -b BINARY   Ruta al binario compilado              (default: auto-detectado)
#   -c CONFIG   Ruta al fichero config.toml            (default: junto al binario)
#   -h          Muestra esta ayuda
#
# Ejemplos:
#   # Instalación mínima en Pi (detecta automáticamente):
#   sudo ./scripts/install-service.sh
#
#   # Instalación con rutas explícitas en workstation:
#   sudo ./scripts/install-service.sh -u myuser -b /usr/local/bin/smartworkstation -c /etc/smartworkstation/config.toml

set -euo pipefail

# ── Colores ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()    { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# ── Constantes ────────────────────────────────────────────────────────────────
SERVICE_NAME="smartworkstation"
UNIT_TEMPLATE="$(dirname "$0")/../systemd/${SERVICE_NAME}.service"
UNIT_DEST="/etc/systemd/system/${SERVICE_NAME}.service"

# ── Defaults ──────────────────────────────────────────────────────────────────
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Detecta el usuario que debe correr el servicio (no root)
if [[ -n "${SUDO_USER:-}" ]]; then
    DEFAULT_USER="$SUDO_USER"
elif id pi &>/dev/null; then
    DEFAULT_USER="pi"
else
    DEFAULT_USER="$(whoami)"
fi

# Detecta el binario más adecuado en bin/
detect_binary() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64)       echo "${REPO_ROOT}/bin/${SERVICE_NAME}-linux-amd64" ;;
        armv6l)       echo "${REPO_ROOT}/bin/${SERVICE_NAME}-linux-arm6"  ;;
        armv7l)       echo "${REPO_ROOT}/bin/${SERVICE_NAME}-linux-arm7"  ;;
        aarch64|arm64) echo "${REPO_ROOT}/bin/${SERVICE_NAME}-linux-arm64" ;;
        *)            echo "${REPO_ROOT}/bin/${SERVICE_NAME}" ;;
    esac
}

SVC_USER="$DEFAULT_USER"
SVC_BINARY="$(detect_binary)"
SVC_CONFIG=""   # se rellena después del parsing de args

# ── Parseo de argumentos ──────────────────────────────────────────────────────
while getopts "u:b:c:h" opt; do
    case "$opt" in
        u) SVC_USER="$OPTARG" ;;
        b) SVC_BINARY="$OPTARG" ;;
        c) SVC_CONFIG="$OPTARG" ;;
        h)
            sed -n '2,20p' "$0"
            exit 0
            ;;
        *) error "Opción desconocida. Usa -h para ver la ayuda." ;;
    esac
done

# Config por defecto: junto al binario
if [[ -z "$SVC_CONFIG" ]]; then
    SVC_CONFIG="$(dirname "$SVC_BINARY")/config.toml"
fi

SVC_WORKDIR="$(dirname "$SVC_BINARY")"

# ── Validaciones ──────────────────────────────────────────────────────────────
[[ "$EUID" -eq 0 ]] || error "Este script debe ejecutarse como root (usa sudo)."

[[ -f "$UNIT_TEMPLATE" ]] || \
    error "No se encuentra la plantilla: $UNIT_TEMPLATE\n       Ejecuta desde la raíz del repo."

if [[ ! -f "$SVC_BINARY" ]]; then
    warn "El binario no existe todavía: $SVC_BINARY"
    warn "Compila primero con 'make build-amd64' (workstation) o 'make build-arm6' (Pi Zero W)."
    read -rp "¿Continuar de todas formas? [s/N] " ans
    [[ "$ans" =~ ^[sS]$ ]] || { info "Instalación cancelada."; exit 0; }
fi

if [[ ! -f "$SVC_CONFIG" ]]; then
    warn "No se encuentra el fichero de configuración: $SVC_CONFIG"
    warn "Crea config.toml a partir de config.toml.example antes de arrancar el servicio."
fi

id "$SVC_USER" &>/dev/null || error "El usuario '$SVC_USER' no existe en el sistema."

# ── Generar el unit file ──────────────────────────────────────────────────────
info "Generando unit file para systemd..."
TMP_UNIT="$(mktemp)"
trap 'rm -f "$TMP_UNIT"' EXIT

sed \
    -e "s|@USER@|${SVC_USER}|g" \
    -e "s|@BINARY@|${SVC_BINARY}|g" \
    -e "s|@CONFIG@|${SVC_CONFIG}|g" \
    -e "s|@WORKDIR@|${SVC_WORKDIR}|g" \
    "$UNIT_TEMPLATE" > "$TMP_UNIT"

# ── Instalar ──────────────────────────────────────────────────────────────────
info "Copiando unit file → ${UNIT_DEST}"
cp "$TMP_UNIT" "$UNIT_DEST"
chmod 644 "$UNIT_DEST"

info "Recargando daemon de systemd..."
systemctl daemon-reload

info "Habilitando servicio (arrancará automáticamente al inicio)..."
systemctl enable "$SERVICE_NAME"

# Arrancamos sólo si el binario y la config existen
if [[ -f "$SVC_BINARY" && -f "$SVC_CONFIG" ]]; then
    info "Arrancando servicio..."
    systemctl start "$SERVICE_NAME"
    sleep 1
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        info "Servicio '${SERVICE_NAME}' arrancado correctamente."
    else
        warn "El servicio no está activo. Revisa los logs:"
        echo "       journalctl -u ${SERVICE_NAME} -n 30"
    fi
else
    warn "El servicio está habilitado pero NO arrancado (faltan binario o config)."
    warn "Una vez que tengas ambos, arrancar con:"
    echo "       sudo systemctl start ${SERVICE_NAME}"
fi

# ── Resumen ───────────────────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════"
echo "  Instalación completada"
echo "═══════════════════════════════════════════════════════"
echo "  Servicio  : ${SERVICE_NAME}"
echo "  Usuario   : ${SVC_USER}"
echo "  Binario   : ${SVC_BINARY}"
echo "  Config    : ${SVC_CONFIG}"
echo "  Unit file : ${UNIT_DEST}"
echo ""
echo "  Comandos útiles:"
echo "    sudo systemctl start   ${SERVICE_NAME}"
echo "    sudo systemctl stop    ${SERVICE_NAME}"
echo "    sudo systemctl restart ${SERVICE_NAME}"
echo "    sudo systemctl status  ${SERVICE_NAME}"
echo "    journalctl -u ${SERVICE_NAME} -f"
echo "═══════════════════════════════════════════════════════"
