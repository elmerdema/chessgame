package main

import (
	"errors"
	"net/http"
)

var ErrorAuth = errors.New("Unauthorized")

func Authorize(r *http.Request) error {
	username := r.FormValue("username")
	user, ok := users[username]
	if !ok {
		return ErrorAuth
	}
	// get session token from the cookie
	st, err := r.Cookie("session_token")
	if err != nil || st.Value == "" || st.Value != user.SessionToken {
		return ErrorAuth
	}

	csrf := r.Header.Get("X-CSRF-Token")
	if csrf != user.CSRFToken || csrf == "" {
		return ErrorAuth
	}
	return nil
}
