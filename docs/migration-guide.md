# Migration Guide

This guide maps common patterns from popular Go web frameworks to bebo.

## From Gin
- Router: `gin.Default()` -> `bebo.New()` plus middleware
- Routes: `r.GET` -> `app.GET`
- Params: `c.Param("id")` -> `ctx.Param("id")`
- JSON: `c.JSON(code, payload)` -> `ctx.JSON(code, payload)`
- Binding: `c.BindJSON(&dst)` -> `ctx.BindJSON(&dst)`
- Errors: return errors and let the error handler respond

## From Echo
- Router: `echo.New()` -> `bebo.New()`
- Middleware: `e.Use` -> `app.Use`
- Params: `c.Param("id")` -> `ctx.Param("id")`
- JSON: `return c.JSON(code, payload)` -> `return ctx.JSON(code, payload)`
- Templates: `c.Render` -> `ctx.HTML` with render.Engine

## From Fiber
- Router: `fiber.New()` -> `bebo.New()`
- Groups: `app.Group("/api")` -> `app.Group("/api")`
- Params: `c.Params("id")` -> `ctx.Param("id")`
- JSON: `return c.JSON(payload)` -> `return ctx.JSON(status, payload)`

## Router notes
- Params use `:id` and wildcards use `*path`.
- Host-based routing is supported via bebo.WithHost.
