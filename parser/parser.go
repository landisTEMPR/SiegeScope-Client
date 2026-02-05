package parser

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"r6-replay-recorder/database"
	"r6-replay-recorder/models"

	"github.com/redraskal/r6-dissect/dissect"
)

// Parser handles reading .rec files and storing them in the database
type Parser struct {
	db *database.Database
}

// New creates a new parser instance
func New(db *database.Database) *Parser {
	return &Parser{db: db}
}

// ImportMatch imports a match folder (containing multiple .rec files) into the database
func (p *Parser) ImportMatch(matchFolderPath string) (*models.Match, error) {
	log.Printf("ImportMatch called with path: %s", matchFolderPath)

	// Check if path exists and is a directory
	info, err := os.Stat(matchFolderPath)
	if err != nil {
		log.Printf("ERROR: Path does not exist: %v", err)
		return nil, err
	}
	if !info.IsDir() {
		log.Printf("ERROR: Path is not a directory: %s", matchFolderPath)
		// If it's a file, try to import as single round
		if filepath.Ext(matchFolderPath) == ".rec" {
			return p.ImportSingleRound(matchFolderPath)
		}
		return nil, err
	}

	// Open the match folder
	folder, err := os.Open(matchFolderPath)
	if err != nil {
		log.Printf("ERROR: Cannot open folder: %v", err)
		return nil, err
	}
	defer folder.Close()

	// Check if folder contains .rec files
	entries, err := os.ReadDir(matchFolderPath)
	if err != nil {
		log.Printf("ERROR: Cannot read directory: %v", err)
		return nil, err
	}

	hasRecFiles := false
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".rec" {
			hasRecFiles = true
			log.Printf("Found .rec file: %s", entry.Name())
			break
		}
	}

	if !hasRecFiles {
		log.Printf("WARNING: No .rec files found in folder: %s", matchFolderPath)
		return nil, nil
	}

	// Create match reader
	matchReader, err := dissect.NewMatchReader(folder)
	if err != nil {
		log.Printf("ERROR: Cannot create match reader: %v", err)
		return nil, err
	}

	// Get the first round to access header info
	firstRound, err := matchReader.FirstRound()
	if err != nil {
		log.Printf("ERROR: Cannot get first round: %v", err)
		return nil, err
	}

	// Read the first round fully
	if err := firstRound.Read(); !dissect.Ok(err) {
		log.Printf("ERROR: Cannot read first round: %v", err)
		return nil, err
	}

	header := firstRound.Header

	// Check if match already exists
	exists, err := p.db.MatchExists(header.MatchID)
	if err != nil {
		log.Printf("ERROR: Database error checking match existence: %v", err)
		return nil, err
	}
	if exists {
		log.Printf("INFO: Match %s already exists in database", header.MatchID)
		return nil, nil // Already imported
	}

	// Calculate final scores and winner from all rounds
	teamScore, opponentScore, won := p.calculateFinalScore(matchFolderPath)

	log.Printf("Creating match record for MatchID: %s", header.MatchID)

	// Create match record
	match := &models.Match{
		MatchID:         header.MatchID,
		GameVersion:     header.GameVersion,
		CodeVersion:     header.CodeVersion,
		Timestamp:       header.Timestamp,
		MatchType:       matchTypeToString(header.MatchType),
		GameMode:        header.GameMode.String(),
		Map:             header.Map.String(),
		RecordingPlayer: header.RecordingPlayer().Username,
		ProfileID:       header.RecordingProfileID,
		TeamScore:       teamScore,
		OpponentScore:   opponentScore,
		Won:             won,
		RoundsPlayed:    matchReader.NumRounds(),
		FilePath:        matchFolderPath,
	}

	// Insert match
	matchDBID, err := p.db.InsertMatch(match)
	if err != nil {
		log.Printf("ERROR: Cannot insert match: %v", err)
		return nil, err
	}
	match.ID = matchDBID
	log.Printf("Successfully inserted match with DB ID: %d", matchDBID)

	// Import all rounds
	if err := p.importRounds(matchFolderPath, matchDBID); err != nil {
		// Rollback by deleting the match
		log.Printf("ERROR: Failed to import rounds, rolling back: %v", err)
		p.db.DeleteMatch(matchDBID)
		return nil, err
	}

	log.Printf("Successfully imported match: %s", header.MatchID)
	return match, nil
}

