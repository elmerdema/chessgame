package main

// go run ./server
// curl -X POST -d "username=testuser&password=testpass" "http://localhost:8081/register"
// curl -X POST -d "username=testuser&password=testpass" "http://localhost:8081/login"
import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
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

	r := newRoom()

	// Handle the WebSocket connection for the chat room.
	// This path must be different from the file server paths.
	http.Handle("/ws", r)

	// Serve all static files from the 'www' directory.
	// This will serve index.html for "/" and all other files like checkmate.css, index.js etc.
	http.Handle("/", http.FileServer(http.Dir("./www")))
	http.HandleFunc("/login", login)
	http.HandleFunc("/register", register)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/protected", protected)
	// get the room going
	go r.run()

	// start the web server
	log.Println("Starting web server on", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
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
		err := http.StatusMethodNotAllowed
		http.Error(w, "Invalid username/password", err)
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
