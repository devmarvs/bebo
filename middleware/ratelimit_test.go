package middleware

import (
	"testing"
	"time"
)

func TestLimiterTTLEvictsIdleBuckets(t *testing.T) {
	ttl := 30 * time.Millisecond
	limiter := NewLimiter(1, 1, LimiterTTL(ttl))

	if !limiter.Allow("a") {
		t.Fatalf("expected allow for key a")
	}
	if !limiter.Allow("b") {
		t.Fatalf("expected allow for key b")
	}
	if got := len(limiter.buckets); got != 2 {
		t.Fatalf("expected 2 buckets, got %d", got)
	}

	time.Sleep(ttl + 20*time.Millisecond)

	_ = limiter.Allow("a")

	if _, ok := limiter.buckets["b"]; ok {
		t.Fatalf("expected bucket b to be evicted")
	}
	if got := len(limiter.buckets); got != 1 {
		t.Fatalf("expected 1 bucket, got %d", got)
	}
}
