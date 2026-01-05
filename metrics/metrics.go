package metrics

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Snapshot captures current metrics values.
type Snapshot struct {
	Requests int64           `json:"requests"`
	Errors   int64           `json:"errors"`
	InFlight int64           `json:"in_flight"`
	Latency  LatencySnapshot `json:"latency"`
	Statuses map[int]int64   `json:"statuses"`
}

// LatencySnapshot captures latency statistics.
type LatencySnapshot struct {
	Count   int64           `json:"count"`
	Total   time.Duration   `json:"total"`
	Min     time.Duration   `json:"min"`
	Max     time.Duration   `json:"max"`
	Buckets []LatencyBucket `json:"buckets"`
}

// LatencyBucket captures histogram counts.
type LatencyBucket struct {
	UpperBound time.Duration `json:"le"`
	Count      int64         `json:"count"`
}

// Registry tracks request metrics.
type Registry struct {
	mu       sync.Mutex
	requests int64
	errors   int64
	inFlight int64
	latency  LatencySnapshot
	statuses map[int]int64
	buckets  []time.Duration
	counts   []int64
}

// New creates a new registry with default buckets.
func New() *Registry {
	return NewWithBuckets(defaultBuckets())
}

// NewWithBuckets creates a new registry with custom buckets.
func NewWithBuckets(buckets []time.Duration) *Registry {
	if len(buckets) == 0 {
		buckets = defaultBuckets()
	}
	return &Registry{
		statuses: make(map[int]int64),
		buckets:  buckets,
		counts:   make([]int64, len(buckets)),
	}
}

// Start marks the start of a request.
func (r *Registry) Start() time.Time {
	r.mu.Lock()
	r.inFlight++
	r.mu.Unlock()
	return time.Now()
}

// End records a completed request.
func (r *Registry) End(start time.Time, status int, err error) {
	duration := time.Since(start)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.inFlight--
	r.requests++
	if err != nil || status >= 500 {
		r.errors++
	}

	r.latency.Count++
	r.latency.Total += duration
	if r.latency.Min == 0 || duration < r.latency.Min {
		r.latency.Min = duration
	}
	if duration > r.latency.Max {
		r.latency.Max = duration
	}

	for i, bound := range r.buckets {
		if duration <= bound {
			r.counts[i]++
			break
		}
	}

	if status != 0 {
		r.statuses[status]++
	}
}

// Snapshot returns a copy of metrics data.
func (r *Registry) Snapshot() Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	statuses := make(map[int]int64, len(r.statuses))
	for code, count := range r.statuses {
		statuses[code] = count
	}

	buckets := make([]LatencyBucket, len(r.buckets))
	for i, bound := range r.buckets {
		buckets[i] = LatencyBucket{UpperBound: bound, Count: r.counts[i]}
	}

	latency := r.latency
	latency.Buckets = buckets

	return Snapshot{
		Requests: r.requests,
		Errors:   r.errors,
		InFlight: r.inFlight,
		Latency:  latency,
		Statuses: statuses,
	}
}

// Handler exposes metrics as JSON.
func Handler(registry *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if registry == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(registry.Snapshot())
	})
}

func defaultBuckets() []time.Duration {
	return []time.Duration{
		5 * time.Millisecond,
		10 * time.Millisecond,
		25 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		5 * time.Second,
	}
}
