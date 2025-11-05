package config

import (
	"encoding/json"
	"os"
)

// Config represents the sidecar service configuration
type Config struct {
	Server struct {
		Port int    `json:"port"`
		Host string `json:"host"`
	} `json:"server"`

	Jellyfin struct {
		URL    string `json:"url"`
		APIKey string `json:"api_key"`
	} `json:"jellyfin"`

	Symlink struct {
		BasePath          string `json:"base_path"`
		VirtualFolderName string `json:"virtual_folder_name"`
		CollectionType    string `json:"collection_type"` // "movies" or "tvshows"
	} `json:"symlink"`

	Security struct {
		APIKey string `json:"api_key"` // API key for Prunarr to authenticate with this service
	} `json:"security"`
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set defaults
	if config.Server.Port == 0 {
		config.Server.Port = 8090
	}
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Symlink.VirtualFolderName == "" {
		config.Symlink.VirtualFolderName = "Leaving Soon"
	}
	if config.Symlink.CollectionType == "" {
		config.Symlink.CollectionType = "mixed"
	}

	return &config, nil
}

// Example returns an example configuration
func Example() *Config {
	return &Config{}
}
