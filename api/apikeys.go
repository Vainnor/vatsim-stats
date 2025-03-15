package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/vainnor/vatsim-stats/db"
)

type APIKey struct {
	ID          int       `json:"id"`
	Key         string    `json:"key"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  time.Time `json:"last_used_at,omitempty"`
	IsActive    bool      `json:"is_active"`
}

// generateAPIKey generates a random 32-byte hex string
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// validateMasterKey checks if the provided key matches the master key
func validateMasterKey(key string) bool {
	return key == os.Getenv("MASTER_API_KEY")
}

// CreateAPIKey creates a new API key
func CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Validate master key from Authorization header
	authHeader := r.Header.Get("Authorization")
	if !validateMasterKey(authHeader) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate new API key
	key, err := generateAPIKey()
	if err != nil {
		http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}

	// Insert into database
	var apiKey APIKey
	err = db.DB.QueryRow(`
		INSERT INTO api_keys (key, description)
		VALUES ($1, $2)
		RETURNING id, key, description, created_at, is_active
	`, key, req.Description).Scan(
		&apiKey.ID,
		&apiKey.Key,
		&apiKey.Description,
		&apiKey.CreatedAt,
		&apiKey.IsActive,
	)
	if err != nil {
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiKey)
}

// DeleteAPIKey deletes an API key
func DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	// Validate master key from Authorization header
	authHeader := r.Header.Get("Authorization")
	if !validateMasterKey(authHeader) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Delete from database
	result, err := db.DB.Exec("DELETE FROM api_keys WHERE id = $1", req.ID)
	if err != nil {
		http.Error(w, "Failed to delete API key", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to get rows affected", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "API key not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListAPIKeys lists all API keys (only accessible with master key)
func ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	// Validate master key from Authorization header
	authHeader := r.Header.Get("Authorization")
	if !validateMasterKey(authHeader) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := db.DB.Query(`
		SELECT id, key, description, created_at, last_used_at, is_active
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, "Failed to list API keys", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var apiKeys []APIKey
	for rows.Next() {
		var apiKey APIKey
		var lastUsedAt *time.Time
		err := rows.Scan(
			&apiKey.ID,
			&apiKey.Key,
			&apiKey.Description,
			&apiKey.CreatedAt,
			&lastUsedAt,
			&apiKey.IsActive,
		)
		if err != nil {
			continue
		}
		if lastUsedAt != nil {
			apiKey.LastUsedAt = *lastUsedAt
		}
		apiKeys = append(apiKeys, apiKey)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiKeys)
}

// ValidateAPIKey checks if an API key is valid and updates its last_used_at timestamp
func ValidateAPIKey(key string) bool {
	var exists bool
	err := db.DB.QueryRow(`
		UPDATE api_keys
		SET last_used_at = NOW()
		WHERE key = $1 AND is_active = true
		RETURNING true
	`, key).Scan(&exists)
	return err == nil && exists
}
