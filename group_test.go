package bebo

import "testing"

func TestJoinPaths(t *testing.T) {
	cases := []struct {
		base string
		path string
		want string
	}{
		{"", "/users", "/users"},
		{"/api", "/v1", "/api/v1"},
		{"/api/", "v1", "/api/v1"},
		{"/", "/health", "/health"},
		{"/api", "/", "/api"},
		{"api", "v1/users", "/api/v1/users"},
	}

	for _, tc := range cases {
		if got := joinPaths(tc.base, tc.path); got != tc.want {
			t.Fatalf("joinPaths(%q, %q) = %q, want %q", tc.base, tc.path, got, tc.want)
		}
	}
}
