# Scaling Guidelines

## Stateless services
- Keep app instances stateless and move state to Postgres/Redis.
- Use Redis/Postgres session stores for multi-instance deployments.
- Use Redis-backed rate limits so limits apply across instances.

## Database
- Use connection pooling (db.Open with Options).
- Index high-cardinality lookups and avoid N+1 queries.
- Use timeouts for all queries (db.Helper).

## Background jobs
- Make job handlers idempotent.
- Use retries with backoff and dead-letter hooks.
- Propagate request metadata into job contexts.

## Caching
- Cache read-heavy endpoints with explicit invalidation.
- Avoid caching responses containing personalized data without keying correctly.
