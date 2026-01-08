package config

import (
	"errors"
	"strings"
)

// Validate validates config values.
func Validate(cfg Config) error {
	var issues []string

	if cfg.ReadTimeout < 0 {
		issues = append(issues, "read_timeout must be >= 0")
	}
	if cfg.WriteTimeout < 0 {
		issues = append(issues, "write_timeout must be >= 0")
	}
	if cfg.IdleTimeout < 0 {
		issues = append(issues, "idle_timeout must be >= 0")
	}
	if cfg.ReadHeaderTimeout < 0 {
		issues = append(issues, "read_header_timeout must be >= 0")
	}
	if cfg.ShutdownTimeout < 0 {
		issues = append(issues, "shutdown_timeout must be >= 0")
	}
	if cfg.MaxHeaderBytes < 0 {
		issues = append(issues, "max_header_bytes must be >= 0")
	}

	if cfg.LogLevel != "" && !validLogLevel(cfg.LogLevel) {
		issues = append(issues, "log_level must be one of debug|info|warn|error")
	}
	if cfg.LogFormat != "" && !validLogFormat(cfg.LogFormat) {
		issues = append(issues, "log_format must be one of text|json")
	}

	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validLogLevel(level string) bool {
	switch strings.ToLower(level) {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}

func validLogFormat(format string) bool {
	switch strings.ToLower(format) {
	case "text", "json":
		return true
	default:
		return false
	}
}
