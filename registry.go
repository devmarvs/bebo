package bebo

import (
	"errors"
	"strings"
	"sync"

	"github.com/devmarvs/bebo/cache"
	"github.com/devmarvs/bebo/validate"
)

var (
	// ErrRegistryNil indicates the registry is nil.
	ErrRegistryNil = errors.New("registry is nil")
	// ErrRegistryNameRequired indicates a missing registry name.
	ErrRegistryNameRequired = errors.New("registry name is required")
	// ErrRegistryExists indicates a registry entry already exists.
	ErrRegistryExists = errors.New("registry entry already exists")
	// ErrRegistryNotFound indicates a registry entry was not found.
	ErrRegistryNotFound = errors.New("registry entry not found")
)

// Plugin registers components with a registry.
type Plugin interface {
	Name() string
	Register(*Registry) error
}

// MiddlewareFactory builds middleware using a config map.
type MiddlewareFactory func(config map[string]any) (Middleware, error)

// AuthenticatorFactory builds an authenticator using a config map.
type AuthenticatorFactory func(config map[string]any) (Authenticator, error)

// CacheFactory builds a cache store using a config map.
type CacheFactory func(config map[string]any) (cache.Store, error)

// Registry stores registered middleware, auth, cache, and validators.
type Registry struct {
	mu             sync.RWMutex
	plugins        map[string]Plugin
	middleware     map[string]MiddlewareFactory
	authenticators map[string]AuthenticatorFactory
	caches         map[string]CacheFactory
	validators     map[string]validate.ValidatorFunc
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins:        make(map[string]Plugin),
		middleware:     make(map[string]MiddlewareFactory),
		authenticators: make(map[string]AuthenticatorFactory),
		caches:         make(map[string]CacheFactory),
		validators:     make(map[string]validate.ValidatorFunc),
	}
}

// Use registers a plugin and its components.
func (r *Registry) Use(plugin Plugin) error {
	if r == nil {
		return ErrRegistryNil
	}
	if plugin == nil {
		return errors.New("plugin is nil")
	}
	name, err := normalizeRegistryName(plugin.Name())
	if err != nil {
		return err
	}

	r.mu.Lock()
	if _, exists := r.plugins[name]; exists {
		r.mu.Unlock()
		return ErrRegistryExists
	}
	r.mu.Unlock()

	if err := plugin.Register(r); err != nil {
		return err
	}

	r.mu.Lock()
	r.plugins[name] = plugin
	r.mu.Unlock()

	return nil
}

// RegisterMiddleware registers a named middleware factory.
func (r *Registry) RegisterMiddleware(name string, factory MiddlewareFactory) error {
	if r == nil {
		return ErrRegistryNil
	}
	key, err := normalizeRegistryName(name)
	if err != nil {
		return err
	}
	if factory == nil {
		return errors.New("middleware factory is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.middleware[key]; exists {
		return ErrRegistryExists
	}
	r.middleware[key] = factory
	return nil
}

// Middleware builds a named middleware.
func (r *Registry) Middleware(name string, config map[string]any) (Middleware, error) {
	if r == nil {
		return nil, ErrRegistryNil
	}
	key, err := normalizeRegistryName(name)
	if err != nil {
		return nil, err
	}

	r.mu.RLock()
	factory := r.middleware[key]
	r.mu.RUnlock()

	if factory == nil {
		return nil, ErrRegistryNotFound
	}
	return factory(config)
}

// RegisterAuthenticator registers a named authenticator factory.
func (r *Registry) RegisterAuthenticator(name string, factory AuthenticatorFactory) error {
	if r == nil {
		return ErrRegistryNil
	}
	key, err := normalizeRegistryName(name)
	if err != nil {
		return err
	}
	if factory == nil {
		return errors.New("authenticator factory is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.authenticators[key]; exists {
		return ErrRegistryExists
	}
	r.authenticators[key] = factory
	return nil
}

// Authenticator builds a named authenticator.
func (r *Registry) Authenticator(name string, config map[string]any) (Authenticator, error) {
	if r == nil {
		return nil, ErrRegistryNil
	}
	key, err := normalizeRegistryName(name)
	if err != nil {
		return nil, err
	}

	r.mu.RLock()
	factory := r.authenticators[key]
	r.mu.RUnlock()

	if factory == nil {
		return nil, ErrRegistryNotFound
	}
	return factory(config)
}

// RegisterCache registers a named cache store factory.
func (r *Registry) RegisterCache(name string, factory CacheFactory) error {
	if r == nil {
		return ErrRegistryNil
	}
	key, err := normalizeRegistryName(name)
	if err != nil {
		return err
	}
	if factory == nil {
		return errors.New("cache factory is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.caches[key]; exists {
		return ErrRegistryExists
	}
	r.caches[key] = factory
	return nil
}

// Cache builds a named cache store.
func (r *Registry) Cache(name string, config map[string]any) (cache.Store, error) {
	if r == nil {
		return nil, ErrRegistryNil
	}
	key, err := normalizeRegistryName(name)
	if err != nil {
		return nil, err
	}

	r.mu.RLock()
	factory := r.caches[key]
	r.mu.RUnlock()

	if factory == nil {
		return nil, ErrRegistryNotFound
	}
	return factory(config)
}

// RegisterValidator registers a named validator and exposes it via validate.Register.
func (r *Registry) RegisterValidator(name string, fn validate.ValidatorFunc) error {
	if r == nil {
		return ErrRegistryNil
	}
	key, err := normalizeRegistryName(name)
	if err != nil {
		return err
	}
	if fn == nil {
		return errors.New("validator is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.validators[key]; exists {
		return ErrRegistryExists
	}
	validate.Register(key, fn)
	r.validators[key] = fn
	return nil
}

// Validator returns a named validator if registered.
func (r *Registry) Validator(name string) (validate.ValidatorFunc, error) {
	if r == nil {
		return nil, ErrRegistryNil
	}
	key, err := normalizeRegistryName(name)
	if err != nil {
		return nil, err
	}

	r.mu.RLock()
	fn := r.validators[key]
	r.mu.RUnlock()

	if fn == nil {
		return nil, ErrRegistryNotFound
	}
	return fn, nil
}

func normalizeRegistryName(name string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return "", ErrRegistryNameRequired
	}
	return key, nil
}
