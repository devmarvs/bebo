package render

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkJSON(b *testing.B) {
	payload := map[string]any{
		"id":     42,
		"status": "ok",
		"items":  []string{"a", "b", "c"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		_ = JSON(rec, http.StatusOK, payload)
	}
}

func BenchmarkTemplateRender(b *testing.B) {
	tmp := b.TempDir()
	page := filepath.Join(tmp, "index.html")
	if err := os.WriteFile(page, []byte("<h1>Hello {{.Name}}</h1>"), 0o600); err != nil {
		b.Fatalf("write template: %v", err)
	}

	engine, err := NewEngineWithOptions(tmp, Options{})
	if err != nil {
		b.Fatalf("engine: %v", err)
	}

	data := map[string]string{"Name": "bebo"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		_ = engine.Render(rec, http.StatusOK, "index.html", data)
	}
}
