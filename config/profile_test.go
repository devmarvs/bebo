package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProfileLayering(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.json")
	envPath := filepath.Join(dir, "env.json")
	secretsPath := filepath.Join(dir, "secrets.json")

	if err := os.WriteFile(basePath, []byte(`{"address":":8080","log_level":"info"}`), 0o600); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := os.WriteFile(envPath, []byte(`{"address":":9090"}`), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}
	if err := os.WriteFile(secretsPath, []byte(`{"log_level":"error"}`), 0o600); err != nil {
		t.Fatalf("write secrets: %v", err)
	}

	t.Setenv("BEBO_ADDRESS", ":7070")

	cfg, err := LoadProfile(Profile{
		BasePath:    basePath,
		EnvPath:     envPath,
		SecretsPath: secretsPath,
		EnvPrefix:   "BEBO_",
	})
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}

	if cfg.Address != ":7070" {
		t.Fatalf("expected address override, got %q", cfg.Address)
	}
	if cfg.LogLevel != "error" {
		t.Fatalf("expected log_level from secrets, got %q", cfg.LogLevel)
	}
}

func TestLoadProfileValidation(t *testing.T) {
	cfg := Default()
	cfg.ReadTimeout = -1
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestTypedLoader(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
	}

	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.json")
	if err := os.WriteFile(basePath, []byte(`{"name":"fromfile"}`), 0o600); err != nil {
		t.Fatalf("write base: %v", err)
	}

	loader := Loader[sample]{
		Defaults: func() sample { return sample{Name: "default"} },
		Validate: func(cfg sample) error {
			if cfg.Name == "" {
				return errors.New("name required")
			}
			return nil
		},
	}

	cfg, err := loader.Load(Profile{BasePath: basePath})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Name != "fromfile" {
		t.Fatalf("expected name from file, got %q", cfg.Name)
	}
}
