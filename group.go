package bebo

import "strings"

// Group defines a route group with a common prefix and middleware.
type Group struct {
	app        *App
	prefix     string
	middleware []Middleware
}

// Group creates a new route group.
func (a *App) Group(prefix string, middleware ...Middleware) *Group {
	return &Group{app: a, prefix: cleanPrefix(prefix), middleware: middleware}
}

// Version creates a versioned group under /api/{version}.
func (a *App) Version(version string, middleware ...Middleware) *Group {
	version = strings.Trim(version, "/")
	return a.Group("/api/"+version, middleware...)
}

// Group creates a nested group.
func (g *Group) Group(prefix string, middleware ...Middleware) *Group {
	joined := joinPaths(g.prefix, prefix)
	combined := append([]Middleware{}, g.middleware...)
	combined = append(combined, middleware...)
	return &Group{app: g.app, prefix: joined, middleware: combined}
}

// Route registers a route with options in the group.
func (g *Group) Route(method, path string, handler Handler, options ...RouteOption) {
	g.handle(method, path, handler, nil, options...)
}

// GET registers a GET route in the group.
func (g *Group) GET(path string, handler Handler, middleware ...Middleware) {
	g.handle("GET", path, handler, middleware)
}

func (g *Group) HEAD(path string, handler Handler, middleware ...Middleware) {
	g.handle("HEAD", path, handler, middleware)
}

// POST registers a POST route in the group.
func (g *Group) POST(path string, handler Handler, middleware ...Middleware) {
	g.handle("POST", path, handler, middleware)
}

// PUT registers a PUT route in the group.
func (g *Group) PUT(path string, handler Handler, middleware ...Middleware) {
	g.handle("PUT", path, handler, middleware)
}

// PATCH registers a PATCH route in the group.
func (g *Group) PATCH(path string, handler Handler, middleware ...Middleware) {
	g.handle("PATCH", path, handler, middleware)
}

// DELETE registers a DELETE route in the group.
func (g *Group) DELETE(path string, handler Handler, middleware ...Middleware) {
	g.handle("DELETE", path, handler, middleware)
}

// Handle registers a route in the group for an arbitrary method.
func (g *Group) Handle(method, path string, handler Handler, middleware ...Middleware) {
	g.handle(method, path, handler, middleware)
}

func (g *Group) handle(method, path string, handler Handler, middleware []Middleware, options ...RouteOption) {
	fullPath := joinPaths(g.prefix, path)
	combined := append([]Middleware{}, g.middleware...)
	combined = append(combined, middleware...)
	g.app.handleWithOptions(method, fullPath, handler, combined, options...)
}

func joinPaths(base, path string) string {
	if base == "" {
		return cleanPrefix(path)
	}
	if path == "" || path == "/" {
		return cleanPrefix(base)
	}

	base = cleanPrefix(base)
	path = cleanPrefix(path)

	if base == "/" {
		return path
	}
	return strings.TrimRight(base, "/") + path
}

func cleanPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if len(prefix) > 1 {
		prefix = strings.TrimRight(prefix, "/")
	}
	return prefix
}
