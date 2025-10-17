package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"

	"github.com/corentings/chess"
)

// Commonly used K-factor in Elo rating systems
// wikipedia.org/wiki/Elo_rating_system
const KFactor = 32

type MoveRequest struct {
	Move string `json:"move"`
}

type MoveResponse struct {
	GameID  string `json:"gameID"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	NewFEN  string `json:"newFEN"`
	Outcome string `json:"outcome"`
	Turn    string `json:"turn"`
}

func sendJSONResponse(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func calculateNewRatings(player1Elo, player2Elo int, outcome chess.Outcome) (int, int) {
	p1 := 1.0 / (1.0 + float64(math.Pow(10, float64(player2Elo-player1Elo)/400.0)))
	p2 := 1.0 - p1

	var score1, score2 float64
	switch outcome {
	//  player1 is white, player2 is blakc
	case chess.WhiteWon:
		score1 = 1.0
		score2 = 0.0
	case chess.BlackWon:
		score1 = 0.0
		score2 = 1.0
	case chess.Draw:
		score1 = 0.5
		score2 = 0.5
	}

	newElo1 := int(float64(player1Elo) + KFactor*(score1-p1))
	newElo2 := int(float64(player2Elo) + KFactor*(score2-p2))

	return newElo1, newElo2
}

// updateElo updates the ratings for two players after a game.
func updateElo(player1 string, player2 string, outcome chess.Outcome) {
	usersMutex.Lock()
	defer usersMutex.Unlock()

	user1, ok1 := users[player1]
	user2, ok2 := users[player2]

	if !ok1 || !ok2 {
		log.Printf("Could not find one or both users to update ELO: %s, %s", player1, player2)
		return
	}

	// No outcome means no ELO change
	if outcome == chess.NoOutcome {
		return
	}

	// player1 is white, player2 is black
	newElo1, newElo2 := calculateNewRatings(user1.Elo, user2.Elo, outcome)

	log.Printf("ELO Update (%s): %s vs %s. %d -> %d, %d -> %d", outcome, player1, player2, user1.Elo, newElo1, user2.Elo, newElo2)

	user1.Elo = newElo1
	user2.Elo = newElo2
	users[player1] = user1
	users[player2] = user2
}
