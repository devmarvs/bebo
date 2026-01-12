package db

import (
	"context"
	"database/sql"
	"time"
)

// QueryHook receives query timing information.
type QueryHook func(ctx context.Context, query string, args []any, duration time.Duration, err error)

// LoggedDB wraps a database with query hooks.
type LoggedDB struct {
	DB   QueryDB
	Hook QueryHook
}

// WithQueryHook wraps a database with a query hook.
func WithQueryHook(db QueryDB, hook QueryHook) LoggedDB {
	return LoggedDB{DB: db, Hook: hook}
}

// ExecContext executes a statement and emits hook timing.
func (l LoggedDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	res, err := l.DB.ExecContext(ctx, query, args...)
	if l.Hook != nil {
		l.Hook(ctx, query, args, time.Since(start), err)
	}
	return res, err
}

// QueryContext executes a query and emits hook timing.
func (l LoggedDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := l.DB.QueryContext(ctx, query, args...)
	if l.Hook != nil {
		l.Hook(ctx, query, args, time.Since(start), err)
	}
	return rows, err
}

// QueryRowContext executes a row query and emits hook timing.
func (l LoggedDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()
	row := l.DB.QueryRowContext(ctx, query, args...)
	if l.Hook != nil {
		l.Hook(ctx, query, args, time.Since(start), nil)
	}
	return row
}
