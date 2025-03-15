package api

import (
	"github.com/gorilla/mux"
	"github.com/vainnor/vatsim-stats/types"
)

type Collector interface {
	GetStats() types.CollectionStats
	GetCurrentData() (*types.VatsimData, error)
}

// NewRouter creates and configures a new router with all API endpoints
func NewRouter(collector Collector) *mux.Router {
	r := mux.NewRouter()

	// Add API key management endpoints
	r.HandleFunc("/api/keys", CreateAPIKey).Methods("POST")
	r.HandleFunc("/api/keys", ListAPIKeys).Methods("GET")
	r.HandleFunc("/api/keys", DeleteAPIKey).Methods("DELETE")

	// Apply rate limiting middleware to all other routes
	api := r.PathPrefix("/api").Subrouter()
	api.Use(RateLimit)

	// Membership endpoints
	api.HandleFunc("/membership/{cid}/pilot", GetMembershipHandler).Methods("GET")
	api.HandleFunc("/membership/{cid}/debug", GetPilotDebug).Methods("GET")
	api.HandleFunc("/collector/stats", GetCollectorStats(collector)).Methods("GET")

	// Airport traffic endpoint
	api.HandleFunc("/airports/{icao}/traffic", GetAirportTraffic).Methods("GET")

	// Flight search endpoint
	api.HandleFunc("/flights/search", SearchFlights).Methods("GET")

	// Network statistics endpoint
	api.HandleFunc("/network/stats", GetNetworkStatisticsHandler(collector)).Methods("GET")

	// Add facility statistics endpoint
	api.HandleFunc("/facilities/{facility}/stats", GetFacilityStats).Methods("GET")

	// Route statistics endpoints
	api.HandleFunc("/routes/popular", GetPopularRoutes).Methods("GET")
	api.HandleFunc("/routes/{origin}/{destination}/stats", GetRouteStats).Methods("GET")

	// Add analytics endpoints
	api.HandleFunc("/analytics/network-stats", GetNetworkStatistics).Methods("GET")
	api.HandleFunc("/analytics/trends", GetNetworkTrends).Methods("GET")

	return r
}
