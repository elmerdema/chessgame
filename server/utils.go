package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func HashedPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateToken(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Failde to generate token: %v", err)
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

func getUsernameFromSession(r *http.Request) (string, error) {
	sessionCookie, err := r.Cookie("session_token")
	if err != nil {
		return "", fmt.Errorf("no session cookie found")
	}

	sessionToken := sessionCookie.Value
	for username, loginData := range users {
		if loginData.SessionToken == sessionToken {
			return username, nil
		}
	}
	return "", fmt.Errorf("invalid session token")
}
