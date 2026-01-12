package auth

import (
	"errors"

	"github.com/devmarvs/bebo"
)

// PermissionClaim is the JWT claim used for permissions.
const PermissionClaim = "permissions"

var (
	// ErrMissingRole indicates a missing required role.
	ErrMissingRole = errors.New("missing required role")
	// ErrMissingPermission indicates a missing required permission.
	ErrMissingPermission = errors.New("missing required permission")
)

// HasRole reports whether a principal has the given role.
func HasRole(principal *bebo.Principal, role string) bool {
	if principal == nil || role == "" {
		return false
	}
	for _, item := range principal.Roles {
		if item == role {
			return true
		}
	}
	return false
}

// HasAnyRole reports whether a principal has any of the provided roles.
func HasAnyRole(principal *bebo.Principal, roles ...string) bool {
	if principal == nil || len(roles) == 0 {
		return false
	}
	for _, role := range roles {
		if HasRole(principal, role) {
			return true
		}
	}
	return false
}

// HasAllRoles reports whether a principal has all of the provided roles.
func HasAllRoles(principal *bebo.Principal, roles ...string) bool {
	if principal == nil || len(roles) == 0 {
		return false
	}
	for _, role := range roles {
		if !HasRole(principal, role) {
			return false
		}
	}
	return true
}

// HasPermission reports whether a principal has the given permission.
func HasPermission(principal *bebo.Principal, permission string) bool {
	if principal == nil || permission == "" {
		return false
	}
	for _, item := range permissionsFromClaims(principal) {
		if item == permission {
			return true
		}
	}
	return false
}

// HasAnyPermission reports whether a principal has any of the provided permissions.
func HasAnyPermission(principal *bebo.Principal, permissions ...string) bool {
	if principal == nil || len(permissions) == 0 {
		return false
	}
	for _, permission := range permissions {
		if HasPermission(principal, permission) {
			return true
		}
	}
	return false
}

// HasAllPermissions reports whether a principal has all of the provided permissions.
func HasAllPermissions(principal *bebo.Principal, permissions ...string) bool {
	if principal == nil || len(permissions) == 0 {
		return false
	}
	for _, permission := range permissions {
		if !HasPermission(principal, permission) {
			return false
		}
	}
	return true
}

// RoleAuthorizer requires roles for authorization.
type RoleAuthorizer struct {
	Any []string
	All []string
}

// PermissionAuthorizer requires permissions for authorization.
type PermissionAuthorizer struct {
	Any []string
	All []string
}

// RequireRoles enforces all roles.
func RequireRoles(roles ...string) RoleAuthorizer {
	return RoleAuthorizer{All: roles}
}

// RequireAnyRole enforces at least one role.
func RequireAnyRole(roles ...string) RoleAuthorizer {
	return RoleAuthorizer{Any: roles}
}

// RequirePermissions enforces all permissions.
func RequirePermissions(permissions ...string) PermissionAuthorizer {
	return PermissionAuthorizer{All: permissions}
}

// RequireAnyPermission enforces at least one permission.
func RequireAnyPermission(permissions ...string) PermissionAuthorizer {
	return PermissionAuthorizer{Any: permissions}
}

// Authorize validates required roles.
func (r RoleAuthorizer) Authorize(_ *bebo.Context, principal *bebo.Principal) error {
	if len(r.All) == 0 && len(r.Any) == 0 {
		return nil
	}
	if principal == nil {
		return ErrMissingRole
	}
	if len(r.All) > 0 && !HasAllRoles(principal, r.All...) {
		return ErrMissingRole
	}
	if len(r.Any) > 0 && !HasAnyRole(principal, r.Any...) {
		return ErrMissingRole
	}
	return nil
}

// Authorize validates required permissions.
func (p PermissionAuthorizer) Authorize(_ *bebo.Context, principal *bebo.Principal) error {
	if len(p.All) == 0 && len(p.Any) == 0 {
		return nil
	}
	if principal == nil {
		return ErrMissingPermission
	}
	if len(p.All) > 0 && !HasAllPermissions(principal, p.All...) {
		return ErrMissingPermission
	}
	if len(p.Any) > 0 && !HasAnyPermission(principal, p.Any...) {
		return ErrMissingPermission
	}
	return nil
}

func permissionsFromClaims(principal *bebo.Principal) []string {
	if principal == nil || principal.Claims == nil {
		return nil
	}
	return parseStringList(principal.Claims[PermissionClaim])
}

func parseStringList(value any) []string {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return append([]string{}, typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	case string:
		return []string{typed}
	default:
		return nil
	}
}
