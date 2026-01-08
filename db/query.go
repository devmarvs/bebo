package db

import (
	"context"
	"database/sql"
	"time"
)

// Execer runs exec statements with context.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Queryer runs queries with context.
type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// QueryRower runs row queries with context.
type QueryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Helper wraps query helpers with a default timeout.
type Helper struct {
	Timeout time.Duration
}

// Exec runs an exec statement with timeout.
func (h Helper) Exec(ctx context.Context, db Execer, query string, args ...any) (sql.Result, error) {
	return Exec(ctx, h.Timeout, db, query, args...)
}

// Query runs a query with timeout.
func (h Helper) Query(ctx context.Context, db Queryer, query string, args ...any) (*sql.Rows, error) {
	return Query(ctx, h.Timeout, db, query, args...)
}

// QueryRow runs a row query with timeout.
func (h Helper) QueryRow(ctx context.Context, db QueryRower, query string, args ...any) (*sql.Row, context.CancelFunc) {
	return QueryRow(ctx, h.Timeout, db, query, args...)
}

// Exec runs an exec statement with timeout.
func Exec(ctx context.Context, timeout time.Duration, db Execer, query string, args ...any) (sql.Result, error) {
	ctx, cancel := WithTimeout(ctx, timeout)
	defer cancel()
	return db.ExecContext(ctx, query, args...)
}

// Query runs a query with timeout.
func Query(ctx context.Context, timeout time.Duration, db Queryer, query string, args ...any) (*sql.Rows, error) {
	ctx, cancel := WithTimeout(ctx, timeout)
	defer cancel()
	return db.QueryContext(ctx, query, args...)
}

// QueryRow runs a row query with timeout.
func QueryRow(ctx context.Context, timeout time.Duration, db QueryRower, query string, args ...any) (*sql.Row, context.CancelFunc) {
	ctx, cancel := WithTimeout(ctx, timeout)
	return db.QueryRowContext(ctx, query, args...), cancel
}

// WithTimeout returns a context with timeout when provided.
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}
