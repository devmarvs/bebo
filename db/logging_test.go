package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

type stubQueryDB struct {
	execErr  error
	queryErr error
}

type stubQueryResult struct{}

func (stubQueryResult) LastInsertId() (int64, error) { return 0, nil }
func (stubQueryResult) RowsAffected() (int64, error) { return 0, nil }

func (s *stubQueryDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return stubQueryResult{}, s.execErr
}

func (s *stubQueryDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, s.queryErr
}

func (s *stubQueryDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return &sql.Row{}
}

func TestLoggedDBExec(t *testing.T) {
	stub := &stubQueryDB{}
	calls := 0
	logged := WithQueryHook(stub, func(ctx context.Context, query string, args []any, duration time.Duration, err error) {
		calls++
		if err != nil {
			t.Fatalf("unexpected error")
		}
	})

	_, err := logged.ExecContext(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected hook to be called")
	}
}

func TestLoggedDBQueryError(t *testing.T) {
	stub := &stubQueryDB{queryErr: errors.New("boom")}
	var gotErr error
	logged := WithQueryHook(stub, func(ctx context.Context, query string, args []any, duration time.Duration, err error) {
		gotErr = err
	})

	_, _ = logged.QueryContext(context.Background(), "SELECT 1")
	if gotErr == nil {
		t.Fatalf("expected hook error")
	}
}
