package models

// Controller represents an ATC controller session
type Controller struct {
	CID         int    `json:"cid"`
	Name        string `json:"name"`
	Callsign    string `json:"callsign"`
	Frequency   string `json:"frequency"`
	Facility    int    `json:"facility"`
	Rating      int    `json:"rating"`
	Server      string `json:"server"`
	VisualRange int    `json:"visual_range"`
	LogonTime   string `json:"logon_time"`
	LastUpdated string `json:"last_updated"`
}
