package cache

import (
	"context"
	"testing"
	"time"

	"github.com/devmarvs/bebo/internal/redistest"
)

func TestRedisStoreRoundTrip(t *testing.T) {
	addr, shutdown := redistest.Start(t)
	defer shutdown()

	store := NewRedisStore(RedisOptions{
		Address:      addr,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})

	ctx := context.Background()
	if _, ok, err := store.Get(ctx, "missing"); err != nil || ok {
		t.Fatalf("expected cache miss, ok=%v err=%v", ok, err)
	}

	if err := store.Set(ctx, "key", []byte("value"), time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}

	payload, ok, err := store.Get(ctx, "key")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if string(payload) != "value" {
		t.Fatalf("expected value, got %q", payload)
	}

	if err := store.Delete(ctx, "key"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok, _ := store.Get(ctx, "key"); ok {
		t.Fatalf("expected cache miss after delete")
	}
}
