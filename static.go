package bebo

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/devmarvs/bebo/apperr"
)

type staticConfig struct {
	cacheControl string
	etag         bool
	indexFile    string
	paramName    string
}

// StaticOption configures static file handling.
type StaticOption func(*staticConfig)

// StaticCacheControl sets the Cache-Control header value.
func StaticCacheControl(value string) StaticOption {
	return func(cfg *staticConfig) {
		cfg.cacheControl = value
	}
}

// StaticETag enables ETag handling.
func StaticETag(enabled bool) StaticOption {
	return func(cfg *staticConfig) {
		cfg.etag = enabled
	}
}

// StaticIndex sets the index file name to serve for directories.
func StaticIndex(name string) StaticOption {
	return func(cfg *staticConfig) {
		cfg.indexFile = name
	}
}

// File registers a static route for a single file on disk.
func (a *App) File(route, filePath string, options ...StaticOption) {
	cfg := staticConfig{
		cacheControl: "public, max-age=86400",
		etag:         true,
	}
	for _, opt := range options {
		opt(&cfg)
	}

	if filePath == "" {
		return
	}

	dir := filepath.Dir(filePath)
	rel := filepath.Base(filePath)
	handler := func(ctx *Context) error {
		return serveStatic(ctx, dir, rel, cfg)
	}

	a.GET(route, handler)
	a.HEAD(route, handler)
}

// Static registers a static file route.
func (a *App) Static(prefix, dir string, options ...StaticOption) {
	cfg := staticConfig{
		cacheControl: "public, max-age=86400",
		etag:         true,
		indexFile:    "index.html",
		paramName:    "path",
	}
	for _, opt := range options {
		opt(&cfg)
	}

	pattern := buildStaticPattern(prefix, cfg.paramName)
	handler := func(ctx *Context) error {
		rel := ctx.Param(cfg.paramName)
		return serveStatic(ctx, dir, rel, cfg)
	}

	a.GET(pattern, handler)
	a.HEAD(pattern, handler)
}

// StaticFS registers a static file route from an fs.FS.
func (a *App) StaticFS(prefix string, fsys fs.FS, options ...StaticOption) {
	cfg := staticConfig{
		cacheControl: "public, max-age=86400",
		etag:         true,
		indexFile:    "index.html",
		paramName:    "path",
	}
	for _, opt := range options {
		opt(&cfg)
	}

	pattern := buildStaticPattern(prefix, cfg.paramName)
	handler := func(ctx *Context) error {
		rel := ctx.Param(cfg.paramName)
		return serveStaticFS(ctx, fsys, rel, cfg)
	}

	a.GET(pattern, handler)
	a.HEAD(pattern, handler)
}

func buildStaticPattern(prefix, param string) string {
	prefix = cleanPrefix(prefix)
	if prefix == "" || prefix == "/" {
		return "/*" + param
	}
	return strings.TrimRight(prefix, "/") + "/*" + param
}

func serveStatic(ctx *Context, dir, rel string, cfg staticConfig) error {
	fs := http.Dir(dir)
	if rel == "" {
		rel = cfg.indexFile
	}
	if rel == "" {
		return apperr.NotFound("not found", nil)
	}

	clean := path.Clean("/" + rel)
	clean = strings.TrimPrefix(clean, "/")

	file, err := fs.Open(clean)
	if err != nil {
		return apperr.NotFound("not found", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return apperr.Internal("file stat failed", err)
	}

	if info.IsDir() {
		if cfg.indexFile == "" {
			return apperr.NotFound("not found", nil)
		}
		return serveStatic(ctx, dir, path.Join(clean, cfg.indexFile), cfg)
	}

	w := ctx.ResponseWriter
	r := ctx.Request
	if cfg.cacheControl != "" {
		w.Header().Set("Cache-Control", cfg.cacheControl)
	}

	if cfg.etag {
		etag := buildETag(info.ModTime(), info.Size())
		w.Header().Set("ETag", etag)
		if matchETag(r.Header.Get("If-None-Match"), etag) {
			w.WriteHeader(http.StatusNotModified)
			return nil
		}
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	return nil
}

func serveStaticFS(ctx *Context, fsys fs.FS, rel string, cfg staticConfig) error {
	if fsys == nil {
		return apperr.Internal("static fs missing", nil)
	}
	if rel == "" {
		rel = cfg.indexFile
	}
	if rel == "" {
		return apperr.NotFound("not found", nil)
	}

	clean := path.Clean("/" + rel)
	clean = strings.TrimPrefix(clean, "/")

	file, err := fsys.Open(clean)
	if err != nil {
		return apperr.NotFound("not found", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return apperr.Internal("file stat failed", err)
	}

	if info.IsDir() {
		if cfg.indexFile == "" {
			return apperr.NotFound("not found", nil)
		}
		return serveStaticFS(ctx, fsys, path.Join(clean, cfg.indexFile), cfg)
	}

	w := ctx.ResponseWriter
	r := ctx.Request
	if cfg.cacheControl != "" {
		w.Header().Set("Cache-Control", cfg.cacheControl)
	}

	if cfg.etag {
		etag := buildETag(info.ModTime(), info.Size())
		w.Header().Set("ETag", etag)
		if matchETag(r.Header.Get("If-None-Match"), etag) {
			w.WriteHeader(http.StatusNotModified)
			return nil
		}
	}

	reader, ok := file.(io.ReadSeeker)
	if !ok {
		data, err := io.ReadAll(file)
		if err != nil {
			return apperr.Internal("file read failed", err)
		}
		reader = bytes.NewReader(data)
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), reader)
	return nil
}

func buildETag(modTime time.Time, size int64) string {
	return fmt.Sprintf("\"%x-%x\"", modTime.UnixNano(), size)
}

func matchETag(header, etag string) bool {
	for _, part := range strings.Split(header, ",") {
		if strings.TrimSpace(part) == etag {
			return true
		}
	}
	return false
}
