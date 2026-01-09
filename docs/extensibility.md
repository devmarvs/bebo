# Extensibility

bebo provides a registry and hook points to keep integrations decoupled.

## Registry
```go
reg := bebo.NewRegistry()
_ = reg.RegisterMiddleware("request_id", func(_ map[string]any) (bebo.Middleware, error) {
    return middleware.RequestID(), nil
})

app := bebo.New(bebo.WithRegistry(reg))
mw, _ := app.Registry().Middleware("request_id", nil)
app.Use(mw)
```

Plugins can bundle registrations:
```go
type MyPlugin struct{}

func (MyPlugin) Name() string { return "my-plugin" }
func (MyPlugin) Register(r *bebo.Registry) error {
    return r.RegisterValidator("starts_with", func(field string, value reflect.Value, param string) *validate.FieldError {
        return nil
    })
}
```

## Hook points
- Auth: `bebo.WithAuthHooks` receives callbacks before and after authentication.
- Cache: `cache.WithHooks` wraps a cache store with hit/miss/set hooks.
- Validation: `validate.SetHooks` lets you observe validation errors.
