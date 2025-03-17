package jsonfetcher

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/vainnor/vatsim-stats/models"
)

// FetchVatsimData fetches the JSON data from VATSIM
func FetchVatsimData() (*models.VatsimData, error) {
	url := "https://data.vatsim.net/v3/vatsim-data.json"
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching VATSIM data: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var vatsimData models.VatsimData
	err = json.NewDecoder(resp.Body).Decode(&vatsimData)
	if err != nil {
		log.Printf("Error decoding VATSIM data: %v", err)
		return nil, err
	}

	log.Println("Fetched VATSIM data successfully")
	return &vatsimData, nil
}
