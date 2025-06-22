package main

//go run .
// curl -X POST -d "username=testuser&password=testpass" "http://localhost:8081/api/register"
// curl -X POST -d "username=testuser&password=testpass" "http://localhost:8081//apilogin"
//curl -X POST "http://localhost:8081/api/game/new"
// curl -X POST \
//   -H "Content-Type: application/json" \
//   -d '{"move": "e2e4"}' \
//   "http://localhost:8081/api/game/YOUR_GAME_ID_HERE/move"
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

type Login struct {
	HashedPassword string
	SessionToken   string
	CSRFToken      string
}

// TODO: setup real db here in the future
var users = map[string]Login{}

var games = make(map[string]*chess.Game)
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

	api.HandleFunc("/game/new", createNewGame).Methods("POST", "OPTIONS")

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

// lexoje prap kyt funksion
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

func createNewGame(w http.ResponseWriter, r *http.Request) {
	gamesMutex.Lock()
	defer gamesMutex.Unlock()

	gameID := uuid.New().String() //generate new unique id for tjhe game
	game := chess.NewGame()
	games[gameID] = game
	log.Printf("New game created with Id %s", &gameID)

	response := map[string]string{
		"gameID": gameID,
		"fen":    game.FEN(),
	}
	sendJSONResponse(w, http.StatusAccepted, response)
}

func makeMoveHandler(room *room) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		game, ok := games[gameID]
		if !ok {
			http.Error(w, "Game not found", http.StatusNotFound)
			return
		}

		//  core verification logic
		err := game.MoveStr(req.Move)
		if err != nil {
			sendJSONResponse(w, http.StatusBadRequest, MoveResponse{
				Status:  "error",
				Message: fmt.Sprintf("Illegal move: %s", err.Error()),
			})
			return
		}

		// if move was valid, the game state is updated.
		log.Printf("Game %s: Valid move %s. New FEN: %s", gameID, req.Move, game.FEN())

		// Authoritative response
		response := MoveResponse{
			Status:  "ok",
			NewFEN:  game.FEN(),
			Outcome: string(game.Outcome()),
			Turn:    game.Position().Turn().Name(),
		}

		//Broadcast the new state to all clients in the room via WebSockets
		broadcastMessage, err := json.Marshal(response)
		if err == nil {
			room.forward <- broadcastMessage
		}

		sendJSONResponse(w, http.StatusOK, response)
	}
}
