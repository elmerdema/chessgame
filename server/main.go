package main

//go run .
// curl -X POST -d "username=testuser&password=testpass" "http://localhost:8081/register"
// curl -X POST -d "username=testuser&password=testpass" "http://localhost:8081/login"

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rs/cors"
)

type Login struct {
	HashedPassword string
	SessionToken   string
	CSRFToken      string
}

// TODO: setup real db here in the future
var users = map[string]Login{}

func main() {
	var addr = flag.String("addr", ":8081", "The addr of the application.")
	flag.Parse()

	// Create a new router (mux). This is better practice than using the default.
	mux := http.NewServeMux()

	mux.HandleFunc("/login", login)
	mux.HandleFunc("/register", register)
	mux.HandleFunc("/logout", logout)
	mux.HandleFunc("/protected", protected)

	// --- CORS Configuration ---
	c := cors.New(cors.Options{
		// IMPORTANT: This must be the address of your frontend server
		AllowedOrigins:   []string{"http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	})

	// Wrap your entire mux with the CORS handler.
	handler := c.Handler(mux)

	// Start the API server on port 8081 with the CORS handler
	log.Println("Starting API server on", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		err := http.StatusMethodNotAllowed
		http.Error(w, "Invalid method", err)
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
	//ca asht %s????
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
