package metrics

import (
	"fmt"
	"net/http"
	"sort"
)

// PrometheusHandler exposes metrics in Prometheus text format.
func PrometheusHandler(registry *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if registry == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		snap := registry.Snapshot()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		fmt.Fprintf(w, "# HELP bebo_requests_total Total HTTP requests\n")
		fmt.Fprintf(w, "# TYPE bebo_requests_total counter\n")
		fmt.Fprintf(w, "bebo_requests_total %d\n", snap.Requests)

		fmt.Fprintf(w, "# HELP bebo_errors_total Total HTTP errors\n")
		fmt.Fprintf(w, "# TYPE bebo_errors_total counter\n")
		fmt.Fprintf(w, "bebo_errors_total %d\n", snap.Errors)

		fmt.Fprintf(w, "# HELP bebo_in_flight In-flight HTTP requests\n")
		fmt.Fprintf(w, "# TYPE bebo_in_flight gauge\n")
		fmt.Fprintf(w, "bebo_in_flight %d\n", snap.InFlight)

		fmt.Fprintf(w, "# HELP bebo_latency_seconds Request latency\n")
		fmt.Fprintf(w, "# TYPE bebo_latency_seconds histogram\n")

		cumulative := int64(0)
		for _, bucket := range snap.Latency.Buckets {
			cumulative += bucket.Count
			fmt.Fprintf(w, "bebo_latency_seconds_bucket{le=\"%g\"} %d\n", bucket.UpperBound.Seconds(), cumulative)
		}
		fmt.Fprintf(w, "bebo_latency_seconds_bucket{le=\"+Inf\"} %d\n", snap.Latency.Count)
		fmt.Fprintf(w, "bebo_latency_seconds_sum %g\n", snap.Latency.Total.Seconds())
		fmt.Fprintf(w, "bebo_latency_seconds_count %d\n", snap.Latency.Count)

		if len(snap.Statuses) > 0 {
			fmt.Fprintf(w, "# HELP bebo_statuses_total HTTP responses by status\n")
			fmt.Fprintf(w, "# TYPE bebo_statuses_total counter\n")
			codes := make([]int, 0, len(snap.Statuses))
			for code := range snap.Statuses {
				codes = append(codes, code)
			}
			sort.Ints(codes)
			for _, code := range codes {
				fmt.Fprintf(w, "bebo_statuses_total{code=\"%d\"} %d\n", code, snap.Statuses[code])
			}
		}
	})
}
