package generator

import (
	"encoding/json"
	"os"
)

// Config holds generator configuration
type Config struct {
	Host               string
	Port               uint16
	TerritoryURL       string
	StaticDir          string
	DisableCommands    bool   // Disable game commands from map
	DisableTerritory   bool   // Disable territory generation
	EntityFetchRateInSeconds int    // Polling rate for colonies
	ColonyFetchRateInSeconds int // Polling rate for ships and beds
}

// LoadConfig loads and returns generator config from specified file
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	cfg := Config{
		Host:               "",
		Port:               8880,
		TerritoryURL:       "http://localhost:8881/territoryTiles/",
		DisableCommands:    true,
		StaticDir:          "./www",
		ColonyFetchRateInSeconds: 1800,
		EntityFetchRateInSeconds: 300,
	}

	if err = decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
