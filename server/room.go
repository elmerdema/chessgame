package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type room struct {

	// clients holds all current clients,organized by clientid
	games map[string]map[*Client]bool

	// join is a channel for clients wishing to join the room.
	join chan *Client

	// leave is a channel for clients wishing to leave the room.
	leave chan *Client

	// forward is a channel that holds incoming messages that should be forwarded to the other clients.
	forward chan []byte
}

// newRoom create a new chat room

func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *Client),
		leave:   make(chan *Client),
		games:   make(map[string]map[*Client]bool),
	}
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			// When a client joins, create a game room if it doesn't exist
			if r.games[client.GameID] == nil {
				r.games[client.GameID] = make(map[*Client]bool)
			}
			// Add the client to their specific game room
			r.games[client.GameID][client] = true
			log.Printf("Client joined game room %s", client.GameID)

		case client := <-r.leave:
			// When a client leaves, remove them from their game room
			if _, ok := r.games[client.GameID]; ok {
				delete(r.games[client.GameID], client)
				close(client.receive)
				// If the room is now empty, delete it to save memory
				if len(r.games[client.GameID]) == 0 {
					delete(r.games, client.GameID)
					log.Printf("Game room %s closed.", client.GameID)
				}
			}

		case msg := <-r.forward:
			log.Printf("ROOM RECEIVED message to forward: %s", string(msg))
			// To broadcast, we must first find out which game this message is for.
			// We expect the message to be JSON with a "gameID" field.
			var moveData struct {
				GameID string `json:"gameID"`
			}

			if err := json.Unmarshal(msg, &moveData); err == nil {
				// If we successfully found a gameID, forward the message
				// to all clients in that specific game room.
				if gameClients, ok := r.games[moveData.GameID]; ok {
					for client := range gameClients {
						select {
						case client.receive <- msg:
							// message sent
						default:
							// failed to send
							delete(r.games[moveData.GameID], client)
							close(client.receive)
						}
					}
				}
			}
		}
	}
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{ReadBufferSize: socketBufferSize, WriteBufferSize: socketBufferSize,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	username, _ := GetUsernameFromContext(req)
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}

	// Get the GameID from the WebSocket URL /ws?gameId=...
	gameID := req.URL.Query().Get("gameId")
	if gameID == "" {
		log.Println("WebSocket connection rejected: no gameId provided.")
		socket.Close()
		return
	}

	client := &Client{
		socket:   socket,
		receive:  make(chan []byte, messageBufferSize),
		room:     r,
		GameID:   gameID,
		Username: username,
	}
	r.join <- client
	defer func() { r.leave <- client }()
	go client.write()
	client.read()
}