// ImportSingleRound imports just a single .rec file
func (p *Parser) ImportSingleRound(recFilePath string) (*models.Match, error) {
	log.Printf("ImportSingleRound called with path: %s", recFilePath)

	f, err := os.Open(recFilePath)
	if err != nil {
		log.Printf("ERROR: Cannot open file: %v", err)
		return nil, err
	}
	defer f.Close()

	reader, err := dissect.NewReader(f)
	if err != nil {
		log.Printf("ERROR: Cannot create reader: %v", err)
		return nil, err
	}

	if err := reader.Read(); !dissect.Ok(err) {
		log.Printf("ERROR: Cannot read replay: %v", err)
		return nil, err
	}

	header := reader.Header

	// Check if match already exists
	exists, err := p.db.MatchExists(header.MatchID)
	if err != nil {
		return nil, err
	}
	if exists {
		log.Printf("INFO: Match %s already exists in database", header.MatchID)
		return nil, nil
	}

	// Determine winner from teams
	var teamScore, opponentScore int
	var won bool
	teamScore = header.Teams[0].Score
	won = header.Teams[0].Won
	opponentScore = header.Teams[1].Score

	// Create match record
	match := &models.Match{
		MatchID:         header.MatchID,
		GameVersion:     header.GameVersion,
		CodeVersion:     header.CodeVersion,
		Timestamp:       header.Timestamp,
		MatchType:       matchTypeToString(header.MatchType),
		GameMode:        header.GameMode.String(),
		Map:             header.Map.String(),
		RecordingPlayer: header.RecordingPlayer().Username,
		ProfileID:       header.RecordingProfileID,
		TeamScore:       teamScore,
		OpponentScore:   opponentScore,
		Won:             won,
		RoundsPlayed:    1,
		FilePath:        recFilePath,
	}

	matchDBID, err := p.db.InsertMatch(match)
	if err != nil {
		return nil, err
	}
	match.ID = matchDBID

	// Import the single round
	if err := p.importRoundFromReader(reader, matchDBID, 1); err != nil {
		p.db.DeleteMatch(matchDBID)
		return nil, err
	}

	log.Printf("Successfully imported single round match: %s", header.MatchID)
	return match, nil
}

func (p *Parser) calculateFinalScore(matchFolderPath string) (teamScore, opponentScore int, won bool) {
	folder, err := os.Open(matchFolderPath)
	if err != nil {
		return 0, 0, false
	}
	defer folder.Close()

	matchReader, err := dissect.NewMatchReader(folder)
	if err != nil {
		return 0, 0, false
	}

	// Get the last round for final scores
	lastRound, err := matchReader.LastRound()
	if err != nil {
		return 0, 0, false
	}

	if err := lastRound.Read(); !dissect.Ok(err) {
		return 0, 0, false
	}

	teamScore = lastRound.Header.Teams[0].Score
	won = lastRound.Header.Teams[0].Won
	opponentScore = lastRound.Header.Teams[1].Score

	return teamScore, opponentScore, won
}

