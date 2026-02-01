package main

import (
	"database/sql"
	"sync"
	"time"

	"github.com/corentings/chess"
)

type Server struct {
	db *sql.DB
}

type GameSession struct {
	Game        *chess.Game
	PlayerWhite string
	PlayerBlack string
	State       string
}

type LeaderboardEntry struct {
	Username string `json:"username"`
	Elo      int    `json:"elo"`
}

type Login struct {
	HashedPassword string
	SessionToken   string
	CSRFToken      string
	Elo            int
}

type MatchmakingRequest struct {
	Username  string
	Elo       int
	Timestamp time.Time
}

type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	GameID  string      `json:"gameID"`
}

// GameTimer tracks time for each player in a game
type GameTimer struct {
	WhiteTimeRemaining time.Duration
	BlackTimeRemaining time.Duration
	LastMoveTime       time.Time
	CurrentTurn        chess.Color
	TimeControl        time.Duration
	Increment          time.Duration
	IsActive           bool
	// Protect concurrent access
	mu sync.RWMutex
}

// TimerState represents the current state sent to clients
type TimerState struct {
	WhiteTime   float64 `json:"whiteTime"`
	BlackTime   float64 `json:"blackTime"`
	CurrentTurn string  `json:"currentTurn"`
	IsActive    bool    `json:"isActive"`
}
