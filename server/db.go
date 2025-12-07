package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = os.Getenv("DATABASE_PASSWORD")
)

var psqlInfo = fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable", host, port, user, password)

func InitDB() (*sql.DB, error) {

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to the database")
	return db, nil
}

func CreateDatabase(dbname string) error {
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("CREATE DATABASE " + dbname)
	return err
}

func DropDatabase(dbname string) error {
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("DROP DATABASE IF EXISTS " + dbname)
	return err
}

func GetDBConnectionString(dbname string) string {
	return fmt.Sprintf("%s dbname=%s", psqlInfo, dbname)
}

func ConnectToDB(dbname string) (*sql.DB, error) {
	connStr := GetDBConnectionString(dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func CloseDB(db *sql.DB) {
	err := db.Close()
	if err != nil {
		log.Println("Error closing database:", err)
	}
}
