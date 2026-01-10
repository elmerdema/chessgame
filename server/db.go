package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var (
	host = "localhost"
	port = 5432
	user = "postgres"
)

var psqlInfo string

func InitDB() (*sql.DB, error) {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error while loading .env file", err)
	}
	password := os.Getenv("DATABASE_PASSWORD")
	psqlInfo = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=chessgame sslmode=disable", host, port, user, password)
	schema, err := os.ReadFile("server/schemas/schema.sql")

	if err != nil {
		log.Fatal("Error when reading schema.sql", err)
	}

	db, err := sql.Open("postgres", psqlInfo)

	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to the database")

	_, err = db.Exec(string(schema))

	if err != nil {
		log.Fatal("Error when querying the DB", err)
	}

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
