package testutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// UpdateGolden reports whether golden files should be updated.
func UpdateGolden() bool {
	return os.Getenv("BEBO_UPDATE_GOLDEN") != ""
}

// AssertGolden compares the data to a golden file.
func AssertGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	if UpdateGolden() {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("golden mkdir: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("golden write: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden read: %v", err)
	}
	if !bytes.Equal(want, got) {
		t.Fatalf("golden mismatch: %s", path)
	}
}
