#!/bin/bash

# macOS DMG Creation Script for R6 Replay Recorder
# Requires: create-dmg (brew install create-dmg)

APP_NAME="R6 Replay Recorder"
VERSION="1.0.0"
DMG_NAME="R6ReplayRecorder_${VERSION}"
APP_BUNDLE="$APP_NAME.app"

echo "Creating macOS app bundle and DMG..."

# Create app bundle structure
mkdir -p "build/$APP_BUNDLE/Contents/MacOS"
mkdir -p "build/$APP_BUNDLE/Contents/Resources"

# Copy binary (use arm64 or amd64 based on target)
if [ -f "build/R6ReplayRecorder_darwin_arm64" ]; then
    cp "build/R6ReplayRecorder_darwin_arm64" "build/$APP_BUNDLE/Contents/MacOS/R6ReplayRecorder"
elif [ -f "build/R6ReplayRecorder_darwin_amd64" ]; then
    cp "build/R6ReplayRecorder_darwin_amd64" "build/$APP_BUNDLE/Contents/MacOS/R6ReplayRecorder"
else
    echo "No macOS binary found. Build the app first."
    exit 1
fi

chmod +x "build/$APP_BUNDLE/Contents/MacOS/R6ReplayRecorder"

# Create Info.plist
cat > "build/$APP_BUNDLE/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>R6ReplayRecorder</string>
    <key>CFBundleIdentifier</key>
    <string>com.r6replayrecorder.app</string>
    <key>CFBundleName</key>
    <string>R6 Replay Recorder</string>
    <key>CFBundleDisplayName</key>
    <string>R6 Replay Recorder</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleSignature</key>
    <string>????</string>
    <key>CFBundleIconFile</key>
    <string>icon</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSMinimumSystemVersion</key>
    <string>10.14</string>
    <key>NSSupportsAutomaticGraphicsSwitching</key>
    <true/>
</dict>
</plist>
EOF

# Copy icon if exists
if [ -f "assets/icon.icns" ]; then
    cp "assets/icon.icns" "build/$APP_BUNDLE/Contents/Resources/icon.icns"
fi

# Create DMG
mkdir -p installers

if command -v create-dmg &> /dev/null; then
    create-dmg \
        --volname "$APP_NAME" \
        --volicon "assets/icon.icns" \
        --window-pos 200 120 \
        --window-size 600 400 \
        --icon-size 100 \
        --icon "$APP_BUNDLE" 150 190 \
        --hide-extension "$APP_BUNDLE" \
        --app-drop-link 450 185 \
        "installers/${DMG_NAME}.dmg" \
        "build/$APP_BUNDLE"
else
    echo "create-dmg not found. Creating simple DMG..."
    hdiutil create -volname "$APP_NAME" -srcfolder "build/$APP_BUNDLE" -ov -format UDZO "installers/${DMG_NAME}.dmg"
fi

echo "Done! DMG created at: installers/${DMG_NAME}.dmg"
