package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/corentings/chess"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if len(username) < 5 || len(password) < 5 {
		http.Error(w, "Username and password must be at least 5 characters", http.StatusBadRequest)
		return
	}

	hashedPassword, err := HashedPassword(password)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	_, err = s.db.Exec(
		"INSERT INTO users (username, password_hash, elo) VALUES ($1, $2, $3)",
		username, hashedPassword, DefaultElo,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			http.Error(w, "User already exists", http.StatusConflict)
			return
		}
		log.Println("DB Register Error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "User registered successfully!")
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var storedHash string
	err := s.db.QueryRow("SELECT password_hash FROM users WHERE username=$1", username).Scan(&storedHash)

	if err == sql.ErrNoRows || !CheckPasswordHash(password, storedHash) {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	sessionToken := generateToken(32)
	csrfToken := generateToken(32)

	_, err = s.db.Exec("UPDATE users SET session_token=$1, csrf_token=$2 WHERE username=$3",
		sessionToken, csrfToken, username)

	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: false,
	})

	fmt.Fprintln(w, "Login Successful")
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	username, ok := GetUsernameFromContext(r)
	if ok {
		s.db.Exec("UPDATE users SET session_token=NULL, csrf_token=NULL WHERE username=$1", username)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: false,
	})

	fmt.Fprintln(w, "Logged out successfully!")
}

