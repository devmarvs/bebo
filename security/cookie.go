package security

import (
	"net/http"
	"time"
)

// CookieOptions configures secure cookie defaults.
type CookieOptions struct {
	DisableDefaults bool
	Path            string
	Domain          string
	MaxAge          int
	Expires         time.Time
	Secure          bool
	HTTPOnly        bool
	SameSite        http.SameSite
	Partitioned     bool
}

// NewSecureCookie creates a cookie with secure defaults.
func NewSecureCookie(name, value string, options CookieOptions) *http.Cookie {
	cfg := options
	if !cfg.DisableDefaults {
		if cfg.Path == "" {
			cfg.Path = "/"
		}
		if !cfg.Secure {
			cfg.Secure = true
		}
		if !cfg.HTTPOnly {
			cfg.HTTPOnly = true
		}
		if cfg.SameSite == 0 {
			cfg.SameSite = http.SameSiteLaxMode
		}
	}

	cookie := &http.Cookie{
		Name:        name,
		Value:       value,
		Path:        cfg.Path,
		Domain:      cfg.Domain,
		MaxAge:      cfg.MaxAge,
		Expires:     cfg.Expires,
		Secure:      cfg.Secure,
		HttpOnly:    cfg.HTTPOnly,
		SameSite:    cfg.SameSite,
		Partitioned: cfg.Partitioned,
	}
	return cookie
}

// SetSecureCookie writes a secure cookie to the response.
func SetSecureCookie(w http.ResponseWriter, name, value string, options CookieOptions) {
	http.SetCookie(w, NewSecureCookie(name, value, options))
}
