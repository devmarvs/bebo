package config

import (
	"os"
	"strconv"
	"time"
)

// LoadFromEnv applies environment overrides with a prefix (e.g. BEBO_).
func LoadFromEnv(prefix string, base Config) Config {
	get := func(key string) string { return os.Getenv(prefix + key) }

	if value := get("ADDRESS"); value != "" {
		base.Address = value
	}
	if value := get("READ_TIMEOUT"); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			base.ReadTimeout = d
		}
	}
	if value := get("WRITE_TIMEOUT"); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			base.WriteTimeout = d
		}
	}
	if value := get("IDLE_TIMEOUT"); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			base.IdleTimeout = d
		}
	}
	if value := get("READ_HEADER_TIMEOUT"); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			base.ReadHeaderTimeout = d
		}
	}
	if value := get("SHUTDOWN_TIMEOUT"); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			base.ShutdownTimeout = d
		}
	}
	if value := get("MAX_HEADER_BYTES"); value != "" {
		if n, err := strconv.Atoi(value); err == nil {
			base.MaxHeaderBytes = n
		}
	}
	if value := get("TEMPLATES_DIR"); value != "" {
		base.TemplatesDir = value
	}
	if value := get("LAYOUT_TEMPLATE"); value != "" {
		base.LayoutTemplate = value
	}
	if value := get("TEMPLATE_RELOAD"); value != "" {
		if enabled, err := strconv.ParseBool(value); err == nil {
			base.TemplateReload = enabled
		}
	}
	if value := get("LOG_LEVEL"); value != "" {
		base.LogLevel = value
	}
	if value := get("LOG_FORMAT"); value != "" {
		base.LogFormat = value
	}

	return base
}
