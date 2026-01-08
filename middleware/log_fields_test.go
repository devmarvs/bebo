package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/router"
)

func TestLogTraceSpanIDs(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	rec := httptest.NewRecorder()
	ctx := bebo.NewContext(rec, req, router.Params{}, bebo.New())
	recorder := newResponseRecorder(rec)

	traceAttr := LogTraceID()(ctx, recorder, 0)
	if got := traceAttr.Value.String(); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("expected trace_id, got %q", got)
	}

	spanAttr := LogSpanID()(ctx, recorder, 0)
	if got := spanAttr.Value.String(); got != "00f067aa0ba902b7" {
		t.Fatalf("expected span_id, got %q", got)
	}
}

func TestLogRequestBytes(t *testing.T) {
	body := "payload"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	rec := httptest.NewRecorder()
	ctx := bebo.NewContext(rec, req, router.Params{}, bebo.New())
	recorder := newResponseRecorder(rec)

	attr := LogRequestBytes()(ctx, recorder, 0)
	if got := attr.Value.Int64(); got != int64(len(body)) {
		t.Fatalf("expected request_bytes %d, got %d", len(body), got)
	}
}
