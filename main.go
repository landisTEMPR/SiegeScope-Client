package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"r6-replay-recorder/database"
	"r6-replay-recorder/parser"
	"r6-replay-recorder/ui"
)

func main() {
	// 1. Initialize Database
	log.Println("Initializing database...")
	db, err := database.New()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()
	log.Println("Database initialized")

	// 2. Initialize App
	log.Println("Creating app...")
	a := app.New()

	// 3. Apply custom SiegeScope theme
	log.Println("Applying theme...")
	a.Settings().SetTheme(&ui.SiegeScopeTheme{})
	log.Println("Theme applied")

	// 4. Create window
	log.Println("Creating window...")
	w := a.NewWindow("SiegeScope")

	// 5. Create parser and UI
	log.Println("Creating parser and UI...")
	p := parser.New(db)
	u := ui.New(w, db, p)

	// 6. Build and set content
	log.Println("Building UI...")
	content := u.Build()
	log.Println("Setting content...")
	w.SetContent(container.NewMax(content))

	// 7. Resize and show
	log.Println("Resizing window...")
	w.Resize(fyne.NewSize(1500, 750))

	log.Println("Showing window - app should stay open now...")
	w.ShowAndRun()

	// 8. Cleanup (only runs after window closes)
	u.StopWatcher()
	log.Println("SiegeScope closed.")
}
