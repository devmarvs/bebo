package auth

import (
	"errors"
	"testing"

	"github.com/devmarvs/bebo"
)

func TestRoleHelpers(t *testing.T) {
	principal := &bebo.Principal{Roles: []string{"admin", "editor"}}

	if !HasRole(principal, "admin") {
		t.Fatalf("expected role match")
	}
	if HasRole(principal, "missing") {
		t.Fatalf("unexpected role match")
	}
	if !HasAnyRole(principal, "missing", "editor") {
		t.Fatalf("expected any role match")
	}
	if HasAnyRole(principal, "missing", "other") {
		t.Fatalf("unexpected any role match")
	}
	if !HasAllRoles(principal, "admin", "editor") {
		t.Fatalf("expected all roles match")
	}
	if HasAllRoles(principal, "admin", "missing") {
		t.Fatalf("unexpected all roles match")
	}
}

func TestPermissionHelpers(t *testing.T) {
	principal := &bebo.Principal{Claims: map[string]any{"permissions": []string{"read", "write"}}}

	if !HasPermission(principal, "read") {
		t.Fatalf("expected permission match")
	}
	if HasPermission(principal, "delete") {
		t.Fatalf("unexpected permission match")
	}
	if !HasAnyPermission(principal, "delete", "write") {
		t.Fatalf("expected any permission match")
	}
	if HasAnyPermission(principal, "delete", "other") {
		t.Fatalf("unexpected any permission match")
	}
	if !HasAllPermissions(principal, "read", "write") {
		t.Fatalf("expected all permissions match")
	}
	if HasAllPermissions(principal, "read", "missing") {
		t.Fatalf("unexpected all permissions match")
	}
}

func TestPermissionClaimsVariants(t *testing.T) {
	principal := &bebo.Principal{Claims: map[string]any{"permissions": []any{"read", "write"}}}
	if !HasPermission(principal, "write") {
		t.Fatalf("expected permission match for []any")
	}

	principal = &bebo.Principal{Claims: map[string]any{"permissions": "read"}}
	if !HasPermission(principal, "read") {
		t.Fatalf("expected permission match for string")
	}
}

func TestAuthorizers(t *testing.T) {
	principal := &bebo.Principal{Roles: []string{"admin"}, Claims: map[string]any{"permissions": []string{"read"}}}

	if err := RequireRoles("admin").Authorize(nil, principal); err != nil {
		t.Fatalf("expected role authorizer ok: %v", err)
	}
	if !errors.Is(RequireRoles("missing").Authorize(nil, principal), ErrMissingRole) {
		t.Fatalf("expected role authorizer error")
	}

	if err := RequirePermissions("read").Authorize(nil, principal); err != nil {
		t.Fatalf("expected permission authorizer ok: %v", err)
	}
	if !errors.Is(RequirePermissions("write").Authorize(nil, principal), ErrMissingPermission) {
		t.Fatalf("expected permission authorizer error")
	}
}
