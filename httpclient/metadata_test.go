package httpclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devmarvs/bebo"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestMetadataRoundTripperInjects(t *testing.T) {
	metadata := bebo.RequestMetadata{
		RequestID:   "req-1",
		Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Tracestate:  "vendor=demo",
	}
	ctx := bebo.WithRequestMetadata(context.Background(), metadata)
	request := httptest.NewRequest(http.MethodGet, "http://example.com", nil).WithContext(ctx)

	rt := &MetadataRoundTripper{Base: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get(bebo.RequestIDHeader); got != metadata.RequestID {
			t.Fatalf("expected request id header")
		}
		if got := req.Header.Get(bebo.TraceparentHeader); got != metadata.Traceparent {
			t.Fatalf("expected traceparent header")
		}
		if got := req.Header.Get(bebo.TracestateHeader); got != metadata.Tracestate {
			t.Fatalf("expected tracestate header")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})}

	resp, err := rt.RoundTrip(request)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	_ = resp.Body.Close()
}

func TestMetadataRoundTripperRespectsExistingHeaders(t *testing.T) {
	metadata := bebo.RequestMetadata{RequestID: "req-1"}
	ctx := bebo.WithRequestMetadata(context.Background(), metadata)
	request := httptest.NewRequest(http.MethodGet, "http://example.com", nil).WithContext(ctx)
	request.Header.Set(bebo.RequestIDHeader, "req-override")

	rt := &MetadataRoundTripper{Base: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get(bebo.RequestIDHeader); got != "req-override" {
			t.Fatalf("expected existing request id to remain")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})}

	resp, err := rt.RoundTrip(request)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	_ = resp.Body.Close()
}
