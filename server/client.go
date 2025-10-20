package main

import (
	"github.com/gorilla/websocket"
)

// Client represents a single chatting user.
type Client struct {

	// socket is the web socket for this client.
	socket *websocket.Conn

	// receive is a channel to receive messages from other clients.
	receive chan []byte

	// room is the room this client is chatting in.
	room *room

	GameID string

	Username string
}

func (c *Client) read() {
	defer c.socket.Close()
	for {
		_, msg, err := c.socket.ReadMessage()
		if err != nil {
			return
		}
		c.room.forward <- &Message{client: c, content: msg}
	}
}

func (c *Client) write() {
	defer c.socket.Close()
	for msg := range c.receive {
		err := c.socket.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			return
		}
	}
}
