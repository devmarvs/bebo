package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrometheusHandler(t *testing.T) {
	reg := New()
	start := reg.Start()
	reg.End(start, http.StatusOK, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	PrometheusHandler(reg).ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "bebo_requests_total 1") {
		t.Fatalf("expected requests metric")
	}
	if !strings.Contains(body, "bebo_latency_seconds_count 1") {
		t.Fatalf("expected latency count")
	}
}
