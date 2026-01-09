# Hardening Guide

This guide covers practical hardening steps for production bebo deployments.

## Baseline middleware
```go
app.Use(
    middleware.RequestID(),
    middleware.Recover(),
    middleware.Logger(),
    middleware.SecurityHeaders(middleware.DefaultSecurityHeaders()),
    middleware.CSRF(middleware.CSRFOptions{}),
    middleware.BodyLimit(2<<20),
    middleware.Timeout(10*time.Second),
    middleware.Gzip(0),
)
app.UsePre(middleware.MethodOverride(middleware.MethodOverrideOptions{}))
```

## Headers and TLS
- Terminate TLS at the edge and enable HSTS.
- Use a CSP tuned to your assets and third-party dependencies.
- Disable framing unless explicitly required.

## Sessions and cookies
- Use Redis/Postgres session stores in multi-instance deployments.
- Set Secure, HTTPOnly, and SameSite for cookies.
- Rotate signing keys and keep a short grace window for old keys.

## Input validation
- Use ctx.BindJSON/BindForm and validate.Struct for structured payloads.
- Enforce body size limits for uploads.
- Avoid logging raw payloads containing secrets or PII.

## AuthN/AuthZ
- Put authentication middleware early in the chain for protected routes.
- Keep authorization checks close to data access.

## Rate limiting and abuse protection
- Use Redis-backed rate limiting for shared limits across instances.
- Apply per-route policies for high-risk endpoints.

## Observability and profiling
- Use pprof endpoints only behind authentication and restricted networks.
- Include request IDs and trace IDs in logs and pass them through background jobs.
