# R6 Replay Recorder

A desktop application for recording and tracking your Rainbow Six: Siege match replays. Built with Go and Fyne, using the [r6-dissect](https://github.com/redraskal/r6-dissect) library for parsing `.rec` files.

## Features

- **Import Match Replays**: Import individual matches or bulk import entire replay folders
- **Persistent Storage**: All data stored locally in SQLite - survives app restarts
- **Match History**: Browse all your recorded matches with filtering by map, match type, and result
- **Round Details**: View round-by-round breakdown including players, operators, and events
- **Statistics Dashboard**: Track your win rate, performance per map, and more
- **Auto-Import**: Optionally watch your replay folder for new matches
- **Cross-Platform**: Runs on Windows, macOS, and Linux

## Installation

### Windows
1. Download `R6ReplayRecorder_Setup_x.x.x.exe` from the [Releases](https://github.com/yourusername/r6-replay-recorder/releases) page
2. Run the installer
3. Launch from Start Menu or Desktop shortcut

### macOS
1. Download `R6ReplayRecorder_x.x.x.dmg` from Releases
2. Open the DMG and drag the app to Applications
3. Launch from Applications folder

### Linux
**Debian/Ubuntu:**
```bash
sudo dpkg -i r6-replay-recorder_x.x.x_amd64.deb
```

**Other distributions:**
Download the binary and run directly, or use the AppImage.

## Building from Source

### Prerequisites
- Go 1.21 or later
- GCC (for SQLite CGO compilation)
- Fyne dependencies:
  - **Windows**: MinGW-w64
  - **macOS**: Xcode command line tools
  - **Linux**: `sudo apt install libgl1-mesa-dev xorg-dev`

### Build Steps

```bash
# Clone the repository
git clone https://github.com/yourusername/r6-replay-recorder.git
cd r6-replay-recorder

# Install Go dependencies
go mod tidy

# Build for current platform
go build -o R6ReplayRecorder .

# Or use the build script for all platforms
chmod +x build.sh
./build.sh
```

### Creating Installers

**Windows (requires Inno Setup):**
```bash
# Build the Windows binary first
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui" -o build/R6ReplayRecorder_windows_amd64.exe .

# Then run Inno Setup with windows-installer.iss
```

**macOS:**
```bash
chmod +x create-macos-dmg.sh
./create-macos-dmg.sh
```

**Linux (.deb):**
```bash
chmod +x create-linux-deb.sh
./create-linux-deb.sh
```

## Usage

### First Launch
1. Open R6 Replay Recorder
2. Go to **Settings** tab
3. Set your R6 replay folder (usually `Documents/My Games/Rainbow Six - Siege/replays`)
4. Enable **Watch folder for new replays** if you want auto-import

### Importing Matches
- **Import Match**: Import a single match folder
- **Import All**: Bulk import all matches from a folder

### Viewing Data
- **Matches Tab**: Browse all imported matches, click on a match for details
- **Stats Tab**: View aggregated statistics

### Filtering
Use the filter dropdowns to narrow down matches by:
- Map
- Match Type (Ranked, QuickMatch, Unranked)
- Result (Wins, Losses)

## Data Location

Your match data is stored locally:
- **Windows**: `%APPDATA%\R6ReplayRecorder\replays.db`
- **macOS**: `~/Library/Application Support/R6ReplayRecorder/replays.db`
- **Linux**: `~/.config/R6ReplayRecorder/replays.db`

## Dependencies

- [Fyne](https://fyne.io/) - Cross-platform GUI toolkit
- [r6-dissect](https://github.com/redraskal/r6-dissect) - R6 replay file parser
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details.

## Credits

- [redraskal](https://github.com/redraskal) for the excellent r6-dissect library
- [stnokott](https://github.com/stnokott) and [draguve](https://github.com/draguve) for additional reverse engineering work on the .rec format

## Troubleshooting

### App won't start
- Ensure you have the required system libraries installed
- On Linux, install: `libgl1-mesa-dev xorg-dev`

### Replays not importing
- Make sure you're selecting the match folder (e.g., `Match-2024-01-01_12-00-00-000`) not individual .rec files
- Check that the replay files aren't corrupted

### Database errors
- Try deleting the `replays.db` file to reset (you'll lose saved data)
- Make sure the app has write permissions to the data directory
# SiegeScope-Client
