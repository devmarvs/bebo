package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Do executes a request against a handler.
func Do(t *testing.T, handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// MustStatus asserts the response status code.
func MustStatus(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("expected status %d, got %d", status, rec.Code)
	}
}

// MustHeader asserts a response header value.
func MustHeader(t *testing.T, rec *httptest.ResponseRecorder, key, value string) {
	t.Helper()
	if got := rec.Header().Get(key); got != value {
		t.Fatalf("expected header %s=%q, got %q", key, value, got)
	}
}

// DecodeJSON decodes a JSON response into dst.
func DecodeJSON(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(dst); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}
