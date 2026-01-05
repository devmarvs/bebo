package openapi

import "testing"

func TestBuilderAddRoute(t *testing.T) {
	builder := New(Info{Title: "bebo", Version: "0.1"})
	if err := builder.AddRoute("GET", "/health", Operation{Summary: "health"}); err != nil {
		t.Fatalf("add route: %v", err)
	}

	doc := builder.Document()
	item := doc.Paths["/health"]
	if item == nil || item.Get == nil {
		t.Fatalf("expected GET operation")
	}
}

func TestBuilderUnsupportedMethod(t *testing.T) {
	builder := New(Info{Title: "bebo", Version: "0.1"})
	if err := builder.AddRoute("FOO", "/health", Operation{}); err == nil {
		t.Fatalf("expected unsupported method error")
	}
}
