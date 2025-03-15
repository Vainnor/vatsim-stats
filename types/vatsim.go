package types

import "time"

type VatsimData struct {
	General     General      `json:"general"`
	Pilots      []Pilot      `json:"pilots"`
	Controllers []Controller `json:"controllers"`
	Facilities  []Facility   `json:"facilities"`
	Ratings     []Rating     `json:"ratings"`
}

type General struct {
	Version          int       `json:"version"`
	Reload           int       `json:"reload"`
	Update           string    `json:"update"`
	UpdateTimestamp  time.Time `json:"update_timestamp"`
	ConnectedClients int       `json:"connected_clients"`
	UniqueUsers      int       `json:"unique_users"`
}

type Pilot struct {
	CID            int         `json:"cid"`
	Name           string      `json:"name"`
	Callsign       string      `json:"callsign"`
	Server         string      `json:"server"`
	PilotRating    int         `json:"pilot_rating"`
	MilitaryRating int         `json:"military_rating"`
	Latitude       float64     `json:"latitude"`
	Longitude      float64     `json:"longitude"`
	Altitude       int         `json:"altitude"`
	Groundspeed    int         `json:"groundspeed"`
	Transponder    string      `json:"transponder"`
	Heading        int         `json:"heading"`
	QNHiHg         float64     `json:"qnh_i_hg"`
	QNHMb          int         `json:"qnh_mb"`
	FlightPlan     *FlightPlan `json:"flight_plan,omitempty"`
	LogonTime      time.Time   `json:"logon_time"`
	LastUpdated    time.Time   `json:"last_updated"`
}

type FlightPlan struct {
	FlightRules         string `json:"flight_rules"`
	Aircraft            string `json:"aircraft"`
	AircraftFaa         string `json:"aircraft_faa"`
	AircraftShort       string `json:"aircraft_short"`
	Departure           string `json:"departure"`
	Arrival             string `json:"arrival"`
	Alternate           string `json:"alternate"`
	CruiseTAS           string `json:"cruise_tas"`
	Altitude            string `json:"altitude"`
	DepTime             string `json:"deptime"`
	EnrouteTime         string `json:"enroute_time"`
	FuelTime            string `json:"fuel_time"`
	Remarks             string `json:"remarks"`
	Route               string `json:"route"`
	RevisionID          int    `json:"revision_id"`
	AssignedTransponder string `json:"assigned_transponder"`
}

type Facility struct {
	ID    int    `json:"id"`
	Short string `json:"short"`
	Long  string `json:"long"`
}

type Rating struct {
	ID    int    `json:"id"`
	Short string `json:"short"`
	Long  string `json:"long"`
}

type Controller struct {
	CID         int       `json:"cid"`
	Name        string    `json:"name"`
	Callsign    string    `json:"callsign"`
	Frequency   string    `json:"frequency"`
	Facility    int       `json:"facility"`
	Rating      int       `json:"rating"`
	Server      string    `json:"server"`
	VisualRange int       `json:"visual_range"`
	TextAtis    []string  `json:"text_atis"`
	LastUpdated time.Time `json:"last_updated"`
	LogonTime   time.Time `json:"logon_time"`
}
