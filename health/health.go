package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"
)

// CheckFunc runs a health or readiness check.
type CheckFunc func(context.Context) error

// CheckResult reports a single check.
type CheckResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

// Report describes health status.
type Report struct {
	Status      string        `json:"status"`
	Checks      []CheckResult `json:"checks"`
	DurationMS  int64         `json:"duration_ms"`
	CheckedAt   time.Time     `json:"checked_at"`
	ChecksReady bool          `json:"ready"`
}

// Option configures a Registry.
type Option func(*Registry)

// WithTimeout sets a timeout for all checks.
func WithTimeout(timeout time.Duration) Option {
	return func(r *Registry) {
		r.timeout = timeout
	}
}

// Registry stores health and readiness checks.
type Registry struct {
	mu      sync.RWMutex
	checks  map[string]CheckFunc
	ready   map[string]CheckFunc
	timeout time.Duration
}

// New creates a Registry.
func New(options ...Option) *Registry {
	registry := &Registry{
		checks: make(map[string]CheckFunc),
		ready:  make(map[string]CheckFunc),
	}
	for _, opt := range options {
		opt(registry)
	}
	return registry
}

// Add registers a liveness check.
func (r *Registry) Add(name string, check CheckFunc) {
	r.mu.Lock()
	r.checks[name] = check
	r.mu.Unlock()
}

// AddReady registers a readiness check.
func (r *Registry) AddReady(name string, check CheckFunc) {
	r.mu.Lock()
	r.ready[name] = check
	r.mu.Unlock()
}

// Remove deletes a liveness check.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	delete(r.checks, name)
	r.mu.Unlock()
}

// RemoveReady deletes a readiness check.
func (r *Registry) RemoveReady(name string) {
	r.mu.Lock()
	delete(r.ready, name)
	r.mu.Unlock()
}

// Handler returns a handler for liveness checks.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		report, status := r.report(req.Context(), r.snapshot(r.checks), false)
		writeReport(w, report, status)
	})
}

// ReadyHandler returns a handler for readiness checks.
func (r *Registry) ReadyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		report, status := r.report(req.Context(), r.snapshot(r.ready), true)
		writeReport(w, report, status)
	})
}

func (r *Registry) report(ctx context.Context, checks map[string]CheckFunc, ready bool) (Report, int) {
	start := time.Now()
	results := make([]CheckResult, 0, len(checks))

	names := make([]string, 0, len(checks))
	for name := range checks {
		names = append(names, name)
	}
	sort.Strings(names)

	status := http.StatusOK
	for _, name := range names {
		check := checks[name]
		result := runCheck(ctx, check, r.timeout)
		result.Name = name
		results = append(results, result)
		if result.Status != "ok" {
			status = http.StatusServiceUnavailable
		}
	}

	report := Report{
		Status:      statusLabel(status),
		Checks:      results,
		DurationMS:  time.Since(start).Milliseconds(),
		CheckedAt:   time.Now().UTC(),
		ChecksReady: ready,
	}
	return report, status
}

func runCheck(ctx context.Context, check CheckFunc, timeout time.Duration) CheckResult {
	if check == nil {
		return CheckResult{Status: "ok"}
	}

	checkCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		checkCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	start := time.Now()
	err := check(checkCtx)
	result := CheckResult{DurationMS: time.Since(start).Milliseconds()}
	if err != nil {
		result.Status = "fail"
		result.Error = err.Error()
		return result
	}
	result.Status = "ok"
	return result
}

func (r *Registry) snapshot(source map[string]CheckFunc) map[string]CheckFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()

	copy := make(map[string]CheckFunc, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}

func statusLabel(code int) string {
	if code >= http.StatusBadRequest {
		return "fail"
	}
	return "ok"
}

func writeReport(w http.ResponseWriter, report Report, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(report)
}
