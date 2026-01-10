package config

import "time"

// Config holds app configuration.
type Config struct {
	Address           string        `json:"address"`
	ReadTimeout       time.Duration `json:"read_timeout"`
	WriteTimeout      time.Duration `json:"write_timeout"`
	IdleTimeout       time.Duration `json:"idle_timeout"`
	ReadHeaderTimeout time.Duration `json:"read_header_timeout"`
	ShutdownTimeout   time.Duration `json:"shutdown_timeout"`
	MaxHeaderBytes    int           `json:"max_header_bytes"`

	TemplatesDir   string `json:"templates_dir"`
	LayoutTemplate string `json:"layout_template"`
	TemplateReload bool   `json:"template_reload"`

	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`
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
		TemplateReload:    false,
		LogLevel:          "info",
		LogFormat:         "text",
	}
}
