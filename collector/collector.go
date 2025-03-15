package collector

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/lib/pq"
	"github.com/vainnor/vatsim-stats/api"
	"github.com/vainnor/vatsim-stats/db"
	"github.com/vainnor/vatsim-stats/types"
)

const vatsimDataURL = "https://data.vatsim.net/v3/vatsim-data.json"

type Collector struct {
	lastUpdate string
	client     *http.Client
	// Track active connections by CID and callsign
	activeConnections map[string]activeConnection
	// Collection stats
	stats types.CollectionStats
}

type activeConnection struct {
	cid            string
	callsign       string
	connectionType api.ConnectionType
	rating         int
	server         string
	startTime      time.Time
	lastSeen       time.Time
	// Statistics tracking
	aircraftTracked    int
	aircraftSeen       int
	flightsAmended     int
	handoffsInitiated  int
	handoffsReceived   int
	handoffsRefused    int
	squawksAssigned    int
	cruiseAltsModified int
	tempAltsModified   int
	scratchpadMods     int
	// Pilot specific stats
	hasFlightPlan bool
}

func NewCollector() *Collector {
	return &Collector{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		activeConnections: make(map[string]activeConnection),
		stats: types.CollectionStats{
			StartTime: time.Now(),
		},
	}
}

func (c *Collector) GetStats() types.CollectionStats {
	return c.stats
}

// GetCurrentData returns the most recently fetched VATSIM data
func (c *Collector) GetCurrentData() (*types.VatsimData, error) {
	return c.fetchData()
}

func (c *Collector) FetchAndStore() error {
	data, err := c.fetchData()
	if err != nil {
		return fmt.Errorf("error fetching data: %v", err)
	}

	// Check if data has changed
	if data.General.Update == c.lastUpdate {
		return nil
	}

	// Store new data
	if err := c.storeData(data); err != nil {
		return fmt.Errorf("error storing data: %v", err)
	}

	// Store network and related statistics
	if err := c.storeNetworkStats(data); err != nil {
		log.Printf("Error storing network stats: %v", err)
	}

	// Store airport statistics
	if err := c.storeAirportStats(); err != nil {
		log.Printf("Error storing airport stats: %v", err)
	}

	c.lastUpdate = data.General.Update
	c.stats.LastUpdate = time.Now()
	c.stats.TotalSnapshots++
	c.stats.ActivePilots = len(data.Pilots)

	// Count ATC and ATIS controllers
	c.stats.ActiveATCs = 0
	c.stats.ActiveATIS = 0
	for _, controller := range data.Controllers {
		if len(controller.TextAtis) > 0 {
			c.stats.ActiveATIS++
		} else {
			c.stats.ActiveATCs++
		}
	}

	c.stats.ProcessedPilots += int64(len(data.Pilots))

	log.Printf("Collection update: Active pilots: %d, Active ATCs: %d, Active ATIS: %d, Total snapshots: %d, Running for: %v",
		c.stats.ActivePilots,
		c.stats.ActiveATCs,
		c.stats.ActiveATIS,
		c.stats.TotalSnapshots,
		time.Since(c.stats.StartTime).Round(time.Second))

	return nil
}

