package main

import (
	"encoding/json"
	"net/http"
)

type MoveRequest struct {
	Move string `json:"move"`
}

type MoveResponse struct {
	GameID  string `json:"gameID"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	NewFEN  string `json:"newFEN"`
	Outcome string `json:"outcome"`
	Turn    string `json:"turn"`
}

func sendJSONResponse(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}
