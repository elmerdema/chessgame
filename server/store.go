package main

import (
	"database/sql"

	"github.com/corentings/chess"
)

func CreateUser(db *sql.DB, username, hash string) error {
	_, err := db.Exec("INSERT INTO users (username, password_hash, elo) VALUES ($1, $2, $3)",
		username, hash, DefaultElo)
	return err
}

func GetUser(db *sql.DB, username string) (*Login, error) {
	var u Login

	// use COALESCE(column, '') to ensure we get an empty string instead of NULL???
	query := `
		SELECT 
			password_hash, 
			COALESCE(session_token, ''), 
			COALESCE(csrf_token, ''), 
			elo 
		FROM users 
		WHERE username=$1`

	err := db.QueryRow(query, username).
		Scan(&u.HashedPassword, &u.SessionToken, &u.CSRFToken, &u.Elo)

	if err != nil {
		return nil, err
	}
	return &u, nil
}

func UpdateSession(db *sql.DB, username, sessionToken, csrfToken string) error {
	_, err := db.Exec("UPDATE users SET session_token=$1, csrf_token=$2 WHERE username=$3", sessionToken, csrfToken, username)
	return err
}

func SaveGameState(db *sql.DB, gameID string, fen string, state string) error {
	_, err := db.Exec("UPDATE games SET fen=$1, state=$2, updated_at=NOW() WHERE id=$3",
		fen, state, gameID)
	return err
}

func UpdateEloInDB(white, black string, outcome chess.Outcome, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var eloWhite, eloBlack int

	err = tx.QueryRow("SELECT elo FROM users WHERE username=$1", white).Scan(&eloWhite)
	if err != nil {
		return err
	}

	err = tx.QueryRow("SELECT elo FROM users WHERE username=$1", black).Scan(&eloBlack)
	if err != nil {
		return err
	}

	newEloWhite, newEloBlack := calculateNewRatings(eloWhite, eloBlack, outcome)

	_, err = tx.Exec("UPDATE users SET elo=$1 WHERE username=$2", newEloWhite, white)
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE users SET elo=$1 WHERE username=$2", newEloBlack, black)
	if err != nil {
		return err
	}

	return tx.Commit()
}
