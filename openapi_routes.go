package bebo

import (
	"errors"
	"sort"
	"strings"

	"github.com/devmarvs/bebo/openapi"
)

// OpenAPIOptions configures automatic OpenAPI route derivation.
type OpenAPIOptions struct {
	IncludeUnnamed bool
	SkipPaths      []string
	SkipMethods    []string
	TagFromHost    bool
}

// OpenAPIOption customizes OpenAPI route derivation.
type OpenAPIOption func(*OpenAPIOptions)

// DefaultOpenAPIOptions returns the default OpenAPI options.
func DefaultOpenAPIOptions() OpenAPIOptions {
	return OpenAPIOptions{IncludeUnnamed: true}
}

// WithOpenAPIIncludeUnnamed toggles inclusion of unnamed routes.
func WithOpenAPIIncludeUnnamed(enabled bool) OpenAPIOption {
	return func(options *OpenAPIOptions) {
		options.IncludeUnnamed = enabled
	}
}

// WithOpenAPISkipPaths skips paths that match the provided patterns.
func WithOpenAPISkipPaths(paths ...string) OpenAPIOption {
	return func(options *OpenAPIOptions) {
		options.SkipPaths = append([]string{}, paths...)
	}
}

// WithOpenAPISkipMethods skips specific HTTP methods.
func WithOpenAPISkipMethods(methods ...string) OpenAPIOption {
	return func(options *OpenAPIOptions) {
		options.SkipMethods = append([]string{}, methods...)
	}
}

// WithOpenAPITagFromHost applies host values as OpenAPI tags.
func WithOpenAPITagFromHost(enabled bool) OpenAPIOption {
	return func(options *OpenAPIOptions) {
		options.TagFromHost = enabled
	}
}

// AddOpenAPIRoutes derives OpenAPI operations from registered routes.
func (a *App) AddOpenAPIRoutes(builder *openapi.Builder, options ...OpenAPIOption) error {
	if builder == nil {
		return errors.New("openapi builder is required")
	}

	cfg := DefaultOpenAPIOptions()
	for _, opt := range options {
		opt(&cfg)
	}

	skipMethods := make(map[string]struct{}, len(cfg.SkipMethods))
	for _, method := range cfg.SkipMethods {
		skipMethods[strings.ToUpper(strings.TrimSpace(method))] = struct{}{}
	}

	routes := a.RoutesAll()
	for _, route := range routes {
		if route.Method == "*" {
			continue
		}
		if !cfg.IncludeUnnamed && route.Name == "" {
			continue
		}
		if shouldSkipPath(route.Pattern, cfg.SkipPaths) {
			continue
		}
		if _, ok := skipMethods[strings.ToUpper(route.Method)]; ok {
			continue
		}

		path, params := openAPIPath(route.Pattern)
		operation := openapi.Operation{
			OperationID: route.Name,
			Summary:     route.Name,
			Parameters:  params,
			Responses:   defaultOpenAPIResponses(route.Method),
		}
		if operation.Summary == "" {
			operation.Summary = strings.ToUpper(route.Method) + " " + route.Pattern
		}
		if cfg.TagFromHost && route.Host != "" {
			operation.Tags = []string{route.Host}
		}

		if err := builder.AddRoute(route.Method, path, operation); err != nil {
			return err
		}
	}

	return nil
}

// RoutesAll returns metadata for all registered routes.
func (a *App) RoutesAll() []RouteInfo {
	items := make([]RouteInfo, 0, len(a.routes))
	for _, entry := range a.routes {
		items = append(items, RouteInfo{
			Name:    entry.name,
			Method:  entry.method,
			Host:    entry.host,
			Pattern: entry.pattern,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Pattern == items[j].Pattern {
			if items[i].Method == items[j].Method {
				if items[i].Host == items[j].Host {
					return items[i].Name < items[j].Name
				}
				return items[i].Host < items[j].Host
			}
			return items[i].Method < items[j].Method
		}
		return items[i].Pattern < items[j].Pattern
	})

	return items
}

func openAPIPath(pattern string) (string, []openapi.Parameter) {
	if pattern == "" || pattern == "/" {
		return "/", nil
	}

	parts := strings.Split(strings.Trim(pattern, "/"), "/")
	segments := make([]string, 0, len(parts))
	params := make([]openapi.Parameter, 0)
	seen := make(map[string]struct{})

	for _, part := range parts {
		if part == "" {
			continue
		}
		switch {
		case strings.HasPrefix(part, ":"):
			name := strings.TrimPrefix(part, ":")
			segments = append(segments, "{"+name+"}")
			if _, ok := seen[name]; !ok {
				params = append(params, openapi.Parameter{
					Name:     name,
					In:       "path",
					Required: true,
					Schema:   &openapi.Schema{Type: "string"},
				})
				seen[name] = struct{}{}
			}
		case strings.HasPrefix(part, "*"):
			name := strings.TrimPrefix(part, "*")
			segments = append(segments, "{"+name+"}")
			if _, ok := seen[name]; !ok {
				params = append(params, openapi.Parameter{
					Name:        name,
					In:          "path",
					Required:    true,
					Description: "wildcard",
					Schema:      &openapi.Schema{Type: "string"},
				})
				seen[name] = struct{}{}
			}
		default:
			segments = append(segments, part)
		}
	}

	return "/" + strings.Join(segments, "/"), params
}

func defaultOpenAPIResponses(method string) map[string]openapi.Response {
	switch strings.ToUpper(method) {
	case "POST":
		return map[string]openapi.Response{"201": {Description: "created"}}
	case "DELETE":
		return map[string]openapi.Response{"204": {Description: "no content"}}
	default:
		return map[string]openapi.Response{"200": {Description: "ok"}}
	}
}

func shouldSkipPath(path string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
			continue
		}
		if path == pattern {
			return true
		}
	}
	return false
}
