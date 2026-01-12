# CRUD example (real app)

This example demonstrates CRUD routes, sessions, form auth, and migrations.

## Prerequisites
- PostgreSQL running locally
- Go 1.25

## Setup
```sh
export BEBO_DATABASE_URL="postgres://postgres:postgres@localhost:5432/bebo_crud?sslmode=disable"
export BEBO_SESSION_KEY="replace-with-32-byte-random-value"
```

## Run migrations
```sh
go run ./examples/crud -migrate
```

## Start the app
```sh
go run ./examples/crud
```

## Routes
- Web: /signup, /login, /notes
- API: /api/notes
- Health: /health, /ready

## Production runbook
See `examples/crud/RUNBOOK.md`.
