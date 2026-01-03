package render

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
)

// RenderFunc allows custom rendering.
type RenderFunc func(http.ResponseWriter) error

// Engine renders HTML templates.
type Engine struct {
	dir       string
	layout    string
	templates map[string]*template.Template
	loaded    bool
}

// NewEngine builds a template engine and loads templates.
func NewEngine(dir, layout string) (*Engine, error) {
	engine := &Engine{dir: dir, layout: layout}
	return engine, engine.Load()
}

// Load parses templates from disk.
func (e *Engine) Load() error {
	if e.dir == "" {
		return nil
	}

	entries, err := filepath.Glob(filepath.Join(e.dir, "*.html"))
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return errors.New("no templates found")
	}

	layoutPath := ""
	if e.layout != "" {
		layoutPath = filepath.Join(e.dir, e.layout)
	}

	e.templates = make(map[string]*template.Template)
	for _, file := range entries {
		if layoutPath != "" && filepath.Clean(file) == filepath.Clean(layoutPath) {
			continue
		}

		name := filepath.Base(file)
		tmpl, err := parseTemplate(layoutPath, file)
		if err != nil {
			return err
		}
		e.templates[name] = tmpl
	}

	if len(e.templates) == 0 {
		return errors.New("no page templates found")
	}

	e.loaded = true
	return nil
}

// Render writes a template response.
func (e *Engine) Render(w http.ResponseWriter, status int, name string, data any) error {
	if !e.loaded || len(e.templates) == 0 {
		return http.ErrMissingFile
	}

	tmpl, ok := e.templates[name]
	if !ok && !strings.HasSuffix(name, ".html") {
		tmpl, ok = e.templates[name+".html"]
	}
	if !ok {
		return http.ErrMissingFile
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	return tmpl.Execute(w, data)
}

// JSON writes a JSON response.
func JSON(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}

// Text writes a text response.
func Text(w http.ResponseWriter, status int, message string) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write([]byte(message))
	return err
}

// Custom invokes a custom render function with status set.
func Custom(w http.ResponseWriter, status int, fn RenderFunc) error {
	w.WriteHeader(status)
	return fn(w)
}

func parseTemplate(layoutPath, pagePath string) (*template.Template, error) {
	if layoutPath == "" {
		return template.ParseFiles(pagePath)
	}
	return template.ParseFiles(layoutPath, pagePath)
}