func (p *Parser) importRounds(matchFolderPath string, matchDBID int64) error {
	// Read directory entries directly instead of using ListReplayFiles
	entries, err := os.ReadDir(matchFolderPath)
	if err != nil {
		log.Printf("ERROR reading directory %s: %v", matchFolderPath, err)
		return err
	}

	// Find all .rec files
	var recFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".rec" {
			recFiles = append(recFiles, entry.Name())
		}
	}

	log.Printf("Found %d .rec files in %s", len(recFiles), matchFolderPath)

	// Sort to ensure round order
	sort.Strings(recFiles)

	for i, recFile := range recFiles {
		recPath := filepath.Join(matchFolderPath, recFile)
		log.Printf("Processing round %d: %s", i+1, recPath)

		f, err := os.Open(recPath)
		if err != nil {
			log.Printf("ERROR opening file: %v", err)
			continue
		}

		reader, err := dissect.NewReader(f)
		if err != nil {
			log.Printf("ERROR creating reader: %v", err)
			f.Close()
			continue
		}

		if err := reader.Read(); !dissect.Ok(err) {
			log.Printf("ERROR reading replay: %v", err)
			f.Close()
			continue
		}

		log.Printf("Successfully read round %d, importing...", i+1)
		if err := p.importRoundFromReader(reader, matchDBID, i+1); err != nil {
			log.Printf("ERROR importing round: %v", err)
		}
		f.Close()
	}

	return nil
}

func (p *Parser) importRoundFromReader(reader *dissect.Reader, matchDBID int64, roundNum int) error {
	header := reader.Header

	// Determine team role and win status
	var teamRole string
	var won bool
	var winCondition string
	var teamScore, opponentScore int

	team0 := header.Teams[0]
	team1 := header.Teams[1]

	teamRole = string(team0.Role)
	won = team0.Won
	if team0.Won {
		winCondition = string(team0.WinCondition)
	} else {
		winCondition = string(team1.WinCondition)
	}
	teamScore = team0.Score
	opponentScore = team1.Score

	// Create round record
	round := &models.Round{
		MatchID:       matchDBID,
		RoundNumber:   roundNum,
		Site:          header.Site,
		TeamRole:      teamRole,
		Won:           won,
		WinCondition:  winCondition,
		TeamScore:     teamScore,
		OpponentScore: opponentScore,
	}

	roundDBID, err := p.db.InsertRound(round)
	if err != nil {
		return err
	}

	// Import players
	for _, player := range header.Players {
		p.db.InsertPlayer(&models.Player{
			RoundID:   roundDBID,
			MatchID:   matchDBID,
			ProfileID: player.ProfileID,
			Username:  player.Username,
			TeamIndex: player.TeamIndex,
			Operator:  player.Operator.String(),
			Spawn:     player.Spawn,
		})
	}

	// Import match events (kills, plants, etc.)
	for _, event := range reader.MatchFeedback {
		headshot := false
		if event.Headshot != nil {
			headshot = *event.Headshot
		}

		p.db.InsertEvent(&models.MatchEvent{
			RoundID:       roundDBID,
			MatchID:       matchDBID,
			EventType:     event.Type.String(),
			Time:          event.Time,
			TimeInSeconds: int(event.TimeInSeconds),
			Username:      event.Username,
			Target:        event.Target,
			Headshot:      headshot,
		})
	}

	// Get all match events
	events := reader.MatchFeedback

	// Analyze events for advanced stats
	playerAdvancedStats := p.calculateAdvancedStats(events, header.Players, reader.PlayerStats())

	// Import player round stats with advanced stats
	for username, advStats := range playerAdvancedStats {
		// Find base stats
		var baseStats *dissect.PlayerRoundStats
		for _, stat := range reader.PlayerStats() {
			if stat.Username == username {
				baseStats = &stat
				break
			}
		}

		if baseStats == nil {
			continue
		}

		// Find operator and team
		operator := ""
		teamIndex := 0
		for _, player := range header.Players {
			if player.Username == username {
				operator = player.Operator.String()
				teamIndex = player.TeamIndex
				break
			}
		}

		p.db.InsertPlayerRoundStats(&models.PlayerRoundStats{
			RoundID:            roundDBID,
			MatchID:            matchDBID,
			Username:           username,
			TeamIndex:          teamIndex,
			Operator:           operator,
			Kills:              baseStats.Kills,
			Died:               baseStats.Died,
			Assists:            baseStats.Assists,
			Headshots:          baseStats.Headshots,
			HeadshotPercentage: baseStats.HeadshotPercentage,
			EntryKill:          advStats.EntryKill,
			EntryDeath:         advStats.EntryDeath,

			// Advanced stats
			DefuserPlants:  advStats.DefuserPlants,
			DefuserDefuses: advStats.DefuserDefuses,
			DefuserPickups: advStats.DefuserPickups,
			PlantDenials:   advStats.PlantDenials,
			ClutchAttempts: advStats.ClutchAttempts,
			ClutchWins:     advStats.ClutchWins,
			Clutch1v1:      advStats.Clutch1v1,
			Clutch1v2:      advStats.Clutch1v2,
			Clutch1v3:      advStats.Clutch1v3,
			Clutch1v4:      advStats.Clutch1v4,
			Clutch1v5:      advStats.Clutch1v5,
			DoubleKills:    advStats.DoubleKills,
			TripleKills:    advStats.TripleKills,
			QuadKills:      advStats.QuadKills,
			Ace:            advStats.Ace,
			TradeKills:     advStats.TradeKills,
			TradeDeaths:    advStats.TradeDeaths,
			SurvivalTime:   advStats.SurvivalTime,
			Survived:       advStats.Survived,
		})
	}

	return nil
}

