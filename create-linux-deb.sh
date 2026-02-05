#!/bin/bash

# Linux .deb Package Creation Script for R6 Replay Recorder

APP_NAME="r6-replay-recorder"
VERSION="1.0.0"
MAINTAINER="Your Name <your.email@example.com>"
DESCRIPTION="Match replay recorder for Rainbow Six: Siege"

echo "Creating Linux .deb package..."

# Create package directory structure
PKG_DIR="build/${APP_NAME}_${VERSION}_amd64"
mkdir -p "$PKG_DIR/DEBIAN"
mkdir -p "$PKG_DIR/usr/bin"
mkdir -p "$PKG_DIR/usr/share/applications"
mkdir -p "$PKG_DIR/usr/share/icons/hicolor/256x256/apps"
mkdir -p "$PKG_DIR/usr/share/$APP_NAME"

# Copy binary
cp "build/R6ReplayRecorder_linux_amd64" "$PKG_DIR/usr/bin/$APP_NAME"
chmod +x "$PKG_DIR/usr/bin/$APP_NAME"

# Create control file
cat > "$PKG_DIR/DEBIAN/control" << EOF
Package: $APP_NAME
Version: $VERSION
Section: games
Priority: optional
Architecture: amd64
Depends: libc6, libgl1, libx11-6, libxcursor1, libxrandr2, libxinerama1, libxi6, libxxf86vm1
Maintainer: $MAINTAINER
Description: $DESCRIPTION
 R6 Replay Recorder parses Rainbow Six: Siege replay files (.rec)
 and stores match data locally for statistics tracking and analysis.
EOF

# Create desktop entry
cat > "$PKG_DIR/usr/share/applications/$APP_NAME.desktop" << EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=R6 Replay Recorder
Comment=Match replay recorder for Rainbow Six: Siege
Exec=$APP_NAME
Icon=$APP_NAME
Terminal=false
Categories=Game;Utility;
Keywords=rainbow;six;siege;replay;stats;
EOF

# Copy icon if exists
if [ -f "assets/icon.png" ]; then
    cp "assets/icon.png" "$PKG_DIR/usr/share/icons/hicolor/256x256/apps/$APP_NAME.png"
fi

# Create post-install script
cat > "$PKG_DIR/DEBIAN/postinst" << 'EOF'
#!/bin/bash
update-desktop-database 2>/dev/null || true
gtk-update-icon-cache /usr/share/icons/hicolor 2>/dev/null || true
EOF
chmod 755 "$PKG_DIR/DEBIAN/postinst"

# Build the .deb package
mkdir -p installers
dpkg-deb --build "$PKG_DIR" "installers/${APP_NAME}_${VERSION}_amd64.deb"

echo "Done! Package created at: installers/${APP_NAME}_${VERSION}_amd64.deb"
echo ""
echo "To install: sudo dpkg -i installers/${APP_NAME}_${VERSION}_amd64.deb"
echo "To uninstall: sudo apt remove $APP_NAME"
