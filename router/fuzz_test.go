package router

import (
	"strings"
	"testing"
)

func FuzzRouterMatch(f *testing.F) {
	f.Add("GET", "/users/:id", "/users/123")
	f.Add("POST", "/files/*path", "/files/a/b")
	f.Add("GET", "/", "/")

	f.Fuzz(func(t *testing.T, method, pattern, path string) {
		if method == "" {
			return
		}
		if pattern == "" {
			pattern = "/"
		}
		if !strings.HasPrefix(pattern, "/") {
			pattern = "/" + pattern
		}
		if path == "" {
			path = "/"
		}

		r := New()
		if _, err := r.Add(method, pattern); err != nil {
			return
		}
		_, _, _ = r.Match(method, path)
	})
}
