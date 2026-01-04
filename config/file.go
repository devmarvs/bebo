package config

import (
	"encoding/json"
	"os"
)

// LoadFromFile loads configuration from a JSON file into the base config.
func LoadFromFile(path string, base Config) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return base, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&base); err != nil {
		return base, err
	}
	return base, nil
}

// Load loads config from file (if provided) and applies env overrides.
func Load(path, envPrefix string) (Config, error) {
	cfg := Default()
	var err error
	if path != "" {
		cfg, err = LoadFromFile(path, cfg)
		if err != nil {
			return cfg, err
		}
	}
	if envPrefix != "" {
		cfg = LoadFromEnv(envPrefix, cfg)
	}
	return cfg, nil
}
