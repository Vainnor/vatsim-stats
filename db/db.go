package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // Postgres driver
)

// Global DB variable
var DB *sql.DB

// Connect initializes the connection to the PostgreSQL database
func Connect() {
	// Get the database connection string from environment variables
	connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	// Open a database connection
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}

	// Verify the connection
	err = DB.Ping()
	if err != nil {
		log.Fatal("Error pinging the database: ", err)
	}

	log.Println("Connected to the database successfully!")
}

// Close closes the database connection
func Close() {
	if err := DB.Close(); err != nil {
		log.Fatal("Error closing the database: ", err)
	}
}
