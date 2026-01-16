package main

import (
	"context"
	"database/sql"
	"net/http"
)

// contextKey is a custom type to use as a key for context values.
// This prevents collisions with other packages' context keys.
type contextKey string

const userContextKey = contextKey("username")

func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized: No session cookie", http.StatusUnauthorized)
			return
		}

		sessionToken := sessionCookie.Value
		var username string

		err = s.db.QueryRow("SELECT username FROM users WHERE session_token=$1", sessionToken).Scan(&username)

		if err == sql.ErrNoRows {
			http.Error(w, "Unauthorized: Invalid session token", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, username)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUsernameFromContext(r *http.Request) (string, bool) {
	username, ok := r.Context().Value(userContextKey).(string)
	return username, ok
}
