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

	// Membership endpoints
	r.HandleFunc("/membership/{cid}/pilot", GetMembershipHandler).Methods("GET")
	r.HandleFunc("/membership/{cid}/debug", GetPilotDebug).Methods("GET")
	r.HandleFunc("/collector/stats", GetCollectorStats(collector)).Methods("GET")

	// Airport traffic endpoint
	r.HandleFunc("/airports/{icao}/traffic", GetAirportTraffic).Methods("GET")

	// Flight search endpoint
	r.HandleFunc("/flights/search", SearchFlights).Methods("GET")

	// Network statistics endpoint
	r.HandleFunc("/network/stats", GetNetworkStatisticsHandler(collector)).Methods("GET")

	// Add facility statistics endpoint
	r.HandleFunc("/facilities/{facility}/stats", GetFacilityStats).Methods("GET")

	// Add analytics endpoints
	r.HandleFunc("/analytics/network-stats", GetNetworkStatistics).Methods("GET")
	r.HandleFunc("/analytics/trends", GetNetworkTrends).Methods("GET")

	// Route statistics endpoints
	r.HandleFunc("/routes/popular", GetPopularRoutes).Methods("GET")
	r.HandleFunc("/routes/{origin}/{destination}/stats", GetRouteStats).Methods("GET")

	return r
}
