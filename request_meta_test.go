package bebo

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTraceIDs(t *testing.T) {
	traceID, spanID, ok := TraceIDs("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	if !ok {
		t.Fatalf("expected trace ids")
	}
	if traceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("unexpected trace id %q", traceID)
	}
	if spanID != "00f067aa0ba902b7" {
		t.Fatalf("unexpected span id %q", spanID)
	}
}

func TestRequestMetadataHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "req-1")
	req.Header.Set(TraceparentHeader, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	req.Header.Set(TracestateHeader, "congo=t61rcWkgMzE")

	metadata := RequestMetadataFromRequest(req)
	if metadata.RequestID != "req-1" {
		t.Fatalf("expected request id req-1")
	}

	ctx := WithRequestMetadata(req.Context(), RequestMetadata{RequestID: "ctx-id"})
	req = req.WithContext(ctx)

	metadata = RequestMetadataFromRequest(req)
	if metadata.RequestID != "ctx-id" {
		t.Fatalf("expected context request id")
	}

	out := httptest.NewRequest(http.MethodGet, "/", nil)
	InjectRequestMetadata(out, metadata)
	if out.Header.Get(RequestIDHeader) != "ctx-id" {
		t.Fatalf("expected injected request id")
	}
}