func (s *Server) checkAuth(w http.ResponseWriter, r *http.Request) {
	username, ok := GetUsernameFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"isLoggedIn": true, "username": "%s"}`, username)
}

func (s *Server) joinGame(w http.ResponseWriter, r *http.Request) {
	username, ok := GetUsernameFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	gameID := vars["id"]

	var state, playerWhite, fen string
	err := s.db.QueryRow("SELECT state, player_white, fen FROM games WHERE id=$1", gameID).
		Scan(&state, &playerWhite, &fen)

	if err == sql.ErrNoRows {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	if state != "waiting" {
		http.Error(w, "Game is not available to join", http.StatusConflict)
		return
	}

	if playerWhite == username {
		http.Error(w, "You cannot join your own game.", http.StatusBadRequest)
		return
	}

	_, err = s.db.Exec("UPDATE games SET player_black=$1, state='in_progress', updated_at=NOW() WHERE id=$2",
		username, gameID)

	if err != nil {
		http.Error(w, "Failed to join game", http.StatusInternalServerError)
		return
	}

	log.Printf("User %s joined game %s as Black.", username, gameID)

	response := map[string]string{
		"gameID": gameID,
		"fen":    fen,
		"color":  "black",
	}
	sendJSONResponse(w, http.StatusOK, response)
}

func (s *Server) getGame(w http.ResponseWriter, r *http.Request) {
	username, ok := GetUsernameFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	gameID := vars["id"]

	var fen, white, black, state string
	// Handle NULL black player using sql.NullString or COALESCE
	err := s.db.QueryRow("SELECT fen, player_white, COALESCE(player_black, ''), state FROM games WHERE id=$1", gameID).
		Scan(&fen, &white, &black, &state)

	if err == sql.ErrNoRows {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	playerColor := ""
	if username == white {
		playerColor = "white"
	} else if username == black {
		playerColor = "black"
	}

	response := map[string]interface{}{
		"gameID":      gameID,
		"fen":         fen,
		"playerWhite": white,
		"playerBlack": black,
		"state":       state,
		"playerColor": playerColor,
	}
	sendJSONResponse(w, http.StatusOK, response)
}

func (s *Server) makeMoveHandler(room *room) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := GetUsernameFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		gameID, ok := vars["id"]

		var req MoveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		var fen, playerWhite, playerBlack, dbState string
		err := s.db.QueryRow(`SELECT fen, player_white, player_black, state FROM games WHERE id = $1`, gameID).
			Scan(&fen, &playerWhite, &playerBlack, &dbState)

		if err != nil {
			http.Error(w, "Game not found", http.StatusNotFound)
			return
		}

		if dbState != "in_progress" {
			http.Error(w, "Game is not in progress", http.StatusBadRequest)
			return
		}

		fenFunc, err := chess.FEN(fen)
		if err != nil {
			http.Error(w, "Invalid FEN in database", http.StatusInternalServerError)
			return
		}
		game := chess.NewGame(fenFunc)

		// Check turn
		turn := game.Position().Turn()
		if (turn == chess.White && username != playerWhite) || (turn == chess.Black && username != playerBlack) {
			http.Error(w, "Not your turn", http.StatusForbidden)
			return
		}

		// Validate Move
		moveStr := req.Move
		var foundMove *chess.Move
		for _, validMove := range game.ValidMoves() {
			if validMove.String() == moveStr {
				foundMove = validMove
				break
			}
		}

		if foundMove == nil {
			sendJSONResponse(w, http.StatusBadRequest, MoveResponse{Status: "error", Message: "Illegal move"})
			return
		}

		if err := game.Move(foundMove); err != nil {
			http.Error(w, "Error applying move", http.StatusInternalServerError)
			return
		}

		outcome := game.Outcome()
		newFen := game.FEN()
		newState := "in_progress"
		var winner *string = nil

		if outcome != chess.NoOutcome {
			newState = "finished"
			UpdateEloInDB(playerWhite, playerBlack, outcome, s.db)

			wStr := ""
			if outcome == chess.WhiteWon {
				wStr = "white"
			} else if outcome == chess.BlackWon {
				wStr = "black"
			} else {
				wStr = "draw"
			}
			winner = &wStr

			_, err = s.db.Exec(`UPDATE games SET fen = $1, state = $2, winner = $3, updated_at = NOW() WHERE id = $4`,
				newFen, newState, winner, gameID)

			if err != nil {
				http.Error(w, "Failed to save game", http.StatusInternalServerError)
				return
			}

			response := MoveResponse{
				GameID:  gameID,
				Status:  "ok",
				NewFEN:  newFen,
				Outcome: string(outcome),
				Turn:    game.Position().Turn().Name(),
			}

			wsMessage := WebSocketMessage{
				Type:    "gameStateUpdate",
				Payload: response,
				GameID:  gameID,
			}

			bytes, _ := json.Marshal(wsMessage)
			room.forward <- &Message{content: bytes}
			sendJSONResponse(w, http.StatusOK, response)
		} else {
			_, err = s.db.Exec(`UPDATE games SET fen = $1, updated_at = NOW() WHERE id = $2`,
				newFen, gameID)

			if err != nil {
				http.Error(w, "Failed to save game", http.StatusInternalServerError)
				return
			}

			response := MoveResponse{
				GameID:  gameID,
				Status:  "ok",
				NewFEN:  newFen,
				Outcome: "",
				Turn:    game.Position().Turn().Name(),
			}

			wsMessage := WebSocketMessage{
				Type:    "gameStateUpdate",
				Payload: response,
				GameID:  gameID,
			}

			bytes, _ := json.Marshal(wsMessage)
			room.forward <- &Message{content: bytes}
			sendJSONResponse(w, http.StatusOK, response)
		}
	}
}

// --- Matchmaking & Leaderboard ---

func (s *Server) findMatch(w http.ResponseWriter, r *http.Request) {
	username, ok := GetUsernameFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userElo int
	err := s.db.QueryRow("SELECT elo FROM users WHERE username=$1", username).Scan(&userElo)
	if err != nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	// Check Queue
	matchmakingMutex.Lock()
	for _, req := range matchmakingQueue {
		if req.Username == username {
			matchmakingMutex.Unlock()
			sendJSONResponse(w, http.StatusOK, map[string]string{"message": "Already in queue"})
			return
		}
	}
	matchmakingMutex.Unlock()

	// Check Active Games in DB
	var existingGameID string
	err = s.db.QueryRow("SELECT id FROM games WHERE (player_white=$1 OR player_black=$1) AND state='in_progress'", username).
		Scan(&existingGameID)

	if err == nil {
		s.getMatchmakingStatus(w, r)
		return
	}

	matchmakingMutex.Lock()
	matchmakingQueue = append(matchmakingQueue, MatchmakingRequest{
		Username:  username,
		Elo:       userElo,
		Timestamp: time.Now(),
	})
	matchmakingMutex.Unlock()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"message": "Searching..."})
}

func (s *Server) getMatchmakingStatus(w http.ResponseWriter, r *http.Request) {
	username, ok := GetUsernameFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var gameID string
	err := s.db.QueryRow("SELECT id FROM games WHERE (player_white=$1 OR player_black=$1) AND state='in_progress'", username).
		Scan(&gameID)

	if err == nil {
		sendJSONResponse(w, http.StatusOK, map[string]string{"status": "found", "gameID": gameID})
	} else {
		sendJSONResponse(w, http.StatusOK, map[string]string{"status": "searching"})
	}
}

func (s *Server) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query("SELECT username, elo FROM users ORDER BY elo DESC LIMIT 10")
	if err != nil {
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var leaderboard []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		rows.Scan(&e.Username, &e.Elo)
		leaderboard = append(leaderboard, e)
	}
	sendJSONResponse(w, http.StatusOK, leaderboard)
}
