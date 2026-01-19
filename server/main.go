package main

//go run .\server

import (
	"flag"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

var DefaultElo = 500
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
	go runMatchmaker(srv) // passedd server to matchmaker to access DB

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
