package db

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

type stubExecer struct {
	ctx context.Context
}

type stubResult struct{}

func (stubResult) LastInsertId() (int64, error) { return 0, nil }
func (stubResult) RowsAffected() (int64, error) { return 0, nil }

func (s *stubExecer) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	s.ctx = ctx
	return stubResult{}, nil
}

func TestWithTimeout(t *testing.T) {
	ctx := context.Background()
	ctx2, cancel := WithTimeout(ctx, 0)
	defer cancel()
	if _, ok := ctx2.Deadline(); ok {
		t.Fatalf("expected no deadline")
	}
}

func TestExecWithTimeout(t *testing.T) {
	stub := &stubExecer{}
	_, err := Exec(context.Background(), 50*time.Millisecond, stub, "SELECT 1")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if stub.ctx == nil {
		t.Fatalf("expected context")
	}
	if _, ok := stub.ctx.Deadline(); !ok {
		t.Fatalf("expected deadline")
	}
}
