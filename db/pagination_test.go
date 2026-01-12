package db

import "testing"

func TestPaginationDefaults(t *testing.T) {
	p := Pagination{}
	normalized := p.Normalize()
	if normalized.Page != 1 {
		t.Fatalf("expected page 1, got %d", normalized.Page)
	}
	if normalized.Size != DefaultPageSize {
		t.Fatalf("expected default size")
	}
	if normalized.MaxSize != MaxPageSize {
		t.Fatalf("expected max size")
	}
}

func TestPaginationLimitOffset(t *testing.T) {
	p := Pagination{Page: 2, Size: 25}
	limit, offset := p.LimitOffset()
	if limit != 25 || offset != 25 {
		t.Fatalf("expected limit 25 offset 25, got %d %d", limit, offset)
	}
}

func TestPaginationClamp(t *testing.T) {
	p := Pagination{Page: -1, Size: 500, MaxSize: 50}
	limit, offset := p.LimitOffset()
	if limit != 50 {
		t.Fatalf("expected clamped limit")
	}
	if offset != 0 {
		t.Fatalf("expected offset 0, got %d", offset)
	}
}
