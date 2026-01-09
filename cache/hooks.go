package cache

import (
	"context"
	"time"
)

// Hooks defines cache lifecycle callbacks.
type Hooks struct {
	OnHit    func(ctx context.Context, key string)
	OnMiss   func(ctx context.Context, key string)
	OnSet    func(ctx context.Context, key string, ttl time.Duration)
	OnDelete func(ctx context.Context, key string)
	OnError  func(ctx context.Context, op string, key string, err error)
}

// WithHooks wraps a store with hook callbacks.
func WithHooks(store Store, hooks Hooks) Store {
	if store == nil {
		return nil
	}
	if hooks.isZero() {
		return store
	}
	return &hookStore{base: store, hooks: hooks}
}

type hookStore struct {
	base  Store
	hooks Hooks
}

func (h *hookStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, ok, err := h.base.Get(ctx, key)
	if err != nil {
		h.fireError(ctx, "get", key, err)
		return value, ok, err
	}
	if ok {
		h.fireHit(ctx, key)
	} else {
		h.fireMiss(ctx, key)
	}
	return value, ok, err
}

func (h *hookStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := h.base.Set(ctx, key, value, ttl); err != nil {
		h.fireError(ctx, "set", key, err)
		return err
	}
	h.fireSet(ctx, key, ttl)
	return nil
}

func (h *hookStore) Delete(ctx context.Context, key string) error {
	if err := h.base.Delete(ctx, key); err != nil {
		h.fireError(ctx, "delete", key, err)
		return err
	}
	h.fireDelete(ctx, key)
	return nil
}

func (h Hooks) isZero() bool {
	return h.OnHit == nil && h.OnMiss == nil && h.OnSet == nil && h.OnDelete == nil && h.OnError == nil
}

func (h *hookStore) fireHit(ctx context.Context, key string) {
	if h.hooks.OnHit != nil {
		h.hooks.OnHit(ctx, key)
	}
}

func (h *hookStore) fireMiss(ctx context.Context, key string) {
	if h.hooks.OnMiss != nil {
		h.hooks.OnMiss(ctx, key)
	}
}

func (h *hookStore) fireSet(ctx context.Context, key string, ttl time.Duration) {
	if h.hooks.OnSet != nil {
		h.hooks.OnSet(ctx, key, ttl)
	}
}

func (h *hookStore) fireDelete(ctx context.Context, key string) {
	if h.hooks.OnDelete != nil {
		h.hooks.OnDelete(ctx, key)
	}
}

func (h *hookStore) fireError(ctx context.Context, op string, key string, err error) {
	if h.hooks.OnError != nil {
		h.hooks.OnError(ctx, op, key, err)
	}
}