type advancedPlayerStats struct {
	Username       string
	EntryKill      bool
	EntryDeath     bool
	DefuserPlants  int
	DefuserDefuses int
	DefuserPickups int
	PlantDenials   int
	ClutchAttempts int
	ClutchWins     int
	Clutch1v1      bool
	Clutch1v2      bool
	Clutch1v3      bool
	Clutch1v4      bool
	Clutch1v5      bool
	DoubleKills    int
	TripleKills    int
	QuadKills      int
	Ace            bool
	TradeKills     int
	TradeDeaths    int
	SurvivalTime   float64
	Survived       bool
	KOST           bool
}

func (p *Parser) calculateAdvancedStats(events []dissect.MatchUpdate, players []dissect.Player, baseStats []dissect.PlayerRoundStats) map[string]*advancedPlayerStats {
	stats := make(map[string]*advancedPlayerStats)

	// Initialize stats for all players
	for _, player := range players {
		stats[player.Username] = &advancedPlayerStats{
			Username: player.Username,
		}
	}

	// Track alive players for clutch detection
	alivePlayers := make(map[int][]string) // teamIndex -> []username
	for _, player := range players {
		alivePlayers[player.TeamIndex] = append(alivePlayers[player.TeamIndex], player.Username)
	}

	// Track kills per player for multi-kill detection
	killCounts := make(map[string]int)
	killTimestamps := make(map[string][]float64)

	// Track recent deaths for trade detection (within 3 seconds)
	type deathEvent struct {
		victim string
		killer string
		time   float64
	}
	recentDeaths := []deathEvent{}

	// Process events chronologically
	for _, event := range events {
		eventType := event.Type.String()

		switch eventType {
		case "Kill":
			killer := event.Username
			victim := event.Target
			eventTime := event.TimeInSeconds

			// Remove victim from alive players
			for team := range alivePlayers {
				for i, player := range alivePlayers[team] {
					if player == victim {
						alivePlayers[team] = append(alivePlayers[team][:i], alivePlayers[team][i+1:]...)
						break
					}
				}
			}

			// Track kill for multi-kill detection
			killCounts[killer]++
			killTimestamps[killer] = append(killTimestamps[killer], eventTime)

			// Check for trades (kill within 3 seconds of teammate death)
			for i := len(recentDeaths) - 1; i >= 0; i-- {
				rd := recentDeaths[i]
				timeDiff := eventTime - rd.time

				if timeDiff > 3.0 {
					// Remove old deaths
					recentDeaths = recentDeaths[i+1:]
					break
				}

				// Check if this kill avenges a teammate
				if rd.killer == victim && killer != victim {
					if stats[killer] != nil {
						stats[killer].TradeKills++
					}
					if stats[victim] != nil {
						stats[victim].TradeDeaths++
					}
				}
			}

			// Add to recent deaths
			recentDeaths = append(recentDeaths, deathEvent{
				victim: victim,
				killer: killer,
				time:   eventTime,
			})

			// Check for clutch situations
			killerTeam := p.getPlayerTeam(killer, players)
			if killerTeam != -1 {
				aliveOnTeam := len(alivePlayers[killerTeam])
				enemyTeam := 1 - killerTeam
				aliveEnemies := len(alivePlayers[enemyTeam])

				if aliveOnTeam == 1 && aliveEnemies > 0 {
					// Player is in a 1vX situation
					if stats[killer] != nil {
						stats[killer].ClutchAttempts++

						if aliveEnemies == 1 {
							stats[killer].Clutch1v1 = true
						} else if aliveEnemies == 2 {
							stats[killer].Clutch1v2 = true
						} else if aliveEnemies == 3 {
							stats[killer].Clutch1v3 = true
						} else if aliveEnemies == 4 {
							stats[killer].Clutch1v4 = true
						} else if aliveEnemies == 5 {
							stats[killer].Clutch1v5 = true
						}

						// Check if clutch was won (no more enemies alive after this kill)
						if aliveEnemies == 1 {
							stats[killer].ClutchWins++
						}
					}
				}
			}

		case "DefuserPlantStart", "DefuserPlantComplete":
			if event.Username != "" && stats[event.Username] != nil {
				stats[event.Username].DefuserPlants++
			}

		case "DefuserDisableStart", "DefuserDisableComplete":
			if event.Username != "" && stats[event.Username] != nil {
				stats[event.Username].DefuserDefuses++
			}

		case "DefuserPickedUp":
			if event.Username != "" && stats[event.Username] != nil {
				stats[event.Username].DefuserPickups++
			}
		}
	}

	// Calculate multi-kills (kills within 10 seconds of each other)
	for username, timestamps := range killTimestamps {
		if len(timestamps) < 2 {
			continue
		}

		consecutiveKills := 1
		for i := 1; i < len(timestamps); i++ {
			if timestamps[i]-timestamps[i-1] <= 10.0 {
				consecutiveKills++
			} else {
				p.recordMultiKill(stats[username], consecutiveKills)
				consecutiveKills = 1
			}
		}
		p.recordMultiKill(stats[username], consecutiveKills)

		// Check for ace
		if killCounts[username] >= 5 && stats[username] != nil {
			stats[username].Ace = true
		}
	}

	// Calculate entry kills/deaths
	if len(events) > 0 {
		var firstKill *dissect.MatchUpdate
		for _, event := range events {
			if event.Type.String() == "Kill" {
				firstKill = &event
				break
			}
		}

		if firstKill != nil {
			if stats[firstKill.Username] != nil {
				stats[firstKill.Username].EntryKill = true
			}
			if stats[firstKill.Target] != nil {
				stats[firstKill.Target].EntryDeath = true
			}
		}
	}

	// Calculate survival stats
	for _, baseStat := range baseStats {
		if stat, exists := stats[baseStat.Username]; exists {
			stat.Survived = !baseStat.Died
			// Survival time would need to be calculated from death event timestamp
		}
	}

	for username, stat := range stats {
		// Find base stats
		for _, baseStat := range baseStats {
			if baseStat.Username == username {
				// KOST = Kill OR Objective (plant/defuse) OR Survived OR Traded
				hasKill := baseStat.Kills > 0
				hasObjective := stat.DefuserPlants > 0 || stat.DefuserDefuses > 0
				survived := !baseStat.Died
				traded := stat.TradeDeaths > 0 // If player was traded, they contributed

				stat.KOST = hasKill || hasObjective || survived || traded
				stat.Survived = survived
				break
			}
		}
	}

	return stats

}

