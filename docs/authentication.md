# Authentication and Authorization

This guide covers default-safe patterns for auth in bebo.

## Sessions
- Use `middleware.Session` to load sessions and keep persistence explicit with `Save`/`Clear`.
- Prefer Redis/Postgres stores for multi-instance deployments.
- Set `Secure`, `HTTPOnly`, and `SameSite` on cookies.

```go
store := session.NewCookieStore("bebo_session", []byte(os.Getenv("SESSION_KEY")))
app.Use(middleware.Session(store))

app.POST("/login", func(ctx *bebo.Context) error {
    sess, _ := middleware.SessionFromContext(ctx)
    sess.Set("user_id", "user-1")
    return sess.Save(ctx.ResponseWriter)
})
```

## JWT
- Set `Issuer` and `Audience` for each environment.
- Keep tokens short-lived and rotate keys with `JWTKeySet`.
- Use `ClockSkew` to tolerate minor drift.

```go
authenticator := auth.JWTAuthenticator{
    Key:      []byte(os.Getenv("JWT_KEY")),
    Issuer:   "bebo",
    Audience: "api",
    ClockSkew: 30 * time.Second,
}

app.GET("/private", handler, middleware.RequireJWT(authenticator))
```

Key rotation:
```go
keys := auth.JWTKeySet{
    Primary: auth.JWTKey{ID: "v2", Secret: []byte(os.Getenv("JWT_KEY_V2"))},
    Fallback: []auth.JWTKey{
        {ID: "v1", Secret: []byte(os.Getenv("JWT_KEY_V1"))},
    },
}
app.GET("/private", handler, middleware.RequireJWT(auth.JWTAuthenticator{KeySet: &keys}))
```

## Roles and permissions
Use the built-in helpers with the auth middleware:
```go
authz := auth.RequireRoles("admin")
app.GET("/admin", handler, middleware.RequireAuthorization(authenticator, authz))
```
Permissions are read from the `permissions` claim:
```go
authz := auth.RequireAnyPermission("reports.read", "reports.write")
app.GET("/reports", handler, middleware.RequireAuthorization(authenticator, authz))
```

## Secure defaults checklist
- Keep session signing keys 32+ bytes and rotate them.
- Use `Secure` cookies in production and set a `SameSite` policy.
- Validate issuer/audience for JWTs and enforce expiry.
- Scope authorization checks close to data access.
