package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migration describes a migration pair.
type Migration struct {
	Version  int
	Name     string
	UpPath   string
	DownPath string
}

// PlanEntry describes a migration and whether it has been applied.
type PlanEntry struct {
	Migration
	Applied bool
}

// Locker handles migration locking.
type Locker interface {
	Lock(context.Context, *sql.DB) error
	Unlock(context.Context, *sql.DB) error
}

// ErrLockTimeout indicates a lock timeout.
var ErrLockTimeout = errors.New("migration lock timeout")

// AdvisoryLocker uses PostgreSQL advisory locks.
type AdvisoryLocker struct {
	ID           int64
	Timeout      time.Duration
	PollInterval time.Duration
}

// Lock acquires a PostgreSQL advisory lock.
func (a AdvisoryLocker) Lock(ctx context.Context, db *sql.DB) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if a.Timeout <= 0 {
		_, err := db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", a.ID)
		return err
	}

	poll := a.PollInterval
	if poll <= 0 {
		poll = 200 * time.Millisecond
	}
	deadline := time.Now().Add(a.Timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}

	for {
		locked, err := tryAdvisoryLock(ctx, db, a.ID)
		if err != nil {
			return err
		}
		if locked {
			return nil
		}
		if time.Now().After(deadline) {
			return ErrLockTimeout
		}
		if err := sleepWithContext(ctx, poll); err != nil {
			return err
		}
	}
}

// Unlock releases a PostgreSQL advisory lock.
func (a AdvisoryLocker) Unlock(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", a.ID)
	return err
}

// Runner executes migrations from a directory.
type Runner struct {
	DB     *sql.DB
	Dir    string
	Table  string
	Locker Locker
}

// New creates a new Runner.
func New(db *sql.DB, dir string) *Runner {
	return &Runner{DB: db, Dir: dir, Table: "schema_migrations"}
}

// Plan returns a migration plan, optionally marking applied migrations.
func (r *Runner) Plan(ctx context.Context) ([]PlanEntry, error) {
	migrations, err := r.loadMigrations()
	if err != nil {
		return nil, err
	}

	appliedSet := make(map[int]struct{})
	if r.DB != nil {
		if err := r.ensureTable(ctx); err != nil {
			return nil, err
		}
		applied, err := r.appliedVersions(ctx)
		if err != nil {
			return nil, err
		}
		for _, v := range applied {
			appliedSet[v] = struct{}{}
		}
	}

	plan := make([]PlanEntry, 0, len(migrations))
	for _, migration := range migrations {
		_, applied := appliedSet[migration.Version]
		plan = append(plan, PlanEntry{Migration: migration, Applied: applied})
	}

	return plan, nil
}

// Up applies all pending migrations.
func (r *Runner) Up(ctx context.Context) (int, error) {
	if r.DB == nil {
		return 0, errors.New("db is required")
	}
	if err := r.ensureTable(ctx); err != nil {
		return 0, err
	}
	if r.Locker != nil {
		if err := r.Locker.Lock(ctx, r.DB); err != nil {
			return 0, err
		}
		defer func() {
			_ = r.Locker.Unlock(ctx, r.DB)
		}()
	}

	migrations, err := r.loadMigrations()
	if err != nil {
		return 0, err
	}
	applied, err := r.appliedVersions(ctx)
	if err != nil {
		return 0, err
	}

	appliedSet := make(map[int]struct{}, len(applied))
	for _, v := range applied {
		appliedSet[v] = struct{}{}
	}

	count := 0
	for _, m := range migrations {
		if _, ok := appliedSet[m.Version]; ok {
			continue
		}
		if m.UpPath == "" {
			return count, fmt.Errorf("missing up migration for version %d", m.Version)
		}
		if err := r.apply(ctx, m, true); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

// Down rolls back the latest migrations.
func (r *Runner) Down(ctx context.Context, steps int) (int, error) {
	if r.DB == nil {
		return 0, errors.New("db is required")
	}
	if steps <= 0 {
		return 0, nil
	}
	if err := r.ensureTable(ctx); err != nil {
		return 0, err
	}
	if r.Locker != nil {
		if err := r.Locker.Lock(ctx, r.DB); err != nil {
			return 0, err
		}
		defer func() {
			_ = r.Locker.Unlock(ctx, r.DB)
		}()
	}

	migrations, err := r.loadMigrations()
	if err != nil {
		return 0, err
	}
	applied, err := r.appliedVersions(ctx)
	if err != nil {
		return 0, err
	}

	byVersion := make(map[int]Migration, len(migrations))
	for _, m := range migrations {
		byVersion[m.Version] = m
	}

	count := 0
	for i := 0; i < steps && i < len(applied); i++ {
		version := applied[i]
		m, ok := byVersion[version]
		if !ok {
			return count, fmt.Errorf("missing migration for version %d", version)
		}
		if m.DownPath == "" {
			return count, fmt.Errorf("missing down migration for version %d", version)
		}
		if err := r.apply(ctx, m, false); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

// List returns migrations found in a directory.
func List(dir string) ([]Migration, error) {
	runner := New(nil, dir)
	return runner.loadMigrations()
}

func (r *Runner) ensureTable(ctx context.Context) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (version bigint primary key, name text not null, applied_at timestamptz not null)`, r.Table)
	_, err := r.DB.ExecContext(ctx, query)
	return err
}

func (r *Runner) appliedVersions(ctx context.Context) ([]int, error) {
	query := fmt.Sprintf(`SELECT version FROM %s ORDER BY version DESC`, r.Table)
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}

	return versions, rows.Err()
}

func (r *Runner) loadMigrations() ([]Migration, error) {
	entries, err := os.ReadDir(r.Dir)
	if err != nil {
		return nil, err
	}

	byVersion := make(map[int]Migration)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		parts := strings.Split(name, ".")
		if len(parts) != 3 || parts[2] != "sql" {
			continue
		}
		versionName := parts[0]
		direction := parts[1]

		versionParts := strings.SplitN(versionName, "_", 2)
		version, err := strconv.Atoi(versionParts[0])
		if err != nil {
			continue
		}
		migration := byVersion[version]
		migration.Version = version
		if len(versionParts) > 1 {
			migration.Name = versionParts[1]
		}

		fullPath := filepath.Join(r.Dir, name)
		switch direction {
		case "up":
			migration.UpPath = fullPath
		case "down":
			migration.DownPath = fullPath
		default:
			continue
		}

		byVersion[version] = migration
	}

	migrations := make([]Migration, 0, len(byVersion))
	for _, migration := range byVersion {
		migrations = append(migrations, migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (r *Runner) apply(ctx context.Context, migration Migration, up bool) error {
	path := migration.UpPath
	if !up {
		path = migration.DownPath
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, string(contents)); err != nil {
		_ = tx.Rollback()
		return err
	}

	if up {
		query := fmt.Sprintf(`INSERT INTO %s (version, name, applied_at) VALUES ($1, $2, $3)`, r.Table)
		if _, err := tx.ExecContext(ctx, query, migration.Version, migration.Name, time.Now().UTC()); err != nil {
			_ = tx.Rollback()
			return err
		}
	} else {
		query := fmt.Sprintf(`DELETE FROM %s WHERE version = $1`, r.Table)
		if _, err := tx.ExecContext(ctx, query, migration.Version); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func tryAdvisoryLock(ctx context.Context, db *sql.DB, id int64) (bool, error) {
	row := db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", id)
	var locked bool
	if err := row.Scan(&locked); err != nil {
		return false, err
	}
	return locked, nil
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	if ctx == nil {
		time.Sleep(delay)
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
