#!/usr/bin/env bash

# Naviger Uninstall Script for Linux and macOS
# Reverses install.sh: stops and disables service/agent, removes files and symlinks

set -euo pipefail

INSTALL_DIR="/opt/naviger"
BIN_DIR="/usr/local/bin"
SERVICE_FILE="/etc/systemd/system/naviger.service"
PLIST_NAME="com.naviger.server.plist"

usage() {
  cat <<EOF
Usage: $0 [--yes|-y] [--help|-h]

Options:
  -y, --yes    Don't prompt, run non-interactively
  -h, --help   Show this help message

This script must be run as root (use sudo).
It will attempt to stop and remove the Naviger systemd service (Linux) or launchd agent (macOS),
remove installation directory (${INSTALL_DIR}), and remove symlinks in ${BIN_DIR}.
EOF
}

# Parse args
FORCE=no
for arg in "$@"; do
  case "$arg" in
    -y|--yes) FORCE=yes; shift || true;;
    -h|--help) usage; exit 0;;
    *) ;;
  esac
done

if [ "${EUID:-$(id -u)}" -ne 0 ]; then
  echo "This script must be run as root. Use sudo ./uninstall.sh"
  exit 1
fi

REAL_USER="${SUDO_USER:-$USER}"
if [ -z "$REAL_USER" ]; then
  REAL_USER="$USER"
fi

OS="$(uname -s)"
case "$OS" in
  Linux*) OS_TYPE=linux ;;
  Darwin*) OS_TYPE=macos ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

if [ "$FORCE" != "yes" ]; then
  echo "This will remove Naviger installation at: ${INSTALL_DIR}"
  echo "It will also remove symlinks in ${BIN_DIR} and any service/agent configuration."
  read -r -p "Are you sure you want to continue? [y/N]: " confirm
  case "$confirm" in
    [yY]|[yY][eE][sS]) ;;
    *) echo "Aborted."; exit 0 ;;
  esac
fi

echo "Stopping and removing service/agent (if present)..."

if [ "$OS_TYPE" = "linux" ]; then
  if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet naviger; then
      echo "Stopping naviger service..."
      systemctl stop naviger || true
    fi

    if systemctl is-enabled --quiet naviger 2>/dev/null; then
      echo "Disabling naviger service..."
      systemctl disable naviger || true
    fi

    if [ -f "$SERVICE_FILE" ]; then
      echo "Removing systemd service file ${SERVICE_FILE}..."
      rm -f "$SERVICE_FILE"
      echo "Reloading systemd daemon..."
      systemctl daemon-reload || true
    fi
  else
    echo "systemctl not found; skipping systemd cleanup."
  fi

elif [ "$OS_TYPE" = "macos" ]; then
  # Determine user's home (when run as sudo the SUDO_USER is the original user)
  if [ "$REAL_USER" = "root" ]; then
    USER_HOME="/var/root"
  else
    USER_HOME="/Users/$REAL_USER"
  fi
  PLIST_FILE="$USER_HOME/Library/LaunchAgents/${PLIST_NAME}"

  if [ -f "$PLIST_FILE" ]; then
    echo "Unloading launchd agent $PLIST_FILE..."
    if [ "$REAL_USER" != "root" ]; then
      sudo -u "$REAL_USER" launchctl unload "$PLIST_FILE" 2>/dev/null || true
    else
      launchctl unload "$PLIST_FILE" 2>/dev/null || true
    fi

    echo "Removing plist file..."
    rm -f "$PLIST_FILE"
  else
    echo "No launchd agent plist found at $PLIST_FILE"
  fi
fi

# Remove symlinks (only if they point to the install dir or exist)
echo "Removing symlinks in ${BIN_DIR}..."
if [ -L "${BIN_DIR}/naviger-cli" ] || [ -e "${BIN_DIR}/naviger-cli" ]; then
  rm -f "${BIN_DIR}/naviger-cli" || true
  echo "Removed ${BIN_DIR}/naviger-cli"
fi
if [ -L "${BIN_DIR}/naviger-server" ] || [ -e "${BIN_DIR}/naviger-server" ]; then
  rm -f "${BIN_DIR}/naviger-server" || true
  echo "Removed ${BIN_DIR}/naviger-server"
fi

# Remove installation directory
if [ -d "$INSTALL_DIR" ]; then
  echo "Removing installation directory ${INSTALL_DIR}..."
  rm -rf "$INSTALL_DIR" || true
  echo "Removed ${INSTALL_DIR}"
else
  echo "No installation directory found at ${INSTALL_DIR}"
fi

# Additional cleanup: logs in /tmp
if [ -f "/tmp/naviger.out" ]; then
  rm -f /tmp/naviger.out || true
fi
if [ -f "/tmp/naviger.err" ]; then
  rm -f /tmp/naviger.err || true
fi

echo "Uninstall complete."

exit 0

