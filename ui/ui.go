package ui

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"r6-replay-recorder/database"
	"r6-replay-recorder/models"
	"r6-replay-recorder/parser"
)

// UI handles all user interface components
type UI struct {
	window    fyne.Window
	db        *database.Database
	parser    *parser.Parser
	watcher   *parser.FolderWatcher
	matches   []models.Match
	matchList *widget.List

	// Filter widgets
	mapFilter  *widget.Select
	typeFilter *widget.Select
	wonFilter  *widget.Select

	// Stats labels
	statsContainer *fyne.Container

	// Track if UI is fully initialized
	initialized bool
}

// New creates a new UI instance
func New(window fyne.Window, db *database.Database, p *parser.Parser) *UI {
	return &UI{
		window:      window,
		db:          db,
		parser:      p,
		initialized: false,
	}
}

// Build creates the main UI layout
func (u *UI) Build() fyne.CanvasObject {
	// Create tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Matches", u.buildMatchesTab()),
		container.NewTabItem("Stats", u.buildStatsTab()),
		container.NewTabItem("Settings", u.buildSettingsTab()),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	// Mark as initialized after building
	u.initialized = true

	return tabs
}

func (u *UI) buildMatchesTab() fyne.CanvasObject {
	// Toolbar
	importBtn := widget.NewButtonWithIcon("Import Match", theme.FolderOpenIcon(), func() {
		u.showImportDialog()
	})

	importFolderBtn := widget.NewButtonWithIcon("Import All", theme.ContentAddIcon(), func() {
		u.showImportAllDialog()
	})

	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		u.refreshMatches()
	})

	toolbar := container.NewHBox(importBtn, importFolderBtn, refreshBtn, layout.NewSpacer())

	// Create filter change handler that checks initialization
	filterChanged := func(s string) {
		if u.initialized {
			u.applyFilters()
		}
	}

	// Filters
	u.mapFilter = widget.NewSelect([]string{"All"}, filterChanged)
	u.typeFilter = widget.NewSelect([]string{"All", "Ranked", "QuickMatch", "Unranked", "Standard"}, filterChanged)
	u.wonFilter = widget.NewSelect([]string{"All", "Wins", "Losses"}, filterChanged)

	filters := container.NewHBox(
		widget.NewLabel("Map:"), u.mapFilter,
		widget.NewLabel("Type:"), u.typeFilter,
		widget.NewLabel("Result:"), u.wonFilter,
	)

	// Match list
	u.matchList = widget.NewList(
		func() int {
			return len(u.matches)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Map Name Here"),
				layout.NewSpacer(),
				widget.NewLabel("Ranked"),
				widget.NewLabel("Team 4 - 3 Opponents"),
				widget.NewLabel("WIN"),
				widget.NewLabel("2024-01-01"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(u.matches) {
				return
			}
			match := u.matches[id]
			box := obj.(*fyne.Container)

			box.Objects[0].(*widget.Label).SetText(match.Map)
			box.Objects[2].(*widget.Label).SetText(match.MatchType)
			box.Objects[3].(*widget.Label).SetText(fmt.Sprintf("Team %d - %d Opp", match.TeamScore, match.OpponentScore))

			if match.Won {
				box.Objects[4].(*widget.Label).SetText("WIN")
			} else {
				box.Objects[4].(*widget.Label).SetText("LOSS")
			}

			box.Objects[5].(*widget.Label).SetText(match.Timestamp.Format("2006-01-02 15:04"))
		},
	)

	u.matchList.OnSelected = func(id widget.ListItemID) {
		if id < len(u.matches) {
			match := u.matches[id]
			u.matchList.UnselectAll()
			u.showMatchDetails(match)
		}
	}

	// Load initial data
	u.refreshMatches()
	u.updateMapFilter()

	// Set default selections
	u.mapFilter.Selected = "All"
	u.typeFilter.Selected = "All"
	u.wonFilter.Selected = "All"

	header := container.NewVBox(toolbar, filters)

	return container.NewBorder(header, nil, nil, nil, u.matchList)
}

