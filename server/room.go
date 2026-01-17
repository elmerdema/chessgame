package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Message struct {
	client  *Client
	content []byte
}

type room struct {

	// clients holds all current clients,organized by clientid
	games map[string]map[*Client]bool

	// join is a channel for clients wishing to join the room.
	join chan *Client

	// leave is a channel for clients wishing to leave the room.
	leave chan *Client

	// forward is a channel that holds incoming messages that should be forwarded to the other clients.
	forward chan *Message
}

// newRoom create a new chat room

func newRoom() *room {
	return &room{
		forward: make(chan *Message),
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
			log.Printf("Client %s joined game room %s", client.Username, client.GameID)

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
			var data map[string]interface{}
			if err := json.Unmarshal(msg.content, &data); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}
			gameID, ok := data["gameID"].(string)
			if !ok {
				log.Printf("Message missing gameID: %s", string(msg.content))
				continue
			}

			// If it's a chat message, add the username to the payload.
			if msgType, ok := data["type"].(string); ok && msgType == "chat" {
				payload, payloadOk := data["payload"].(string)
				if payloadOk {
					data["payload"] = fmt.Sprintf("%s: %s", msg.client.Username, payload)
				}
			}

			broadcastMsg, err := json.Marshal(data)
			if err != nil {
				log.Printf("Error marshaling broadcast message: %v", err)
				continue
			}

			// Broadcast the message to all clients in the specific game room.
			if gameClients, ok := r.games[gameID]; ok {
				for client := range gameClients {
					// Don't send chat messages back to the original sender
					if msgType, ok := data["type"].(string); ok && msgType == "chat" && client == msg.client {
						continue
					}

					select {
					case client.receive <- broadcastMsg:
						// message sent
					default:
						// failed to send, clean up client
						delete(r.games[gameID], client)
						close(client.receive)
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
