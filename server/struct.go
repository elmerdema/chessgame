package main

import (
	"database/sql"
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
