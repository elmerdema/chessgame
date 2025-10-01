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

type Login struct {
	HashedPassword string
	SessionToken   string
	CSRFToken      string
}

// TODO: setup real db here in the future
var users = map[string]Login{}

var games = make(map[string]*GameSession)
var gamesMutex = &sync.RWMutex{}

func main() {
	var addr = flag.String("addr", ":8081", "The addr of the application.")
	flag.Parse()

	router := mux.NewRouter()

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/login", login).Methods("POST", "OPTIONS")
	api.HandleFunc("/register", register).Methods("POST", "OPTIONS")
	api.HandleFunc("/logout", logout).Methods("POST", "OPTIONS")
	api.HandleFunc("/protected", protected).Methods("POST", "OPTIONS")
	api.HandleFunc("/check-auth", checkAuth).Methods("GET", "OPTIONS")
	api.HandleFunc("/game/{id}/join", joinGame).Methods("POST", "OPTIONS")
	api.HandleFunc("/game/{id}", getGame).Methods("GET", "OPTIONS")
	api.HandleFunc("/matchmaking/find", findMatch).Methods("POST", "OPTIONS")
	room := newRoom()

	api.HandleFunc("/game/{id}/move", makeMoveHandler(room)).Methods("POST", "OPTIONS")
	router.Handle("/ws", room) //room is passed to handlers that need to broadcast it

	go room.run()

	c := cors.New(cors.Options{
		// localhost of the frontend server
		AllowedOrigins:   []string{"http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	})

	//CORS handler.
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
	if _, ok := users[username]; ok {
		err := http.StatusConflict
		http.Error(w, "User already exists", err)
		return
	}
	hashedPassword, _ := HashedPassword(password)
	users[username] = Login{
		HashedPassword: hashedPassword,
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
	if err := Authorize(r); err != nil {
		err := http.StatusUnauthorized
		http.Error(w, "Unauthorized", err)
		return
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
	username := r.FormValue("username")
	user, _ := users[username]
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
	username, err := getUsernameFromSession(r)
	if err != nil {
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
		username, err := getUsernameFromSession(r)
		if err != nil {
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
			http.Error(w, "Not your turn.", http.StatusForbidden) // 403 Forbidden is appropriate here
			return

		}

		var moveErr error
		// The move string from JS is in coordinate notation (e.g., "e2e4", "d1h5")
		moveStr := req.Move

		// We need to find the specific move from the list of valid moves.
		var foundMove *chess.Move
		for _, validMove := range gameSession.Game.ValidMoves() {
			// validMove.String() formats the move in the same "e2e4" coordinate style
			if validMove.String() == moveStr {
				foundMove = validMove
				break
			}
		}

		if foundMove != nil {
			// We found a valid move object, apply it directly
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

		response := MoveResponse{
			GameID:  gameID,
			Status:  "ok",
			NewFEN:  gameSession.Game.FEN(),
			Outcome: string(gameSession.Game.Outcome()),
			Turn:    gameSession.Game.Position().Turn().Name(),
		}

		broadcastMessage, err := json.Marshal(response)
		if err == nil {
			log.Printf("BROADCASTING MOVE for game %s: %s", gameID, string(broadcastMessage))
			room.forward <- broadcastMessage
		}

		sendJSONResponse(w, http.StatusOK, response)
	}
}

func getGame(w http.ResponseWriter, r *http.Request) {
	username, err := getUsernameFromSession(r)
	if err != nil {
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
	username, err := getUsernameFromSession(r)
	if err != nil {
		http.Error(w, "Unauthorized: Please login to find a match", http.StatusUnauthorized)
		return
	}

	gamesMutex.Lock()
	defer gamesMutex.Unlock()

	for gameID, session := range games {
		if session.State == "waiting" {
			// Found a waiting game
			log.Printf("Match found! User %s is joining game %s created by %s", username, gameID, session.PlayerWhite)
			session.PlayerBlack = username
			session.State = "in_progress"

			// TODO: In the future, send a WebSocket message to PlayerWhite to notify them!

			response := map[string]string{
				"gameID": gameID,
			}
			sendJSONResponse(w, http.StatusOK, response)
			return
		}
	}

	// --- 2. No waiting games found. Create a new one. ---
	log.Printf("No waiting games found. User %s is creating a new one.", username)
	gameID := uuid.New().String()
	gameSession := &GameSession{
		Game:        chess.NewGame(),
		PlayerWhite: username,
		PlayerBlack: "",
		State:       "waiting",
	}
	games[gameID] = gameSession

	log.Printf("New game created by %s with ID %s. Waiting for opponent.", username, gameID)

	response := map[string]string{
		"gameID": gameID,
	}
	sendJSONResponse(w, http.StatusCreated, response)
}
