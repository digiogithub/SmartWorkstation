#!/usr/bin/env bash
# uninstall-service.sh — elimina el servicio systemd de SmartWorkstation.
#
# Uso:
#   sudo ./scripts/uninstall-service.sh
#
# No borra el binario ni config.toml; solo gestiona la integración con systemd.

set -euo pipefail

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

SERVICE_NAME="smartworkstation"
UNIT_DEST="/etc/systemd/system/${SERVICE_NAME}.service"

[[ "$EUID" -eq 0 ]] || error "Ejecuta con sudo."

if ! systemctl list-unit-files "${SERVICE_NAME}.service" &>/dev/null; then
    warn "El servicio '${SERVICE_NAME}' no está instalado. Nada que hacer."
    exit 0
fi

# Detener primero
if systemctl is-active --quiet "$SERVICE_NAME"; then
    info "Deteniendo servicio..."
    systemctl stop "$SERVICE_NAME"
fi

# Deshabilitar (elimina el symlink de WantedBy)
if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
    info "Deshabilitando servicio..."
    systemctl disable "$SERVICE_NAME"
fi

# Borrar el unit file
if [[ -f "$UNIT_DEST" ]]; then
    info "Eliminando unit file: ${UNIT_DEST}"
    rm -f "$UNIT_DEST"
fi

info "Recargando daemon de systemd..."
systemctl daemon-reload
systemctl reset-failed 2>/dev/null || true

echo ""
echo "═══════════════════════════════════════════════════════"
echo "  Servicio '${SERVICE_NAME}' desinstalado correctamente."
echo "  El binario y config.toml no han sido modificados."
echo "═══════════════════════════════════════════════════════"
