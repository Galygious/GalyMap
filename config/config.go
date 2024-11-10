package config

import (
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

// Settings defines the structure for configuration options
type Settings struct {
	PerformanceMode int    `yaml:"performanceMode"`
	FpsCap          int    `yaml:"fpscap"`
	GameWindowId    string `yaml:"gameWindowId"`
	Debug           bool   `yaml:"debug"`
}

// defaultSettings provides default values for settings
var defaultSettings = Settings{
	PerformanceMode: 1,            // Default performance mode
	FpsCap:          60,           // Default FPS cap
	GameWindowId:    "D2R Window", // Example default window ID
	Debug:           false,        // Debug mode off by default
}

// LoadConfig loads settings from a YAML file, creating the file with defaults if it doesn't exist
func LoadConfig(filePath string) (*Settings, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err := createDefaultConfig(filePath)
		if err != nil {
			return nil, err
		}
		log.Printf("Created default config file at %s\n", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Settings
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// createDefaultConfig creates a config file with default settings
func createDefaultConfig(filePath string) error {
	data, err := yaml.Marshal(&defaultSettings)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}
