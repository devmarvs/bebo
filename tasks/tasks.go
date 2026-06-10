package tasks

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrRunnerClosed indicates the runner is shutting down.
var ErrRunnerClosed = errors.New("task runner is closed")

// ErrHandlerMissing indicates a job handler was not provided.
var ErrHandlerMissing = errors.New("task handler is required")

// Handler executes a background job.
type Handler func(context.Context) error

// BackoffFunc returns the backoff duration for a retry attempt.
type BackoffFunc func(attempt int) time.Duration

// RetryDecider decides whether a retry should be attempted.
type RetryDecider func(err error) bool

// RetryPolicy defines retry behavior.
type RetryPolicy struct {
	MaxRetries int
	Backoff    BackoffFunc
	RetryIf    RetryDecider
}

// Job describes a background task.
type Job struct {
	Name         string
	Handler      Handler
	Context      context.Context
	Retry        *RetryPolicy
	Timeout      time.Duration
	OnRetry      func(RetryInfo)
	OnDeadLetter func(DeadLetter)
}

// RetryInfo describes a retry attempt.
type RetryInfo struct {
	Name      string
	Attempt   int
	Err       error
	NextDelay time.Duration
}

// DeadLetter captures a job that failed permanently.
type DeadLetter struct {
	Name     string
	Attempts int
	Err      error
}

// Options configures a Runner.
type Options struct {
	Workers      int
	QueueSize    int
	Retry        *RetryPolicy
	Timeout      time.Duration
	OnRetry      func(RetryInfo)
	OnDeadLetter func(DeadLetter)
	Sleep        func(time.Duration)
}

type runnerOptions struct {
	workers      int
	queueSize    int
	retry        RetryPolicy
	timeout      time.Duration
	onRetry      func(RetryInfo)
	onDeadLetter func(DeadLetter)
	sleep        func(time.Duration)
}

// DefaultRetryPolicy returns a retry configuration with exponential backoff.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries: 2,
		Backoff:    ExponentialBackoff(100*time.Millisecond, 2*time.Second),
		RetryIf:    DefaultRetryDecider,
	}
}

// DefaultOptions returns a Runner configuration with sane defaults.
func DefaultOptions() Options {
	retry := DefaultRetryPolicy()
	return Options{
		Workers:   1,
		QueueSize: 100,
		Retry:     &retry,
		Sleep:     time.Sleep,
	}
}

// Runner manages background jobs.
type Runner struct {
	opts      runnerOptions
	queue     chan Job
	closing   chan struct{}
	startOnce sync.Once
	closeOnce sync.Once
	mu        sync.Mutex
	closed    bool
	enqueueWG sync.WaitGroup
	workerWG  sync.WaitGroup
}

// New creates a new Runner.
func New(options Options) *Runner {
	opts := normalizeOptions(options)
	return &Runner{
		opts:    opts,
		queue:   make(chan Job, opts.queueSize),
		closing: make(chan struct{}),
	}
}

// Start launches worker goroutines.
func (r *Runner) Start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	r.startOnce.Do(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.closed {
			return
		}

		for i := 0; i < r.opts.workers; i++ {
			r.workerWG.Add(1)
			go r.worker(ctx)
		}
	})
}

// Enqueue schedules a job for execution.
func (r *Runner) Enqueue(job Job) error {
	return r.EnqueueContext(context.Background(), job)
}

// EnqueueContext schedules a job, honoring context cancellation while waiting.
func (r *Runner) EnqueueContext(ctx context.Context, job Job) error {
	if job.Handler == nil {
		return ErrHandlerMissing
	}
	if ctx == nil {
		ctx = context.Background()
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return ErrRunnerClosed
	}
	r.enqueueWG.Add(1)
	r.mu.Unlock()
	defer r.enqueueWG.Done()

	select {
	case r.queue <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-r.closing:
		return ErrRunnerClosed
	}
}

// Shutdown stops accepting new jobs and waits for workers to finish.
func (r *Runner) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	r.closeOnce.Do(func() {
		r.mu.Lock()
		r.closed = true
		close(r.closing)
		r.mu.Unlock()

		r.enqueueWG.Wait()
		close(r.queue)
	})

	done := make(chan struct{})
	go func() {
		r.workerWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runner) worker(ctx context.Context) {
	defer r.workerWG.Done()
	for job := range r.queue {
		r.runJob(ctx, job)
	}
}

