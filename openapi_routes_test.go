package bebo

import (
	"net/http"
	"testing"

	"github.com/devmarvs/bebo/openapi"
)

func TestAddOpenAPIRoutes(t *testing.T) {
	app := New()
	app.Route(http.MethodGet, "/users/:id", func(*Context) error { return nil }, WithName("user.show"))

	builder := openapi.New(openapi.Info{Title: "bebo", Version: "v0.1"})
	if err := app.AddOpenAPIRoutes(builder); err != nil {
		t.Fatalf("add openapi routes: %v", err)
	}

	doc := builder.Document()
	item := doc.Paths["/users/{id}"]
	if item == nil || item.Get == nil {
		t.Fatalf("expected GET operation")
	}
	if len(item.Get.Parameters) != 1 {
		t.Fatalf("expected 1 param, got %d", len(item.Get.Parameters))
	}
	if item.Get.Parameters[0].Name != "id" {
		t.Fatalf("expected param id")
	}
}

func TestAddOpenAPIRoutesSkipUnnamed(t *testing.T) {
	app := New()
	app.GET("/health", func(*Context) error { return nil })

	builder := openapi.New(openapi.Info{Title: "bebo", Version: "v0.1"})
	if err := app.AddOpenAPIRoutes(builder, WithOpenAPIIncludeUnnamed(false)); err != nil {
		t.Fatalf("add openapi routes: %v", err)
	}

	if _, ok := builder.Document().Paths["/health"]; ok {
		t.Fatalf("expected unnamed route to be skipped")
	}
}