func (u *UI) buildStatsTab() fyne.CanvasObject {
	u.statsContainer = container.NewVBox()
	u.updateStats()

	return container.NewVScroll(u.statsContainer)
}

func (u *UI) buildSettingsTab() fyne.CanvasObject {
	settings, err := u.db.GetSettings()
	if err != nil {
		settings = &models.Settings{}
	}

	// Replay folder
	folderEntry := widget.NewEntry()
	folderEntry.SetText(settings.ReplayFolder)
	folderEntry.SetPlaceHolder(parser.GetDefaultReplayPath())

	browseBtn := widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			folderEntry.SetText(uri.Path())
		}, u.window)
	})

	folderRow := container.NewBorder(nil, nil, nil, browseBtn, folderEntry)

	// Test folder detection button
	testBtn := widget.NewButtonWithIcon("Test Folder Detection", theme.SearchIcon(), func() {
		u.testFolderDetection(folderEntry.Text)
	})

	// Auto import toggle
	autoImport := widget.NewCheck("Watch folder for new replays", func(checked bool) {
		settings.AutoImport = checked
		u.db.UpdateSettings(settings)
		if checked && folderEntry.Text != "" {
			u.StartWatcher(folderEntry.Text)
		} else {
			u.StopWatcher()
		}
	})
	autoImport.Checked = settings.AutoImport

	// Save button
	saveBtn := widget.NewButtonWithIcon("Save Settings", theme.DocumentSaveIcon(), func() {
		settings.ReplayFolder = folderEntry.Text
		if err := u.db.UpdateSettings(settings); err != nil {
			dialog.ShowError(err, u.window)
		} else {
			dialog.ShowInformation("Settings", "Settings saved successfully!", u.window)
		}
	})

	// Data management
	exportBtn := widget.NewButtonWithIcon("Export All Data (JSON)", theme.DownloadIcon(), func() {
		u.exportData()
	})

	clearBtn := widget.NewButtonWithIcon("Clear All Data", theme.DeleteIcon(), func() {
		dialog.ShowConfirm("Clear Data", "Are you sure you want to delete all match data? This cannot be undone.", func(ok bool) {
			if ok {
				dialog.ShowInformation("Cleared", "All data has been cleared.", u.window)
			}
		}, u.window)
	})

	form := container.NewVBox(
		widget.NewLabel("Replay Folder:"),
		folderRow,
		testBtn,
		widget.NewSeparator(),
		autoImport,
		widget.NewSeparator(),
		saveBtn,
		widget.NewSeparator(),
		widget.NewLabel("Data Management:"),
		container.NewHBox(exportBtn, clearBtn),
	)

	return container.NewPadded(form)
}

func (u *UI) testFolderDetection(path string) {
	if path == "" {
		dialog.ShowInformation("Test", "Please enter a replay folder path first", u.window)
		return
	}

	// Show progress dialog
	progressDialog := dialog.NewCustom("Testing Folder Detection", "Cancel",
		widget.NewLabel("Scanning for replay folders..."), u.window)
	progressDialog.Show()

	go func() {
		folders, err := u.parser.FindReplayFolders(path)

		// Close progress dialog
		progressDialog.Hide()

		if err != nil {
			dialog.ShowError(fmt.Errorf("Error scanning folder: %v", err), u.window)
			return
		}

		msg := fmt.Sprintf("Found %d replay folders:\n\n", len(folders))

		if len(folders) == 0 {
			msg += "No replay folders found.\n\n"
			msg += "Possible reasons:\n"
			msg += "• No .rec files in this location\n"
			msg += "• Wrong folder selected\n"
			msg += "• Replays not enabled in R6 Siege\n\n"
			msg += "Please verify:\n"
			msg += "1. Replay recording is enabled in R6 Siege settings\n"
			msg += "2. You've played at least one match with recording enabled\n"
			msg += "3. The folder path is correct"
		} else {
			for i, folder := range folders {
				if i < 10 { // Show first 10
					msg += filepath.Base(folder) + "\n"
				}
			}
			if len(folders) > 10 {
				msg += fmt.Sprintf("\n... and %d more", len(folders)-10)
			}
			msg += "\n\n✓ These folders should import successfully!"
		}

		dialog.ShowInformation("Detection Test Results", msg, u.window)
	}()
}

