# Timeouts and Circuit Breakers

## Server timeouts
Use config defaults or BEBO_ environment overrides:
- BEBO_READ_TIMEOUT
- BEBO_WRITE_TIMEOUT
- BEBO_IDLE_TIMEOUT
- BEBO_READ_HEADER_TIMEOUT
- BEBO_SHUTDOWN_TIMEOUT

## Request timeouts
```go
app.Use(middleware.Timeout(10 * time.Second))
```

## Downstream HTTP clients
```go
breaker := httpclient.NewCircuitBreaker(httpclient.CircuitBreakerOptions{})
client := httpclient.NewClient(httpclient.ClientOptions{
    Timeout: 5 * time.Second,
    Retry:   httpclient.DefaultRetryOptions(),
    Breaker: breaker,
})
_ = client
```

## Guidance
- Set time budgets for each hop and keep them consistent.
- Retry only idempotent requests.
- Use circuit breakers for flaky dependencies.
