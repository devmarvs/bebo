package bebo

import (
	"reflect"
	"testing"

	"github.com/devmarvs/bebo/validate"
)

func TestRegistryMiddleware(t *testing.T) {
	reg := NewRegistry()
	called := false

	err := reg.RegisterMiddleware("request_id", func(config map[string]any) (Middleware, error) {
		called = true
		return func(next Handler) Handler { return next }, nil
	})
	if err != nil {
		t.Fatalf("register middleware: %v", err)
	}

	if _, err := reg.Middleware("request_id", nil); err != nil {
		t.Fatalf("build middleware: %v", err)
	}
	if !called {
		t.Fatalf("expected factory to be called")
	}

	if err := reg.RegisterMiddleware("request_id", func(config map[string]any) (Middleware, error) { return nil, nil }); err == nil {
		t.Fatalf("expected duplicate middleware registration error")
	}
}

func TestRegistryPlugin(t *testing.T) {
	reg := NewRegistry()
	plug := &testPlugin{}

	if err := reg.Use(plug); err != nil {
		t.Fatalf("use plugin: %v", err)
	}
	if !plug.called {
		t.Fatalf("expected plugin register to be called")
	}
	if err := reg.Use(plug); err == nil {
		t.Fatalf("expected duplicate plugin error")
	}

	if _, err := reg.Validator("starts_with"); err != nil {
		t.Fatalf("expected validator registered: %v", err)
	}
}

type testPlugin struct {
	called bool
}

func (p *testPlugin) Name() string {
	return "test"
}

func (p *testPlugin) Register(r *Registry) error {
	p.called = true
	return r.RegisterValidator("starts_with", func(field string, value reflect.Value, param string) *validate.FieldError {
		return nil
	})
}
