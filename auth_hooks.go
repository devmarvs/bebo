package bebo

// AuthHooks provides hook points around authentication.
type AuthHooks struct {
	BeforeAuthenticate func(ctx *Context)
	AfterAuthenticate  func(ctx *Context, principal *Principal, err error)
}
