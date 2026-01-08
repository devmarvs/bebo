package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/devmarvs/bebo/internal/redistest"
)

func TestRedisLimiter(t *testing.T) {
	addr, shutdown := redistest.Start(t)
	defer shutdown()

	now := time.Unix(0, 0)
	current := now
	limiter := NewRedisLimiter(1, 1, RedisLimiterOptions{
		Address:      addr,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		Now: func() time.Time {
			return current
		},
	})

	allowed, err := limiter.Allow(context.Background(), "client")
	if err != nil {
		t.Fatalf("allow: %v", err)
	}
	if !allowed {
		t.Fatalf("expected first request allowed")
	}

	allowed, err = limiter.Allow(context.Background(), "client")
	if err != nil {
		t.Fatalf("allow: %v", err)
	}
	if allowed {
		t.Fatalf("expected rate limit")
	}

	current = current.Add(time.Second)
	allowed, err = limiter.Allow(context.Background(), "client")
	if err != nil {
		t.Fatalf("allow: %v", err)
	}
	if !allowed {
		t.Fatalf("expected token refill")
	}
}
