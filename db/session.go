package db

import (
	"log"
	"time"
	"your_project_name/models"
)

// CreateSession creates a new session entry
func CreateSession(session models.Session) error {
	query := `INSERT INTO sessions (cid, session_type, start_time, details) VALUES ($1, $2, $3, $4)`
	_, err := DB.Exec(query, session.CID, session.SessionType, session.StartTime, session.Details)
	if err != nil {
		log.Printf("Error creating session for CID %d: %v", session.CID, err)
		return err
	}
	log.Printf("Session created for CID %d", session.CID)
	return nil
}

// EndSession ends a session and updates the end time and duration
func EndSession(cid int) error {
	query := `UPDATE sessions SET end_time = $1, duration = $2 WHERE cid = $3 AND end_time IS NULL`
	endTime := time.Now().Format(time.RFC3339)
	duration := int(time.Since(time.Parse(time.RFC3339, endTime)).Seconds())

	_, err := DB.Exec(query, endTime, duration, cid)
	if err != nil {
		log.Printf("Error ending session for CID %d: %v", cid, err)
		return err
	}
	log.Printf("Session ended for CID %d", cid)
	return nil
}
