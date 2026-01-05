package render

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
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
	// IncludeSubdirs enables loading templates from nested directories.
	IncludeSubdirs bool
	// Partials lists glob patterns (relative to the templates dir) treated as partials.
	Partials []string
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
	partials  []string
	recursive bool
	mu        sync.RWMutex
}

// NewEngine builds a template engine and loads templates.
func NewEngine(dir, layout string) (*Engine, error) {
	return NewEngineWithOptions(dir, Options{Layout: layout})
}

// NewEngineWithOptions builds a template engine with options.
func NewEngineWithOptions(dir string, options Options) (*Engine, error) {
	engine := &Engine{
		dir:       dir,
		layout:    options.Layout,
		funcs:     options.Funcs,
		reload:    options.Reload,
		partials:  options.Partials,
		recursive: options.IncludeSubdirs,
	}
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

	files, err := findTemplateFiles(e.dir, e.recursive)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("no templates found")
	}

	layoutPath := ""
	if e.layout != "" {
		layoutPath = filepath.Join(e.dir, e.layout)
	}

	pages, partials, err := classifyTemplates(e.dir, files, layoutPath, e.partials)
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return errors.New("no page templates found")
	}

	templates := make(map[string]*template.Template)
	for _, page := range pages {
		pageName, err := templateName(e.dir, page)
		if err != nil {
			return err
		}
		tmpl, err := parseTemplateSet(e.dir, layoutPath, page, partials, e.funcs)
		if err != nil {
			return err
		}
		templates[pageName] = tmpl
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

func findTemplateFiles(dir string, recursive bool) ([]string, error) {
	if !recursive {
		entries, err := filepath.Glob(filepath.Join(dir, "*.html"))
		if err != nil {
			return nil, err
		}
		sort.Strings(entries)
		return entries, nil
	}

	var entries []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".html") {
			entries = append(entries, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)
	return entries, nil
}

func classifyTemplates(dir string, files []string, layoutPath string, patterns []string) ([]string, []string, error) {
	var pages []string
	var partials []string
	layoutPath = filepath.Clean(layoutPath)

	for _, file := range files {
		clean := filepath.Clean(file)
		if layoutPath != "" && clean == layoutPath {
			continue
		}
		name, err := templateName(dir, clean)
		if err != nil {
			return nil, nil, err
		}
		if isPartial(name, patterns) {
			partials = append(partials, clean)
			continue
		}
		pages = append(pages, clean)
	}

	sort.Strings(pages)
	sort.Strings(partials)
	return pages, partials, nil
}

func templateName(dir, file string) (string, error) {
	rel, err := filepath.Rel(dir, file)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func isPartial(name string, patterns []string) bool {
	if len(patterns) > 0 {
		for _, pattern := range patterns {
			if matchPattern(pattern, name) {
				return true
			}
		}
		return false
	}

	base := path.Base(name)
	if strings.HasPrefix(base, "_") {
		return true
	}
	if strings.HasPrefix(name, "partials/") || strings.Contains(name, "/partials/") {
		return true
	}
	return false
}

func matchPattern(pattern, name string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	pattern = path.Clean(strings.TrimPrefix(pattern, "/"))
	if strings.Contains(pattern, "**") {
		re, err := globToRegex(pattern)
		if err != nil {
			return false
		}
		return re.MatchString(name)
	}

	matched, err := path.Match(pattern, name)
	if err != nil {
		return false
	}
	return matched
}

func globToRegex(pattern string) (*regexp.Regexp, error) {
	var builder strings.Builder
	builder.WriteString("^")

	for i := 0; i < len(pattern); {
		switch {
		case strings.HasPrefix(pattern[i:], "**"):
			builder.WriteString(".*")
			i += 2
		case pattern[i] == '*':
			builder.WriteString("[^/]*")
			i++
		case pattern[i] == '?':
			builder.WriteString("[^/]")
			i++
		default:
			builder.WriteString(regexp.QuoteMeta(string(pattern[i])))
			i++
		}
	}

	builder.WriteString("$")
	return regexp.Compile(builder.String())
}

func parseTemplateSet(dir, layoutPath, pagePath string, partials []string, funcs FuncMap) (*template.Template, error) {
	pageName, err := templateName(dir, pagePath)
	if err != nil {
		return nil, err
	}

	layoutName := ""
	if layoutPath != "" {
		layoutName, err = templateName(dir, layoutPath)
		if err != nil {
			return nil, err
		}
	}

	baseName := pageName
	if layoutName != "" {
		baseName = layoutName
	}

	base := template.New(baseName)
	if funcs != nil {
		base = base.Funcs(template.FuncMap(funcs))
	}

	if layoutPath != "" {
		if err := parseTemplateFile(base, layoutPath, baseName); err != nil {
			return nil, err
		}
	}

	for _, partial := range partials {
		name, err := templateName(dir, partial)
		if err != nil {
			return nil, err
		}
		if name == baseName {
			continue
		}
		if err := parseTemplateFile(base, partial, name); err != nil {
			return nil, err
		}
	}

	if err := parseTemplateFile(base, pagePath, pageName); err != nil {
		return nil, err
	}

	return base, nil
}

func parseTemplateFile(base *template.Template, filePath, name string) error {
	contents, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if name == base.Name() {
		_, err = base.Parse(string(contents))
		return err
	}

	if base.Lookup(name) != nil {
		return fmt.Errorf("template %s already defined", name)
	}

	_, err = base.New(name).Parse(string(contents))
	return err
}
