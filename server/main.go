package main

//go run .
// curl -X POST -d "username=testuser&password=testpass" "http://localhost:8081/api/register"
// curl -X POST -d "username=testuser&password=testpass" "http://localhost:8081//apilogin"
//curl -X POST "http://localhost:8081/api/game/new"
// curl -X POST -H "Content-Type: application/json" -d "{\"move\": \"e2e4\"}" "http://localhost:8081/api/game/YOUR_GAME_ID_HERE/move"
import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/corentings/chess"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

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

var DefaultElo = 500

// TODO: setup real db here in the future
var users = map[string]Login{}
var usersMutex = &sync.RWMutex{}

var games = make(map[string]*GameSession)
var gamesMutex = &sync.RWMutex{}

var userGameMap = make(map[string]string)
var userGameMapMutex = &sync.RWMutex{}

var matchmakingQueue []MatchmakingRequest
var matchmakingMutex = &sync.Mutex{}

func main() {
	var addr = flag.String("addr", ":8081", "The addr of the application.")
	flag.Parse()

	router := mux.NewRouter()

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/login", login).Methods("POST", "OPTIONS")
	api.HandleFunc("/register", register).Methods("POST", "OPTIONS")

	auth := api.PathPrefix("").Subrouter()
	auth.Use(AuthMiddleware)

	auth.HandleFunc("/logout", logout).Methods("POST", "OPTIONS")
	auth.HandleFunc("/protected", protected).Methods("POST", "OPTIONS")
	auth.HandleFunc("/check-auth", checkAuth).Methods("GET", "OPTIONS")
	auth.HandleFunc("/game/{id}/join", joinGame).Methods("POST", "OPTIONS")
	auth.HandleFunc("/game/{id}", getGame).Methods("GET", "OPTIONS")
	auth.HandleFunc("/matchmaking/find", findMatch).Methods("POST", "OPTIONS")
	auth.HandleFunc("/matchmaking/status", getMatchmakingStatus).Methods("GET", "OPTIONS")
	auth.HandleFunc("/leaderboard", getLeaderboard).Methods("GET", "OPTIONS")
	room := newRoom()

	auth.HandleFunc("/game/{id}/move", makeMoveHandler(room)).Methods("POST", "OPTIONS")
	auth.Handle("/ws", room).Methods("GET") //room is passed to handlers that need to broadcast it

	go room.run()
	go runMatchmaker()

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8080"}, // frontend server, careful, if you start 2 different builds, the localhost may change
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "X-Requested-With"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	// Start the API server on port 8081 with the CORS handler
	log.Println("Starting API server on", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func register(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		err := http.StatusMethodNotAllowed
		http.Error(w, "Invalid method", err)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if len(username) < 5 || len(password) < 5 {
		err := http.StatusBadRequest
		http.Error(w, "Username and password must be at least 5 characters", err)
		return
	}
	usersMutex.Lock()
	defer usersMutex.Unlock()

	if _, ok := users[username]; ok {
		err := http.StatusConflict
		http.Error(w, "User already exists", err)
		return
	}
	hashedPassword, _ := HashedPassword(password)
	users[username] = Login{
		HashedPassword: hashedPassword,
		Elo:            DefaultElo,
	}
	fmt.Fprintln(w, "User registered successfully!")
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		err := http.StatusMethodNotAllowed
		http.Error(w, "Invalid request method", err)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, ok := users[username]
	if !ok || !CheckPasswordHash(password, user.HashedPassword) {
		err := http.StatusUnauthorized
		http.Error(w, "Invalid username or password", err)
		return
	}

	sessionToken := generateToken(32)
	csrfToken := generateToken(32)
	//session cookie
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

	//store the token in our defacto "db"
	user.SessionToken = sessionToken
	users[username] = user
	fmt.Println(w, "Login Successful")
}

func protected(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		err := http.StatusMethodNotAllowed
		http.Error(w, "Invalid request method", err)
		return
	}
	if err := Authorize(r); err != nil {
		err := http.StatusUnauthorized
		http.Error(w, "Unathorized, ", err)
		return
	}
	username := r.FormValue("username")
	fmt.Fprintf(w, "CSRF validation successful! Welcome %s", username)
}

func logout(w http.ResponseWriter, r *http.Request) {
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
	username := r.FormValue("username")
	user := users[username]
	user.CSRFToken = ""
	users[username] = user
	fmt.Fprintln(w, "Logged out successfully!")
}

func checkAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid Method, expected GET request", http.StatusMethodNotAllowed)
		return
	}
	sessionCookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Unathorized", http.StatusUnauthorized)
		return
	}
	sessionToken := sessionCookie.Value
	var foundUsername string
	for username, loginData := range users {
		if loginData.SessionToken == sessionToken {
			foundUsername = username
			break
		}
	}
	if foundUsername == "" {
		http.Error(w, "Unathorized", http.StatusUnauthorized)
		return
	}

	//user is authenticated
	w.Header().Set("Content-Type", "application/json")

	fmt.Fprintf(w, `{"isLoggedIn": true, "username": "%s"}`, foundUsername)
}

