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

# Ask for installation mode
echo "Select installation mode:"
echo "1) Headless (Service/Daemon)"
echo "2) Desktop (App/Shortcut)"
read -p "Enter choice [1-2]: " INSTALL_MODE

if [ "$INSTALL_MODE" != "1" ] && [ "$INSTALL_MODE" != "2" ]; then
    echo "Invalid choice. Exiting."
    exit 1
fi

# Check for dependencies
command -v curl >/dev/null 2>&1 || { echo >&2 "curl is required but not installed. Aborting."; exit 1; }
command -v unzip >/dev/null 2>&1 || { echo >&2 "unzip is required but not installed. Aborting."; exit 1; }

# Check for root privileges if installing as service or to system dirs
if [ "$INSTALL_MODE" = "1" ] && [ "$EUID" -ne 0 ]; then
  echo "Please run as root (sudo ./install.sh) for Headless installation."
  exit 1
fi

# Determine actual user
REAL_USER="${SUDO_USER:-$USER}"

# Define paths
INSTALL_DIR="/opt/naviger"
BIN_DIR="/usr/local/bin"

# Stop existing service if running
echo "Checking for existing installation..."
if [ "$OS_TYPE" = "linux" ]; then
    if systemctl is-active --quiet naviger; then
        echo "Stopping existing Naviger service..."
        sudo systemctl stop naviger
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
    sudo rm -rf "${INSTALL_DIR}"
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

# Installation Logic
if [ "$INSTALL_MODE" = "1" ]; then
    # HEADLESS MODE
    echo "Installing Headless Mode..."

    sudo mkdir -p "${INSTALL_DIR}"
    sudo rm -rf "${INSTALL_DIR}/*"

    # Check if extracted content is inside a folder (common with zips) or flat
    if [ -d "${TMP_DIR}/extracted/Naviger.app" ]; then
         # macOS app bundle case - we need the binary inside
         sudo cp -r "${TMP_DIR}/extracted/Naviger.app/Contents/MacOS/Naviger" "${INSTALL_DIR}/naviger-server"
         # Also copy web_dist if it exists inside Resources or MacOS
         if [ -d "${TMP_DIR}/extracted/Naviger.app/Contents/MacOS/web_dist" ]; then
             sudo cp -r "${TMP_DIR}/extracted/Naviger.app/Contents/MacOS/web_dist" "${INSTALL_DIR}/"
         fi
    else
         sudo cp -r "${TMP_DIR}/extracted/"* "${INSTALL_DIR}/"
    fi

    # Ensure CLI is installed if it was outside the app bundle
    if [ -f "${TMP_DIR}/extracted/naviger-cli" ]; then
        sudo cp "${TMP_DIR}/extracted/naviger-cli" "${INSTALL_DIR}/"
    fi

    # Cleanup
    rm -rf "${TMP_DIR}"

    # Set permissions
    sudo chmod +x "${INSTALL_DIR}/naviger-server"
    if [ -f "${INSTALL_DIR}/naviger-cli" ]; then
        sudo chmod +x "${INSTALL_DIR}/naviger-cli"
        sudo ln -sf "${INSTALL_DIR}/naviger-cli" "${BIN_DIR}/naviger-cli"
    fi
    sudo chown -R "$REAL_USER" "$INSTALL_DIR"

    # Service Configuration
    if [ "$OS_TYPE" = "linux" ]; then
        SERVICE_FILE="/etc/systemd/system/naviger.service"
        echo "Setting up systemd service..."

        sudo bash -c "cat > ${SERVICE_FILE}" <<EOF
[Unit]
Description=Naviger Server Daemon
After=network.target

[Service]
Type=simple
User=$REAL_USER
ExecStart=${INSTALL_DIR}/naviger-server --headless
WorkingDirectory=${INSTALL_DIR}
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

        echo "Reloading systemd daemon..."
        sudo systemctl daemon-reload
        echo "Enabling and starting naviger service..."
        sudo systemctl enable naviger
        sudo systemctl start naviger
        echo "Naviger service installed and started (Headless)."

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
        <string>--headless</string>
    </array>
    <key>WorkingDirectory</key>
    <string>${INSTALL_DIR}</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
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
        echo "Naviger agent installed and loaded (Headless)."
    fi

else
    # DESKTOP MODE
    echo "Installing Desktop Mode..."

    if [ "$OS_TYPE" = "macos" ]; then
        APP_DEST="/Applications/Naviger.app"
        echo "Installing to ${APP_DEST}..."

        # Look for Naviger.app in extracted files
        if [ -d "${TMP_DIR}/extracted/Naviger.app" ]; then
            sudo rm -rf "${APP_DEST}"
            sudo cp -r "${TMP_DIR}/extracted/Naviger.app" "/Applications/"
            echo "Naviger.app installed to /Applications."
        else
            echo "Error: Naviger.app not found in the downloaded package."
        fi

    elif [ "$OS_TYPE" = "linux" ]; then
        # Install binaries to /opt/naviger
        sudo mkdir -p "${INSTALL_DIR}"
        sudo rm -rf "${INSTALL_DIR}/*"
        sudo cp -r "${TMP_DIR}/extracted/"* "${INSTALL_DIR}/"

        # Set permissions
        sudo chmod +x "${INSTALL_DIR}/naviger-server"
        sudo chown -R "$REAL_USER" "$INSTALL_DIR"

        # Install Desktop Entry
        DESKTOP_FILE="${INSTALL_DIR}/naviger.desktop"
        ICON_FILE="${INSTALL_DIR}/naviger.png"

        # Ensure paths in desktop file are correct
        if [ -f "$DESKTOP_FILE" ]; then
             # The build script should have set Exec=/opt/naviger/naviger-server
             # We copy it to system applications
             sudo cp "$DESKTOP_FILE" "/usr/share/applications/naviger.desktop"
             echo "Desktop entry installed."
        else
             echo "Warning: naviger.desktop not found in package. Creating one..."
             cat > naviger.desktop <<EOF
[Desktop Entry]
Type=Application
Name=Naviger
Comment=Naviger Server Manager
Exec=${INSTALL_DIR}/naviger-server
Icon=${ICON_FILE}
Terminal=false
Categories=Development;Server;
EOF
             sudo mv naviger.desktop "/usr/share/applications/"
        fi

        echo "Naviger installed in Desktop mode."
    fi

    # Install CLI if available
    if [ -f "${TMP_DIR}/extracted/naviger-cli" ]; then
        echo "Installing CLI tool..."
        sudo cp "${TMP_DIR}/extracted/naviger-cli" "${BIN_DIR}/naviger-cli"
        sudo chmod +x "${BIN_DIR}/naviger-cli"
    fi

    # Cleanup
    rm -rf "${TMP_DIR}"
fi

echo "Installation complete!"
