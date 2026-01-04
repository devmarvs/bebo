package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type cacheEntry struct {
	value   string
	modTime time.Time
	size    int64
}

// Resolver adds fingerprint query params for static assets.
type Resolver struct {
	root  string
	mu    sync.RWMutex
	cache map[string]cacheEntry
	dev   bool
}

// Option configures the resolver.
type Option func(*Resolver)

// WithDevMode disables caching to pick up asset changes.
func WithDevMode(enabled bool) Option {
	return func(r *Resolver) {
		r.dev = enabled
	}
}

// NewResolver creates a resolver rooted at a directory.
func NewResolver(root string, options ...Option) *Resolver {
	resolver := &Resolver{root: root, cache: make(map[string]cacheEntry)}
	for _, opt := range options {
		opt(resolver)
	}
	return resolver
}

// Resolve returns a path with a cache-busting fingerprint.
func (r *Resolver) Resolve(assetPath string) string {
	clean := path.Clean("/" + assetPath)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" {
		return assetPath
	}

	fullPath := filepath.Join(r.root, filepath.FromSlash(clean))
	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		return assetPath
	}

	if !r.dev {
		r.mu.RLock()
		entry, ok := r.cache[clean]
		r.mu.RUnlock()
		if ok && entry.modTime.Equal(info.ModTime()) && entry.size == info.Size() {
			return entry.value
		}
	}

	hash, err := hashFile(fullPath)
	if err != nil {
		return assetPath
	}

	fingerprinted := assetPath + "?v=" + hash[:8]
	if !r.dev {
		r.mu.Lock()
		r.cache[clean] = cacheEntry{value: fingerprinted, modTime: info.ModTime(), size: info.Size()}
		r.mu.Unlock()
	}

	return fingerprinted
}

// Func returns a template helper function.
func (r *Resolver) Func() func(string) string {
	return func(assetPath string) string {
		return r.Resolve(assetPath)
	}
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
