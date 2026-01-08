package tasks

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerExecutesJob(t *testing.T) {
	runner := New(Options{QueueSize: 1})
	runner.Start(context.Background())

	done := make(chan struct{})
	job := Job{
		Name: "example",
		Handler: func(ctx context.Context) error {
			close(done)
			return nil
		},
	}

	if err := runner.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("job did not run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runner.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestRunnerRetriesAndDeadLetter(t *testing.T) {
	retry := RetryPolicy{
		MaxRetries: 2,
		Backoff:    func(int) time.Duration { return 0 },
		RetryIf:    DefaultRetryDecider,
	}
	attempts := int64(0)
	retries := []int{}
	var retryMu sync.Mutex
	dead := make(chan DeadLetter, 1)

	runner := New(Options{QueueSize: 1, Retry: &retry, Sleep: func(time.Duration) {}})
	runner.Start(context.Background())

	job := Job{
		Name: "flaky",
		Handler: func(ctx context.Context) error {
			atomic.AddInt64(&attempts, 1)
			return errors.New("boom")
		},
		OnRetry: func(info RetryInfo) {
			retryMu.Lock()
			retries = append(retries, info.Attempt)
			retryMu.Unlock()
		},
		OnDeadLetter: func(info DeadLetter) {
			dead <- info
		},
	}

	if err := runner.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	select {
	case info := <-dead:
		if info.Attempts != 3 {
			t.Fatalf("expected 3 attempts, got %d", info.Attempts)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("dead letter not called")
	}

	if got := atomic.LoadInt64(&attempts); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
	retryMu.Lock()
	defer retryMu.Unlock()
	if len(retries) != 2 || retries[0] != 1 || retries[1] != 2 {
		t.Fatalf("unexpected retry attempts: %v", retries)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runner.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestEnqueueGuards(t *testing.T) {
	runner := New(Options{QueueSize: 1})

	if err := runner.Enqueue(Job{}); !errors.Is(err, ErrHandlerMissing) {
		t.Fatalf("expected handler error, got %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runner.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	if err := runner.Enqueue(Job{Name: "late", Handler: func(context.Context) error { return nil }}); !errors.Is(err, ErrRunnerClosed) {
		t.Fatalf("expected closed error, got %v", err)
	}
}
