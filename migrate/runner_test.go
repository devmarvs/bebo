package migrate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestListMigrations(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "0001_init.up.sql"), []byte("-- up"), 0o644); err != nil {
		t.Fatalf("write up: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0001_init.down.sql"), []byte("-- down"), 0o644); err != nil {
		t.Fatalf("write down: %v", err)
	}

	migrations, err := List(dir)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(migrations) != 1 {
		t.Fatalf("expected 1 migration, got %d", len(migrations))
	}
}

func TestPlanWithoutDB(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "0002_add.up.sql"), []byte("-- up"), 0o644); err != nil {
		t.Fatalf("write up: %v", err)
	}

	runner := New(nil, dir)
	plan, err := runner.Plan(context.Background())
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan) != 1 {
		t.Fatalf("expected 1 plan entry, got %d", len(plan))
	}
	if plan[0].Applied {
		t.Fatalf("expected plan entry to be pending")
	}
}
