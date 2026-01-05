package bebo

import "time"

type routeConfig struct {
	name       string
	timeout    time.Duration
	host       string
	middleware []Middleware
}

// RouteOption customizes route registration.
type RouteOption func(*routeConfig)

// WithName assigns a name to a route.
func WithName(name string) RouteOption {
	return func(cfg *routeConfig) {
		cfg.name = name
	}
}

// WithTimeout sets a per-route timeout.
func WithTimeout(timeout time.Duration) RouteOption {
	return func(cfg *routeConfig) {
		cfg.timeout = timeout
	}
}

// WithHost scopes a route to a host or wildcard (e.g. "api.example.com", "*.example.com").
func WithHost(host string) RouteOption {
	return func(cfg *routeConfig) {
		cfg.host = host
	}
}

// WithMiddleware attaches middleware to a route.
func WithMiddleware(middleware ...Middleware) RouteOption {
	return func(cfg *routeConfig) {
		cfg.middleware = append(cfg.middleware, middleware...)
	}
}