func (r *Runner) runJob(ctx context.Context, job Job) {
	retry := r.opts.retry
	if job.Retry != nil {
		retry = *job.Retry
		if retry.MaxRetries < 0 {
			retry.MaxRetries = 0
		}
		if retry.Backoff == nil {
			retry.Backoff = r.opts.retry.Backoff
		}
		if retry.RetryIf == nil {
			retry.RetryIf = r.opts.retry.RetryIf
		}
	}
	if retry.Backoff == nil {
		retry.Backoff = r.opts.retry.Backoff
	}
	if retry.RetryIf == nil {
		retry.RetryIf = r.opts.retry.RetryIf
	}

	timeout := job.Timeout
	if timeout <= 0 {
		timeout = r.opts.timeout
	}
	onRetry := job.OnRetry
	if onRetry == nil {
		onRetry = r.opts.onRetry
	}
	onDeadLetter := job.OnDeadLetter
	if onDeadLetter == nil {
		onDeadLetter = r.opts.onDeadLetter
	}

	baseCtx := ctx
	if job.Context != nil {
		baseCtx = job.Context
	}

	attempts := 0
	retries := 0
	for {
		attempts++

		runCtx := baseCtx
		cancel := func() {}
		if timeout > 0 {
			runCtx, cancel = context.WithTimeout(baseCtx, timeout)
		}
		err := job.Handler(runCtx)
		cancel()

		if err == nil {
			return
		}
		if !retry.RetryIf(err) {
			if onDeadLetter != nil {
				onDeadLetter(DeadLetter{Name: job.Name, Attempts: attempts, Err: err})
			}
			return
		}
		if retries >= retry.MaxRetries {
			if onDeadLetter != nil {
				onDeadLetter(DeadLetter{Name: job.Name, Attempts: attempts, Err: err})
			}
			return
		}

		retries++
		delay := retry.Backoff(retries)
		if onRetry != nil {
			onRetry(RetryInfo{Name: job.Name, Attempt: retries, Err: err, NextDelay: delay})
		}

		if delay > 0 {
			if err := sleepWithContext(baseCtx, delay, r.opts.sleep); err != nil {
				if onDeadLetter != nil {
					onDeadLetter(DeadLetter{Name: job.Name, Attempts: attempts, Err: err})
				}
				return
			}
		}
	}
}

func normalizeOptions(options Options) runnerOptions {
	defaults := DefaultOptions()
	if options.Workers <= 0 {
		options.Workers = defaults.Workers
	}
	if options.QueueSize <= 0 {
		options.QueueSize = defaults.QueueSize
	}
	retry := defaults.Retry
	if options.Retry != nil {
		copied := *options.Retry
		retry = &copied
	}
	policy := normalizeRetryPolicy(retry)

	sleep := options.Sleep
	if sleep == nil {
		sleep = defaults.Sleep
	}

	return runnerOptions{
		workers:      options.Workers,
		queueSize:    options.QueueSize,
		retry:        policy,
		timeout:      options.Timeout,
		onRetry:      options.OnRetry,
		onDeadLetter: options.OnDeadLetter,
		sleep:        sleep,
	}
}

func normalizeRetryPolicy(policy *RetryPolicy) RetryPolicy {
	defaults := DefaultRetryPolicy()
	if policy == nil {
		return defaults
	}
	if policy.MaxRetries < 0 {
		policy.MaxRetries = 0
	}
	if policy.Backoff == nil {
		policy.Backoff = defaults.Backoff
	}
	if policy.RetryIf == nil {
		policy.RetryIf = defaults.RetryIf
	}
	return *policy
}

// ExponentialBackoff returns a backoff function with exponential growth.
func ExponentialBackoff(base, max time.Duration) BackoffFunc {
	if base <= 0 {
		base = 100 * time.Millisecond
	}
	if max <= 0 {
		max = 2 * time.Second
	}
	return func(attempt int) time.Duration {
		if attempt <= 0 {
			return base
		}
		delay := base << (attempt - 1)
		if delay > max {
			return max
		}
		return delay
	}
}

// DefaultRetryDecider retries unless the error is from context cancellation.
func DefaultRetryDecider(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

func sleepWithContext(ctx context.Context, delay time.Duration, sleep func(time.Duration)) error {
	if delay <= 0 {
		return nil
	}
	if ctx == nil {
		sleep(delay)
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
