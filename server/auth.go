package main

import (
	"context"
	"net/http"
)

// contextKey is a custom type to use as a key for context values.
// This prevents collisions with other packages' context keys.
type contextKey string

const userContextKey = contextKey("username")

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized: No session cookie", http.StatusUnauthorized)
			return
		}

		sessionToken := sessionCookie.Value

		usersMutex.RLock()
		var foundUsername string
		for username, loginData := range users {
			if loginData.SessionToken == sessionToken && sessionToken != "" {
				foundUsername = username
				break
			}
		}
		usersMutex.RUnlock()

		if foundUsername == "" {
			http.Error(w, "Unauthorized: Invalid session token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, foundUsername)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUsernameFromContext(r *http.Request) (string, bool) {
	username, ok := r.Context().Value(userContextKey).(string)
	return username, ok
}
