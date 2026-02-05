#!/bin/bash

# R6 Replay Recorder Build Script
# This script builds the application for Windows, macOS, and Linux

APP_NAME="R6ReplayRecorder"
VERSION="1.0.0"

echo "Building $APP_NAME v$VERSION..."

# Create build directory
mkdir -p build

# Install dependencies
echo "Installing dependencies..."
go mod tidy

# Build for current platform first
echo "Building for current platform..."
go build -o "build/$APP_NAME" .

# Cross-compile for Windows
echo "Building for Windows..."
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
    go build -ldflags="-H windowsgui" -o "build/${APP_NAME}_windows_amd64.exe" .

# Cross-compile for macOS (requires macOS SDK)
echo "Building for macOS..."
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
    go build -o "build/${APP_NAME}_darwin_amd64" . 2>/dev/null || echo "macOS build requires macOS SDK"

CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
    go build -o "build/${APP_NAME}_darwin_arm64" . 2>/dev/null || echo "macOS ARM build requires macOS SDK"

# Build for Linux
echo "Building for Linux..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -o "build/${APP_NAME}_linux_amd64" .

echo ""
echo "Build complete! Binaries are in the 'build' directory."
echo ""
echo "For creating installers:"
echo "  Windows: Use Inno Setup with windows-installer.iss"
echo "  macOS: Use create-dmg or the provided script"
echo "  Linux: Use the .deb or AppImage scripts"
