package api

import "time"

type ConnectionType int

const (
	TypePilot ConnectionType = 1
	TypeATC   ConnectionType = 2
	TypeATIS  ConnectionType = 3
)

type ConnectionID struct {
	ID       int64          `json:"id"`
	VatsimID string         `json:"vatsim_id"`
	Type     ConnectionType `json:"type"`
	Rating   int            `json:"rating"`
	Callsign string         `json:"callsign"`
	Start    time.Time      `json:"start"`
	End      time.Time      `json:"end"`
	Server   string         `json:"server"`
}

type ATCStats struct {
	ConnectionID       ConnectionID `json:"connection_id"`
	AircraftTracked    int          `json:"aircrafttracked"`
	AircraftSeen       int          `json:"aircraftseen"`
	FlightsAmended     int          `json:"flightsamended"`
	HandoffsInitiated  int          `json:"handoffsinitiated"`
	HandoffsReceived   int          `json:"handoffsreceived"`
	HandoffsRefused    int          `json:"handoffsrefused"`
	SquawksAssigned    int          `json:"squawksassigned"`
	CruiseAltsModified int          `json:"cruisealtsmodified"`
	TempAltsModified   int          `json:"tempaltsmodified"`
	ScratchpadMods     int          `json:"scratchpadmods"`
}

type PilotStats struct {
	ConnectionID    ConnectionID `json:"connection_id"`
	TotalHours      int          `json:"total_hours"`
	TotalFlights    int          `json:"total_flights"`
	StudentHours    int          `json:"student_hours"`
	PPLHours        int          `json:"ppl_hours"`
	InstrumentHours int          `json:"instrument_hours"`
	CPLHours        int          `json:"cpl_hours"`
	ATPLHours       int          `json:"atpl_hours"`
	CurrentSession  *SessionInfo `json:"current_session,omitempty"`
}

type SessionInfo struct {
	StartTime     time.Time `json:"start_time"`
	Duration      int       `json:"duration_minutes"`
	HasFlightPlan bool      `json:"has_flight_plan"`
	Rating        int       `json:"rating"`
}

type ATISStats struct {
	ConnectionID ConnectionID `json:"connection_id"`
	Updates      int          `json:"updates"`
	Frequency    string       `json:"frequency"`
	Letter       string       `json:"letter"`
}

type MembershipResponse struct {
	Items []interface{} `json:"items"`
}
