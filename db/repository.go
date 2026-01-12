package db

import (
	"context"
	"database/sql"
	"time"
)

// QueryDB groups query interfaces for repositories.
type QueryDB interface {
	Execer
	Queryer
	QueryRower
}

// Repository wraps common query helpers and timeouts.
type Repository struct {
	DB     QueryDB
	Helper Helper
}

// NewRepository creates a repository with a timeout.
func NewRepository(db QueryDB, timeout time.Duration) Repository {
	return Repository{DB: db, Helper: Helper{Timeout: timeout}}
}

// WithTimeout returns a copy with an updated timeout.
func (r Repository) WithTimeout(timeout time.Duration) Repository {
	r.Helper.Timeout = timeout
	return r
}

// Exec executes a statement with the repository timeout.
func (r Repository) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return r.Helper.Exec(ctx, r.DB, query, args...)
}

// Query executes a query with the repository timeout.
func (r Repository) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return r.Helper.Query(ctx, r.DB, query, args...)
}

// QueryRow executes a query row with the repository timeout.
func (r Repository) QueryRow(ctx context.Context, query string, args ...any) (*sql.Row, context.CancelFunc) {
	return r.Helper.QueryRow(ctx, r.DB, query, args...)
}

// Paginate returns limit/offset for a pagination config.
func (r Repository) Paginate(p Pagination) (int, int) {
	return p.LimitOffset()
}
