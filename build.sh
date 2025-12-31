#!/bin/bash

set -e

echo "Cleaning up previous build..."
rm -rf dist

echo "Building web frontend..."
cd web || exit
npm install
npm run build
cd ..

mkdir -p dist/web_dist
cp -r web/dist/* dist/web_dist/

echo "Building Go backend..."
echo "Building server..."
go build -v -o dist/naviger-server ./cmd/server
echo "Building CLI..."
go build -v -o dist/naviger-cli ./cmd/cli

if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Creating macOS Application Bundle..."
    APP_NAME="Naviger"
    APP_DIR="dist/${APP_NAME}.app"
    CONTENTS_DIR="${APP_DIR}/Contents"
    MACOS_DIR="${CONTENTS_DIR}/MacOS"
    RESOURCES_DIR="${CONTENTS_DIR}/Resources"

    mkdir -p "${MACOS_DIR}"
    mkdir -p "${RESOURCES_DIR}"

    cp "dist/naviger-server" "${MACOS_DIR}/${APP_NAME}"

    cp -r "dist/web_dist" "${MACOS_DIR}/"

    cat > "${CONTENTS_DIR}/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>${APP_NAME}</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>com.naviger.server</string>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF

    if command -v sips >/dev/null 2>&1; then
        ICON_SRC="cmd/server/icon.png"
        if [ -f "$ICON_SRC" ]; then
            echo "Generating AppIcon.icns..."
            ICONSET="${RESOURCES_DIR}/AppIcon.iconset"
            mkdir -p "${ICONSET}"

            sips -z 16 16     "$ICON_SRC" --out "${ICONSET}/icon_16x16.png" > /dev/null 2>&1
            sips -z 32 32     "$ICON_SRC" --out "${ICONSET}/icon_16x16@2x.png" > /dev/null 2>&1
            sips -z 32 32     "$ICON_SRC" --out "${ICONSET}/icon_32x32.png" > /dev/null 2>&1
            sips -z 64 64     "$ICON_SRC" --out "${ICONSET}/icon_32x32@2x.png" > /dev/null 2>&1
            sips -z 128 128   "$ICON_SRC" --out "${ICONSET}/icon_128x128.png" > /dev/null 2>&1
            sips -z 256 256   "$ICON_SRC" --out "${ICONSET}/icon_128x128@2x.png" > /dev/null 2>&1
            sips -z 256 256   "$ICON_SRC" --out "${ICONSET}/icon_256x256.png" > /dev/null 2>&1
            sips -z 512 512   "$ICON_SRC" --out "${ICONSET}/icon_256x256@2x.png" > /dev/null 2>&1
            sips -z 512 512   "$ICON_SRC" --out "${ICONSET}/icon_512x512.png" > /dev/null 2>&1

            if command -v iconutil >/dev/null 2>&1; then
                iconutil -c icns "${ICONSET}" -o "${RESOURCES_DIR}/AppIcon.icns"
                rm -rf "${ICONSET}"
            else
                rm -rf "${ICONSET}"
            fi
        fi
    fi

    echo "macOS App Bundle created at ${APP_DIR}"

    rm "dist/naviger-server"
    rm -rf "dist/web_dist"

elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "Creating Linux Desktop Entry..."

    ICON_SRC="cmd/server/icon.png"
    if [ -f "$ICON_SRC" ]; then
        cp "$ICON_SRC" "dist/naviger.png"
    fi

    cat > "dist/naviger.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=Naviger
Comment=Naviger Server Manager
Exec=/opt/naviger/naviger-server
Icon=/opt/naviger/naviger.png
Terminal=false
Categories=Development;Server;
EOF

    chmod +x "dist/naviger.desktop"

    echo "Linux desktop files created in dist/"
fi

echo "Build finished successfully!"