func (c *Collector) fetchData() (*types.VatsimData, error) {
	resp, err := c.client.Get(vatsimDataURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data types.VatsimData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (c *Collector) storeData(data *types.VatsimData) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Store facilities
	for _, facility := range data.Facilities {
		_, err = tx.Exec(`
			INSERT INTO facilities (id, short_name, long_name)
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO UPDATE
			SET short_name = $2, long_name = $3
		`, facility.ID, facility.Short, facility.Long)
		if err != nil {
			return err
		}
	}

	// Store ratings
	for _, rating := range data.Ratings {
		_, err = tx.Exec(`
			INSERT INTO ratings (id, short_name, long_name)
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO UPDATE
			SET short_name = $2, long_name = $3
		`, rating.ID, rating.Short, rating.Long)
		if err != nil {
			return err
		}
	}

	// Insert snapshot
	var snapshotID int
	err = tx.QueryRow(`
		INSERT INTO snapshots (
			timestamp, version, reload, update_str,
			connected_clients, unique_users
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, data.General.UpdateTimestamp, data.General.Version,
		data.General.Reload, data.General.Update,
		data.General.ConnectedClients, data.General.UniqueUsers).Scan(&snapshotID)
	if err != nil {
		return err
	}

	// Track current connections to detect disconnects
	currentConnections := make(map[string]bool)

	// Process pilots
	for _, pilot := range data.Pilots {
		key := fmt.Sprintf("%d-%s", pilot.CID, pilot.Callsign)
		currentConnections[key] = true

		// Store pilot data
		var pilotID int
		err = tx.QueryRow(`
			INSERT INTO pilots (
				snapshot_id, cid, name, callsign, server,
				pilot_rating, military_rating, latitude, longitude,
				altitude, groundspeed, transponder, heading,
				qnh_i_hg, qnh_mb, logon_time, last_updated
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
			RETURNING id
		`, snapshotID, pilot.CID, pilot.Name, pilot.Callsign, pilot.Server,
			pilot.PilotRating, pilot.MilitaryRating, pilot.Latitude, pilot.Longitude,
			pilot.Altitude, pilot.Groundspeed, pilot.Transponder, pilot.Heading,
			pilot.QNHiHg, pilot.QNHMb, pilot.LogonTime, pilot.LastUpdated).Scan(&pilotID)
		if err != nil {
			return err
		}

		if pilot.FlightPlan != nil {
			_, err = tx.Exec(`
				INSERT INTO flight_plans (
					pilot_id, flight_rules, aircraft, aircraft_faa,
					aircraft_short, departure, arrival, alternate,
					cruise_tas, altitude, deptime, enroute_time,
					fuel_time, remarks, route, revision_id,
					assigned_transponder
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
			`, pilotID, pilot.FlightPlan.FlightRules, pilot.FlightPlan.Aircraft,
				pilot.FlightPlan.AircraftFaa, pilot.FlightPlan.AircraftShort,
				pilot.FlightPlan.Departure, pilot.FlightPlan.Arrival,
				pilot.FlightPlan.Alternate, pilot.FlightPlan.CruiseTAS,
				pilot.FlightPlan.Altitude, pilot.FlightPlan.DepTime,
				pilot.FlightPlan.EnrouteTime, pilot.FlightPlan.FuelTime,
				pilot.FlightPlan.Remarks, pilot.FlightPlan.Route,
				pilot.FlightPlan.RevisionID, pilot.FlightPlan.AssignedTransponder)
			if err != nil {
				return err
			}
		}

		// Check for existing active connection
		var existingConnID int64
		err = tx.QueryRow(`
			SELECT id FROM connections 
			WHERE vatsim_id = $1 
			AND callsign = $2 
			AND type = $3 
			AND end_time > $4
			ORDER BY start_time DESC 
			LIMIT 1
		`, fmt.Sprintf("%d", pilot.CID), pilot.Callsign, api.TypePilot,
			time.Now().Add(-5*time.Minute)).Scan(&existingConnID)

		if err != nil && err != sql.ErrNoRows {
			return err
		}

		if err == sql.ErrNoRows {
			// No existing active connection found, create new one
			var connID int64
			err = tx.QueryRow(`
				INSERT INTO connections (
					vatsim_id, type, rating, callsign,
					start_time, end_time, server
				) VALUES ($1, $2, $3, $4, $5, $6, $7)
				RETURNING id
			`, fmt.Sprintf("%d", pilot.CID), api.TypePilot, pilot.PilotRating, pilot.Callsign,
				pilot.LogonTime, pilot.LastUpdated, pilot.Server).Scan(&connID)
			if err != nil {
				return err
			}

			// Track the new connection in memory
			c.activeConnections[key] = activeConnection{
				cid:            fmt.Sprintf("%d", pilot.CID),
				callsign:       pilot.Callsign,
				connectionType: api.TypePilot,
				rating:         pilot.PilotRating,
				server:         pilot.Server,
				startTime:      pilot.LogonTime,
				lastSeen:       pilot.LastUpdated,
				hasFlightPlan:  pilot.FlightPlan != nil,
			}
		} else {
			// Update existing connection's end time
			_, err = tx.Exec(`
				UPDATE connections 
				SET end_time = $1, rating = $2
				WHERE id = $3
			`, pilot.LastUpdated, pilot.PilotRating, existingConnID)
			if err != nil {
				return err
			}

			// Update active connection tracking
			if active, exists := c.activeConnections[key]; exists {
				active.lastSeen = pilot.LastUpdated
				active.rating = pilot.PilotRating
				active.hasFlightPlan = pilot.FlightPlan != nil
				c.activeConnections[key] = active
			} else {
				// Restore active connection state
				c.activeConnections[key] = activeConnection{
					cid:            fmt.Sprintf("%d", pilot.CID),
					callsign:       pilot.Callsign,
					connectionType: api.TypePilot,
					rating:         pilot.PilotRating,
					server:         pilot.Server,
					startTime:      pilot.LogonTime,
					lastSeen:       pilot.LastUpdated,
					hasFlightPlan:  pilot.FlightPlan != nil,
				}
			}
		}
	}

	// Process controllers (ATC and ATIS)
	for _, controller := range data.Controllers {
		key := fmt.Sprintf("%d-%s", controller.CID, controller.Callsign)
		currentConnections[key] = true

		// Store controller data
		_, err = tx.Exec(`
			INSERT INTO controllers (
				snapshot_id, cid, name, callsign, frequency,
				facility, rating, server, visual_range, text_atis,
				last_updated, logon_time
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, snapshotID, controller.CID, controller.Name, controller.Callsign,
			controller.Frequency, controller.Facility, controller.Rating,
			controller.Server, controller.VisualRange, pq.Array(controller.TextAtis),
			controller.LastUpdated, controller.LogonTime)
		if err != nil {
			return err
		}

		// Determine if this is an ATC or ATIS position
		var connType api.ConnectionType
		if len(controller.TextAtis) > 0 {
			connType = api.TypeATIS
		} else {
			connType = api.TypeATC
		}

		// Check for existing active connection
		var existingConnID int64
		err = tx.QueryRow(`
			SELECT id FROM connections 
			WHERE vatsim_id = $1 
			AND callsign = $2 
			AND type = $3 
			AND end_time > $4
			ORDER BY start_time DESC 
			LIMIT 1
		`, fmt.Sprintf("%d", controller.CID), controller.Callsign, connType,
			time.Now().Add(-5*time.Minute)).Scan(&existingConnID)

		if err != nil && err != sql.ErrNoRows {
			return err
		}

		if err == sql.ErrNoRows {
			// New connection
			var connID int64
			err = tx.QueryRow(`
				INSERT INTO connections (
					vatsim_id, type, rating, callsign,
					start_time, end_time, server
				) VALUES ($1, $2, $3, $4, $5, $6, $7)
				RETURNING id
			`, fmt.Sprintf("%d", controller.CID), connType, controller.Rating, controller.Callsign,
				controller.LogonTime, controller.LastUpdated, controller.Server).Scan(&connID)
			if err != nil {
				return err
			}

			c.activeConnections[key] = activeConnection{
				cid:            fmt.Sprintf("%d", controller.CID),
				callsign:       controller.Callsign,
				connectionType: connType,
				rating:         controller.Rating,
				server:         controller.Server,
				startTime:      controller.LogonTime,
				lastSeen:       controller.LastUpdated,
			}
		} else {
			// Update existing connection's end time
			_, err = tx.Exec(`
				UPDATE connections 
				SET end_time = $1, rating = $2
				WHERE id = $3
			`, controller.LastUpdated, controller.Rating, existingConnID)
			if err != nil {
				return err
			}

			// Update active connection tracking
			if active, exists := c.activeConnections[key]; exists {
				active.lastSeen = controller.LastUpdated
				active.rating = controller.Rating
				c.activeConnections[key] = active
			} else {
				// Restore active connection state
				c.activeConnections[key] = activeConnection{
					cid:            fmt.Sprintf("%d", controller.CID),
					callsign:       controller.Callsign,
					connectionType: connType,
					rating:         controller.Rating,
					server:         controller.Server,
					startTime:      controller.LogonTime,
					lastSeen:       controller.LastUpdated,
				}
			}
		}
	}

	// Check for disconnections
	for key, conn := range c.activeConnections {
		if !currentConnections[key] {
			// Connection ended, store the stats
			err = c.storeConnectionStats(tx, conn)
			if err != nil {
				return err
			}
			delete(c.activeConnections, key)
		}
	}

	return tx.Commit()
}

func (c *Collector) storeConnectionStats(tx *sql.Tx, conn activeConnection) error {
	// Insert final connection record
	var connID int64
	err := tx.QueryRow(`
		INSERT INTO connections (
			vatsim_id, type, rating, callsign,
			start_time, end_time, server
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, conn.cid, conn.connectionType, conn.rating, conn.callsign,
		conn.startTime, conn.lastSeen, conn.server).Scan(&connID)
	if err != nil {
		return err
	}

	// Store type-specific stats
	switch conn.connectionType {
	case api.TypePilot:
		// Calculate flight time in hours
		flightTime := int(conn.lastSeen.Sub(conn.startTime).Minutes() / 60)

		// Store individual connection stats
		_, err = tx.Exec(`
			INSERT INTO pilot_stats (
				connection_id, flight_time, pilot_rating,
				has_flight_plan
			) VALUES ($1, $2, $3, $4)
		`, connID, flightTime, conn.rating, conn.hasFlightPlan)
		if err != nil {
			return err
		}

		// Update total stats
		_, err = tx.Exec(`
			INSERT INTO pilot_total_stats (
				vatsim_id, total_hours, total_flights,
				student_hours, ppl_hours, instrument_hours,
				cpl_hours, atpl_hours, last_updated
			) VALUES (
				$1, $2, 1,
				CASE WHEN $3 = 1 THEN $2 ELSE 0 END,
				CASE WHEN $3 = 2 THEN $2 ELSE 0 END,
				CASE WHEN $3 = 3 THEN $2 ELSE 0 END,
				CASE WHEN $3 = 4 THEN $2 ELSE 0 END,
				CASE WHEN $3 = 5 THEN $2 ELSE 0 END,
				$4
			)
			ON CONFLICT (vatsim_id) DO UPDATE SET
				total_hours = pilot_total_stats.total_hours + $2,
				total_flights = pilot_total_stats.total_flights + 1,
				student_hours = pilot_total_stats.student_hours + CASE WHEN $3 = 1 THEN $2 ELSE 0 END,
				ppl_hours = pilot_total_stats.ppl_hours + CASE WHEN $3 = 2 THEN $2 ELSE 0 END,
				instrument_hours = pilot_total_stats.instrument_hours + CASE WHEN $3 = 3 THEN $2 ELSE 0 END,
				cpl_hours = pilot_total_stats.cpl_hours + CASE WHEN $3 = 4 THEN $2 ELSE 0 END,
				atpl_hours = pilot_total_stats.atpl_hours + CASE WHEN $3 = 5 THEN $2 ELSE 0 END,
				last_updated = $4
		`, conn.cid, flightTime, conn.rating, conn.lastSeen)

	case api.TypeATC:
		_, err = tx.Exec(`
			INSERT INTO atc_stats (
				connection_id, aircraft_tracked, aircraft_seen,
				flights_amended, handoffs_initiated, handoffs_received,
				handoffs_refused, squawks_assigned, cruise_alts_modified,
				temp_alts_modified, scratchpad_mods
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, connID,
			conn.aircraftTracked, conn.aircraftSeen,
			conn.flightsAmended, conn.handoffsInitiated,
			conn.handoffsReceived, conn.handoffsRefused,
			conn.squawksAssigned, conn.cruiseAltsModified,
			conn.tempAltsModified, conn.scratchpadMods)

	case api.TypeATIS:
		_, err = tx.Exec(`
			INSERT INTO atis_stats (
				connection_id, updates
			) VALUES ($1, $2)
		`, connID, conn.aircraftTracked) // Using aircraftTracked as updates counter
	}

	return err
}

// storeNetworkStats stores current network-wide statistics
func (c *Collector) storeNetworkStats(data *types.VatsimData) error {
	// Store network stats
	_, err := db.DB.Exec(`
		INSERT INTO network_stats (
			timestamp, total_pilots, total_atcs, active_pilots
		) VALUES (
			NOW(), 
			(SELECT COUNT(DISTINCT cid) FROM pilots WHERE last_updated > NOW() - INTERVAL '24 hours'),
			(SELECT COUNT(DISTINCT cid) FROM controllers WHERE last_updated > NOW() - INTERVAL '24 hours'),
			(SELECT COUNT(DISTINCT cid) FROM pilots WHERE last_updated > NOW() - INTERVAL '5 minutes')
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to store network stats: %v", err)
	}

	// Store server stats
	servers := make(map[string]int)
	for _, pilot := range data.Pilots {
		servers[pilot.Server]++
	}
	for _, controller := range data.Controllers {
		servers[controller.Server]++
	}

	for server, count := range servers {
		_, err = db.DB.Exec(`
			INSERT INTO server_stats (
				timestamp, server_name, connected_users
			) VALUES (
				NOW(), $1, $2
			)
		`, server, count)
		if err != nil {
			return fmt.Errorf("failed to store server stats: %v", err)
		}
	}

	// Store rating stats
	_, err = db.DB.Exec(`
		INSERT INTO rating_stats (
			timestamp, rating, pilot_count, atc_count
		)
		SELECT 
			NOW(),
			rating,
			COUNT(DISTINCT CASE WHEN type = 1 THEN vatsim_id END),
			COUNT(DISTINCT CASE WHEN type = 2 THEN vatsim_id END)
		FROM connections
		WHERE end_time > NOW() - INTERVAL '24 hours'
		GROUP BY rating
	`)
	if err != nil {
		return fmt.Errorf("failed to store rating stats: %v", err)
	}

	// Store aircraft stats
	_, err = db.DB.Exec(`
		INSERT INTO aircraft_stats (
			timestamp, aircraft_type, count
		)
		SELECT 
			NOW(),
			aircraft_short,
			COUNT(*)
		FROM flight_plans fp
		JOIN pilots p ON p.id = fp.pilot_id
		WHERE p.last_updated > NOW() - INTERVAL '5 minutes'
		GROUP BY aircraft_short
	`)
	if err != nil {
		return fmt.Errorf("failed to store aircraft stats: %v", err)
	}

	return nil
}

// storeAirportStats stores traffic statistics for airports
func (c *Collector) storeAirportStats() error {
	// Get current traffic for all airports
	_, err := db.DB.Exec(`
		INSERT INTO airport_stats (
			icao, timestamp, hourly_movements, arrival_count, departure_count
		)
		SELECT 
			airport,
			NOW(),
			COUNT(*) as hourly_movements,
			COUNT(CASE WHEN type = 'arrival' THEN 1 END) as arrival_count,
			COUNT(CASE WHEN type = 'departure' THEN 1 END) as departure_count
		FROM (
			SELECT 
				departure as airport,
				'departure' as type
			FROM flight_plans fp
			JOIN pilots p ON p.id = fp.pilot_id
			WHERE p.last_updated > NOW() - INTERVAL '1 hour'
			UNION ALL
			SELECT 
				arrival as airport,
				'arrival' as type
			FROM flight_plans fp
			JOIN pilots p ON p.id = fp.pilot_id
			WHERE p.last_updated > NOW() - INTERVAL '1 hour'
		) movements
		GROUP BY airport
	`)
	if err != nil {
		return fmt.Errorf("failed to store airport stats: %v", err)
	}

	return nil
}

// Add a status endpoint to the API
func GetCollectorStats(c *Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(c.GetStats())
	}
}
