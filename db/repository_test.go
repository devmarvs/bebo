package db

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

type stubRepoDB struct {
	ctx context.Context
}

type stubRepoResult struct{}

func (stubRepoResult) LastInsertId() (int64, error) { return 0, nil }
func (stubRepoResult) RowsAffected() (int64, error) { return 0, nil }

func (s *stubRepoDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	s.ctx = ctx
	return stubRepoResult{}, nil
}

func (s *stubRepoDB) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (s *stubRepoDB) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return &sql.Row{}
}

func TestRepositoryExecTimeout(t *testing.T) {
	stub := &stubRepoDB{}
	repo := NewRepository(stub, 50*time.Millisecond)

	_, err := repo.Exec(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}
	if stub.ctx == nil {
		t.Fatalf("expected context")
	}
	if _, ok := stub.ctx.Deadline(); !ok {
		t.Fatalf("expected deadline")
	}
}

func TestRepositoryPaginate(t *testing.T) {
	repo := Repository{}
	limit, offset := repo.Paginate(Pagination{Page: 3, Size: 10})
	if limit != 10 || offset != 20 {
		t.Fatalf("unexpected pagination")
	}
}
