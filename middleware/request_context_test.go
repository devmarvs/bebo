package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/testutil"
)

func TestRequestContextMetadata(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(bebo.RequestIDHeader, "req-1")
	req.Header.Set(bebo.TraceparentHeader, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	req.Header.Set(bebo.TracestateHeader, "congo=t61rcWkgMzE")

	var meta bebo.RequestMetadata
	mw := RequestContext()
	handler := func(ctx *bebo.Context) error {
		meta = bebo.RequestMetadataFromContext(ctx.Request.Context())
		return nil
	}

	_, err := testutil.RunMiddleware(t, []bebo.Middleware{mw}, handler, req)
	if err != nil {
		t.Fatalf("middleware: %v", err)
	}

	if meta.RequestID != "req-1" {
		t.Fatalf("expected request id req-1, got %q", meta.RequestID)
	}
	if meta.Traceparent == "" {
		t.Fatalf("expected traceparent")
	}
	if meta.Tracestate != "congo=t61rcWkgMzE" {
		t.Fatalf("expected tracestate, got %q", meta.Tracestate)
	}
}

func TestRequestIDUpdatesMetadata(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	var meta bebo.RequestMetadata
	handler := func(ctx *bebo.Context) error {
		meta = bebo.RequestMetadataFromContext(ctx.Request.Context())
		return nil
	}

	_, err := testutil.RunMiddleware(t, []bebo.Middleware{RequestContext(), RequestID()}, handler, req)
	if err != nil {
		t.Fatalf("middleware: %v", err)
	}

	if meta.RequestID == "" {
		t.Fatalf("expected request id to be set")
	}
	if got := req.Header.Get(bebo.RequestIDHeader); got == "" {
		t.Fatalf("expected request id header to be set")
	}
}
