package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds application configuration
type Config struct {
	Port         int    `json:"port"`
	Host         string `json:"host"`
	DataDir      string `json:"data_dir"`
	DatabasePath string `json:"database_path"`
	Debug        bool   `json:"debug"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		Port:         7979,
		Host:         "0.0.0.0",
		DataDir:      "./data",
		DatabasePath: "./data/maggpi.db",
		Debug:        false,
	}
}

// Load loads configuration from a JSON file, creating it with defaults if it doesn't exist
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Ensure data directory exists
	dataDir := filepath.Dir(path)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	// Try to read existing config
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config file
			if err := cfg.Save(path); err != nil {
				return nil, err
			}
			return &cfg, nil
		}
		return nil, err
	}

	// Parse existing config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save saves configuration to a JSON file
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
