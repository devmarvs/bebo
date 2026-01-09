package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubStore struct {
	value     []byte
	ok        bool
	getErr    error
	setErr    error
	deleteErr error
}

func (s stubStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	return s.value, s.ok, s.getErr
}

func (s stubStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return s.setErr
}

func (s stubStore) Delete(ctx context.Context, key string) error {
	return s.deleteErr
}

func TestHooksHitMiss(t *testing.T) {
	var hit, miss int
	store := stubStore{value: []byte("ok"), ok: true}
	wrapped := WithHooks(store, Hooks{
		OnHit: func(ctx context.Context, key string) { hit++ },
		OnMiss: func(ctx context.Context, key string) {
			miss++
		},
	})

	_, ok, err := wrapped.Get(context.Background(), "key")
	if err != nil || !ok {
		t.Fatalf("expected hit, got err=%v ok=%v", err, ok)
	}
	if hit != 1 {
		t.Fatalf("expected hit hook, got %d", hit)
	}
	if miss != 0 {
		t.Fatalf("expected no miss hook, got %d", miss)
	}

	store = stubStore{ok: false}
	wrapped = WithHooks(store, Hooks{
		OnMiss: func(ctx context.Context, key string) { miss++ },
	})

	_, ok, err = wrapped.Get(context.Background(), "missing")
	if err != nil || ok {
		t.Fatalf("expected miss, got err=%v ok=%v", err, ok)
	}
	if miss != 1 {
		t.Fatalf("expected miss hook, got %d", miss)
	}
}

func TestHooksSetDeleteAndError(t *testing.T) {
	var setCount, deleteCount, errCount int
	store := stubStore{setErr: errors.New("set failed"), deleteErr: errors.New("delete failed")}
	wrapped := WithHooks(store, Hooks{
		OnSet: func(ctx context.Context, key string, ttl time.Duration) { setCount++ },
		OnDelete: func(ctx context.Context, key string) {
			deleteCount++
		},
		OnError: func(ctx context.Context, op string, key string, err error) { errCount++ },
	})

	if err := wrapped.Set(context.Background(), "key", []byte("v"), time.Second); err == nil {
		t.Fatalf("expected set error")
	}
	if setCount != 0 {
		t.Fatalf("expected no set hook on error, got %d", setCount)
	}

	if err := wrapped.Delete(context.Background(), "key"); err == nil {
		t.Fatalf("expected delete error")
	}
	if deleteCount != 0 {
		t.Fatalf("expected no delete hook on error, got %d", deleteCount)
	}
	if errCount != 2 {
		t.Fatalf("expected error hooks, got %d", errCount)
	}
}
