package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/corentings/chess"
)

var (
	gameTimers      = make(map[string]*GameTimer)
	gameTimersMutex sync.RWMutex
)

// 10 minutes per player with 5 second increment
const (
	DefaultTimeControl = 10 * time.Minute
	DefaultIncrement   = 5 * time.Second
)

// used to create a new timer for a game
func initializeGameTimer(gameID string, timeControl, increment time.Duration) {
	gameTimersMutex.Lock()
	defer gameTimersMutex.Unlock()

	gameTimers[gameID] = &GameTimer{
		WhiteTimeRemaining: timeControl,
		BlackTimeRemaining: timeControl,
		LastMoveTime:       time.Now(),
		CurrentTurn:        chess.White,
		TimeControl:        timeControl,
		Increment:          increment,
		IsActive:           true,
	}

	log.Printf("Timer initialized for game %s: %v + %v increment", gameID, timeControl, increment)
}

func updateGameTimer(gameID string, game *chess.Game) error {
	gameTimersMutex.RLock()
	timer, exists := gameTimers[gameID]
	gameTimersMutex.RUnlock()

	if !exists {
		return fmt.Errorf("timer not found for game %s", gameID)
	}

	timer.mu.Lock()
	defer timer.mu.Unlock()

	if !timer.IsActive {
		return fmt.Errorf("timer is not active for game %s", gameID)
	}

	elapsed := time.Since(timer.LastMoveTime)

	if timer.CurrentTurn == chess.White {
		timer.WhiteTimeRemaining -= elapsed
		timer.WhiteTimeRemaining += timer.Increment
		if timer.WhiteTimeRemaining <= 0 {
			timer.IsActive = false
			return fmt.Errorf("white ran out of time")
		}
	} else {
		timer.BlackTimeRemaining -= elapsed
		timer.BlackTimeRemaining += timer.Increment
		if timer.BlackTimeRemaining <= 0 {
			timer.IsActive = false
			return fmt.Errorf("black ran out of time")
		}
	}

	timer.CurrentTurn = game.Position().Turn()
	timer.LastMoveTime = time.Now()

	log.Printf("Timer updated for game %s: White=%v, Black=%v",
		gameID, timer.WhiteTimeRemaining, timer.BlackTimeRemaining)

	return nil
}

func getTimerState(gameID string) (*TimerState, error) {
	gameTimersMutex.RLock()
	timer, exists := gameTimers[gameID]
	gameTimersMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("timer not found for game %s", gameID)
	}

	timer.mu.RLock()
	defer timer.mu.RUnlock()

	whiteTime := timer.WhiteTimeRemaining
	blackTime := timer.BlackTimeRemaining

	if timer.IsActive {
		elapsed := time.Since(timer.LastMoveTime)
		if timer.CurrentTurn == chess.White {
			whiteTime -= elapsed
		} else {
			blackTime -= elapsed
		}
	}

	return &TimerState{
		WhiteTime:   whiteTime.Seconds(),
		BlackTime:   blackTime.Seconds(),
		CurrentTurn: timer.CurrentTurn.String(),
		IsActive:    timer.IsActive,
	}, nil
}

func cleanupGameTimer(gameID string) {
	gameTimersMutex.Lock()
	defer gameTimersMutex.Unlock()

	delete(gameTimers, gameID)
	log.Printf("Timer cleaned up for game %s", gameID)
}

func runTimerChecker(room *room) {
	// Update every 100ms
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	log.Println("Timer checker started")

	for range ticker.C {
		gameTimersMutex.RLock()
		timersCopy := make(map[string]*GameTimer, len(gameTimers))
		for k, v := range gameTimers {
			timersCopy[k] = v
		}
		gameTimersMutex.RUnlock()

		for gameID, timer := range timersCopy {
			timer.mu.RLock()

			if !timer.IsActive {
				timer.mu.RUnlock()
				continue
			}

			elapsed := time.Since(timer.LastMoveTime)
			whiteTime := timer.WhiteTimeRemaining
			blackTime := timer.BlackTimeRemaining

			if timer.CurrentTurn == chess.White {
				whiteTime -= elapsed
			} else {
				blackTime -= elapsed
			}

			currentTurn := timer.CurrentTurn.String()
			timer.mu.RUnlock()

			// Check if time expired
			if whiteTime <= 0 || blackTime <= 0 {
				// Lock for write to deactivate
				timer.mu.Lock()
				timer.IsActive = false
				timer.mu.Unlock()

				winner := "black"
				if blackTime <= 0 {
					winner = "white"
				}

				// Broadcast time forfeit
				forfeitMsg := WebSocketMessage{
					Type: "time_forfeit",
					Payload: map[string]interface{}{
						"reason": "time_expired",
						"winner": winner,
					},
					GameID: gameID,
				}

				broadcastToGame(room, gameID, forfeitMsg)
				log.Printf("Game %s ended by time forfeit, winner: %s", gameID, winner)
				continue
			}

			// Broadcast regular timer update
			timerUpdate := WebSocketMessage{
				Type: "timer_update",
				Payload: TimerState{
					WhiteTime:   whiteTime.Seconds(),
					BlackTime:   blackTime.Seconds(),
					CurrentTurn: currentTurn,
					IsActive:    true,
				},
				GameID: gameID,
			}

			broadcastToGame(room, gameID, timerUpdate)
		}
	}
}

// BroadcastToGame sends a message to all clients in a specific game
func broadcastToGame(room *room, gameID string, message WebSocketMessage) {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling timer message: %v", err)
		return
	}

	room.forward <- &Message{
		client:  nil,
		content: msgBytes,
	}
}
