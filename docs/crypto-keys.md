# Crypto and Key Rotation

## Key inventory
- JWT signing keys (auth.JWTKeySet)
- Session signing keys (session.NewCookieStore)
- CSRF tokens (middleware.CSRF)

## Rotation process
1. Add a new primary key and keep old keys as fallback.
2. Deploy and allow existing sessions/tokens to expire.
3. Remove old keys once the maximum TTL passes.

## JWT rotation example
```go
keys := auth.JWTKeySet{
    Primary: auth.JWTKey{ID: "v2", Secret: []byte(os.Getenv("JWT_KEY_V2"))},
    Fallback: []auth.JWTKey{
        {ID: "v1", Secret: []byte(os.Getenv("JWT_KEY_V1"))},
    },
}
authenticator := auth.JWTAuthenticator{KeySet: &keys}
```

## Session key rotation example
```go
current := []byte(os.Getenv("SESSION_KEY_V2"))
old := []byte(os.Getenv("SESSION_KEY_V1"))
store := session.NewCookieStore("bebo_session", current, old)
```

## Key storage guidance
- Use 32+ random bytes for HMAC keys.
- Store keys in a secret manager and restrict access.
- Audit key usage and rotate when staff or environments change.
