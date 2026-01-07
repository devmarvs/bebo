package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIEndpoints(t *testing.T) {
	app := NewApp()
	server := httptest.NewServer(app)
	defer server.Close()

	t.Run("health", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/health", nil)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var payload map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if payload["status"] != "ok" {
			t.Fatalf("expected status ok, got %q", payload["status"])
		}
	})

	t.Run("user", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/users/42", nil)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var payload map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if payload["id"] != "42" {
			t.Fatalf("expected id 42, got %q", payload["id"])
		}
	})
}