func (p *Parser) recordMultiKill(stat *advancedPlayerStats, count int) {
	if stat == nil {
		return
	}
	if count >= 5 {
		stat.Ace = true
		stat.QuadKills++
		stat.TripleKills++
		stat.DoubleKills++
	} else if count == 4 {
		stat.QuadKills++
		stat.TripleKills++
		stat.DoubleKills++
	} else if count == 3 {
		stat.TripleKills++
		stat.DoubleKills++
	} else if count == 2 {
		stat.DoubleKills++
	}
}

func (p *Parser) getPlayerTeam(username string, players []dissect.Player) int {
	for _, player := range players {
		if player.Username == username {
			return player.TeamIndex
		}
	}
	return -1
}

// matchTypeToString converts MatchType to a readable string
func matchTypeToString(mt dissect.MatchType) string {
	switch mt {
	case dissect.QuickMatch:
		return "QuickMatch"
	case dissect.Ranked:
		return "Ranked"
	case dissect.CustomGameLocal:
		return "CustomGameLocal"
	case dissect.CustomGameOnline:
		return "CustomGameOnline"
	case dissect.Standard:
		return "Standard"
	default:
		return "Unknown"
	}
}

// FindReplayFolders scans a directory for R6 replay match folders
// IMPROVED VERSION: More flexible folder detection with recursive search
func (p *Parser) FindReplayFolders(rootPath string) ([]string, error) {
	log.Printf("FindReplayFolders called with rootPath: %s", rootPath)

	var folders []string

	// Check if root path exists
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		log.Printf("ERROR: Root path does not exist: %s", rootPath)
		return nil, err
	}

	// Walk the directory tree
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("ERROR walking path %s: %v", path, err)
			return nil // Continue walking despite errors
		}

		// Skip the root directory itself
		if path == rootPath {
			return nil
		}

		// Only process directories
		if !info.IsDir() {
			return nil
		}

		// Check if this directory contains .rec files
		hasRecFiles, err := containsRecFiles(path)
		if err != nil {
			log.Printf("ERROR checking for .rec files in %s: %v", path, err)
			return nil
		}

		if hasRecFiles {
			log.Printf("Found replay folder: %s", path)
			folders = append(folders, path)
			// Don't descend into folders that contain .rec files
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		log.Printf("ERROR during directory walk: %v", err)
		return nil, err
	}

	log.Printf("Found %d replay folders in total", len(folders))
	return folders, nil
}

