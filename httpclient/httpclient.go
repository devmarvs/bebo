package httpclient

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

// ErrCircuitOpen indicates the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker open")

// ErrBodyNotReplayable indicates a request body cannot be retried.
var ErrBodyNotReplayable = errors.New("request body is not replayable")

// BackoffFunc returns the backoff duration for a retry attempt.
type BackoffFunc func(attempt int) time.Duration

// RetryDecider decides whether a request should be retried.
type RetryDecider func(req *http.Request, resp *http.Response, err error) bool

// RetryOptions configures retry behavior.
type RetryOptions struct {
	MaxRetries int
	Backoff    BackoffFunc
	RetryIf    RetryDecider
	OnRetry    func(attempt int, err error, resp *http.Response)
}

// RetryRoundTripper retries requests based on RetryOptions.
type RetryRoundTripper struct {
	Base    http.RoundTripper
	Options RetryOptions
	Sleep   func(time.Duration)
}

// RoundTrip executes the request with retry behavior.
func (r *RetryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := r.Base
	if base == nil {
		base = http.DefaultTransport
	}

	opts := normalizeRetryOptions(r.Options)
	sleep := r.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	if req == nil {
		return nil, errors.New("request is nil")
	}

	attempt := 0
	var resp *http.Response
	var err error
	for {
		currentReq, cloneErr := cloneRequest(req, attempt)
		if cloneErr != nil {
			if errors.Is(cloneErr, ErrBodyNotReplayable) && attempt > 0 {
				return resp, err
			}
			return nil, cloneErr
		}

		resp, err = base.RoundTrip(currentReq)
		if attempt >= opts.MaxRetries || !opts.RetryIf(req, resp, err) {
			return resp, err
		}

		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		attempt++
		if opts.OnRetry != nil {
			opts.OnRetry(attempt, err, resp)
		}

		wait := opts.Backoff(attempt)
		if wait > 0 {
			if err := sleepWithContext(req.Context(), wait, sleep); err != nil {
				return nil, err
			}
		}
	}
}

// DefaultRetryOptions returns a retry configuration with exponential backoff.
func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxRetries: 2,
		Backoff:    ExponentialBackoff(100*time.Millisecond, 2*time.Second),
		RetryIf:    DefaultRetryDecider,
	}
}

// ExponentialBackoff returns a backoff function with exponential growth.
func ExponentialBackoff(base, max time.Duration) BackoffFunc {
	if base <= 0 {
		base = 100 * time.Millisecond
	}
	if max <= 0 {
		max = 2 * time.Second
	}
	return func(attempt int) time.Duration {
		if attempt <= 0 {
			return base
		}
		delay := base << (attempt - 1)
		if delay > max {
			return max
		}
		return delay
	}
}

// DefaultRetryDecider retries idempotent methods on network errors or 5xx/429 responses.
func DefaultRetryDecider(req *http.Request, resp *http.Response, err error) bool {
	if req == nil {
		return false
	}
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		return isIdempotent(req.Method)
	}
	if resp == nil {
		return false
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		return isIdempotent(req.Method)
	}
	return false
}

// BreakerDecider decides whether a response/error should trip the breaker.
type BreakerDecider func(req *http.Request, resp *http.Response, err error) bool

// CircuitBreaker implements a simple failure-based breaker.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            breakerState
	failures         int
	openedAt         time.Time
	halfOpenInFlight bool
	maxFailures      int
	resetTimeout     time.Duration
	now              func() time.Time
	onStateChange    func(state string)
}

type breakerState int

const (
	stateClosed breakerState = iota
	stateOpen
	stateHalfOpen
)

// CircuitBreakerOptions configures a CircuitBreaker.
type CircuitBreakerOptions struct {
	MaxFailures   int
	ResetTimeout  time.Duration
	Now           func() time.Time
	OnStateChange func(state string)
}

