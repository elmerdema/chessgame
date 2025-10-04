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

func calculateNewRatings(winnerElo, loserElo int) (int, int) {
	// Probability of winning for each player
	pWinner := 1.0 / (1.0 + float64(math.Pow(10, float64(loserElo-winnerElo)/400.0)))
	pLoser := 1.0 - pWinner

	var newWinnerElo, newLoserElo int
	// If it's a draw, both players have a score of 0.5
	if winnerElo == loserElo {
		// This block is for draws, we can call it with the same ELOs
		newWinnerElo = int(float64(winnerElo) + KFactor*(0.5-pWinner))
		newLoserElo = int(float64(loserElo) + KFactor*(0.5-pLoser))
	} else {
		// For a decisive game, winner gets score 1, loser gets 0
		newWinnerElo = int(float64(winnerElo) + KFactor*(1.0-pWinner))
		newLoserElo = int(float64(loserElo) + KFactor*(0.0-pLoser))
	}

	return newWinnerElo, newLoserElo
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

	var newElo1, newElo2 int
	switch outcome {
	case chess.WhiteWon:
		// Assuming player1 is white, player2 is black. This needs to be checked.
		// We'll pass the winner first to calculateNewRatings.
		newElo1, newElo2 = calculateNewRatings(user1.Elo, user2.Elo)
		log.Printf("ELO Update: %s (W) vs %s (L). %d -> %d, %d -> %d", player1, player2, user1.Elo, newElo1, user2.Elo, newElo2)
	case chess.BlackWon:
		// Player2 (black) won
		newElo2, newElo1 = calculateNewRatings(user2.Elo, user1.Elo)
		log.Printf("ELO Update: %s (L) vs %s (W). %d -> %d, %d -> %d", player1, player2, user1.Elo, newElo1, user2.Elo, newElo2)
	case chess.Draw:
		// For a draw, the order doesn't matter, but we'll use a special call
		newElo1, newElo2 = calculateNewRatings(user1.Elo, user2.Elo) // Pass as if it's a draw
		log.Printf("ELO Update (Draw): %s vs %s. %d -> %d, %d -> %d", player1, player2, user1.Elo, newElo1, user2.Elo, newElo2)
	default:
		// Game not over or other outcome, no ELO change
		return
	}

	user1.Elo = newElo1
	user2.Elo = newElo2
	users[player1] = user1
	users[player2] = user2
}
