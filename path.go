package bebo

import (
	"net/url"
	"strings"
)

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
