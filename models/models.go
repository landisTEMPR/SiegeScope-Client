package models

import "time"

// Match represents a complete match with all rounds
type Match struct {
	ID              int64     `json:"id"`
	MatchID         string    `json:"matchId"`
	GameVersion     string    `json:"gameVersion"`
	CodeVersion     int       `json:"codeVersion"`
	Timestamp       time.Time `json:"timestamp"`
	MatchType       string    `json:"matchType"`
	GameMode        string    `json:"gameMode"`
	Map             string    `json:"map"`
	RecordingPlayer string    `json:"recordingPlayer"`
	ProfileID       string    `json:"profileId"`
	TeamScore       int       `json:"teamScore"`
	OpponentScore   int       `json:"opponentScore"`
	Won             bool      `json:"won"`
	RoundsPlayed    int       `json:"roundsPlayed"`
	ImportedAt      time.Time `json:"importedAt"`
	FilePath        string    `json:"filePath"`
}

// Round represents a single round within a match
type Round struct {
	ID            int64  `json:"id"`
	MatchID       int64  `json:"matchId"`
	RoundNumber   int    `json:"roundNumber"`
	Site          string `json:"site"`
	TeamRole      string `json:"teamRole"`
	Won           bool   `json:"won"`
	WinCondition  string `json:"winCondition"`
	TeamScore     int    `json:"teamScore"`
	OpponentScore int    `json:"opponentScore"`
}

// Player represents a player in a round
type Player struct {
	ID        int64  `json:"id"`
	RoundID   int64  `json:"roundId"`
	MatchID   int64  `json:"matchId"`
	ProfileID string `json:"profileId"`
	Username  string `json:"username"`
	TeamIndex int    `json:"teamIndex"`
	Operator  string `json:"operator"`
	Spawn     string `json:"spawn"`
}

// MatchEvent represents a kill, plant, disable, etc.
type MatchEvent struct {
	ID            int64  `json:"id"`
	RoundID       int64  `json:"roundId"`
	MatchID       int64  `json:"matchId"`
	EventType     string `json:"eventType"`
	Time          string `json:"time"`
	TimeInSeconds int    `json:"timeInSeconds"`
	Username      string `json:"username"`
	Target        string `json:"target"`
	Headshot      bool   `json:"headshot"`
	Message       string `json:"message"`
}

// PlayerRoundStats represents a player's stats for a single round
type PlayerRoundStats struct {
	ID                 int64   `json:"id"`
	RoundID            int64   `json:"roundId"`
	MatchID            int64   `json:"matchId"`
	Username           string  `json:"username"`
	TeamIndex          int     `json:"teamIndex"`
	Operator           string  `json:"operator"`
	Kills              int     `json:"kills"`
	Died               bool    `json:"died"`
	Assists            int     `json:"assists"`
	Headshots          int     `json:"headshots"`
	HeadshotPercentage float64 `json:"headshotPercentage"`
	EntryKill          bool    `json:"entryKill"`
	EntryDeath         bool    `json:"entryDeath"`

	// Defuser stats
	DefuserPlants  int `json:"defuserPlants"`
	DefuserDefuses int `json:"defuserDefuses"`
	DefuserPickups int `json:"defuserPickups"`
	PlantDenials   int `json:"plantDenials"`

	// Clutch stats
	ClutchAttempts int  `json:"clutchAttempts"`
	ClutchWins     int  `json:"clutchWins"`
	Clutch1v1      bool `json:"clutch1v1"`
	Clutch1v2      bool `json:"clutch1v2"`
	Clutch1v3      bool `json:"clutch1v3"`
	Clutch1v4      bool `json:"clutch1v4"`
	Clutch1v5      bool `json:"clutch1v5"`

	// Multi-kill stats
	DoubleKills int  `json:"doubleKills"`
	TripleKills int  `json:"tripleKills"`
	QuadKills   int  `json:"quadKills"`
	Ace         bool `json:"ace"`

	// Trading stats
	TradeKills  int `json:"tradeKills"`
	TradeDeaths int `json:"tradeDeaths"`

	// Survival stats
	SurvivalTime float64 `json:"survivalTime"`
	Survived     bool    `json:"survived"`
}

// PlayerStats aggregated stats for a player across matches
type PlayerStats struct {
	ProfileID          string  `json:"profileId"`
	Username           string  `json:"username"`
	MatchesPlayed      int     `json:"matchesPlayed"`
	RoundsPlayed       int     `json:"roundsPlayed"`
	Kills              int     `json:"kills"`
	Deaths             int     `json:"deaths"`
	Assists            int     `json:"assists"`
	Headshots          int     `json:"headshots"`
	HeadshotPercentage float64 `json:"headshotPercentage"`
	KD                 float64 `json:"kd"`
	WinRate            float64 `json:"winRate"`
}

// MapStats aggregated stats for a specific map
type MapStats struct {
	MapName   string  `json:"mapName"`
	Played    int     `json:"played"`
	Wins      int     `json:"wins"`
	Losses    int     `json:"losses"`
	WinRate   float64 `json:"winRate"`
	AvgRounds float64 `json:"avgRounds"`
}

// OperatorStats aggregated stats for an operator
type OperatorStats struct {
	Operator   string  `json:"operator"`
	TimesUsed  int     `json:"timesUsed"`
	Kills      int     `json:"kills"`
	Deaths     int     `json:"deaths"`
	KD         float64 `json:"kd"`
	RoundsWon  int     `json:"roundsWon"`
	RoundsLost int     `json:"roundsLost"`
	WinRate    float64 `json:"winRate"`
}

// ClutchStats aggregated clutch statistics
type ClutchStats struct {
	Username     string  `json:"username"`
	Clutch1v1    int     `json:"clutch1v1"`
	Clutch1v1Won int     `json:"clutch1v1Won"`
	Clutch1v2    int     `json:"clutch1v2"`
	Clutch1v2Won int     `json:"clutch1v2Won"`
	Clutch1v3    int     `json:"clutch1v3"`
	Clutch1v3Won int     `json:"clutch1v3Won"`
	Clutch1v4    int     `json:"clutch1v4"`
	Clutch1v4Won int     `json:"clutch1v4Won"`
	Clutch1v5    int     `json:"clutch1v5"`
	Clutch1v5Won int     `json:"clutch1v5Won"`
	ClutchRate   float64 `json:"clutchRate"`
}

// DefuserStats aggregated defuser statistics
type DefuserStats struct {
	Username         string  `json:"username"`
	Plants           int     `json:"plants"`
	Defuses          int     `json:"defuses"`
	PlantDenials     int     `json:"plantDenials"`
	PlantSuccessRate float64 `json:"plantSuccessRate"`
}

// Settings represents user application settings
type Settings struct {
	ID              int64  `json:"id"`
	ReplayFolder    string `json:"replayFolder"`
	AutoImport      bool   `json:"autoImport"`
	Theme           string `json:"theme"`
	StartMinimized  bool   `json:"startMinimized"`
	StartWithSystem bool   `json:"startWithSystem"`
	APIKey          string `json:"api_key"`
}
