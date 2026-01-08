package bebo

import "github.com/devmarvs/bebo/config"

// Config is the framework configuration.
type Config = config.Config

// ConfigProfile describes layered config sources.
type ConfigProfile = config.Profile

// DefaultConfig returns default config values.
func DefaultConfig() config.Config {
	return config.Default()
}

// ConfigFromEnv applies environment overrides using the given prefix.
func ConfigFromEnv(prefix string, base config.Config) config.Config {
	return config.LoadFromEnv(prefix, base)
}

// ConfigFromFile loads config from a JSON file onto the base config.
func ConfigFromFile(path string, base config.Config) (config.Config, error) {
	return config.LoadFromFile(path, base)
}

// LoadConfig loads config from file and applies env overrides.
func LoadConfig(path, envPrefix string) (config.Config, error) {
	return config.Load(path, envPrefix)
}

// LoadConfigProfile loads config from base/env/secrets profiles with validation.
func LoadConfigProfile(profile config.Profile) (config.Config, error) {
	return config.LoadProfile(profile)
}

// ValidateConfig validates configuration values.
func ValidateConfig(cfg config.Config) error {
	return config.Validate(cfg)
}
