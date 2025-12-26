#!/bin/bash

# Naviger Installation Script for Linux and macOS

set -e

REPO_OWNER="andre-carbajal"
REPO_NAME="Naviger"

# Determine OS and Architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "${OS}" in
    Linux*)     OS_TYPE="linux";;
    Darwin*)    OS_TYPE="macos";;
    *)          echo "Unsupported operating system: ${OS}"; exit 1;;
esac

if [ "$OS_TYPE" = "macos" ]; then
    if [ "$ARCH" = "x86_64" ]; then
        ASSET_SUFFIX="macos-x64"
    elif [ "$ARCH" = "arm64" ]; then
        ASSET_SUFFIX="macos-arm64"
    else
        echo "Unsupported architecture: ${ARCH}"; exit 1
    fi
else
    # Linux
    if [ "$ARCH" = "x86_64" ]; then
        ASSET_SUFFIX="linux"
    else
        echo "Warning: The official Linux build is optimized for x86_64. Your architecture is ${ARCH}."
        echo "Attempting to install anyway (it might not work)..."
        ASSET_SUFFIX="linux"
    fi
fi

echo "Detected System: ${OS} (${ARCH}) -> Asset: ${ASSET_SUFFIX}"

# Check for dependencies
command -v curl >/dev/null 2>&1 || { echo >&2 "curl is required but not installed. Aborting."; exit 1; }
command -v unzip >/dev/null 2>&1 || { echo >&2 "unzip is required but not installed. Aborting."; exit 1; }

# Check for root privileges
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (sudo ./install.sh)"
  exit 1
fi

# Determine actual user
REAL_USER="${SUDO_USER:-$USER}"
if [ "$REAL_USER" = "root" ]; then
    echo "Warning: Installing as root user. It is recommended to run via sudo from a regular user account."
fi

# Define paths
INSTALL_DIR="/opt/naviger"
BIN_DIR="/usr/local/bin"

# Stop existing service if running
echo "Checking for existing installation..."
if [ "$OS_TYPE" = "linux" ]; then
    if systemctl is-active --quiet naviger; then
        echo "Stopping existing Naviger service..."
        systemctl stop naviger
    fi
elif [ "$OS_TYPE" = "macos" ]; then
    if [ "$REAL_USER" = "root" ]; then
        USER_HOME="/var/root"
    else
        USER_HOME="/Users/$REAL_USER"
    fi
    PLIST_FILE="$USER_HOME/Library/LaunchAgents/com.naviger.server.plist"

    if [ -f "$PLIST_FILE" ]; then
        echo "Stopping existing Naviger agent..."
        if [ "$REAL_USER" != "root" ]; then
            sudo -u "$REAL_USER" launchctl unload "$PLIST_FILE" 2>/dev/null || true
        else
            launchctl unload "$PLIST_FILE" 2>/dev/null || true
        fi
    fi
fi

# Clean up previous installation
if [ -d "$INSTALL_DIR" ]; then
    echo "Removing previous installation at ${INSTALL_DIR}..."
    rm -rf "${INSTALL_DIR}"
fi

# Fetch latest version
echo "Fetching latest release info..."
LATEST_URL=$(curl -Ls -o /dev/null -w %{url_effective} "https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest")
VERSION=$(basename "$LATEST_URL")

if [ -z "$VERSION" ] || [ "$VERSION" = "latest" ]; then
    echo "Error: Could not determine latest version."
    exit 1
fi

echo "Latest version: ${VERSION}"

ASSET_NAME="Naviger-${VERSION}-${ASSET_SUFFIX}.zip"
DOWNLOAD_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${ASSET_NAME}"

# Download and Extract
TMP_DIR=$(mktemp -d)
echo "Downloading ${ASSET_NAME} from ${DOWNLOAD_URL}..."

if curl -L -o "${TMP_DIR}/${ASSET_NAME}" "${DOWNLOAD_URL}" --fail; then
    echo "Download successful."
else
    echo "Error: Failed to download release. Please check if the asset exists for your platform."
    rm -rf "${TMP_DIR}"
    exit 1
fi

echo "Extracting..."
unzip -q "${TMP_DIR}/${ASSET_NAME}" -d "${TMP_DIR}/extracted"

echo "Installing Naviger to ${INSTALL_DIR}..."
mkdir -p "${INSTALL_DIR}"
cp -r "${TMP_DIR}/extracted/"* "${INSTALL_DIR}/"

# Cleanup
rm -rf "${TMP_DIR}"

# Set permissions
chmod +x "${INSTALL_DIR}/naviger-server"
chmod +x "${INSTALL_DIR}/naviger-cli"
chown -R "$REAL_USER" "$INSTALL_DIR"

echo "Creating symlinks..."
ln -sf "${INSTALL_DIR}/naviger-cli" "${BIN_DIR}/naviger-cli"
ln -sf "${INSTALL_DIR}/naviger-server" "${BIN_DIR}/naviger-server"

# Service Configuration
if [ "$OS_TYPE" = "linux" ]; then
    SERVICE_FILE="/etc/systemd/system/naviger.service"
    echo "Setting up systemd service..."

    cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=Naviger Server Daemon
After=network.target

[Service]
Type=simple
User=$REAL_USER
ExecStart=${INSTALL_DIR}/naviger-server
WorkingDirectory=${INSTALL_DIR}
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

    echo "Reloading systemd daemon..."
    systemctl daemon-reload
    echo "Enabling and starting naviger service..."
    systemctl enable naviger
    systemctl start naviger
    echo "Naviger service installed and started."

elif [ "$OS_TYPE" = "macos" ]; then
    # Get user home dir
    if [ "$REAL_USER" = "root" ]; then
        USER_HOME="/var/root"
    else
        USER_HOME="/Users/$REAL_USER"
    fi

    PLIST_FILE="$USER_HOME/Library/LaunchAgents/com.naviger.server.plist"

    echo "Setting up launchd agent at $PLIST_FILE..."

    # Ensure the LaunchAgents directory exists
    mkdir -p "$(dirname "$PLIST_FILE")"
    chown "$REAL_USER" "$(dirname "$PLIST_FILE")"

    cat > "${PLIST_FILE}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.naviger.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/naviger-server</string>
    </array>
    <key>WorkingDirectory</key>
    <string>${INSTALL_DIR}</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/naviger.out</string>
    <key>StandardErrorPath</key>
    <string>/tmp/naviger.err</string>
</dict>
</plist>
EOF

    # Fix permissions for the plist file
    chown "$REAL_USER" "$PLIST_FILE"

    echo "Loading launchd agent..."
    # Run launchctl as the user
    if [ "$REAL_USER" != "root" ]; then
        sudo -u "$REAL_USER" launchctl unload "$PLIST_FILE" 2>/dev/null || true
        sudo -u "$REAL_USER" launchctl load "$PLIST_FILE"
    else
        launchctl unload "$PLIST_FILE" 2>/dev/null || true
        launchctl load "$PLIST_FILE"
    fi
    echo "Naviger agent installed and loaded."
fi

echo "Installation complete!"
echo "You can now use 'naviger-cli' to manage your servers."
