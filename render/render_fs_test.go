package render

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestEngineFromFS(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/layout.html": {Data: []byte("<html><body>{{ template \"content\" . }}</body></html>")},
		"templates/home.html":   {Data: []byte("{{ define \"content\" }}Hello {{ . }}{{ end }}")},
	}

	engine, err := NewEngineFromFS(fsys, "templates", Options{Layout: "layout.html"})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	rec := httptest.NewRecorder()
	if err := engine.Render(rec, http.StatusOK, "home.html", "world"); err != nil {
		t.Fatalf("render: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Hello world") {
		t.Fatalf("expected rendered content, got %q", body)
	}
}
