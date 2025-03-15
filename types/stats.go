package types

import "time"

type CollectionStats struct {
	LastUpdate      time.Time `json:"last_update"`
	TotalSnapshots  int64     `json:"total_snapshots"`
	ActivePilots    int       `json:"active_pilots"`
	ActiveATCs      int       `json:"active_atcs"`
	ActiveATIS      int       `json:"active_atis"`
	ProcessedPilots int64     `json:"processed_pilots"`
	StartTime       time.Time `json:"start_time"`
}
