package app

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"

	"r6-replay-recorder/database"
	"r6-replay-recorder/parser"
	"r6-replay-recorder/ui"
)

// App represents the main application
type App struct {
	fyneApp fyne.App
	window  fyne.Window
	db      *database.Database
	parser  *parser.Parser
	ui      *ui.UI
}

// New creates a new application instance
func New() *App {
	// Create Fyne app with unique ID for settings storage
	fyneApp := app.NewWithID("com.r6replayrecorder.app")
	fyneApp.Settings().SetTheme(theme.DarkTheme())

	// Create main window
	window := fyneApp.NewWindow("R6 Replay Recorder")
	window.Resize(fyne.NewSize(1280, 800))
	window.CenterOnScreen()
	window.SetMaster() // Ensures app quits when this window closes

	// Initialize database
	db, err := database.New()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize parser
	p := parser.New(db)

	// Initialize UI
	userInterface := ui.New(window, db, p)

	return &App{
		fyneApp: fyneApp,
		window:  window,
		db:      db,
		parser:  p,
		ui:      userInterface,
	}
}

// Run starts the application
func (a *App) Run() {
	// Build and set UI content
	a.window.SetContent(a.ui.Build())

	// Handle window close - must quit the app entirely
	a.window.SetCloseIntercept(func() {
		a.cleanup()
		a.fyneApp.Quit()
	})

	// Set up system tray (optional)
	a.setupSystemTray()

	// Check for auto-import on startup
	a.checkAutoImport()

	// Show window and run
	a.window.ShowAndRun()
}

func (a *App) cleanup() {
	if a.db != nil {
		a.db.Close()
	}
}

func (a *App) setupSystemTray() {
	// Fyne v2 supports system tray on desktop
	// This creates a tray icon for quick access
	if desk, ok := a.fyneApp.(interface {
		SetSystemTrayMenu(menu *fyne.Menu)
		SetSystemTrayIcon(icon fyne.Resource)
	}); ok {
		menu := fyne.NewMenu("R6 Replay Recorder",
			fyne.NewMenuItem("Show", func() {
				a.window.Show()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				a.cleanup()
				a.fyneApp.Quit()
			}),
		)
		desk.SetSystemTrayMenu(menu)
	}
}

func (a *App) checkAutoImport() {
	settings, err := a.db.GetSettings()
	if err != nil {
		return
	}

	if settings.AutoImport && settings.ReplayFolder != "" {
		// Start watching the replay folder
		// This is handled by the UI component when it initializes
	}
}
