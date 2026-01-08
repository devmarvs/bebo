package config

import (
	"encoding/json"
	"os"
)

// Profile describes layered config sources.
type Profile struct {
	BasePath     string
	EnvPath      string
	SecretsPath  string
	EnvPrefix    string
	AllowMissing bool
}

// Loader composes layered config with defaults and validation.
type Loader[T any] struct {
	Defaults func() T
	ApplyEnv func(prefix string, base T) T
	Validate func(cfg T) error
}

// Load merges profile layers into a typed config.
func (l Loader[T]) Load(profile Profile) (T, error) {
	var cfg T
	if l.Defaults != nil {
		cfg = l.Defaults()
	}

	var err error
	if profile.BasePath != "" {
		cfg, err = loadJSON(profile.BasePath, cfg, profile.AllowMissing)
		if err != nil {
			return cfg, err
		}
	}
	if profile.EnvPath != "" {
		cfg, err = loadJSON(profile.EnvPath, cfg, profile.AllowMissing)
		if err != nil {
			return cfg, err
		}
	}
	if profile.SecretsPath != "" {
		cfg, err = loadJSON(profile.SecretsPath, cfg, profile.AllowMissing)
		if err != nil {
			return cfg, err
		}
	}
	if profile.EnvPrefix != "" && l.ApplyEnv != nil {
		cfg = l.ApplyEnv(profile.EnvPrefix, cfg)
	}
	if l.Validate != nil {
		if err := l.Validate(cfg); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

// LoadProfile loads Config from a layered profile with validation.
func LoadProfile(profile Profile) (Config, error) {
	loader := Loader[Config]{
		Defaults: Default,
		ApplyEnv: LoadFromEnv,
		Validate: Validate,
	}
	return loader.Load(profile)
}

func loadJSON[T any](path string, base T, allowMissing bool) (T, error) {
	file, err := os.Open(path)
	if err != nil {
		if allowMissing && os.IsNotExist(err) {
			return base, nil
		}
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
