package bebo

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/devmarvs/bebo/apperr"
	"github.com/devmarvs/bebo/render"
	"github.com/devmarvs/bebo/router"
)

// Context holds request-specific data.
type Context struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Params         router.Params

	app    *App
	values map[string]any
}

// NewContext constructs a Context.
func NewContext(w http.ResponseWriter, r *http.Request, params router.Params, app *App) *Context {
	return &Context{
		ResponseWriter: w,
		Request:        r,
		Params:         params,
		app:            app,
		values:         make(map[string]any),
	}
}

// Param returns a route param.
func (c *Context) Param(name string) string {
	return c.Params[name]
}

// Query returns a query param.
func (c *Context) Query(name string) string {
	return c.Request.URL.Query().Get(name)
}

// Set stores a value in the context.
func (c *Context) Set(key string, value any) {
	c.values[key] = value
}

// Get retrieves a stored value.
func (c *Context) Get(key string) (any, bool) {
	value, ok := c.values[key]
	return value, ok
}

// Logger returns the app logger.
func (c *Context) Logger() Logger {
	return Logger{logger: c.app.logger, requestID: RequestIDFromHeader(c.Request)}
}

// JSON responds with JSON.
func (c *Context) JSON(status int, payload any) error {
	return render.JSON(c.ResponseWriter, status, payload)
}

// Text responds with plain text.
func (c *Context) Text(status int, message string) error {
	return render.Text(c.ResponseWriter, status, message)
}

// HTML renders a template.
func (c *Context) HTML(status int, name string, data any) error {
	if c.app.renderer == nil {
		return apperr.Internal("template engine not configured", nil)
	}
	return c.app.renderer.Render(c.ResponseWriter, status, name, data)
}

// BindJSON binds the request body to a struct.
func (c *Context) BindJSON(dst any) error {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return apperr.PayloadTooLarge("request body too large", err)
		}
		return apperr.BadRequest("invalid JSON", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return apperr.BadRequest("unexpected JSON payload", err)
	}
	return nil
}

// Render uses a custom render function.
func (c *Context) Render(status int, fn render.RenderFunc) error {
	return render.Custom(c.ResponseWriter, status, fn)
}

// RequestID returns the request id header.
func (c *Context) RequestID() string {
	return RequestIDFromHeader(c.Request)
}
