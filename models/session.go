package models

// Session represents a user session tracking data (pilot or ATC)
type Session struct {
	CID         int    `json:"cid"`
	SessionType string `json:"session_type"` // 'pilot' or 'controller'
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	Details     string `json:"details"`
}
