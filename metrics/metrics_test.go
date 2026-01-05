package metrics

import (
	"testing"
	"time"
)

func TestMetricsBuckets(t *testing.T) {
	reg := NewWithBuckets([]time.Duration{10 * time.Millisecond, 50 * time.Millisecond})
	start := time.Now().Add(-20 * time.Millisecond)
	reg.End(start, 200, nil)

	snap := reg.Snapshot()
	if len(snap.Latency.Buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(snap.Latency.Buckets))
	}
	if snap.Latency.Buckets[0].Count == 0 && snap.Latency.Buckets[1].Count == 0 {
		t.Fatalf("expected bucket counts")
	}
}