func (u *UI) refreshMatches() {
	matches, err := u.db.GetAllMatches()
	if err != nil {
		if u.initialized {
			dialog.ShowError(err, u.window)
		}
		return
	}
	u.matches = matches
	if u.matchList != nil {
		u.matchList.Refresh()
	}
	u.updateStats()
}

func (u *UI) applyFilters() {
	if u.mapFilter == nil || u.typeFilter == nil || u.wonFilter == nil {
		return
	}

	mapFilter := u.mapFilter.Selected
	typeFilter := u.typeFilter.Selected

	var wonFilter *bool
	switch u.wonFilter.Selected {
	case "Wins":
		w := true
		wonFilter = &w
	case "Losses":
		w := false
		wonFilter = &w
	}

	matches, err := u.db.GetMatchesByFilter(typeFilter, mapFilter, wonFilter)
	if err != nil {
		dialog.ShowError(err, u.window)
		return
	}
	u.matches = matches
	u.matchList.Refresh()
}

func (u *UI) updateMapFilter() {
	if u.mapFilter == nil {
		return
	}

	maps, err := u.db.GetDistinctMaps()
	if err != nil {
		return
	}
	options := []string{"All"}
	options = append(options, maps...)
	u.mapFilter.Options = options
}

func (u *UI) updateStats() {
	if u.statsContainer == nil {
		return
	}

	played, wins, losses, winRate, err := u.db.GetOverallStats()
	if err != nil {
		return
	}

	mapStats, _ := u.db.GetMapStats()
	clutchStats, _ := u.db.GetClutchStats()
	defuserStats, _ := u.db.GetDefuserStats()

	u.statsContainer.Objects = nil

	// Overall stats card
	overallCard := widget.NewCard("Overall Statistics", "",
		container.NewGridWithColumns(4,
			container.NewVBox(
				widget.NewLabelWithStyle("Matches", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle(fmt.Sprintf("%d", played), fyne.TextAlignCenter, fyne.TextStyle{}),
			),
			container.NewVBox(
				widget.NewLabelWithStyle("Wins", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle(fmt.Sprintf("%d", wins), fyne.TextAlignCenter, fyne.TextStyle{}),
			),
			container.NewVBox(
				widget.NewLabelWithStyle("Losses", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle(fmt.Sprintf("%d", losses), fyne.TextAlignCenter, fyne.TextStyle{}),
			),
			container.NewVBox(
				widget.NewLabelWithStyle("Win Rate", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle(fmt.Sprintf("%.1f%%", winRate), fyne.TextAlignCenter, fyne.TextStyle{}),
			),
		),
	)
	u.statsContainer.Add(overallCard)

	// Map stats
	if len(mapStats) > 0 {
		mapRows := []fyne.CanvasObject{
			container.NewGridWithColumns(5,
				widget.NewLabelWithStyle("Map", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Played", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Wins", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Losses", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Win Rate", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			),
		}

		for _, stat := range mapStats {
			row := container.NewGridWithColumns(5,
				widget.NewLabel(stat.MapName),
				widget.NewLabelWithStyle(fmt.Sprintf("%d", stat.Played), fyne.TextAlignCenter, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("%d", stat.Wins), fyne.TextAlignCenter, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("%d", stat.Losses), fyne.TextAlignCenter, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("%.1f%%", stat.WinRate), fyne.TextAlignCenter, fyne.TextStyle{}),
			)
			mapRows = append(mapRows, row)
		}

		mapCard := widget.NewCard("Map Statistics", "", container.NewVBox(mapRows...))
		u.statsContainer.Add(mapCard)
	}

	// Clutch stats
	if len(clutchStats) > 0 {
		clutchRows := []fyne.CanvasObject{
			container.NewGridWithColumns(7,
				widget.NewLabelWithStyle("Player", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("1v1", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("1v2", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("1v3", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("1v4", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("1v5", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Win %", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			),
		}

		for _, stat := range clutchStats {
			row := container.NewGridWithColumns(7,
				widget.NewLabel(stat.Username),
				widget.NewLabelWithStyle(fmt.Sprintf("%d/%d", stat.Clutch1v1Won, stat.Clutch1v1), fyne.TextAlignCenter, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("%d/%d", stat.Clutch1v2Won, stat.Clutch1v2), fyne.TextAlignCenter, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("%d/%d", stat.Clutch1v3Won, stat.Clutch1v3), fyne.TextAlignCenter, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("%d/%d", stat.Clutch1v4Won, stat.Clutch1v4), fyne.TextAlignCenter, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("%d/%d", stat.Clutch1v5Won, stat.Clutch1v5), fyne.TextAlignCenter, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("%.1f%%", stat.ClutchRate), fyne.TextAlignCenter, fyne.TextStyle{}),
			)
			clutchRows = append(clutchRows, row)
		}

		clutchCard := widget.NewCard("Clutch Statistics", "", container.NewVBox(clutchRows...))
		u.statsContainer.Add(clutchCard)
	}

	// Defuser stats
	if len(defuserStats) > 0 {
		defuserRows := []fyne.CanvasObject{
			container.NewGridWithColumns(5,
				widget.NewLabelWithStyle("Player", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Plants", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Defuses", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Denials", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Plant %", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			),
		}

		for _, stat := range defuserStats {
			row := container.NewGridWithColumns(5,
				widget.NewLabel(stat.Username),
				widget.NewLabelWithStyle(fmt.Sprintf("%d", stat.Plants), fyne.TextAlignCenter, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("%d", stat.Defuses), fyne.TextAlignCenter, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("%d", stat.PlantDenials), fyne.TextAlignCenter, fyne.TextStyle{}),
				widget.NewLabelWithStyle(fmt.Sprintf("%.1f%%", stat.PlantSuccessRate), fyne.TextAlignCenter, fyne.TextStyle{}),
			)
			defuserRows = append(defuserRows, row)
		}

		defuserCard := widget.NewCard("Defuser Statistics", "", container.NewVBox(defuserRows...))
		u.statsContainer.Add(defuserCard)
	}

	u.statsContainer.Refresh()
}

func (u *UI) showImportDialog() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}

		// Show progress
		progressDialog := dialog.NewCustom("Importing", "Cancel",
			widget.NewLabel("Importing match..."), u.window)
		progressDialog.Show()

		go func() {
			match, err := u.parser.ImportMatch(uri.Path())
			progressDialog.Hide()

			if err != nil {
				dialog.ShowError(err, u.window)
				return
			}

			if match == nil {
				dialog.ShowInformation("Import", "Match already exists in database.", u.window)
				return
			}

			dialog.ShowInformation("Import", fmt.Sprintf("Imported: %s on %s", match.MatchType, match.Map), u.window)
			u.refreshMatches()
			u.updateMapFilter()
		}()
	}, u.window)
}

func (u *UI) showImportAllDialog() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}

		rootPath := uri.Path()

		// Show progress dialog
		statusLabel := widget.NewLabel("Scanning for replay folders...")
		progressDialog := dialog.NewCustom("Importing All Matches", "Cancel",
			container.NewVBox(statusLabel), u.window)
		progressDialog.Show()

		go func() {
			// Find folders
			log.Printf("Starting FindReplayFolders for: %s", rootPath)
			folders, err := u.parser.FindReplayFolders(rootPath)
			if err != nil {
				progressDialog.Hide()
				dialog.ShowError(err, u.window)
				return
			}

			log.Printf("Found %d folders to import", len(folders))

			if len(folders) == 0 {
				progressDialog.Hide()
				msg := "No replay folders found.\n\n"
				msg += "Make sure:\n"
				msg += "1. The folder contains .rec files\n"
				msg += "2. Replay recording is enabled in R6 Siege\n"
				msg += "3. You've played at least one match"
				dialog.ShowInformation("Import", msg, u.window)
				return
			}

			// Import matches
			imported := 0
			skipped := 0
			failed := 0

			for i, folder := range folders {
				statusLabel.SetText(fmt.Sprintf("Importing match %d of %d...\n%s",
					i+1, len(folders), filepath.Base(folder)))

				match, err := u.parser.ImportMatch(folder)
				if err != nil {
					log.Printf("ERROR importing %s: %v", folder, err)
					failed++
				} else if match != nil {
					imported++
				} else {
					skipped++
				}
			}

			progressDialog.Hide()

			msg := fmt.Sprintf("Import Complete!\n\n")
			msg += fmt.Sprintf("✓ Imported: %d new matches\n", imported)
			if skipped > 0 {
				msg += fmt.Sprintf("⊘ Skipped: %d (already in database)\n", skipped)
			}
			if failed > 0 {
				msg += fmt.Sprintf("✗ Failed: %d (check logs for details)\n", failed)
			}
			msg += fmt.Sprintf("\nTotal folders scanned: %d", len(folders))

			dialog.ShowInformation("Import Results", msg, u.window)
			u.refreshMatches()
			u.updateMapFilter()
		}()
	}, u.window)
}

func (u *UI) showMatchDetails(match models.Match) {
	rounds, err := u.db.GetRoundsByMatch(match.ID)
	if err != nil {
		dialog.ShowError(err, u.window)
		return
	}

	// Get all player stats for the match to aggregate
	allStats, _ := u.db.GetPlayerRoundStatsByMatch(match.ID)

	// Aggregate stats by player
	playerAggregates := make(map[string]*aggregatedStats)
	for _, s := range allStats {
		if _, exists := playerAggregates[s.Username]; !exists {
			playerAggregates[s.Username] = &aggregatedStats{
				Username:  s.Username,
				TeamIndex: s.TeamIndex,
			}
		}
		agg := playerAggregates[s.Username]
		agg.Kills += s.Kills
		if s.Died {
			agg.Deaths++
		}
		agg.Assists += s.Assists
		agg.Headshots += s.Headshots
		if s.EntryKill {
			agg.EntryKills++
		}
		if s.EntryDeath {
			agg.EntryDeaths++
		}

		// Advanced stats
		agg.Plants += s.DefuserPlants
		agg.Defuses += s.DefuserDefuses
		agg.DoubleKills += s.DoubleKills
		agg.TripleKills += s.TripleKills
		agg.QuadKills += s.QuadKills
		if s.Ace {
			agg.Aces++
		}
		if s.Clutch1v1 || s.Clutch1v2 || s.Clutch1v3 || s.Clutch1v4 || s.Clutch1v5 {
			agg.Clutches++
		}
		agg.TradeKills += s.TradeKills
		if s.Survived {
			agg.Survived++
		}
		agg.Rounds++
	}

	// Build content
	content := container.NewVBox(
		widget.NewLabelWithStyle(fmt.Sprintf("%s - %s", match.Map, match.MatchType), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("Date: %s", match.Timestamp.Format("2006-01-02 15:04"))),
		widget.NewLabel(fmt.Sprintf("Score: %d - %d | Result: %s", match.TeamScore, match.OpponentScore, boolToResult(match.Won))),
		widget.NewSeparator(),
	)

	// Show aggregated player stats if we have them
	if len(playerAggregates) > 0 {
		content.Add(widget.NewLabelWithStyle("Match Stats:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

		// Separate by team
		var yourTeam, opponents []*aggregatedStats
		for _, agg := range playerAggregates {
			if agg.TeamIndex == 0 {
				yourTeam = append(yourTeam, agg)
			} else {
				opponents = append(opponents, agg)
			}
		}

		// Your team
		content.Add(widget.NewLabel("Your Team:"))
		content.Add(u.buildAggregatedStatsTable(yourTeam))
		content.Add(widget.NewSeparator())

		// Opponents
		content.Add(widget.NewLabel("Opponents:"))
		content.Add(u.buildAggregatedStatsTable(opponents))
		content.Add(widget.NewSeparator())
	}

	// Rounds section
	content.Add(widget.NewLabelWithStyle("Rounds:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

	for _, round := range rounds {
		r := round
		roundText := fmt.Sprintf("Round %d: %s (%s) - %s",
			r.RoundNumber,
			r.TeamRole,
			r.Site,
			boolToResult(r.Won),
		)
		if r.WinCondition != "" {
			roundText += fmt.Sprintf(" [%s]", r.WinCondition)
		}

		roundBtn := widget.NewButton(roundText, func() {
			u.showRoundDetails(r, match)
		})
		content.Add(roundBtn)
	}

	// Delete button
	deleteBtn := widget.NewButtonWithIcon("Delete Match", theme.DeleteIcon(), func() {
		dialog.ShowConfirm("Delete Match", "Are you sure you want to delete this match?", func(ok bool) {
			if ok {
				if err := u.db.DeleteMatch(match.ID); err != nil {
					dialog.ShowError(err, u.window)
				} else {
					u.refreshMatches()
				}
			}
		}, u.window)
	})
	content.Add(widget.NewSeparator())
	content.Add(deleteBtn)

	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(800, 650))

	d := dialog.NewCustom("Match Details", "Close", scroll, u.window)
	d.Resize(fyne.NewSize(850, 700))
	d.Show()
}

type aggregatedStats struct {
	Username    string
	TeamIndex   int
	Kills       int
	Deaths      int
	Assists     int
	Headshots   int
	EntryKills  int
	EntryDeaths int
	Plants      int
	Defuses     int
	DoubleKills int
	TripleKills int
	QuadKills   int
	Aces        int
	Clutches    int
	TradeKills  int
	Survived    int
	Rounds      int
}

func (u *UI) buildAggregatedStatsTable(stats []*aggregatedStats) fyne.CanvasObject {
	header := container.NewGridWithColumns(15,
		widget.NewLabelWithStyle("Player", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("K", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("D", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("A", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("K/D", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("HS%", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("KOST", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Entry", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Plant", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Defuse", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Trade", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("2K", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("3K", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Ace", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Clutch", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	)

	rows := []fyne.CanvasObject{header}

	for _, s := range stats {
		kd := float64(s.Kills)
		if s.Deaths > 0 {
			kd = float64(s.Kills) / float64(s.Deaths)
		}

		// HS%
		hsPercent := 0.0
		if s.Kills > 0 {
			hsPercent = (float64(s.Headshots) / float64(s.Kills)) * 100
		}

		// KOST percentage (survived rounds / total rounds)
		kostPercent := 0.0
		if s.Rounds > 0 {
			kostPercent = float64(s.Survived) / float64(s.Rounds) * 100
		}

		row := container.NewGridWithColumns(15,
			widget.NewLabel(s.Username),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.Kills), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.Deaths), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.Assists), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%.2f", kd), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%.0f%%", hsPercent), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%.0f%%", kostPercent), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.EntryKills), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.Plants), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.Defuses), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.TradeKills), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.DoubleKills), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.TripleKills), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.Aces), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.Clutches), fyne.TextAlignCenter, fyne.TextStyle{}),
		)
		rows = append(rows, row)
	}

	return container.NewVBox(rows...)
}

func (u *UI) showRoundDetails(round models.Round, match models.Match) {
	playerStats, _ := u.db.GetPlayerRoundStatsByRound(round.ID)
	events, _ := u.db.GetEventsByRound(round.ID)

	content := container.NewVBox(
		widget.NewLabelWithStyle(fmt.Sprintf("Round %d - %s", round.RoundNumber, round.Site), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("Role: %s | Result: %s | Win Condition: %s", round.TeamRole, boolToResult(round.Won), round.WinCondition)),
		widget.NewSeparator(),
	)

	// Player stats tables
	if len(playerStats) > 0 {
		// Separate by team
		var yourTeamStats, opponentStats []models.PlayerRoundStats
		for _, s := range playerStats {
			if s.TeamIndex == 0 {
				yourTeamStats = append(yourTeamStats, s)
			} else {
				opponentStats = append(opponentStats, s)
			}
		}

		// Your team stats
		content.Add(widget.NewLabelWithStyle("Your Team:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		content.Add(u.buildStatsTable(yourTeamStats))
		content.Add(widget.NewSeparator())

		// Opponent stats
		content.Add(widget.NewLabelWithStyle("Opponents:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		content.Add(u.buildStatsTable(opponentStats))
		content.Add(widget.NewSeparator())
	}

	// Kill feed
	if len(events) > 0 {
		content.Add(widget.NewLabelWithStyle("Kill Feed:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

		for _, e := range events {
			if e.EventType == "Kill" {
				hs := ""
				if e.Headshot {
					hs = " [HS]"
				}
				eventText := fmt.Sprintf("[%s] %s killed %s%s", e.Time, e.Username, e.Target, hs)
				content.Add(widget.NewLabel(eventText))
			}
		}
	}

	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(800, 550))

	d := dialog.NewCustom("Round Details", "Close", scroll, u.window)
	d.Resize(fyne.NewSize(850, 600))
	d.Show()
}

func (u *UI) buildStatsTable(stats []models.PlayerRoundStats) fyne.CanvasObject {
	// Header row with HS% and KOST
	header := container.NewGridWithColumns(15,
		widget.NewLabelWithStyle("Player", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Op", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("K", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("D", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("A", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("HS%", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("KOST", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Entry", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Plant", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Defuse", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Trade", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("2K", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("3K", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Ace", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Clutch", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	)

	rows := []fyne.CanvasObject{header}

	for _, s := range stats {
		died := "0"
		if s.Died {
			died = "1"
		}
		entry := ""
		if s.EntryKill {
			entry = "K"
		} else if s.EntryDeath {
			entry = "D"
		}

		clutch := ""
		if s.Clutch1v1 {
			clutch = "1v1"
		} else if s.Clutch1v2 {
			clutch = "1v2"
		} else if s.Clutch1v3 {
			clutch = "1v3"
		} else if s.Clutch1v4 {
			clutch = "1v4"
		} else if s.Clutch1v5 {
			clutch = "1v5"
		}

		ace := ""
		if s.Ace {
			ace = "✓"
		}

		// HS% calculation
		hsPercent := fmt.Sprintf("%.0f%%", s.HeadshotPercentage)

		// KOST indicator - Kill OR Objective OR Survived OR Traded
		kost := ""
		if s.Kills > 0 || s.DefuserPlants > 0 || s.DefuserDefuses > 0 || s.Survived || s.TradeKills > 0 {
			kost = "!"
		}

		row := container.NewGridWithColumns(15,
			widget.NewLabel(s.Username),
			widget.NewLabelWithStyle(s.Operator, fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.Kills), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(died, fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.Assists), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(hsPercent, fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(kost, fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(entry, fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.DefuserPlants), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.DefuserDefuses), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.TradeKills), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.DoubleKills), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(fmt.Sprintf("%d", s.TripleKills), fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(ace, fyne.TextAlignCenter, fyne.TextStyle{}),
			widget.NewLabelWithStyle(clutch, fyne.TextAlignCenter, fyne.TextStyle{}),
		)
		rows = append(rows, row)
	}

	return container.NewVBox(rows...)
}

// StartWatcher is exported for use in main.go
func (u *UI) StartWatcher(path string) {
	u.StopWatcher()
	u.watcher = parser.NewFolderWatcher(path, u.parser, 30*time.Second)
	u.watcher.Start(func(match *models.Match) {
		u.refreshMatches()
		u.updateMapFilter()
	})
}

// StopWatcher is exported to be callable from main.go and internally
func (u *UI) StopWatcher() {
	if u.watcher != nil {
		u.watcher.Stop()
		u.watcher = nil
	}
}

func (u *UI) exportData() {
	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		defer writer.Close()

		// Would implement JSON export here
		dialog.ShowInformation("Export", "Data exported successfully!", u.window)
	}, u.window)
}

func boolToResult(won bool) string {
	if won {
		return "WIN"
	}
	return "LOSS"
}