// NewCircuitBreaker builds a CircuitBreaker with defaults.
func NewCircuitBreaker(options CircuitBreakerOptions) *CircuitBreaker {
	maxFailures := options.MaxFailures
	if maxFailures <= 0 {
		maxFailures = 5
	}
	reset := options.ResetTimeout
	if reset <= 0 {
		reset = 30 * time.Second
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &CircuitBreaker{
		maxFailures:   maxFailures,
		resetTimeout:  reset,
		now:           now,
		onStateChange: options.OnStateChange,
	}
}

// Allow returns ErrCircuitOpen when the breaker is open.
func (c *CircuitBreaker) Allow() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()

	switch c.state {
	case stateOpen:
		if now.Sub(c.openedAt) >= c.resetTimeout {
			c.state = stateHalfOpen
			c.halfOpenInFlight = false
			c.changeState("half-open")
		} else {
			return ErrCircuitOpen
		}
	case stateHalfOpen:
		if c.halfOpenInFlight {
			return ErrCircuitOpen
		}
	}

	if c.state == stateHalfOpen {
		c.halfOpenInFlight = true
	}
	return nil
}

// Record reports the success/failure of a request.
func (c *CircuitBreaker) Record(success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()

	switch c.state {
	case stateHalfOpen:
		c.halfOpenInFlight = false
		if success {
			c.state = stateClosed
			c.failures = 0
			c.changeState("closed")
			return
		}
		c.state = stateOpen
		c.failures = 0
		c.openedAt = now
		c.changeState("open")
		return
	case stateOpen:
		return
	}

	if success {
		c.failures = 0
		return
	}

	c.failures++
	if c.failures >= c.maxFailures {
		c.state = stateOpen
		c.failures = 0
		c.openedAt = now
		c.changeState("open")
	}
}

func (c *CircuitBreaker) changeState(state string) {
	if c.onStateChange != nil {
		c.onStateChange(state)
	}
}

// BreakerRoundTripper wraps a base transport with a CircuitBreaker.
type BreakerRoundTripper struct {
	Base       http.RoundTripper
	Breaker    *CircuitBreaker
	ShouldTrip BreakerDecider
}

// RoundTrip executes the request with circuit breaker protection.
func (b *BreakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := b.Base
	if base == nil {
		base = http.DefaultTransport
	}
	if b.Breaker == nil {
		return base.RoundTrip(req)
	}

	if err := b.Breaker.Allow(); err != nil {
		return nil, err
	}

	resp, err := base.RoundTrip(req)
	trip := b.ShouldTrip
	if trip == nil {
		trip = DefaultBreakerDecider
	}

	success := !trip(req, resp, err)
	b.Breaker.Record(success)
	return resp, err
}

// DefaultBreakerDecider trips on network errors or 5xx responses.
func DefaultBreakerDecider(req *http.Request, resp *http.Response, err error) bool {
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		return true
	}
	if resp == nil {
		return false
	}
	return resp.StatusCode >= http.StatusInternalServerError
}

// ClientOptions configures a default HTTP client.
type ClientOptions struct {
	Timeout           time.Duration
	Transport         http.RoundTripper
	Retry             RetryOptions
	Breaker           *CircuitBreaker
	ShouldTrip        BreakerDecider
	PropagateMetadata bool
}

// DefaultClientOptions returns a baseline client configuration.
func DefaultClientOptions() ClientOptions {
	return ClientOptions{Timeout: 30 * time.Second}
}

// NewClient builds an http.Client with retries and breaker support.
func NewClient(options ClientOptions) *http.Client {
	transport := options.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	if options.Retry.MaxRetries > 0 {
		transport = &RetryRoundTripper{Base: transport, Options: options.Retry}
	}
	if options.Breaker != nil {
		transport = &BreakerRoundTripper{Base: transport, Breaker: options.Breaker, ShouldTrip: options.ShouldTrip}
	}
	if options.PropagateMetadata {
		transport = &MetadataRoundTripper{Base: transport}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   options.Timeout,
	}
}

func normalizeRetryOptions(options RetryOptions) RetryOptions {
	if options.Backoff == nil {
		options.Backoff = ExponentialBackoff(100*time.Millisecond, 2*time.Second)
	}
	if options.RetryIf == nil {
		options.RetryIf = DefaultRetryDecider
	}
	return options
}

func cloneRequest(req *http.Request, attempt int) (*http.Request, error) {
	if attempt == 0 {
		return req, nil
	}

	if req.Body == nil {
		return req.Clone(req.Context()), nil
	}
	if req.GetBody == nil {
		return nil, ErrBodyNotReplayable
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	clone := req.Clone(req.Context())
	clone.Body = body
	return clone, nil
}

func sleepWithContext(ctx context.Context, delay time.Duration, sleep func(time.Duration)) error {
	if delay <= 0 {
		return nil
	}
	if ctx == nil {
		sleep(delay)
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
