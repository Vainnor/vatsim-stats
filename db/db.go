package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() error {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("error opening database: %v", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("error connecting to the database: %v", err)
	}

	if err = createTables(); err != nil {
		return fmt.Errorf("error creating tables: %v", err)
	}

	return nil
}

func createTables() error {
	queries := []string{
		// Core tables
		`CREATE TABLE IF NOT EXISTS api_keys (
			id SERIAL PRIMARY KEY,
			key VARCHAR(64) NOT NULL UNIQUE,
			description TEXT,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			last_used_at TIMESTAMP WITH TIME ZONE,
			is_active BOOLEAN NOT NULL DEFAULT true
		)`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			version INTEGER NOT NULL,
			reload INTEGER NOT NULL,
			update_str VARCHAR(255) NOT NULL,
			connected_clients INTEGER NOT NULL,
			unique_users INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS facilities (
			id INTEGER PRIMARY KEY,
			short_name VARCHAR(10) NOT NULL,
			long_name VARCHAR(255) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS ratings (
			id INTEGER PRIMARY KEY,
			short_name VARCHAR(10) NOT NULL,
			long_name VARCHAR(255) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS pilot_ratings (
			id INTEGER PRIMARY KEY,
			short_name VARCHAR(10) NOT NULL,
			long_name VARCHAR(255) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS military_ratings (
			id INTEGER PRIMARY KEY,
			short_name VARCHAR(10) NOT NULL,
			long_name VARCHAR(255) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS pilots (
			id SERIAL PRIMARY KEY,
			snapshot_id INTEGER REFERENCES snapshots(id),
			cid INTEGER NOT NULL,
			name VARCHAR(255) NOT NULL,
			callsign VARCHAR(255) NOT NULL,
			server VARCHAR(255) NOT NULL,
			pilot_rating INTEGER NOT NULL,
			military_rating INTEGER NOT NULL,
			latitude DOUBLE PRECISION NOT NULL,
			longitude DOUBLE PRECISION NOT NULL,
			altitude INTEGER NOT NULL,
			groundspeed INTEGER NOT NULL,
			transponder VARCHAR(10) NOT NULL,
			heading INTEGER NOT NULL,
			qnh_i_hg DOUBLE PRECISION NOT NULL,
			qnh_mb INTEGER NOT NULL,
			logon_time TIMESTAMP WITH TIME ZONE NOT NULL,
			last_updated TIMESTAMP WITH TIME ZONE NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS controllers (
			id SERIAL PRIMARY KEY,
			snapshot_id INTEGER REFERENCES snapshots(id),
			cid INTEGER NOT NULL,
			name VARCHAR(255) NOT NULL,
			callsign VARCHAR(255) NOT NULL,
			frequency VARCHAR(10) NOT NULL,
			facility INTEGER NOT NULL,
			rating INTEGER NOT NULL,
			server VARCHAR(255) NOT NULL,
			visual_range INTEGER NOT NULL,
			text_atis TEXT[],
			last_updated TIMESTAMP WITH TIME ZONE NOT NULL,
			logon_time TIMESTAMP WITH TIME ZONE NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS flight_plans (
			id SERIAL PRIMARY KEY,
			pilot_id INTEGER REFERENCES pilots(id),
			flight_rules VARCHAR(2) NOT NULL,
			aircraft VARCHAR(255) NOT NULL,
			aircraft_faa VARCHAR(255) NOT NULL,
			aircraft_short VARCHAR(255) NOT NULL,
			departure VARCHAR(4) NOT NULL,
			arrival VARCHAR(4) NOT NULL,
			alternate VARCHAR(4),
			cruise_tas VARCHAR(10) NOT NULL,
			altitude VARCHAR(10) NOT NULL,
			deptime VARCHAR(4) NOT NULL,
			enroute_time VARCHAR(4) NOT NULL,
			fuel_time VARCHAR(4) NOT NULL,
			remarks TEXT,
			route TEXT,
			revision_id INTEGER NOT NULL,
			assigned_transponder VARCHAR(10)
		)`,
		`CREATE TABLE IF NOT EXISTS connections (
			id BIGSERIAL PRIMARY KEY,
			vatsim_id VARCHAR(20) NOT NULL,
			type INTEGER NOT NULL,
			rating INTEGER NOT NULL,
			callsign VARCHAR(255) NOT NULL,
			start_time TIMESTAMP WITH TIME ZONE NOT NULL,
			end_time TIMESTAMP WITH TIME ZONE NOT NULL,
			server VARCHAR(255) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS atc_stats (
			connection_id BIGINT PRIMARY KEY REFERENCES connections(id),
			aircraft_tracked INTEGER NOT NULL DEFAULT 0,
			aircraft_seen INTEGER NOT NULL DEFAULT 0,
			flights_amended INTEGER NOT NULL DEFAULT 0,
			handoffs_initiated INTEGER NOT NULL DEFAULT 0,
			handoffs_received INTEGER NOT NULL DEFAULT 0,
			handoffs_refused INTEGER NOT NULL DEFAULT 0,
			squawks_assigned INTEGER NOT NULL DEFAULT 0,
			cruise_alts_modified INTEGER NOT NULL DEFAULT 0,
			temp_alts_modified INTEGER NOT NULL DEFAULT 0,
			scratchpad_mods INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS pilot_stats (
			connection_id BIGINT PRIMARY KEY REFERENCES connections(id),
			flight_time INTEGER NOT NULL DEFAULT 0,
			pilot_rating INTEGER NOT NULL,
			has_flight_plan BOOLEAN NOT NULL DEFAULT false
		)`,
		`CREATE TABLE IF NOT EXISTS pilot_total_stats (
			vatsim_id VARCHAR(20) PRIMARY KEY,
			total_hours INTEGER NOT NULL DEFAULT 0,
			total_flights INTEGER NOT NULL DEFAULT 0,
			student_hours INTEGER NOT NULL DEFAULT 0,
			ppl_hours INTEGER NOT NULL DEFAULT 0,
			instrument_hours INTEGER NOT NULL DEFAULT 0,
			cpl_hours INTEGER NOT NULL DEFAULT 0,
			atpl_hours INTEGER NOT NULL DEFAULT 0,
			last_updated TIMESTAMP WITH TIME ZONE NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS atis_stats (
			connection_id BIGINT PRIMARY KEY REFERENCES connections(id),
			updates INTEGER NOT NULL DEFAULT 0,
			frequency VARCHAR(10),
			letter CHAR(1)
		)`,

		// Statistics tables
		`CREATE TABLE IF NOT EXISTS airport_stats (
			id SERIAL PRIMARY KEY,
			icao VARCHAR(4) NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			hourly_movements INTEGER NOT NULL DEFAULT 0,
			arrival_count INTEGER NOT NULL DEFAULT 0,
			departure_count INTEGER NOT NULL DEFAULT 0,
			UNIQUE (icao, timestamp)
		)`,
		`CREATE TABLE IF NOT EXISTS network_stats (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			total_pilots INTEGER NOT NULL,
			total_atcs INTEGER NOT NULL,
			active_pilots INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS server_stats (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			server_name TEXT NOT NULL,
			connected_users INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS rating_stats (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			rating INTEGER NOT NULL,
			pilot_count INTEGER NOT NULL,
			atc_count INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS aircraft_stats (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			aircraft_type TEXT NOT NULL,
			count INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS route_stats (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			origin TEXT NOT NULL,
			destination TEXT NOT NULL,
			flight_count INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS network_trends_daily (
			id SERIAL PRIMARY KEY,
			date DATE NOT NULL,
			total_pilots INTEGER NOT NULL DEFAULT 0,
			total_controllers INTEGER NOT NULL DEFAULT 0,
			peak_users INTEGER NOT NULL DEFAULT 0,
			unique_users INTEGER NOT NULL DEFAULT 0,
			UNIQUE(date)
		)`,
		`CREATE TABLE IF NOT EXISTS network_trends_weekly (
			id SERIAL PRIMARY KEY,
			week_start DATE NOT NULL,
			week_end DATE NOT NULL,
			total_pilots INTEGER NOT NULL DEFAULT 0,
			total_controllers INTEGER NOT NULL DEFAULT 0,
			peak_users INTEGER NOT NULL DEFAULT 0,
			unique_users INTEGER NOT NULL DEFAULT 0,
			UNIQUE(week_start)
		)`,
		`CREATE TABLE IF NOT EXISTS network_trends_monthly (
			id SERIAL PRIMARY KEY,
			month DATE NOT NULL,
			total_pilots INTEGER NOT NULL DEFAULT 0,
			total_controllers INTEGER NOT NULL DEFAULT 0,
			peak_users INTEGER NOT NULL DEFAULT 0,
			unique_users INTEGER NOT NULL DEFAULT 0,
			UNIQUE(month)
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_connections_vatsim_id ON connections(vatsim_id)`,
		`CREATE INDEX IF NOT EXISTS idx_connections_type ON connections(type)`,
		`CREATE INDEX IF NOT EXISTS idx_airport_stats_icao_timestamp ON airport_stats (icao, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_network_stats_timestamp ON network_stats (timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_server_stats_timestamp ON server_stats (timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_rating_stats_timestamp ON rating_stats (timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_aircraft_stats_timestamp ON aircraft_stats (timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_controllers_snapshot ON controllers(snapshot_id)`,
		`CREATE INDEX IF NOT EXISTS idx_controllers_callsign ON controllers(callsign)`,
		`CREATE INDEX IF NOT EXISTS idx_route_stats_airports ON route_stats(origin, destination)`,
		`CREATE INDEX IF NOT EXISTS idx_route_stats_timestamp ON route_stats(timestamp)`,
	}

	for _, query := range queries {
		_, err := DB.Exec(query)
		if err != nil {
			return err
		}
	}

	return nil
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
