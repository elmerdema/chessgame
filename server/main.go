package main

//go run .\server

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/corentings/chess"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/rs/cors"
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

var DefaultElo = 500

// in memory
var matchmakingQueue []MatchmakingRequest
var matchmakingMutex = &sync.Mutex{}

func main() {
	db, err := InitDB()
	if err != nil {
		log.Fatal("Failed to connect to database", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping the database", err)
	}

	srv := &Server{db: db}

	var addr = flag.String("addr", ":8081", "The addr of the application.")
	flag.Parse()

	router := mux.NewRouter()

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/login", srv.login).Methods("POST", "OPTIONS")
	api.HandleFunc("/register", srv.register).Methods("POST", "OPTIONS")

	auth := api.PathPrefix("").Subrouter()
	auth.Use(srv.AuthMiddleware)

	auth.HandleFunc("/logout", srv.logout).Methods("POST", "OPTIONS")
	auth.HandleFunc("/check-auth", srv.checkAuth).Methods("GET", "OPTIONS")
	auth.HandleFunc("/game/{id}/join", srv.joinGame).Methods("POST", "OPTIONS")
	auth.HandleFunc("/game/{id}", srv.getGame).Methods("GET", "OPTIONS")
	auth.HandleFunc("/matchmaking/find", srv.findMatch).Methods("POST", "OPTIONS")
	auth.HandleFunc("/matchmaking/status", srv.getMatchmakingStatus).Methods("GET", "OPTIONS")
	auth.HandleFunc("/leaderboard", srv.getLeaderboard).Methods("GET", "OPTIONS")

	room := newRoom()
	auth.HandleFunc("/game/{id}/move", srv.makeMoveHandler(room)).Methods("POST", "OPTIONS")
	auth.Handle("/ws", room).Methods("GET")

	go room.run()
	go runMatchmaker(srv) // Pass server to matchmaker to access DB

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "X-Requested-With"},
		AllowCredentials: true,
	})

	log.Println("Starting API server on", *addr)
	if err := http.ListenAndServe(*addr, c.Handler(router)); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

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
