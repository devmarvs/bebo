package router

import "testing"

func TestMatch(t *testing.T) {
	r := New()

	idRoot, err := r.Add("GET", "/")
	if err != nil {
		t.Fatalf("add root: %v", err)
	}
	idUser, err := r.Add("GET", "/users/:id")
	if err != nil {
		t.Fatalf("add users: %v", err)
	}
	idFiles, err := r.Add("GET", "/assets/*path")
	if err != nil {
		t.Fatalf("add assets: %v", err)
	}

	id, params, ok := r.Match("GET", "/")
	if !ok || id != idRoot {
		t.Fatalf("expected root match")
	}
	if len(params) != 0 {
		t.Fatalf("expected no params")
	}

	id, params, ok = r.Match("GET", "/users/42")
	if !ok || id != idUser {
		t.Fatalf("expected user match")
	}
	if params["id"] != "42" {
		t.Fatalf("expected id param")
	}

	id, params, ok = r.Match("GET", "/assets/css/app.css")
	if !ok || id != idFiles {
		t.Fatalf("expected assets match")
	}
	if params["path"] != "css/app.css" {
		t.Fatalf("expected path param")
	}

	_, _, ok = r.Match("POST", "/users/42")
	if ok {
		t.Fatalf("expected method mismatch")
	}
}

func TestAllowed(t *testing.T) {
	r := New()
	_, _ = r.Add("GET", "/users/:id")
	_, _ = r.Add("POST", "/users/:id")
	_, _ = r.Add("DELETE", "/users/:id")

	allowed := r.Allowed("/users/123")
	if len(allowed) != 3 {
		t.Fatalf("expected 3 allowed methods, got %d", len(allowed))
	}
}

func TestHostMatching(t *testing.T) {
	r := New()
	idWeb, _ := r.AddWithHost("GET", "example.com", "/")
	idAPI, _ := r.AddWithHost("GET", "api.example.com", "/")
	idWildcard, _ := r.AddWithHost("GET", "*.example.com", "/wild")

	id, _, ok := r.MatchHost("GET", "example.com", "/")
	if !ok || id != idWeb {
		t.Fatalf("expected exact host match")
	}

	id, _, ok = r.MatchHost("GET", "api.example.com", "/")
	if !ok || id != idAPI {
		t.Fatalf("expected api host match")
	}

	id, _, ok = r.MatchHost("GET", "foo.example.com", "/wild")
	if !ok || id != idWildcard {
		t.Fatalf("expected wildcard host match")
	}
}

func TestWildcardValidation(t *testing.T) {
	r := New()
	if _, err := r.Add("GET", "/assets/*path/more"); err == nil {
		t.Fatalf("expected wildcard error")
	}
}
