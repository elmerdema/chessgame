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

type GameSession struct { // Used for API responses
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
	WhiteTimeRemaining time.Duration // Time left for white player
	BlackTimeRemaining time.Duration // Time left for black player
	LastMoveTime       time.Time     // When the last move was made
	CurrentTurn        chess.Color   // Whose turn it is
	TimeControl        time.Duration // Initial time per player (e.g., 10 minutes)
	Increment          time.Duration // Increment per move (e.g., 5 seconds)
	IsActive           bool          // Whether the timer is running
	mu                 sync.RWMutex  // Protect concurrent access
}

// TimerState represents the current state sent to clients
type TimerState struct {
	WhiteTime   float64 `json:"whiteTime"`   // Seconds remaining for white
	BlackTime   float64 `json:"blackTime"`   // Seconds remaining for black
	CurrentTurn string  `json:"currentTurn"` // "white" or "black"
	IsActive    bool    `json:"isActive"`
}
