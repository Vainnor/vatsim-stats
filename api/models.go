package api

import "time"

// Airport Traffic Types
type AirportTraffic struct {
	ICAO              string             `json:"icao"`
	Timestamp         time.Time          `json:"timestamp"`
	ActiveControllers []ActiveController `json:"active_controllers"`
	ATIS              *ATISInfo          `json:"atis,omitempty"`
	Traffic           TrafficInfo        `json:"traffic"`
	Statistics        AirportStatistics  `json:"statistics"`
}

type ActiveController struct {
	Position   string           `json:"position"`
	Frequency  string           `json:"frequency"`
	Controller ControllerDetail `json:"controller"`
}

type ControllerDetail struct {
	CID    int    `json:"cid"`
	Rating int    `json:"rating"`
	Name   string `json:"name"`
}

type ATISInfo struct {
	Letter     string   `json:"letter"`
	Text       []string `json:"text"`
	Frequency  string   `json:"frequency"`
	Controller int      `json:"controller_cid"`
}

type TrafficInfo struct {
	Arrivals   []FlightInfo `json:"arrivals"`
	Departures []FlightInfo `json:"departures"`
}

type FlightInfo struct {
	Callsign    string    `json:"callsign"`
	Aircraft    string    `json:"aircraft"`
	Time        time.Time `json:"time"`
	Altitude    int       `json:"altitude"`
	Groundspeed int       `json:"groundspeed"`
	Origin      string    `json:"origin,omitempty"`
	Destination string    `json:"destination,omitempty"`
}

type AirportStatistics struct {
	HourlyMovements int `json:"hourly_movements"`
	DepartureCount  int `json:"departure_count"`
	ArrivalCount    int `json:"arrival_count"`
}

// Active Flights Search Types
type FlightSearchResponse struct {
	Flights []FlightSearchResult `json:"flights"`
	Total   int                  `json:"total"`
}

type FlightSearchResult struct {
	Callsign    string    `json:"callsign"`
	Aircraft    string    `json:"aircraft"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	Altitude    int       `json:"altitude"`
	Groundspeed int       `json:"groundspeed"`
	Position    Position  `json:"position"`
	FlightPlan  string    `json:"flight_plan,omitempty"`
	StartTime   time.Time `json:"start_time"`
}

type Position struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Heading   int     `json:"heading"`
}

// Network Statistics Types
type NetworkStatistics struct {
	Timestamp     time.Time       `json:"timestamp"`
	Global        GlobalStats     `json:"global"`
	ServerStats   []ServerStats   `json:"servers"`
	RegionStats   []RegionStats   `json:"regions"`
	RatingStats   []RatingStats   `json:"ratings"`
	AircraftStats []AircraftStats `json:"aircraft"`
}

type GlobalStats struct {
	TotalClients   int `json:"total_clients"`
	TotalPilots    int `json:"total_pilots"`
	TotalATCs      int `json:"total_atcs"`
	TotalObservers int `json:"total_observers"`
	ActivePilots   int `json:"active_pilots"`
}

type ServerStats struct {
	Name           string `json:"name"`
	ConnectedUsers int    `json:"connected_users"`
	Location       string `json:"location"`
}

type RegionStats struct {
	Region       string `json:"region"`
	ActivePilots int    `json:"active_pilots"`
	ActiveATCs   int    `json:"active_atcs"`
	TopAirports  []int  `json:"top_airports"`
}

type RatingStats struct {
	Rating     int `json:"rating"`
	PilotCount int `json:"pilot_count"`
	ATCCount   int `json:"atc_count"`
}

type AircraftStats struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// Route Statistics Types
type RouteStatistics struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	FlightCount   int    `json:"flight_count"`
	AircraftTypes []struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	} `json:"aircraft_types"`
	AvgFlightTime  int `json:"avg_flight_time"`
	AvgAltitude    int `json:"avg_altitude"`
	AvgGroundspeed int `json:"avg_groundspeed"`
	ActiveFlights  int `json:"active_flights"`
}

type PopularRoutes struct {
	Timestamp time.Time         `json:"timestamp"`
	Routes    []RouteStatistics `json:"routes"`
	Total     int               `json:"total"`
}

// FacilityStatistics represents statistics for an ATC facility
type FacilityStatistics struct {
	Facility    string           `json:"facility"`
	Timestamp   time.Time        `json:"timestamp"`
	Coverage    CoverageStats    `json:"coverage"`
	Traffic     TrafficStats     `json:"traffic"`
	Controllers []ControllerInfo `json:"controllers"`
	PeakTimes   []PeakTime       `json:"peak_times"`
}

// CoverageStats represents controller coverage statistics
type CoverageStats struct {
	TotalHours      int     `json:"total_hours"`
	LastDay         int     `json:"last_24h"`
	LastWeek        int     `json:"last_7d"`
	LastMonth       int     `json:"last_30d"`
	AveragePerDay   float64 `json:"avg_per_day"`
	CoveragePercent float64 `json:"coverage_percent"`
}

// TrafficStats represents traffic handling statistics
type TrafficStats struct {
	TotalFlights     int `json:"total_flights"`
	ArrivingFlights  int `json:"arriving_flights"`
	DepartingFlights int `json:"departing_flights"`
	TransitFlights   int `json:"transit_flights"`
	LastDay          int `json:"last_24h"`
	LastWeek         int `json:"last_7d"`
	LastMonth        int `json:"last_30d"`
}

// ControllerInfo represents a controller's activity at a facility
type ControllerInfo struct {
	CID           string    `json:"cid"`
	Name          string    `json:"name"`
	Rating        int       `json:"rating"`
	TotalHours    int       `json:"total_hours"`
	LastSeen      time.Time `json:"last_seen"`
	ActiveSession *struct {
		StartTime time.Time `json:"start_time"`
		Position  string    `json:"position"`
	} `json:"active_session,omitempty"`
}

// PeakTime represents a peak activity period
type PeakTime struct {
	Hour         int  `json:"hour"`
	DayOfWeek    int  `json:"day_of_week"`
	TrafficCount int  `json:"traffic_count"`
	Coverage     bool `json:"has_coverage"`
}

// NetworkTrends represents network activity trends
type NetworkTrends struct {
	Timestamp time.Time         `json:"timestamp"`
	Daily     []DailyActivity   `json:"daily"`
	Weekly    []WeeklyActivity  `json:"weekly"`
	Monthly   []MonthlyActivity `json:"monthly"`
}

// DailyActivity represents activity metrics for a single day
type DailyActivity struct {
	Date             time.Time `json:"date"`
	TotalPilots      int       `json:"total_pilots"`
	TotalControllers int       `json:"total_controllers"`
	PeakUsers        int       `json:"peak_users"`
	UniqueUsers      int       `json:"unique_users"`
}

// WeeklyActivity represents activity metrics for a week
type WeeklyActivity struct {
	WeekStart        time.Time `json:"week_start"`
	WeekEnd          time.Time `json:"week_end"`
	TotalPilots      int       `json:"total_pilots"`
	TotalControllers int       `json:"total_controllers"`
	PeakUsers        int       `json:"peak_users"`
	UniqueUsers      int       `json:"unique_users"`
}

// MonthlyActivity represents activity metrics for a month
type MonthlyActivity struct {
	Month            time.Time `json:"month"`
	TotalPilots      int       `json:"total_pilots"`
	TotalControllers int       `json:"total_controllers"`
	PeakUsers        int       `json:"peak_users"`
	UniqueUsers      int       `json:"unique_users"`
}
