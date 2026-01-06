package httpclient

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type stubTransport struct {
	responses []int
	errs      []error
	calls     int
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	idx := s.calls
	s.calls++

	if idx < len(s.errs) && s.errs[idx] != nil {
		return nil, s.errs[idx]
	}

	status := http.StatusOK
	if idx < len(s.responses) {
		status = s.responses[idx]
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func TestRetryRoundTripper(t *testing.T) {
	transport := &stubTransport{responses: []int{http.StatusServiceUnavailable, http.StatusOK}}
	retry := RetryRoundTripper{
		Base: transport,
		Options: RetryOptions{
			MaxRetries: 1,
			Backoff:    func(int) time.Duration { return 0 },
		},
		Sleep: func(time.Duration) {},
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := retry.RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if transport.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", transport.calls)
	}
}

func TestRetryRoundTripperNonReplayableBody(t *testing.T) {
	transport := &stubTransport{errs: []error{errors.New("network")}}
	retry := RetryRoundTripper{
		Base: transport,
		Options: RetryOptions{
			MaxRetries: 1,
			Backoff:    func(int) time.Duration { return 0 },
		},
		Sleep: func(time.Duration) {},
	}

	req, _ := http.NewRequest(http.MethodPost, "http://example.com", io.NopCloser(strings.NewReader("body")))
	_, err := retry.RoundTrip(req)
	if err == nil {
		t.Fatalf("expected error")
	}
	if transport.calls != 1 {
		t.Fatalf("expected 1 call, got %d", transport.calls)
	}
}

func TestCircuitBreaker(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	transport := &stubTransport{errs: []error{errors.New("fail"), nil}, responses: []int{http.StatusOK}}
	breaker := NewCircuitBreaker(CircuitBreakerOptions{MaxFailures: 1, ResetTimeout: time.Minute, Now: func() time.Time { return now }})
	wrapper := BreakerRoundTripper{Base: transport, Breaker: breaker}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if _, err := wrapper.RoundTrip(req); err == nil {
		t.Fatalf("expected error")
	}

	if _, err := wrapper.RoundTrip(req); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected circuit open, got %v", err)
	}

	now = now.Add(2 * time.Minute)
	resp, err := wrapper.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected success after reset, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
