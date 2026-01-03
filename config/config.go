package config

import "time"

// Config holds app configuration.
type Config struct {
	Address           string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
	MaxHeaderBytes    int

	TemplatesDir   string
	LayoutTemplate string

	LogLevel  string
	LogFormat string
}

// Default returns safe defaults.
func Default() Config {
	return Config{
		Address:           ":8080",
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		ShutdownTimeout:   10 * time.Second,
		MaxHeaderBytes:    1 << 20,
		TemplatesDir:      "",
		LayoutTemplate:    "",
		LogLevel:          "info",
		LogFormat:         "text",
	}
}
