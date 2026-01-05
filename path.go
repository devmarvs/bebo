package bebo

import (
	"net/url"
	"sort"
	"strings"
)

// PathWithQuery builds a URL path from a named route and attaches query params.
func (a *App) PathWithQuery(name string, params map[string]string, query map[string]string) (string, bool) {
	path, ok := a.Path(name, params)
	if !ok {
		return "", false
	}
	return buildQuery(path, query), true
}

func buildPath(pattern string, params map[string]string) (string, bool) {
	if pattern == "" || pattern == "/" {
		return "/", true
	}

	parts := strings.Split(strings.Trim(pattern, "/"), "/")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			key := strings.TrimPrefix(part, ":")
			value, ok := params[key]
			if !ok {
				return "", false
			}
			out = append(out, url.PathEscape(value))
			continue
		}
		if strings.HasPrefix(part, "*") {
			key := strings.TrimPrefix(part, "*")
			value, ok := params[key]
			if !ok {
				return "", false
			}
			value = strings.TrimPrefix(value, "/")
			out = append(out, value)
			continue
		}
		out = append(out, part)
	}

	return "/" + strings.Join(out, "/"), true
}

func buildQuery(path string, query map[string]string) string {
	if len(query) == 0 {
		return path
	}
	values := url.Values{}
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values.Set(key, query[key])
	}

	return path + "?" + values.Encode()
}
