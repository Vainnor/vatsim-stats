package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/vainnor/vatsim-stats/api"
	"github.com/vainnor/vatsim-stats/collector"
	"github.com/vainnor/vatsim-stats/db"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Initialize database connection
	if err := db.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.CloseDB()

	// Get update interval from environment variable (default to 15 seconds)
	updateInterval := 15
	if intervalStr := os.Getenv("UPDATE_INTERVAL"); intervalStr != "" {
		if interval, err := strconv.Atoi(intervalStr); err == nil {
			updateInterval = interval
		}
	}

	// Create and start collector
	c := collector.NewCollector()
	ticker := time.NewTicker(time.Duration(updateInterval) * time.Second)
	defer ticker.Stop()

	// Set up API routes with the new router
	router := api.NewRouter(c)

	// Start the API server in a goroutine
	go func() {
		log.Printf("Starting API server on :8080")
		if err := http.ListenAndServe(":8080", router); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()

	log.Printf("Starting VATSIM data collector (update interval: %d seconds)", updateInterval)

	// Initial collection
	if err := c.FetchAndStore(); err != nil {
		log.Printf("Error collecting data: %v", err)
	}

	// Continuous collection
	for range ticker.C {
		if err := c.FetchAndStore(); err != nil {
			log.Printf("Error collecting data: %v", err)
		}
	}
}
