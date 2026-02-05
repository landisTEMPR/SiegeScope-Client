package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"r6-replay-recorder/models"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

// GetAppDataPath returns the appropriate application data directory for the OS
func GetAppDataPath() (string, error) {
	var basePath string

	switch runtime.GOOS {
	case "windows":
		basePath = os.Getenv("APPDATA")
		if basePath == "" {
			basePath = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		basePath = filepath.Join(home, "Library", "Application Support")
	default: // Linux and others
		basePath = os.Getenv("XDG_CONFIG_HOME")
		if basePath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			basePath = filepath.Join(home, ".config")
		}
	}

	appPath := filepath.Join(basePath, "R6ReplayRecorder")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(appPath, 0755); err != nil {
		return "", err
	}

	return appPath, nil
}

// New creates and initializes the database
func New() (*Database, error) {
	appPath, err := GetAppDataPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get app data path: %w", err)
	}

	dbPath := filepath.Join(appPath, "replays.db")

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	database := &Database{db: db}

	if err := database.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return database, nil
}

func (d *Database) initialize() error {
	schema := `
	CREATE TABLE IF NOT EXISTS matches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id TEXT UNIQUE NOT NULL,
		game_version TEXT,
		code_version INTEGER,
		timestamp DATETIME,
		match_type TEXT,
		game_mode TEXT,
		map TEXT,
		recording_player TEXT,
		profile_id TEXT,
		team_score INTEGER,
		opponent_score INTEGER,
		won BOOLEAN,
		rounds_played INTEGER,
		imported_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		file_path TEXT
	);

	CREATE TABLE IF NOT EXISTS rounds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id INTEGER NOT NULL,
		round_number INTEGER,
		site TEXT,
		team_role TEXT,
		won BOOLEAN,
		win_condition TEXT,
		team_score INTEGER,
		opponent_score INTEGER,
		FOREIGN KEY (match_id) REFERENCES matches(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS players (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		round_id INTEGER NOT NULL,
		match_id INTEGER NOT NULL,
		profile_id TEXT,
		username TEXT,
		team_index INTEGER,
		operator TEXT,
		spawn TEXT,
		FOREIGN KEY (round_id) REFERENCES rounds(id) ON DELETE CASCADE,
		FOREIGN KEY (match_id) REFERENCES matches(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS match_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		round_id INTEGER NOT NULL,
		match_id INTEGER NOT NULL,
		event_type TEXT,
		time TEXT,
		time_in_seconds INTEGER,
		username TEXT,
		target TEXT,
		headshot BOOLEAN,
		message TEXT,
		FOREIGN KEY (round_id) REFERENCES rounds(id) ON DELETE CASCADE,
		FOREIGN KEY (match_id) REFERENCES matches(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS player_round_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		round_id INTEGER NOT NULL,
		match_id INTEGER NOT NULL,
		username TEXT,
		team_index INTEGER,
		operator TEXT,
		kills INTEGER DEFAULT 0,
		died BOOLEAN DEFAULT 0,
		assists INTEGER DEFAULT 0,
		headshots INTEGER DEFAULT 0,
		headshot_percentage REAL DEFAULT 0,
		entry_kill BOOLEAN DEFAULT 0,
		entry_death BOOLEAN DEFAULT 0,
		defuser_plants INTEGER DEFAULT 0,
		defuser_defuses INTEGER DEFAULT 0,
		defuser_pickups INTEGER DEFAULT 0,
		plant_denials INTEGER DEFAULT 0,
		clutch_attempts INTEGER DEFAULT 0,
		clutch_wins INTEGER DEFAULT 0,
		clutch_1v1 BOOLEAN DEFAULT 0,
		clutch_1v2 BOOLEAN DEFAULT 0,
		clutch_1v3 BOOLEAN DEFAULT 0,
		clutch_1v4 BOOLEAN DEFAULT 0,
		clutch_1v5 BOOLEAN DEFAULT 0,
		double_kills INTEGER DEFAULT 0,
		triple_kills INTEGER DEFAULT 0,
		quad_kills INTEGER DEFAULT 0,
		ace BOOLEAN DEFAULT 0,
		trade_kills INTEGER DEFAULT 0,
		trade_deaths INTEGER DEFAULT 0,
		survival_time REAL DEFAULT 0,
		survived BOOLEAN DEFAULT 0,
		FOREIGN KEY (round_id) REFERENCES rounds(id) ON DELETE CASCADE,
		FOREIGN KEY (match_id) REFERENCES matches(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		replay_folder TEXT,
		auto_import BOOLEAN DEFAULT 0,
		theme TEXT DEFAULT 'dark',
		start_minimized BOOLEAN DEFAULT 0,
		start_with_system BOOLEAN DEFAULT 0,
		api_key TEXT
	);

	-- Insert default settings if not exists
	INSERT OR IGNORE INTO settings (id, theme) VALUES (1, 'dark');

	-- Create indexes for performance
	CREATE INDEX IF NOT EXISTS idx_matches_timestamp ON matches(timestamp);
	CREATE INDEX IF NOT EXISTS idx_matches_map ON matches(map);
	CREATE INDEX IF NOT EXISTS idx_matches_match_type ON matches(match_type);
	CREATE INDEX IF NOT EXISTS idx_rounds_match_id ON rounds(match_id);
	CREATE INDEX IF NOT EXISTS idx_players_match_id ON players(match_id);
	CREATE INDEX IF NOT EXISTS idx_players_username ON players(username);
	CREATE INDEX IF NOT EXISTS idx_match_events_round_id ON match_events(round_id);
	CREATE INDEX IF NOT EXISTS idx_player_round_stats_round_id ON player_round_stats(round_id);
	CREATE INDEX IF NOT EXISTS idx_player_round_stats_match_id ON player_round_stats(match_id);
	`

	if _, err := d.db.Exec(schema); err != nil {
		return err
	}

	// Migration: Add new columns if they don't exist (for existing databases)
	migrations := []string{
		"ALTER TABLE player_round_stats ADD COLUMN defuser_plants INTEGER DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN defuser_defuses INTEGER DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN defuser_pickups INTEGER DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN plant_denials INTEGER DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN clutch_attempts INTEGER DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN clutch_wins INTEGER DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN clutch_1v1 BOOLEAN DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN clutch_1v2 BOOLEAN DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN clutch_1v3 BOOLEAN DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN clutch_1v4 BOOLEAN DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN clutch_1v5 BOOLEAN DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN double_kills INTEGER DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN triple_kills INTEGER DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN quad_kills INTEGER DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN ace BOOLEAN DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN trade_kills INTEGER DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN trade_deaths INTEGER DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN survival_time REAL DEFAULT 0",
		"ALTER TABLE player_round_stats ADD COLUMN survived BOOLEAN DEFAULT 0",
	}

	for _, migration := range migrations {
		_, _ = d.db.Exec(migration) // Ignore errors (column may already exist)
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// MatchExists checks if a match with the given ID already exists
func (d *Database) MatchExists(matchID string) (bool, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM matches WHERE match_id = ?", matchID).Scan(&count)
	return count > 0, err
}

// InsertMatch inserts a new match and returns its database ID
func (d *Database) InsertMatch(match *models.Match) (int64, error) {
	result, err := d.db.Exec(`
		INSERT INTO matches (
			match_id, game_version, code_version, timestamp, match_type,
			game_mode, map, recording_player, profile_id, team_score,
			opponent_score, won, rounds_played, imported_at, file_path
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		match.MatchID, match.GameVersion, match.CodeVersion, match.Timestamp,
		match.MatchType, match.GameMode, match.Map, match.RecordingPlayer,
		match.ProfileID, match.TeamScore, match.OpponentScore, match.Won,
		match.RoundsPlayed, time.Now(), match.FilePath,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// InsertRound inserts a new round
func (d *Database) InsertRound(round *models.Round) (int64, error) {
	result, err := d.db.Exec(`
		INSERT INTO rounds (
			match_id, round_number, site, team_role, won,
			win_condition, team_score, opponent_score
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		round.MatchID, round.RoundNumber, round.Site, round.TeamRole,
		round.Won, round.WinCondition, round.TeamScore, round.OpponentScore,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// InsertPlayer inserts a player record
func (d *Database) InsertPlayer(player *models.Player) error {
	_, err := d.db.Exec(`
		INSERT INTO players (
			round_id, match_id, profile_id, username, team_index, operator, spawn
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		player.RoundID, player.MatchID, player.ProfileID, player.Username,
		player.TeamIndex, player.Operator, player.Spawn,
	)
	return err
}

// InsertEvent inserts a match event
func (d *Database) InsertEvent(event *models.MatchEvent) error {
	_, err := d.db.Exec(`
		INSERT INTO match_events (
			round_id, match_id, event_type, time, time_in_seconds,
			username, target, headshot, message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.RoundID, event.MatchID, event.EventType, event.Time,
		event.TimeInSeconds, event.Username, event.Target,
		event.Headshot, event.Message,
	)
	return err
}

// InsertPlayerRoundStats inserts player stats for a round
func (d *Database) InsertPlayerRoundStats(stats *models.PlayerRoundStats) error {
	_, err := d.db.Exec(`
		INSERT INTO player_round_stats (
			round_id, match_id, username, team_index, operator,
			kills, died, assists, headshots, headshot_percentage,
			entry_kill, entry_death,
			defuser_plants, defuser_defuses, defuser_pickups, plant_denials,
			clutch_attempts, clutch_wins, clutch_1v1, clutch_1v2, clutch_1v3, clutch_1v4, clutch_1v5,
			double_kills, triple_kills, quad_kills, ace,
			trade_kills, trade_deaths, survival_time, survived
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		stats.RoundID, stats.MatchID, stats.Username, stats.TeamIndex,
		stats.Operator, stats.Kills, stats.Died, stats.Assists,
		stats.Headshots, stats.HeadshotPercentage, stats.EntryKill, stats.EntryDeath,
		stats.DefuserPlants, stats.DefuserDefuses, stats.DefuserPickups, stats.PlantDenials,
		stats.ClutchAttempts, stats.ClutchWins, stats.Clutch1v1, stats.Clutch1v2, stats.Clutch1v3, stats.Clutch1v4, stats.Clutch1v5,
		stats.DoubleKills, stats.TripleKills, stats.QuadKills, stats.Ace,
		stats.TradeKills, stats.TradeDeaths, stats.SurvivalTime, stats.Survived,
	)
	return err
}

// GetPlayerRoundStatsByRound returns all player stats for a round
func (d *Database) GetPlayerRoundStatsByRound(roundID int64) ([]models.PlayerRoundStats, error) {
	rows, err := d.db.Query(`
		SELECT id, round_id, match_id, username, team_index, operator,
		       kills, died, assists, headshots, headshot_percentage,
		       entry_kill, entry_death,
		       defuser_plants, defuser_defuses, defuser_pickups, plant_denials,
		       clutch_attempts, clutch_wins, clutch_1v1, clutch_1v2, clutch_1v3, clutch_1v4, clutch_1v5,
		       double_kills, triple_kills, quad_kills, ace,
		       trade_kills, trade_deaths, survival_time, survived
		FROM player_round_stats WHERE round_id = ? ORDER BY team_index, kills DESC
	`, roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.PlayerRoundStats
	for rows.Next() {
		var s models.PlayerRoundStats
		err := rows.Scan(
			&s.ID, &s.RoundID, &s.MatchID, &s.Username, &s.TeamIndex,
			&s.Operator, &s.Kills, &s.Died, &s.Assists, &s.Headshots,
			&s.HeadshotPercentage, &s.EntryKill, &s.EntryDeath,
			&s.DefuserPlants, &s.DefuserDefuses, &s.DefuserPickups, &s.PlantDenials,
			&s.ClutchAttempts, &s.ClutchWins, &s.Clutch1v1, &s.Clutch1v2, &s.Clutch1v3, &s.Clutch1v4, &s.Clutch1v5,
			&s.DoubleKills, &s.TripleKills, &s.QuadKills, &s.Ace,
			&s.TradeKills, &s.TradeDeaths, &s.SurvivalTime, &s.Survived,
		)
		if err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// GetPlayerRoundStatsByMatch returns all player stats for a match (all rounds)
func (d *Database) GetPlayerRoundStatsByMatch(matchID int64) ([]models.PlayerRoundStats, error) {
	rows, err := d.db.Query(`
		SELECT id, round_id, match_id, username, team_index, operator,
		       kills, died, assists, headshots, headshot_percentage,
		       entry_kill, entry_death,
		       defuser_plants, defuser_defuses, defuser_pickups, plant_denials,
		       clutch_attempts, clutch_wins, clutch_1v1, clutch_1v2, clutch_1v3, clutch_1v4, clutch_1v5,
		       double_kills, triple_kills, quad_kills, ace,
		       trade_kills, trade_deaths, survival_time, survived
		FROM player_round_stats WHERE match_id = ? ORDER BY round_id, team_index, kills DESC
	`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.PlayerRoundStats
	for rows.Next() {
		var s models.PlayerRoundStats
		err := rows.Scan(
			&s.ID, &s.RoundID, &s.MatchID, &s.Username, &s.TeamIndex,
			&s.Operator, &s.Kills, &s.Died, &s.Assists, &s.Headshots,
			&s.HeadshotPercentage, &s.EntryKill, &s.EntryDeath,
			&s.DefuserPlants, &s.DefuserDefuses, &s.DefuserPickups, &s.PlantDenials,
			&s.ClutchAttempts, &s.ClutchWins, &s.Clutch1v1, &s.Clutch1v2, &s.Clutch1v3, &s.Clutch1v4, &s.Clutch1v5,
			&s.DoubleKills, &s.TripleKills, &s.QuadKills, &s.Ace,
			&s.TradeKills, &s.TradeDeaths, &s.SurvivalTime, &s.Survived,
		)
		if err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// GetAllMatches returns all matches ordered by timestamp descending
func (d *Database) GetAllMatches() ([]models.Match, error) {
	rows, err := d.db.Query(`
		SELECT id, match_id, game_version, code_version, timestamp, match_type,
		       game_mode, map, recording_player, profile_id, team_score,
		       opponent_score, won, rounds_played, imported_at, file_path
		FROM matches ORDER BY timestamp DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []models.Match
	for rows.Next() {
		var m models.Match
		err := rows.Scan(
			&m.ID, &m.MatchID, &m.GameVersion, &m.CodeVersion, &m.Timestamp,
			&m.MatchType, &m.GameMode, &m.Map, &m.RecordingPlayer, &m.ProfileID,
			&m.TeamScore, &m.OpponentScore, &m.Won, &m.RoundsPlayed,
			&m.ImportedAt, &m.FilePath,
		)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// GetMatchesByFilter returns matches matching the given criteria
func (d *Database) GetMatchesByFilter(matchType, mapName string, won *bool) ([]models.Match, error) {
	query := "SELECT id, match_id, game_version, code_version, timestamp, match_type, game_mode, map, recording_player, profile_id, team_score, opponent_score, won, rounds_played, imported_at, file_path FROM matches WHERE 1=1"
	args := []interface{}{}

	if matchType != "" && matchType != "All" {
		query += " AND match_type = ?"
		args = append(args, matchType)
	}
	if mapName != "" && mapName != "All" {
		query += " AND map = ?"
		args = append(args, mapName)
	}
	if won != nil {
		query += " AND won = ?"
		args = append(args, *won)
	}

	query += " ORDER BY timestamp DESC"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []models.Match
	for rows.Next() {
		var m models.Match
		err := rows.Scan(
			&m.ID, &m.MatchID, &m.GameVersion, &m.CodeVersion, &m.Timestamp,
			&m.MatchType, &m.GameMode, &m.Map, &m.RecordingPlayer, &m.ProfileID,
			&m.TeamScore, &m.OpponentScore, &m.Won, &m.RoundsPlayed,
			&m.ImportedAt, &m.FilePath,
		)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// GetRoundsByMatch returns all rounds for a given match
func (d *Database) GetRoundsByMatch(matchID int64) ([]models.Round, error) {
	rows, err := d.db.Query(`
		SELECT id, match_id, round_number, site, team_role, won, 
		       win_condition, team_score, opponent_score
		FROM rounds WHERE match_id = ? ORDER BY round_number
	`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rounds []models.Round
	for rows.Next() {
		var r models.Round
		err := rows.Scan(
			&r.ID, &r.MatchID, &r.RoundNumber, &r.Site, &r.TeamRole,
			&r.Won, &r.WinCondition, &r.TeamScore, &r.OpponentScore,
		)
		if err != nil {
			return nil, err
		}
		rounds = append(rounds, r)
	}
	return rounds, nil
}

// GetPlayersByRound returns all players in a round
func (d *Database) GetPlayersByRound(roundID int64) ([]models.Player, error) {
	rows, err := d.db.Query(`
		SELECT id, round_id, match_id, profile_id, username, team_index, operator, spawn
		FROM players WHERE round_id = ? ORDER BY team_index, username
	`, roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []models.Player
	for rows.Next() {
		var p models.Player
		err := rows.Scan(
			&p.ID, &p.RoundID, &p.MatchID, &p.ProfileID, &p.Username,
			&p.TeamIndex, &p.Operator, &p.Spawn,
		)
		if err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, nil
}

// GetEventsByRound returns all events in a round
func (d *Database) GetEventsByRound(roundID int64) ([]models.MatchEvent, error) {
	rows, err := d.db.Query(`
		SELECT id, round_id, match_id, event_type, time, time_in_seconds,
		       username, target, headshot, message
		FROM match_events WHERE round_id = ? ORDER BY time_in_seconds DESC
	`, roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.MatchEvent
	for rows.Next() {
		var e models.MatchEvent
		err := rows.Scan(
			&e.ID, &e.RoundID, &e.MatchID, &e.EventType, &e.Time,
			&e.TimeInSeconds, &e.Username, &e.Target, &e.Headshot, &e.Message,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// GetMapStats returns aggregated stats per map
func (d *Database) GetMapStats() ([]models.MapStats, error) {
	rows, err := d.db.Query(`
		SELECT map, 
		       COUNT(*) as played,
		       SUM(CASE WHEN won THEN 1 ELSE 0 END) as wins,
		       SUM(CASE WHEN NOT won THEN 1 ELSE 0 END) as losses,
		       AVG(rounds_played) as avg_rounds
		FROM matches 
		GROUP BY map 
		ORDER BY played DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.MapStats
	for rows.Next() {
		var s models.MapStats
		err := rows.Scan(&s.MapName, &s.Played, &s.Wins, &s.Losses, &s.AvgRounds)
		if err != nil {
			return nil, err
		}
		if s.Played > 0 {
			s.WinRate = float64(s.Wins) / float64(s.Played) * 100
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// GetOverallStats returns overall match statistics
func (d *Database) GetOverallStats() (played, wins, losses int, winRate float64, err error) {
	err = d.db.QueryRow(`
		SELECT COUNT(*) as played,
		       SUM(CASE WHEN won THEN 1 ELSE 0 END) as wins,
		       SUM(CASE WHEN NOT won THEN 1 ELSE 0 END) as losses
		FROM matches
	`).Scan(&played, &wins, &losses)
	if err != nil {
		return
	}
	if played > 0 {
		winRate = float64(wins) / float64(played) * 100
	}
	return
}

// GetDistinctMaps returns all unique maps in the database
func (d *Database) GetDistinctMaps() ([]string, error) {
	rows, err := d.db.Query("SELECT DISTINCT map FROM matches ORDER BY map")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var maps []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		maps = append(maps, m)
	}
	return maps, nil
}

// GetClutchStats returns aggregated clutch statistics for all players
func (d *Database) GetClutchStats() ([]models.ClutchStats, error) {
	rows, err := d.db.Query(`
		SELECT 
			username,
			SUM(CASE WHEN clutch_1v1 THEN 1 ELSE 0 END) as clutch_1v1,
			SUM(CASE WHEN clutch_1v1 AND survived THEN 1 ELSE 0 END) as clutch_1v1_won,
			SUM(CASE WHEN clutch_1v2 THEN 1 ELSE 0 END) as clutch_1v2,
			SUM(CASE WHEN clutch_1v2 AND survived THEN 1 ELSE 0 END) as clutch_1v2_won,
			SUM(CASE WHEN clutch_1v3 THEN 1 ELSE 0 END) as clutch_1v3,
			SUM(CASE WHEN clutch_1v3 AND survived THEN 1 ELSE 0 END) as clutch_1v3_won,
			SUM(CASE WHEN clutch_1v4 THEN 1 ELSE 0 END) as clutch_1v4,
			SUM(CASE WHEN clutch_1v4 AND survived THEN 1 ELSE 0 END) as clutch_1v4_won,
			SUM(CASE WHEN clutch_1v5 THEN 1 ELSE 0 END) as clutch_1v5,
			SUM(CASE WHEN clutch_1v5 AND survived THEN 1 ELSE 0 END) as clutch_1v5_won,
			SUM(clutch_attempts) as total_attempts,
			SUM(clutch_wins) as total_wins
		FROM player_round_stats
		WHERE team_index = 0
		GROUP BY username
		HAVING total_attempts > 0
		ORDER BY total_wins DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.ClutchStats
	for rows.Next() {
		var s models.ClutchStats
		var totalAttempts, totalWins int
		err := rows.Scan(
			&s.Username,
			&s.Clutch1v1, &s.Clutch1v1Won,
			&s.Clutch1v2, &s.Clutch1v2Won,
			&s.Clutch1v3, &s.Clutch1v3Won,
			&s.Clutch1v4, &s.Clutch1v4Won,
			&s.Clutch1v5, &s.Clutch1v5Won,
			&totalAttempts, &totalWins,
		)
		if err != nil {
			return nil, err
		}
		if totalAttempts > 0 {
			s.ClutchRate = float64(totalWins) / float64(totalAttempts) * 100
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// GetDefuserStats returns aggregated defuser statistics for all players
func (d *Database) GetDefuserStats() ([]models.DefuserStats, error) {
	rows, err := d.db.Query(`
		SELECT 
			username,
			SUM(defuser_plants) as plants,
			SUM(defuser_defuses) as defuses,
			SUM(plant_denials) as plant_denials
		FROM player_round_stats
		WHERE team_index = 0
		GROUP BY username
		HAVING (plants > 0 OR defuses > 0 OR plant_denials > 0)
		ORDER BY plants DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.DefuserStats
	for rows.Next() {
		var s models.DefuserStats
		err := rows.Scan(&s.Username, &s.Plants, &s.Defuses, &s.PlantDenials)
		if err != nil {
			return nil, err
		}
		total := s.Plants + s.PlantDenials
		if total > 0 {
			s.PlantSuccessRate = float64(s.Plants) / float64(total) * 100
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// GetSettings returns current settings
func (d *Database) GetSettings() (*models.Settings, error) {
	var s models.Settings
	err := d.db.QueryRow(`
		SELECT id, replay_folder, auto_import, theme, start_minimized, start_with_system
		FROM settings WHERE id = 1
	`).Scan(&s.ID, &s.ReplayFolder, &s.AutoImport, &s.Theme, &s.StartMinimized, &s.StartWithSystem)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpdateSettings updates the settings
func (d *Database) UpdateSettings(s *models.Settings) error {
	_, err := d.db.Exec(`
		UPDATE settings SET 
			replay_folder = ?, auto_import = ?, theme = ?,
			start_minimized = ?, start_with_system = ?
		WHERE id = 1
	`, s.ReplayFolder, s.AutoImport, s.Theme, s.StartMinimized, s.StartWithSystem)
	return err
}

// DeleteMatch removes a match and all its related data
func (d *Database) DeleteMatch(matchID int64) error {
	_, err := d.db.Exec("DELETE FROM matches WHERE id = ?", matchID)
	return err
}

// GetMatchCount returns total number of matches
func (d *Database) GetMatchCount() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM matches").Scan(&count)
	return count, err
}
