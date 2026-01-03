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

func TestWildcardValidation(t *testing.T) {
	r := New()
	if _, err := r.Add("GET", "/assets/*path/more"); err == nil {
		t.Fatalf("expected wildcard error")
	}
}