func joinGame(w http.ResponseWriter, r *http.Request) {
	username, ok := GetUsernameFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized, please login to join a game", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	gameID := vars["id"]

	gamesMutex.Lock()
	defer gamesMutex.Unlock()

	gameSession, ok := games[gameID]
	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	if gameSession.State != "waiting" {
		http.Error(w, "Game is not available to join", http.StatusConflict)
		return
	}

	if gameSession.State == username {
		http.Error(w, "You cannot join your own game.", http.StatusBadRequest)
		return
	}

	gameSession.PlayerBlack = username
	gameSession.State = "in_progress"
	games[gameID] = gameSession

	log.Printf("User %s joined game %s as Black. Game is now in progress", username, gameID)

	// TODO: Use the WebSocket to notify PlayerWhite that the game has started!
	// room.forward <- ... some message ...

	response := map[string]string{
		"gameID": gameID,
		"fen":    gameSession.Game.FEN(),
		"color":  "black",
	}
	sendJSONResponse(w, http.StatusOK, response)

}

func makeMoveHandler(room *room) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := GetUsernameFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized,", http.StatusUnauthorized)
			return
		}
		vars := mux.Vars(r)
		gameID, ok := vars["id"]
		if !ok {
			http.Error(w, "Game ID is missing", http.StatusBadRequest)
			return
		}

		var req MoveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		gamesMutex.Lock()
		defer gamesMutex.Unlock()

		gameSession, ok := games[gameID]
		if !ok {
			http.Error(w, "Game not found", http.StatusNotFound)
			return
		}

		turn := gameSession.Game.Position().Turn()
		isWhitesTurn := turn == chess.White
		isBlacksTurn := turn == chess.Black

		if (isWhitesTurn && username != gameSession.PlayerWhite) ||
			(isBlacksTurn && username != gameSession.PlayerBlack) {
			log.Printf("Illegal move attempt in game %s by user %s. Not their turn.", gameID, username)
			http.Error(w, "Not your turn.", http.StatusForbidden)
			return

		}

		var moveErr error
		moveStr := req.Move

		//  find the specific move from the list of valid moves.
		var foundMove *chess.Move
		for _, validMove := range gameSession.Game.ValidMoves() {
			if validMove.String() == moveStr {
				foundMove = validMove
				break
			}
		}

		if foundMove != nil {
			moveErr = gameSession.Game.Move(foundMove)
		} else {
			// If we couldn't find a matching move, the move is illegal.
			moveErr = fmt.Errorf("illegal move %s", moveStr)
		}

		if moveErr != nil {
			sendJSONResponse(w, http.StatusBadRequest, MoveResponse{
				Status:  "error",
				Message: fmt.Sprintf("Illegal move: %s", moveErr.Error()),
			})
			return
		}

		// if move was valid, the game state is updated.
		log.Printf("Game %s: Valid move %s. New FEN: %s", gameID, req.Move, gameSession.Game.FEN())

		outcome := gameSession.Game.Outcome()
		if outcome != chess.NoOutcome {
			log.Printf("Game: %s finished with outcome: %s", gameID, outcome)
			updateElo(gameSession.PlayerWhite, gameSession.PlayerBlack, outcome)
			gameSession.State = "finished"
		}

		response := MoveResponse{
			GameID:  gameID,
			Status:  "ok",
			NewFEN:  gameSession.Game.FEN(),
			Outcome: string(gameSession.Game.Outcome()),
			Turn:    gameSession.Game.Position().Turn().Name(),
		}

		httpResponse := MoveResponse{
			GameID:  gameID,
			Status:  "ok",
			NewFEN:  gameSession.Game.FEN(),
			Outcome: string(gameSession.Game.Outcome()),
			Turn:    gameSession.Game.Position().Turn().Name(),
		}

		wsMessage := WebSocketMessage{
			Type:    "gameStateUpdate",
			Payload: httpResponse, // add  original response as the payload
			GameID:  gameID,
		}

		broadcastMessage, err := json.Marshal(wsMessage)
		if err == nil {
			log.Printf("BROADCASTING MOVE for game %s: %s", gameID, string(broadcastMessage))
			room.forward <- broadcastMessage
		}

		sendJSONResponse(w, http.StatusOK, response)
	}
}

