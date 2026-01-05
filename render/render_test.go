package render

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderWithPartials(t *testing.T) {
	dir := t.TempDir()

	layout := filepath.Join(dir, "layout.html")
	page := filepath.Join(dir, "home.html")
	partialsDir := filepath.Join(dir, "partials")
	if err := os.MkdirAll(partialsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	partial := filepath.Join(partialsDir, "_header.html")

	if err := os.WriteFile(layout, []byte("{{ template \"content\" . }}"), 0o644); err != nil {
		t.Fatalf("write layout: %v", err)
	}
	if err := os.WriteFile(partial, []byte("{{ define \"header\" }}Header{{ end }}"), 0o644); err != nil {
		t.Fatalf("write partial: %v", err)
	}
	if err := os.WriteFile(page, []byte("{{ define \"content\" }}{{ template \"header\" . }} body{{ end }}"), 0o644); err != nil {
		t.Fatalf("write page: %v", err)
	}

	engine, err := NewEngineWithOptions(dir, Options{Layout: "layout.html", IncludeSubdirs: true})
	if err != nil {
		t.Fatalf("engine: %v", err)
	}

	rec := httptest.NewRecorder()
	if err := engine.Render(rec, 200, "home.html", map[string]string{}); err != nil {
		t.Fatalf("render: %v", err)
	}

	body := rec.Body.String()
	if body != "Header body" {
		t.Fatalf("unexpected body: %s", body)
	}
}
