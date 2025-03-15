package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/vainnor/vatsim-stats/db"
	"github.com/vainnor/vatsim-stats/types"
)

func GetMembershipHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["cid"]
	connectionType := strings.ToLower(vars["type"])

	var typeID ConnectionType
	switch connectionType {
	case "atc":
		typeID = TypeATC
	case "atis":
		typeID = TypeATIS
	case "pilot":
		typeID = TypePilot
	default:
		http.Error(w, "Invalid connection type. Must be 'atc', 'atis', or 'pilot'", http.StatusBadRequest)
		return
	}

	items, err := getMembershipData(cid, typeID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "No data found"})
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MembershipResponse{Items: items})
}

func getMembershipData(cid string, typeID ConnectionType) ([]interface{}, error) {
	var items []interface{}

	// Get all connections for the user and type
	rows, err := db.DB.Query(`
		SELECT 
			c.id, c.vatsim_id, c.rating, c.callsign, 
			c.start_time, c.end_time, c.server
		FROM connections c
		WHERE c.vatsim_id = $1 AND c.type = $2
		ORDER BY c.start_time DESC
	`, cid, typeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var conn ConnectionID
		err := rows.Scan(
			&conn.ID,
			&conn.VatsimID,
			&conn.Rating,
			&conn.Callsign,
			&conn.Start,
			&conn.End,
			&conn.Server,
		)
		if err != nil {
			return nil, err
		}
		conn.Type = typeID

		switch typeID {
		case TypeATC:
			stats, err := getATCStats(conn.ID)
			if err != nil {
				return nil, err
			}
			stats.ConnectionID = conn
			items = append(items, stats)

		case TypePilot:
			stats, err := getPilotStats(conn.ID)
			if err != nil {
				return nil, err
			}
			stats.ConnectionID = conn
			items = append(items, stats)

		case TypeATIS:
			stats, err := getATISStats(conn.ID)
			if err != nil {
				return nil, err
			}
			stats.ConnectionID = conn
			items = append(items, stats)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func getATCStats(connID int64) (*ATCStats, error) {
	stats := &ATCStats{}
	err := db.DB.QueryRow(`
		SELECT 
			aircraft_tracked, aircraft_seen, flights_amended,
			handoffs_initiated, handoffs_received, handoffs_refused,
			squawks_assigned, cruise_alts_modified, temp_alts_modified,
			scratchpad_mods
		FROM atc_stats
		WHERE connection_id = $1
	`, connID).Scan(
		&stats.AircraftTracked, &stats.AircraftSeen, &stats.FlightsAmended,
		&stats.HandoffsInitiated, &stats.HandoffsReceived, &stats.HandoffsRefused,
		&stats.SquawksAssigned, &stats.CruiseAltsModified, &stats.TempAltsModified,
		&stats.ScratchpadMods,
	)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func getPilotStats(connID int64) (*PilotStats, error) {
	stats := &PilotStats{}

	// Get the connection details
	var conn ConnectionID
	err := db.DB.QueryRow(`
		SELECT id, vatsim_id, rating, callsign, start_time, end_time, server
		FROM connections WHERE id = $1
	`, connID).Scan(
		&conn.ID,
		&conn.VatsimID,
		&conn.Rating,
		&conn.Callsign,
		&conn.Start,
		&conn.End,
		&conn.Server,
	)
	if err != nil {
		return nil, err
	}
	stats.ConnectionID = conn

	// Get the total stats for this pilot
	err = db.DB.QueryRow(`
		SELECT 
			total_hours, total_flights,
			student_hours, ppl_hours, instrument_hours,
			cpl_hours, atpl_hours
		FROM pilot_total_stats
		WHERE vatsim_id = $1
	`, conn.VatsimID).Scan(
		&stats.TotalHours, &stats.TotalFlights,
		&stats.StudentHours, &stats.PPLHours, &stats.InstrumentHours,
		&stats.CPLHours, &stats.ATPLHours,
	)
	if err == sql.ErrNoRows {
		// If no stats exist yet, return zeros
		return stats, nil
	}
	if err != nil {
		return nil, err
	}

	// Check if there's an active session
	var startTime time.Time
	var rating int
	var hasFlightPlan bool

	err = db.DB.QueryRow(`
		SELECT p.logon_time, p.pilot_rating, 
		       CASE WHEN fp.id IS NOT NULL THEN true ELSE false END as has_flight_plan
		FROM pilots p
		LEFT JOIN flight_plans fp ON fp.pilot_id = p.id
		WHERE p.cid = $1
		ORDER BY p.last_updated DESC
		LIMIT 1
	`, conn.VatsimID).Scan(&startTime, &rating, &hasFlightPlan)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err != sql.ErrNoRows {
		stats.CurrentSession = &SessionInfo{
			StartTime:     startTime,
			Duration:      int(time.Since(startTime).Minutes()),
			HasFlightPlan: hasFlightPlan,
			Rating:        rating,
		}
	}

	return stats, nil
}

func getATISStats(connID int64) (*ATISStats, error) {
	stats := &ATISStats{}
	err := db.DB.QueryRow(`
		SELECT updates, frequency, letter
		FROM atis_stats
		WHERE connection_id = $1
	`, connID).Scan(
		&stats.Updates, &stats.Frequency, &stats.Letter,
	)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// Add collector stats handler
func GetCollectorStats(collector interface{ GetStats() types.CollectionStats }) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(collector.GetStats())
	}
}

// Add debug endpoint
func GetPilotDebug(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["cid"]

	type DebugInfo struct {
		HasConnection     bool      `json:"has_connection"`
		IsCurrentlyOnline bool      `json:"is_currently_online"`
		HasStats          bool      `json:"has_stats"`
		LastSeen          time.Time `json:"last_seen,omitempty"`
		Rating            int       `json:"rating,omitempty"`
		Callsign          string    `json:"callsign,omitempty"`
	}

	debug := DebugInfo{}

	// Check completed connections
	err := db.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM connections 
			WHERE vatsim_id = $1 AND type = $2
		)
	`, cid, TypePilot).Scan(&debug.HasConnection)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if pilot is currently online
	err = db.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM pilots 
			WHERE cid = $1 AND last_updated > NOW() - INTERVAL '5 minutes'
		)
	`, cid).Scan(&debug.IsCurrentlyOnline)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check pilot_total_stats table
	err = db.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM pilot_total_stats 
			WHERE vatsim_id = $1
		)
	`, cid).Scan(&debug.HasStats)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Get latest pilot info if any
	row := db.DB.QueryRow(`
		SELECT last_updated, pilot_rating, callsign
		FROM pilots 
		WHERE cid = $1 
		ORDER BY last_updated DESC 
		LIMIT 1
	`, cid)
	err = row.Scan(&debug.LastSeen, &debug.Rating, &debug.Callsign)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(debug)
}

// GetAirportTraffic returns current traffic information for a specific airport
func GetAirportTraffic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	icao := strings.ToUpper(vars["icao"])

	// Get active controllers at this airport
	rows, err := db.DB.Query(`
		SELECT 
			c.callsign, c.frequency, c.cid, c.rating, p.name
		FROM pilots p
		JOIN controllers c ON c.callsign LIKE $1
		WHERE c.last_updated > NOW() - INTERVAL '5 minutes'
		ORDER BY c.callsign
	`, icao+"_%")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	traffic := AirportTraffic{
		ICAO:      icao,
		Timestamp: time.Now(),
		Traffic: TrafficInfo{
			Arrivals:   make([]FlightInfo, 0),
			Departures: make([]FlightInfo, 0),
		},
	}

	// Process controllers
	for rows.Next() {
		var ctrl ActiveController
		err := rows.Scan(
			&ctrl.Position,
			&ctrl.Frequency,
			&ctrl.Controller.CID,
			&ctrl.Controller.Rating,
			&ctrl.Controller.Name,
		)
		if err != nil {
			continue
		}
		traffic.ActiveControllers = append(traffic.ActiveControllers, ctrl)

		// Check if this is an ATIS position
		if strings.HasSuffix(ctrl.Position, "_ATIS") {
			atis := &ATISInfo{
				Frequency:  ctrl.Frequency,
				Controller: ctrl.Controller.CID,
			}
			traffic.ATIS = atis
		}
	}

	// Get arrivals
	rows, err = db.DB.Query(`
		SELECT 
			p.callsign, fp.aircraft_short, p.altitude, 
			p.groundspeed, fp.departure, fp.arrival,
			p.logon_time
		FROM pilots p
		JOIN flight_plans fp ON fp.pilot_id = p.id
		WHERE fp.arrival = $1
		AND p.last_updated > NOW() - INTERVAL '5 minutes'
	`, icao)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var flight FlightInfo
		err := rows.Scan(
			&flight.Callsign,
			&flight.Aircraft,
			&flight.Altitude,
			&flight.Groundspeed,
			&flight.Origin,
			&flight.Destination,
			&flight.Time,
		)
		if err != nil {
			continue
		}
		traffic.Traffic.Arrivals = append(traffic.Traffic.Arrivals, flight)
	}

	// Get departures
	rows, err = db.DB.Query(`
		SELECT 
			p.callsign, fp.aircraft_short, p.altitude, 
			p.groundspeed, fp.departure, fp.arrival,
			p.logon_time
		FROM pilots p
		JOIN flight_plans fp ON fp.pilot_id = p.id
		WHERE fp.departure = $1
		AND p.last_updated > NOW() - INTERVAL '5 minutes'
	`, icao)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var flight FlightInfo
		err := rows.Scan(
			&flight.Callsign,
			&flight.Aircraft,
			&flight.Altitude,
			&flight.Groundspeed,
			&flight.Origin,
			&flight.Destination,
			&flight.Time,
		)
		if err != nil {
			continue
		}
		traffic.Traffic.Departures = append(traffic.Traffic.Departures, flight)
	}

	// Calculate statistics
	traffic.Statistics = AirportStatistics{
		HourlyMovements: len(traffic.Traffic.Arrivals) + len(traffic.Traffic.Departures),
		ArrivalCount:    len(traffic.Traffic.Arrivals),
		DepartureCount:  len(traffic.Traffic.Departures),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(traffic)
}

// SearchFlights allows searching for active flights based on various criteria
func SearchFlights(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	callsign := query.Get("callsign")
	aircraft := query.Get("aircraft")
	origin := query.Get("origin")
	destination := query.Get("destination")

	// Build the SQL query based on provided parameters
	sqlQuery := `
		SELECT 
			p.callsign, fp.aircraft_short, fp.departure, 
			fp.arrival, p.altitude, p.groundspeed,
			p.latitude, p.longitude, p.heading,
			fp.route, p.logon_time
		FROM pilots p
		JOIN flight_plans fp ON fp.pilot_id = p.id
		WHERE p.last_updated > NOW() - INTERVAL '5 minutes'
	`
	params := make([]interface{}, 0)
	paramCount := 1

	if callsign != "" {
		sqlQuery += fmt.Sprintf(" AND p.callsign LIKE $%d", paramCount)
		params = append(params, "%"+callsign+"%")
		paramCount++
	}
	if aircraft != "" {
		sqlQuery += fmt.Sprintf(" AND fp.aircraft_short LIKE $%d", paramCount)
		params = append(params, "%"+aircraft+"%")
		paramCount++
	}
	if origin != "" {
		sqlQuery += fmt.Sprintf(" AND fp.departure = $%d", paramCount)
		params = append(params, strings.ToUpper(origin))
		paramCount++
	}
	if destination != "" {
		sqlQuery += fmt.Sprintf(" AND fp.arrival = $%d", paramCount)
		params = append(params, strings.ToUpper(destination))
	}

	rows, err := db.DB.Query(sqlQuery, params...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response := FlightSearchResponse{
		Flights: make([]FlightSearchResult, 0),
	}

	for rows.Next() {
		var flight FlightSearchResult
		err := rows.Scan(
			&flight.Callsign,
			&flight.Aircraft,
			&flight.Origin,
			&flight.Destination,
			&flight.Altitude,
			&flight.Groundspeed,
			&flight.Position.Latitude,
			&flight.Position.Longitude,
			&flight.Position.Heading,
			&flight.FlightPlan,
			&flight.StartTime,
		)
		if err != nil {
			continue
		}
		response.Flights = append(response.Flights, flight)
	}

	response.Total = len(response.Flights)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetNetworkStatisticsHandler returns a handler that uses the collector's data
func GetNetworkStatisticsHandler(collector Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := collector.GetStats()
		data, err := collector.GetCurrentData()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error getting VATSIM data: %v", err), http.StatusInternalServerError)
			return
		}

		networkStats := NetworkStatistics{
			Timestamp: time.Now(),
			Global: GlobalStats{
				TotalClients:   stats.ActivePilots + stats.ActiveATCs + stats.ActiveATIS,
				TotalPilots:    stats.ActivePilots,
				TotalATCs:      stats.ActiveATCs,
				TotalObservers: stats.ActiveATIS,
				ActivePilots:   stats.ActivePilots,
			},
		}

		// Count users per server
		serverCounts := make(map[string]int)
		for _, pilot := range data.Pilots {
			serverCounts[pilot.Server]++
		}
		for _, controller := range data.Controllers {
			serverCounts[controller.Server]++
		}

		// Convert server counts to stats
		for server, count := range serverCounts {
			networkStats.ServerStats = append(networkStats.ServerStats, ServerStats{
				Name:           server,
				ConnectedUsers: count,
			})
		}

		// Count aircraft types
		aircraftCounts := make(map[string]int)
		for _, pilot := range data.Pilots {
			if pilot.FlightPlan != nil {
				aircraftCounts[pilot.FlightPlan.AircraftShort]++
			}
		}

		// Convert aircraft counts to stats (top 10)
		for aircraft, count := range aircraftCounts {
			networkStats.AircraftStats = append(networkStats.AircraftStats, AircraftStats{
				Type:  aircraft,
				Count: count,
			})
		}

		// Sort aircraft stats by count and take top 10
		sort.Slice(networkStats.AircraftStats, func(i, j int) bool {
			return networkStats.AircraftStats[i].Count > networkStats.AircraftStats[j].Count
		})
		if len(networkStats.AircraftStats) > 10 {
			networkStats.AircraftStats = networkStats.AircraftStats[:10]
		}

		// Count ratings
		pilotRatings := make(map[int]int)
		atcRatings := make(map[int]int)
		for _, pilot := range data.Pilots {
			pilotRatings[pilot.PilotRating]++
		}
		for _, controller := range data.Controllers {
			if len(controller.TextAtis) == 0 {
				atcRatings[controller.Rating]++
			}
		}

		// Convert rating counts to stats
		ratingSet := make(map[int]bool)
		for rating := range pilotRatings {
			ratingSet[rating] = true
		}
		for rating := range atcRatings {
			ratingSet[rating] = true
		}
		for rating := range ratingSet {
			networkStats.RatingStats = append(networkStats.RatingStats, RatingStats{
				Rating:     rating,
				PilotCount: pilotRatings[rating],
				ATCCount:   atcRatings[rating],
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(networkStats)
	}
}

// GetNetworkStatistics is deprecated, use GetNetworkStatisticsHandler instead
func GetNetworkStatistics(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "This endpoint has been updated. Please retry your request.", http.StatusGone)
}

// GetPopularRoutes returns the most frequently flown routes
func GetPopularRoutes(w http.ResponseWriter, r *http.Request) {
	// Get query parameters for filtering
	limit := 10 // Default to top 10 routes
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Query for popular routes
	rows, err := db.DB.Query(`
		WITH route_summary AS (
			SELECT 
				fp.departure as origin,
				fp.arrival as destination,
				COUNT(DISTINCT p.cid) as flight_count,
				COALESCE(ROUND(AVG(NULLIF(p.altitude, 0))), 0) as avg_altitude,
				COALESCE(ROUND(AVG(NULLIF(p.groundspeed, 0))), 0) as avg_groundspeed,
				array_agg(DISTINCT p.logon_time::text) as flight_times,
				(
					SELECT json_agg(ac)
					FROM (
						SELECT aircraft_short as type, COUNT(*) as count
						FROM (
							SELECT DISTINCT ON (p2.cid) fp2.aircraft_short
							FROM pilots p2
							JOIN flight_plans fp2 ON fp2.pilot_id = p2.id
							WHERE fp2.departure = fp.departure 
							AND fp2.arrival = fp.arrival
							AND p2.last_updated > NOW() - INTERVAL '24 hours'
							ORDER BY p2.cid, p2.last_updated DESC
						) unique_flights
						GROUP BY aircraft_short
					) ac
				) as aircraft_counts
			FROM pilots p
			JOIN flight_plans fp ON fp.pilot_id = p.id
			WHERE p.last_updated > NOW() - INTERVAL '24 hours'
			GROUP BY fp.departure, fp.arrival
			HAVING COUNT(DISTINCT p.cid) > 0
			ORDER BY COUNT(DISTINCT p.cid) DESC
			LIMIT $1
		)
		SELECT 
			rs.origin,
			rs.destination,
			rs.flight_count,
			rs.avg_altitude::integer,
			rs.avg_groundspeed::integer,
			COALESCE(rs.aircraft_counts::text, '[]'),
			COALESCE(rs.flight_times, ARRAY[]::text[]),
			(
				SELECT COUNT(DISTINCT p2.cid)
				FROM pilots p2
				JOIN flight_plans fp2 ON fp2.pilot_id = p2.id
				WHERE fp2.departure = rs.origin 
				AND fp2.arrival = rs.destination
				AND p2.last_updated > NOW() - INTERVAL '5 minutes'
			) as active_flights
		FROM route_summary rs
	`, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response := PopularRoutes{
		Timestamp: time.Now(),
		Routes:    make([]RouteStatistics, 0),
	}

	for rows.Next() {
		var route RouteStatistics
		var aircraftCountsJSON []byte
		var flightTimes []string
		err := rows.Scan(
			&route.Origin,
			&route.Destination,
			&route.FlightCount,
			&route.AvgAltitude,
			&route.AvgGroundspeed,
			&aircraftCountsJSON,
			pq.Array(&flightTimes),
			&route.ActiveFlights,
		)
		if err != nil {
			continue
		}

		// Calculate average flight time
		var totalMinutes int
		for _, timeStr := range flightTimes {
			if timeStr == "" {
				continue
			}
			logonTime, err := time.Parse(time.RFC3339, timeStr)
			if err != nil {
				continue
			}
			totalMinutes += int(time.Since(logonTime).Minutes())
		}
		if len(flightTimes) > 0 {
			route.AvgFlightTime = totalMinutes / len(flightTimes)
		}

		// Parse aircraft counts from JSON
		type aircraftCount struct {
			Type  string `json:"type"`
			Count int    `json:"count"`
		}
		var counts []aircraftCount
		if err := json.Unmarshal(aircraftCountsJSON, &counts); err != nil {
			continue
		}

		// Convert to route statistics format
		route.AircraftTypes = make([]struct {
			Type  string `json:"type"`
			Count int    `json:"count"`
		}, 0)
		for _, ac := range counts {
			route.AircraftTypes = append(route.AircraftTypes, struct {
				Type  string `json:"type"`
				Count int    `json:"count"`
			}{
				Type:  ac.Type,
				Count: ac.Count,
			})
		}

		response.Routes = append(response.Routes, route)
	}

	response.Total = len(response.Routes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRouteStats returns statistics for a specific route
func GetRouteStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	origin := strings.ToUpper(vars["origin"])
	destination := strings.ToUpper(vars["destination"])

	var route RouteStatistics
	route.Origin = origin
	route.Destination = destination
	route.AircraftTypes = make([]struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}, 0)

	// Query for route statistics
	var aircraftCountsJSON []byte
	var flightTimes []string

	err := db.DB.QueryRow(`
		WITH route_data AS (
			SELECT 
				COUNT(DISTINCT p.cid) as flight_count,
				COALESCE(ROUND(AVG(NULLIF(p.altitude, 0))), 0) as avg_altitude,
				COALESCE(ROUND(AVG(NULLIF(p.groundspeed, 0))), 0) as avg_groundspeed,
				array_agg(DISTINCT p.logon_time::text) as flight_times,
				(
					SELECT json_agg(ac)
					FROM (
						SELECT aircraft_short as type, COUNT(*) as count
						FROM (
							SELECT DISTINCT ON (p2.cid) fp2.aircraft_short
							FROM pilots p2
							JOIN flight_plans fp2 ON fp2.pilot_id = p2.id
							WHERE fp2.departure = $1 
							AND fp2.arrival = $2
							AND p2.last_updated > NOW() - INTERVAL '24 hours'
							ORDER BY p2.cid, p2.last_updated DESC
						) unique_flights
						GROUP BY aircraft_short
					) ac
				) as aircraft_counts
			FROM pilots p
			JOIN flight_plans fp ON fp.pilot_id = p.id
			WHERE fp.departure = $1 
			AND fp.arrival = $2
			AND p.last_updated > NOW() - INTERVAL '24 hours'
		)
		SELECT 
			COALESCE(rd.flight_count, 0),
			COALESCE(rd.avg_altitude::integer, 0),
			COALESCE(rd.avg_groundspeed::integer, 0),
			COALESCE(rd.aircraft_counts::text, '[]'),
			COALESCE(rd.flight_times, ARRAY[]::text[]),
			(
				SELECT COUNT(DISTINCT p2.cid)
				FROM pilots p2
				JOIN flight_plans fp2 ON fp2.pilot_id = p2.id
				WHERE fp2.departure = rs.origin 
				AND fp2.arrival = rs.destination
				AND p2.last_updated > NOW() - INTERVAL '5 minutes'
			) as active_flights
		FROM route_data rd
	`, origin, destination).Scan(
		&route.FlightCount,
		&route.AvgAltitude,
		&route.AvgGroundspeed,
		&aircraftCountsJSON,
		pq.Array(&flightTimes),
		&route.ActiveFlights,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty stats instead of 404
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(route)
			return
		}
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate average flight time
	var totalMinutes int
	for _, timeStr := range flightTimes {
		if timeStr == "" {
			continue
		}
		logonTime, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			continue
		}
		totalMinutes += int(time.Since(logonTime).Minutes())
	}
	if len(flightTimes) > 0 {
		route.AvgFlightTime = totalMinutes / len(flightTimes)
	}

	// Parse aircraft counts from JSON
	type aircraftCount struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}
	var counts []aircraftCount
	if err := json.Unmarshal(aircraftCountsJSON, &counts); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing aircraft counts: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to route statistics format
	for _, ac := range counts {
		route.AircraftTypes = append(route.AircraftTypes, struct {
			Type  string `json:"type"`
			Count int    `json:"count"`
		}{
			Type:  ac.Type,
			Count: ac.Count,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(route)
}

// GetFacilityStats returns statistics for a specific ATC facility
func GetFacilityStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	facility := strings.ToUpper(vars["facility"])

	stats := FacilityStatistics{
		Facility:    facility,
		Timestamp:   time.Now(),
		Controllers: make([]ControllerInfo, 0),
	}

	// Get coverage statistics
	err := db.DB.QueryRow(`
		WITH coverage_data AS (
			SELECT 
				COUNT(DISTINCT c.id) * 15 as total_minutes,
				COUNT(DISTINCT CASE WHEN c.last_updated > NOW() - INTERVAL '24 hours' THEN c.id END) * 15 as last_day_minutes,
				COUNT(DISTINCT CASE WHEN c.last_updated > NOW() - INTERVAL '7 days' THEN c.id END) * 15 as last_week_minutes,
				COUNT(DISTINCT CASE WHEN c.last_updated > NOW() - INTERVAL '30 days' THEN c.id END) * 15 as last_month_minutes
			FROM controllers c
			WHERE c.callsign LIKE $1
			AND c.last_updated > NOW() - INTERVAL '30 days'
		)
		SELECT 
			total_minutes / 60,
			last_day_minutes / 60,
			last_week_minutes / 60,
			last_month_minutes / 60,
			(last_day_minutes::float / (24 * 60)) * 100
		FROM coverage_data
	`, facility+"_%").Scan(
		&stats.Coverage.TotalHours,
		&stats.Coverage.LastDay,
		&stats.Coverage.LastWeek,
		&stats.Coverage.LastMonth,
		&stats.Coverage.CoveragePercent,
	)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate average per day
	if stats.Coverage.LastMonth > 0 {
		stats.Coverage.AveragePerDay = float64(stats.Coverage.LastMonth) / 30.0
	}

	// Get traffic statistics
	err = db.DB.QueryRow(`
		WITH traffic_data AS (
			SELECT 
				COUNT(DISTINCT fp.id) as total_flights,
				COUNT(DISTINCT CASE WHEN fp.arrival = $1 THEN fp.id END) as arrivals,
				COUNT(DISTINCT CASE WHEN fp.departure = $1 THEN fp.id END) as departures,
				COUNT(DISTINCT CASE WHEN p.last_updated > NOW() - INTERVAL '24 hours' THEN fp.id END) as last_day,
				COUNT(DISTINCT CASE WHEN p.last_updated > NOW() - INTERVAL '7 days' THEN fp.id END) as last_week,
				COUNT(DISTINCT CASE WHEN p.last_updated > NOW() - INTERVAL '30 days' THEN fp.id END) as last_month
			FROM pilots p
			JOIN flight_plans fp ON fp.pilot_id = p.id
			WHERE (fp.arrival = $1 OR fp.departure = $1)
			AND p.last_updated > NOW() - INTERVAL '30 days'
		)
		SELECT 
			total_flights,
			arrivals,
			departures,
			last_day,
			last_week,
			last_month
		FROM traffic_data
	`, facility).Scan(
		&stats.Traffic.TotalFlights,
		&stats.Traffic.ArrivingFlights,
		&stats.Traffic.DepartingFlights,
		&stats.Traffic.LastDay,
		&stats.Traffic.LastWeek,
		&stats.Traffic.LastMonth,
	)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Get controller information
	rows, err := db.DB.Query(`
		WITH controller_stats AS (
			SELECT DISTINCT ON (c.cid)
				c.cid,
				COALESCE(p.name, 'Unknown') as name,
				c.rating,
				(
					SELECT COUNT(DISTINCT c2.id) * 15 / 60
					FROM controllers c2 
					WHERE c2.cid = c.cid 
					AND c2.callsign SIMILAR TO $1
					AND c2.last_updated > NOW() - INTERVAL '30 days'
				) as total_hours,
				MAX(c.last_updated) as last_seen,
				CASE 
					WHEN EXISTS (
						SELECT 1 FROM controllers c3 
						WHERE c3.cid = c.cid 
						AND c3.callsign SIMILAR TO $1
						AND c3.last_updated > NOW() - INTERVAL '5 minutes'
					) THEN 
						json_build_object(
							'start_time', (
								SELECT c4.logon_time 
								FROM controllers c4 
								WHERE c4.cid = c.cid 
								AND c4.callsign SIMILAR TO $1
								AND c4.last_updated > NOW() - INTERVAL '5 minutes'
								ORDER BY c4.last_updated DESC
								LIMIT 1
							),
							'position', (
								SELECT c4.callsign 
								FROM controllers c4 
								WHERE c4.cid = c.cid 
								AND c4.callsign SIMILAR TO $1
								AND c4.last_updated > NOW() - INTERVAL '5 minutes'
								ORDER BY c4.last_updated DESC
								LIMIT 1
							)
						)
					ELSE NULL
				END as active_session
			FROM controllers c
			LEFT JOIN pilots p ON p.cid = c.cid
			WHERE c.callsign SIMILAR TO $1
			AND c.last_updated > NOW() - INTERVAL '30 days'
			GROUP BY c.cid, p.name, c.rating
			ORDER BY c.cid, MAX(c.last_updated) DESC
		)
		SELECT 
			cid,
			name,
			rating,
			total_hours,
			last_seen,
			active_session::text
		FROM controller_stats
		WHERE total_hours > 0
		ORDER BY total_hours DESC
	`, facility+"_(DEL|GND|TWR|APP|DEP|CTR|FSS)")
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var ctrl ControllerInfo
		var activeSessionJSON sql.NullString
		err := rows.Scan(
			&ctrl.CID,
			&ctrl.Name,
			&ctrl.Rating,
			&ctrl.TotalHours,
			&ctrl.LastSeen,
			&activeSessionJSON,
		)
		if err != nil {
			continue
		}

		if activeSessionJSON.Valid {
			var session struct {
				StartTime time.Time `json:"start_time"`
				Position  string    `json:"position"`
			}
			if err := json.Unmarshal([]byte(activeSessionJSON.String), &session); err == nil {
				ctrl.ActiveSession = &struct {
					StartTime time.Time `json:"start_time"`
					Position  string    `json:"position"`
				}{
					StartTime: session.StartTime,
					Position:  session.Position,
				}
			}
		}

		stats.Controllers = append(stats.Controllers, ctrl)
	}

	// Get peak times
	rows, err = db.DB.Query(`
		WITH hourly_stats AS (
			SELECT 
				EXTRACT(HOUR FROM p.last_updated AT TIME ZONE 'UTC') as hour,
				EXTRACT(DOW FROM p.last_updated AT TIME ZONE 'UTC') as day_of_week,
				COUNT(DISTINCT p.id) as traffic_count,
				bool_or(EXISTS (
					SELECT 1 FROM controllers c 
					WHERE c.callsign LIKE $1 
					AND c.last_updated >= p.last_updated - INTERVAL '15 minutes'
					AND c.last_updated <= p.last_updated + INTERVAL '15 minutes'
				)) as has_coverage
			FROM pilots p
			JOIN flight_plans fp ON fp.pilot_id = p.id
			WHERE (fp.arrival = $2 OR fp.departure = $2)
			AND p.last_updated > NOW() - INTERVAL '7 days'
			GROUP BY 
				EXTRACT(HOUR FROM p.last_updated AT TIME ZONE 'UTC'),
				EXTRACT(DOW FROM p.last_updated AT TIME ZONE 'UTC')
			HAVING COUNT(DISTINCT p.id) > 0
			ORDER BY traffic_count DESC
			LIMIT 24
		)
		SELECT hour, day_of_week, traffic_count, has_coverage
		FROM hourly_stats
	`, facility+"_%", facility)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var peak PeakTime
		err := rows.Scan(
			&peak.Hour,
			&peak.DayOfWeek,
			&peak.TrafficCount,
			&peak.Coverage,
		)
		if err != nil {
			continue
		}
		stats.PeakTimes = append(stats.PeakTimes, peak)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetNetworkTrends returns network activity trends over time
func GetNetworkTrends(w http.ResponseWriter, r *http.Request) {
	trends := NetworkTrends{
		Timestamp: time.Now(),
		Daily:     make([]DailyActivity, 0),
		Weekly:    make([]WeeklyActivity, 0),
		Monthly:   make([]MonthlyActivity, 0),
	}

	// Get daily trends for the last 30 days
	rows, err := db.DB.Query(`
		WITH dates AS (
			SELECT generate_series(
				date_trunc('day', NOW() - INTERVAL '30 days'),
				date_trunc('day', NOW()),
				'1 day'::interval
			)::date as series_date
		)
		SELECT 
			d.series_date::timestamp,
			COALESCE(ntd.total_pilots, 0) as total_pilots,
			COALESCE(ntd.total_controllers, 0) as total_controllers,
			COALESCE(ntd.peak_users, 0) as peak_users,
			COALESCE(ntd.unique_users, 0) as unique_users
		FROM dates d
		LEFT JOIN network_trends_daily ntd ON ntd.date = d.series_date
		ORDER BY d.series_date DESC
	`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var daily DailyActivity
		err := rows.Scan(
			&daily.Date,
			&daily.TotalPilots,
			&daily.TotalControllers,
			&daily.PeakUsers,
			&daily.UniqueUsers,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
			return
		}
		trends.Daily = append(trends.Daily, daily)
	}

	// Get weekly trends for the last 12 weeks
	rows, err = db.DB.Query(`
		WITH weeks AS (
			SELECT 
				generate_series(
					date_trunc('week', NOW() - INTERVAL '12 weeks'),
					date_trunc('week', NOW()),
					'1 week'::interval
				)::date as series_start,
				(generate_series(
					date_trunc('week', NOW() - INTERVAL '12 weeks'),
					date_trunc('week', NOW()),
					'1 week'::interval
				) + INTERVAL '6 days')::date as series_end
		)
		SELECT 
			w.series_start::timestamp,
			w.series_end::timestamp,
			COALESCE(ntw.total_pilots, 0) as total_pilots,
			COALESCE(ntw.total_controllers, 0) as total_controllers,
			COALESCE(ntw.peak_users, 0) as peak_users,
			COALESCE(ntw.unique_users, 0) as unique_users
		FROM weeks w
		LEFT JOIN network_trends_weekly ntw ON ntw.week_start = w.series_start
		ORDER BY w.series_start DESC
	`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var weekly WeeklyActivity
		err := rows.Scan(
			&weekly.WeekStart,
			&weekly.WeekEnd,
			&weekly.TotalPilots,
			&weekly.TotalControllers,
			&weekly.PeakUsers,
			&weekly.UniqueUsers,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
			return
		}
		trends.Weekly = append(trends.Weekly, weekly)
	}

	// Get monthly trends for the last 12 months
	rows, err = db.DB.Query(`
		WITH months AS (
			SELECT generate_series(
				date_trunc('month', NOW() - INTERVAL '12 months'),
				date_trunc('month', NOW()),
				'1 month'::interval
			)::date as series_month
		)
		SELECT 
			m.series_month::timestamp,
			COALESCE(ntm.total_pilots, 0) as total_pilots,
			COALESCE(ntm.total_controllers, 0) as total_controllers,
			COALESCE(ntm.peak_users, 0) as peak_users,
			COALESCE(ntm.unique_users, 0) as unique_users
		FROM months m
		LEFT JOIN network_trends_monthly ntm ON ntm.month = m.series_month
		ORDER BY m.series_month DESC
	`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var monthly MonthlyActivity
		err := rows.Scan(
			&monthly.Month,
			&monthly.TotalPilots,
			&monthly.TotalControllers,
			&monthly.PeakUsers,
			&monthly.UniqueUsers,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
			return
		}
		trends.Monthly = append(trends.Monthly, monthly)
	}

	// Update daily trends
	_, err = db.DB.Exec(`
		INSERT INTO network_trends_daily (
			date,
			total_pilots,
			total_controllers,
			peak_users,
			unique_users
		)
		WITH daily_stats AS (
			SELECT
				DATE_TRUNC('day', start_time)::date as stat_date,
				COUNT(DISTINCT CASE WHEN type = 1 THEN vatsim_id END) as pilots,
				COUNT(DISTINCT CASE WHEN type = 2 THEN vatsim_id END) as controllers,
				COUNT(DISTINCT vatsim_id) as total_users
			FROM connections
			WHERE start_time >= DATE_TRUNC('day', NOW()) - INTERVAL '30 days'
			GROUP BY DATE_TRUNC('day', start_time)::date
		)
		SELECT
			DATE_TRUNC('day', NOW())::date,
			COALESCE(pilots, 0),
			COALESCE(controllers, 0),
			COALESCE(total_users, 0),
			COALESCE(total_users, 0)
		FROM daily_stats
		WHERE stat_date = DATE_TRUNC('day', NOW())::date
		ON CONFLICT (date) DO UPDATE SET
			total_pilots = EXCLUDED.total_pilots,
			total_controllers = EXCLUDED.total_controllers,
			peak_users = EXCLUDED.peak_users,
			unique_users = EXCLUDED.unique_users
	`)
	if err != nil {
		log.Printf("Error updating daily trends: %v", err)
	}

	// Update weekly trends
	_, err = db.DB.Exec(`
		INSERT INTO network_trends_weekly (
			week_start,
			week_end,
			total_pilots,
			total_controllers,
			peak_users,
			unique_users
		)
		WITH weekly_stats AS (
			SELECT
				DATE_TRUNC('week', start_time)::date as week_start,
				(DATE_TRUNC('week', start_time) + INTERVAL '6 days')::date as week_end,
				COUNT(DISTINCT CASE WHEN type = 1 THEN vatsim_id END) as pilots,
				COUNT(DISTINCT CASE WHEN type = 2 THEN vatsim_id END) as controllers,
				COUNT(DISTINCT vatsim_id) as total_users
			FROM connections
			WHERE start_time >= DATE_TRUNC('week', NOW())
			GROUP BY DATE_TRUNC('week', start_time)
		)
		SELECT
			DATE_TRUNC('week', NOW())::date,
			(DATE_TRUNC('week', NOW()) + INTERVAL '6 days')::date,
			COALESCE(pilots, 0),
			COALESCE(controllers, 0),
			COALESCE(total_users, 0),
			COALESCE(total_users, 0)
		FROM weekly_stats
		WHERE week_start = DATE_TRUNC('week', NOW())::date
		ON CONFLICT (week_start) DO UPDATE SET
			total_pilots = EXCLUDED.total_pilots,
			total_controllers = EXCLUDED.total_controllers,
			peak_users = EXCLUDED.peak_users,
			unique_users = EXCLUDED.unique_users
	`)
	if err != nil {
		log.Printf("Error updating weekly trends: %v", err)
	}

	// Update monthly trends
	_, err = db.DB.Exec(`
		INSERT INTO network_trends_monthly (
			month,
			total_pilots,
			total_controllers,
			peak_users,
			unique_users
		)
		WITH monthly_stats AS (
			SELECT
				DATE_TRUNC('month', start_time)::date as stat_month,
				COUNT(DISTINCT CASE WHEN type = 1 THEN vatsim_id END) as pilots,
				COUNT(DISTINCT CASE WHEN type = 2 THEN vatsim_id END) as controllers,
				COUNT(DISTINCT vatsim_id) as total_users
			FROM connections
			WHERE start_time >= DATE_TRUNC('month', NOW())
			GROUP BY DATE_TRUNC('month', start_time)::date
		)
		SELECT
			DATE_TRUNC('month', NOW())::date,
			COALESCE(pilots, 0),
			COALESCE(controllers, 0),
			COALESCE(total_users, 0),
			COALESCE(total_users, 0)
		FROM monthly_stats
		WHERE stat_month = DATE_TRUNC('month', NOW())::date
		ON CONFLICT (month) DO UPDATE SET
			total_pilots = EXCLUDED.total_pilots,
			total_controllers = EXCLUDED.total_controllers,
			peak_users = EXCLUDED.peak_users,
			unique_users = EXCLUDED.unique_users
	`)
	if err != nil {
		log.Printf("Error updating monthly trends: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(trends); err != nil {
		http.Error(w, fmt.Sprintf("JSON encoding error: %v", err), http.StatusInternalServerError)
		return
	}
}