// containsRecFiles checks if a directory contains any .rec files
func containsRecFiles(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".rec" {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetDefaultReplayPath returns the default R6 replay location for the current OS
func GetDefaultReplayPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Default Windows path for R6 replays
	return filepath.Join(home, "Documents", "My Games", "Rainbow Six - Siege", "replays")
}

// WatchFolder watches a folder for new replay files (simplified polling version)
type FolderWatcher struct {
	path     string
	parser   *Parser
	stopChan chan struct{}
	interval time.Duration
}

// NewFolderWatcher creates a new folder watcher
func NewFolderWatcher(path string, parser *Parser, interval time.Duration) *FolderWatcher {
	return &FolderWatcher{
		path:     path,
		parser:   parser,
		stopChan: make(chan struct{}),
		interval: interval,
	}
}

// Start begins watching for new replays
func (fw *FolderWatcher) Start(onNewMatch func(*models.Match)) {
	go func() {
		ticker := time.NewTicker(fw.interval)
		defer ticker.Stop()

		for {
			select {
			case <-fw.stopChan:
				return
			case <-ticker.C:
				folders, err := fw.parser.FindReplayFolders(fw.path)
				if err != nil {
					continue
				}

				for _, folder := range folders {
					match, err := fw.parser.ImportMatch(folder)
					if err == nil && match != nil {
						onNewMatch(match)
					}
				}
			}
		}
	}()
}

// Stop stops watching
func (fw *FolderWatcher) Stop() {
	close(fw.stopChan)
}
