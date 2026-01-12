# Project Structure Guide

This is a recommended layout for a production bebo service.

## Suggested layout
```text
cmd/
  server/
    main.go
internal/
  app/
    routes.go
  handlers/
    users.go
  store/
    users.go
  jobs/
    workers.go
migrations/
templates/
static/
deploy/
```

## Rationale
- Keep entrypoints small and push logic into internal packages.
- Separate transport (handlers), domain logic, and data access.
- Keep templates and migrations close to the app that owns them.

## bebo integration
- Use config.Default + config.LoadFromEnv for consistent defaults.
- Register middleware early (request ID, recovery, logging, security headers).
- Expose /health and /ready using health.Registry.
- Use db.Helper for query timeouts and migrate.Runner for migrations.

## CLI scaffolding
```sh
bebo new ./myapp -module github.com/me/myapp -web -template
bebo crud new users -dir internal/handlers -package handlers -templates templates
bebo migrate new -dir ./migrations -name create_users
```

The generator writes `bebo.manifest.json` with the template version and project kind.
