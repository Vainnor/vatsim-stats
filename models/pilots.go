package models

// Pilot represents a pilot session from the JSON data
type Pilot struct {
	CID         int        `json:"cid"`
	Name        string     `json:"name"`
	Callsign    string     `json:"callsign"`
	Server      string     `json:"server"`
	PilotRating int        `json:"pilot_rating"`
	Latitude    float64    `json:"latitude"`
	Longitude   float64    `json:"longitude"`
	Altitude    int        `json:"altitude"`
	Groundspeed int        `json:"groundspeed"`
	Heading     int        `json:"heading"`
	FlightPlan  FlightPlan `json:"flight_plan"`
	LogonTime   string     `json:"logon_time"`
	LastUpdated string     `json:"last_updated"`
}

// FlightPlan represents the flight plan data
type FlightPlan struct {
	FlightRules string `json:"flight_rules"`
	Aircraft    string `json:"aircraft"`
	Departure   string `json:"departure"`
	Arrival     string `json:"arrival"`
	Alternate   string `json:"alternate"`
	Altitude    string `json:"altitude"`
}
