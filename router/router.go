package router

import (
	"errors"
	"strings"
)

// RouteID identifies a route.
type RouteID int

// Params holds path parameters.
type Params map[string]string

type segmentType int

const (
	segmentStatic segmentType = iota
	segmentParam
	segmentWildcard
)

type segment struct {
	kind  segmentType
	value string
}

type route struct {
	id       RouteID
	method   string
	pattern  string
	segments []segment
}

// Router matches HTTP methods and paths.
type Router struct {
	routes []route
	nextID RouteID
}

// New creates a Router.
func New() *Router {
	return &Router{}
}

// Add registers a route and returns its id.
func (r *Router) Add(method, pattern string) (RouteID, error) {
	if method == "" {
		return 0, errors.New("method required")
	}
	if pattern == "" || pattern[0] != '/' {
		return 0, errors.New("pattern must start with '/'")
	}

	segments, err := parsePattern(pattern)
	if err != nil {
		return 0, err
	}

	id := r.nextID
	r.nextID++
	r.routes = append(r.routes, route{method: method, pattern: pattern, segments: segments, id: id})
	return id, nil
}

// Match finds a matching route.
func (r *Router) Match(method, path string) (RouteID, Params, bool) {
	if path == "" {
		path = "/"
	}

	parts := splitPath(path)

	for _, rt := range r.routes {
		if !methodMatches(rt.method, method) {
			continue
		}

		params := make(Params)
		if matchSegments(rt.segments, parts, params) {
			return rt.id, params, true
		}
	}

	return 0, nil, false
}

func methodMatches(routeMethod, requestMethod string) bool {
	return routeMethod == "*" || routeMethod == requestMethod
}

func parsePattern(pattern string) ([]segment, error) {
	if pattern == "/" {
		return []segment{}, nil
	}

	parts := splitPath(pattern)
	segments := make([]segment, 0, len(parts))
	for i, part := range parts {
		if part == "" {
			return nil, errors.New("empty path segment")
		}
		if strings.HasPrefix(part, ":") {
			name := strings.TrimPrefix(part, ":")
			if name == "" {
				return nil, errors.New("param name required")
			}
			segments = append(segments, segment{kind: segmentParam, value: name})
			continue
		}
		if strings.HasPrefix(part, "*") {
			name := strings.TrimPrefix(part, "*")
			if name == "" {
				return nil, errors.New("wildcard name required")
			}
			if i != len(parts)-1 {
				return nil, errors.New("wildcard must be last segment")
			}
			segments = append(segments, segment{kind: segmentWildcard, value: name})
			continue
		}

		segments = append(segments, segment{kind: segmentStatic, value: part})
	}

	return segments, nil
}

func splitPath(path string) []string {
	clean := strings.Trim(path, "/")
	if clean == "" {
		return []string{}
	}
	return strings.Split(clean, "/")
}

func matchSegments(pattern []segment, parts []string, params Params) bool {
	if len(pattern) == 0 && len(parts) == 0 {
		return true
	}

	pi := 0
	for pi < len(pattern) {
		seg := pattern[pi]
		switch seg.kind {
		case segmentStatic:
			if pi >= len(parts) || parts[pi] != seg.value {
				return false
			}
		case segmentParam:
			if pi >= len(parts) {
				return false
			}
			params[seg.value] = parts[pi]
		case segmentWildcard:
			params[seg.value] = strings.Join(parts[pi:], "/")
			return true
		}
		pi++
	}

	return pi == len(parts)
}
