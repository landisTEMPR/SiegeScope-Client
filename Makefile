# R6 Replay Recorder Makefile

APP_NAME = R6ReplayRecorder
VERSION = 1.0.0
BUILD_DIR = build
INSTALLERS_DIR = installers

.PHONY: all clean deps build build-windows build-linux build-macos install run

all: deps build

deps:
	go mod tidy

build:
	@echo "Building for current platform..."
	go build -o $(BUILD_DIR)/$(APP_NAME) .

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
		go build -ldflags="-H windowsgui" -o $(BUILD_DIR)/$(APP_NAME)_windows_amd64.exe .

build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		go build -o $(BUILD_DIR)/$(APP_NAME)_linux_amd64 .

build-macos-amd64:
	@echo "Building for macOS (Intel)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
		go build -o $(BUILD_DIR)/$(APP_NAME)_darwin_amd64 .

build-macos-arm64:
	@echo "Building for macOS (Apple Silicon)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
		go build -o $(BUILD_DIR)/$(APP_NAME)_darwin_arm64 .

build-all: build-windows build-linux build-macos-amd64 build-macos-arm64

installer-windows: build-windows
	@echo "Creating Windows installer..."
	@mkdir -p $(INSTALLERS_DIR)
	@echo "Run Inno Setup with windows-installer.iss to create the installer"

installer-linux: build-linux
	@echo "Creating Linux .deb package..."
	@chmod +x create-linux-deb.sh
	@./create-linux-deb.sh

installer-macos: build-macos-amd64 build-macos-arm64
	@echo "Creating macOS DMG..."
	@chmod +x create-macos-dmg.sh
	@./create-macos-dmg.sh

run: build
	./$(BUILD_DIR)/$(APP_NAME)

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(INSTALLERS_DIR)

# Development helpers
test:
	go test ./...

lint:
	golangci-lint run

fmt:
	go fmt ./...

# Show help
help:
	@echo "R6 Replay Recorder Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all              - Install deps and build for current platform"
	@echo "  deps             - Install Go dependencies"
	@echo "  build            - Build for current platform"
	@echo "  build-windows    - Cross-compile for Windows"
	@echo "  build-linux      - Build for Linux"
	@echo "  build-macos-amd64- Build for macOS Intel"
	@echo "  build-macos-arm64- Build for macOS Apple Silicon"
	@echo "  build-all        - Build for all platforms"
	@echo "  installer-windows- Create Windows installer (requires Inno Setup)"
	@echo "  installer-linux  - Create Linux .deb package"
	@echo "  installer-macos  - Create macOS .dmg"
	@echo "  run              - Build and run"
	@echo "  clean            - Remove build artifacts"
	@echo "  test             - Run tests"
	@echo "  help             - Show this help"