func getGame(w http.ResponseWriter, r *http.Request) {
	username, ok := GetUsernameFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
	vars := mux.Vars(r)
	gameID := vars["id"]

	gamesMutex.RLock()
	defer gamesMutex.RUnlock()

	gameSession, ok := games[gameID]

	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	playerColor := ""
	if username == gameSession.PlayerWhite {
		playerColor = "white"
	} else if username == gameSession.PlayerBlack {
		playerColor = "black"
	}

	response := map[string]interface{}{
		"gameID":      gameID,
		"fen":         gameSession.Game.FEN(),
		"playerWhite": gameSession.PlayerWhite,
		"playerBlack": gameSession.PlayerBlack,
		"state":       gameSession.State,
		"playerColor": playerColor,
	}

	sendJSONResponse(w, http.StatusOK, response)

}

func findMatch(w http.ResponseWriter, r *http.Request) {
	username, ok := GetUsernameFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	usersMutex.RLock()
	user, ok := users[username]
	if !ok {
		usersMutex.RUnlock()
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}
	userElo := user.Elo
	usersMutex.RUnlock()

	matchmakingMutex.Lock()
	defer matchmakingMutex.Unlock()

	// Check if user is already in the queue
	for _, req := range matchmakingQueue {
		if req.Username == username {
			sendJSONResponse(w, http.StatusOK, map[string]string{"message": "You are already in the matchmaking queue."})
			return
		}
	}

	// Check if user already has an active game they were matched into
	userGameMapMutex.RLock()
	if _, ok := userGameMap[username]; ok {
		userGameMapMutex.RUnlock()
		// If they are in the map but hit find again, it's likely a UI glitch.
		// We can just guide them to their game.
		getMatchmakingStatus(w, r) // Reuse status logic
		return
	}
	userGameMapMutex.RUnlock()

	// Add user to the queue
	log.Printf("Adding user %s (ELO: %d) to matchmaking queue.", username, userElo)
	request := MatchmakingRequest{
		Username:  username,
		Elo:       userElo,
		Timestamp: time.Now(),
	}
	matchmakingQueue = append(matchmakingQueue, request)

	w.WriteHeader(http.StatusAccepted) // 202 Accepted is a good status code for "request received, processing in background"
	json.NewEncoder(w).Encode(map[string]string{"message": "Searching for a match..."})
}

func runMatchmaker() {
	for {
		// Run every few seconds
		time.Sleep(3 * time.Second)

		matchmakingMutex.Lock()
		if len(matchmakingQueue) < 2 {
			matchmakingMutex.Unlock()
			continue // Not enough players to match
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

				// Dynamic ELO range: expands the longer you wait
				// After 10s, range is +/- 100. After 30s, +/- 300, etc.
				waitDuration := time.Since(p1.Timestamp).Seconds()
				eloRange := 50 + int(waitDuration*5)
				eloDiff := p1.Elo - p2.Elo
				if eloDiff < 0 {
					eloDiff = -eloDiff
				}

				if eloDiff <= eloRange {
					log.Printf("Match found between %s (%d) and %s (%d)", p1.Username, p1.Elo, p2.Username, p2.Elo)

					gamesMutex.Lock()
					gameID := uuid.New().String()
					gameSession := &GameSession{
						Game:        chess.NewGame(),
						PlayerWhite: p1.Username, // P1 is white by default
						PlayerBlack: p2.Username,
						State:       "in_progress",
					}
					games[gameID] = gameSession
					gamesMutex.Unlock()

					// Store the game ID so players can find it
					userGameMapMutex.Lock()
					userGameMap[p1.Username] = gameID
					userGameMap[p2.Username] = gameID
					userGameMapMutex.Unlock()

					// Mark players as matched so they aren't processed again
					matchedIndices[i] = true
					matchedIndices[j] = true
					break // Stop searching for a match for p1
				}
			}
		}

		// Rebuild the queue with only the unmatched players
		for i, req := range matchmakingQueue {
			if !matchedIndices[i] {
				newQueue = append(newQueue, req)
			}
		}
		matchmakingQueue = newQueue
		matchmakingMutex.Unlock()
	}
}

func getMatchmakingStatus(w http.ResponseWriter, r *http.Request) {
	username, ok := GetUsernameFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userGameMapMutex.RLock()
	gameID, found := userGameMap[username]
	userGameMapMutex.RUnlock()

	if found {

		sendJSONResponse(w, http.StatusOK, map[string]string{
			"status": "found",
			"gameID": gameID,
		})
	} else {
		sendJSONResponse(w, http.StatusOK, map[string]string{
			"status": "searching",
		})
	}
}

func getLeaderboard(w http.ResponseWriter, r *http.Request) {

	usersMutex.RLock()
	defer usersMutex.RUnlock()
	// only return top 4 users for now
	var leaderboard []LeaderboardEntry
	for username, data := range users {
		leaderboard = append(leaderboard, LeaderboardEntry{Username: username, Elo: data.Elo})
	}
	sort.Slice(leaderboard, func(i, j int) bool {
		return leaderboard[i].Elo > leaderboard[j].Elo
	})

	if len(leaderboard) > 4 {
		leaderboard = leaderboard[:4]
	}

	sendJSONResponse(w, http.StatusOK, leaderboard)
}
