package bebo

// Principal represents an authenticated actor.
type Principal struct {
	ID     string
	Roles  []string
	Claims map[string]any
}

// Authenticator validates a request and returns a principal.
type Authenticator interface {
	Authenticate(*Context) (*Principal, error)
}

// Authorizer checks if a principal can access a resource.
type Authorizer interface {
	Authorize(*Context, *Principal) error
}

const principalKey = "bebo.principal"

// PrincipalFromContext extracts the principal from context storage.
func PrincipalFromContext(ctx *Context) (*Principal, bool) {
	value, ok := ctx.Get(principalKey)
	if !ok {
		return nil, false
	}
	principal, ok := value.(*Principal)
	return principal, ok
}

// SetPrincipal stores the principal in context storage.
func SetPrincipal(ctx *Context, principal *Principal) {
	ctx.Set(principalKey, principal)
}
