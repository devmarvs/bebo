package render

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
)

// FuncMap defines template functions.
type FuncMap = template.FuncMap

// Options configures the template engine.
type Options struct {
	Layout string
	Funcs  FuncMap
	Reload bool
}

// RenderFunc allows custom rendering.
type RenderFunc func(http.ResponseWriter) error

// Engine renders HTML templates.
type Engine struct {
	dir       string
	layout    string
	templates map[string]*template.Template
	loaded    bool
	funcs     FuncMap
	reload    bool
	mu        sync.RWMutex
}

// NewEngine builds a template engine and loads templates.
func NewEngine(dir, layout string) (*Engine, error) {
	return NewEngineWithOptions(dir, Options{Layout: layout})
}

// NewEngineWithOptions builds a template engine with options.
func NewEngineWithOptions(dir string, options Options) (*Engine, error) {
	engine := &Engine{dir: dir, layout: options.Layout, funcs: options.Funcs, reload: options.Reload}
	return engine, engine.Load()
}

// AddFuncs registers template functions.
func (e *Engine) AddFuncs(funcs FuncMap) error {
	e.mu.Lock()
	if e.funcs == nil {
		e.funcs = FuncMap{}
	}
	for key, fn := range funcs {
		e.funcs[key] = fn
	}
	e.mu.Unlock()

	if e.loaded {
		return e.Load()
	}
	return nil
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

	templates := make(map[string]*template.Template)
	for _, file := range entries {
		if layoutPath != "" && filepath.Clean(file) == filepath.Clean(layoutPath) {
			continue
		}

		name := filepath.Base(file)
		tmpl, err := parseTemplate(layoutPath, file, e.funcs)
		if err != nil {
			return err
		}
		templates[name] = tmpl
	}

	if len(templates) == 0 {
		return errors.New("no page templates found")
	}

	e.mu.Lock()
	e.templates = templates
	e.loaded = true
	e.mu.Unlock()

	return nil
}

// Render writes a template response.
func (e *Engine) Render(w http.ResponseWriter, status int, name string, data any) error {
	if e.reload {
		if err := e.Load(); err != nil {
			return err
		}
	}

	e.mu.RLock()
	if !e.loaded || len(e.templates) == 0 {
		e.mu.RUnlock()
		return http.ErrMissingFile
	}

	tmpl, ok := e.templates[name]
	if !ok && !strings.HasSuffix(name, ".html") {
		tmpl, ok = e.templates[name+".html"]
	}
	e.mu.RUnlock()

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

func parseTemplate(layoutPath, pagePath string, funcs FuncMap) (*template.Template, error) {
	base := template.New(filepath.Base(pagePath))
	if funcs != nil {
		base = base.Funcs(template.FuncMap(funcs))
	}
	if layoutPath == "" {
		return base.ParseFiles(pagePath)
	}
	return base.ParseFiles(layoutPath, pagePath)
}
