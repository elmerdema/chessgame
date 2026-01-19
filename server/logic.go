package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/corentings/chess"
	"github.com/google/uuid"
)

// Commonly used K-factor in Elo rating systems
// wikipedia.org/wiki/Elo_rating_system
const KFactor = 32

type MoveRequest struct {
	Move string `json:"move"`
}
type User struct {
	Elo int
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

func runMatchmaker(s *Server) {
	for {
		time.Sleep(3 * time.Second)
		matchmakingMutex.Lock()
		if len(matchmakingQueue) < 2 {
			matchmakingMutex.Unlock()
			continue
		}

		var matchedIndices = make(map[int]bool)
		var newQueue []MatchmakingRequest

		for i := 0; i < len(matchmakingQueue); i++ {
			if matchedIndices[i] {
				continue
			}
			for j := i + 1; j < len(matchmakingQueue); j++ {
				if matchedIndices[j] {
					continue
				}

				p1 := matchmakingQueue[i]
				p2 := matchmakingQueue[j]

				waitDuration := time.Since(p1.Timestamp).Seconds()
				eloRange := 50 + int(waitDuration*5)
				eloDiff := int(math.Abs(float64(p1.Elo - p2.Elo)))

				if eloDiff <= eloRange {
					gameID := uuid.New().String()
					log.Printf("Match: %s vs %s", p1.Username, p2.Username)

					// Create game in DB
					_, err := s.db.Exec(`INSERT INTO games (id, player_white, player_black, fen, state) 
						VALUES ($1, $2, $3, $4, 'in_progress')`,
						gameID, p1.Username, p2.Username, chess.NewGame().FEN())

					if err == nil {
						matchedIndices[i] = true
						matchedIndices[j] = true
						break
					} else {
						log.Println("Matchmaker DB Error:", err)
					}
				}
			}
		}

		for i, req := range matchmakingQueue {
			if !matchedIndices[i] {
				newQueue = append(newQueue, req)
			}
		}
		matchmakingQueue = newQueue
		matchmakingMutex.Unlock()
	}
}
