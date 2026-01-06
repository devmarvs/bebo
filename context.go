package bebo

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

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

// ParamInt returns a route param as an int.
func (c *Context) ParamInt(name string) (int, error) {
	value := c.Param(name)
	if value == "" {
		return 0, apperr.BadRequest(name+" is required", nil)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, apperr.BadRequest(name+" must be an integer", err)
	}
	return parsed, nil
}

// ParamInt64 returns a route param as an int64.
func (c *Context) ParamInt64(name string) (int64, error) {
	value := c.Param(name)
	if value == "" {
		return 0, apperr.BadRequest(name+" is required", nil)
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, apperr.BadRequest(name+" must be an integer", err)
	}
	return parsed, nil
}

// ParamBool returns a route param as a bool.
func (c *Context) ParamBool(name string) (bool, error) {
	value := c.Param(name)
	if value == "" {
		return false, apperr.BadRequest(name+" is required", nil)
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, apperr.BadRequest(name+" must be a boolean", err)
	}
	return parsed, nil
}

// Query returns a query param.
func (c *Context) Query(name string) string {
	return c.Request.URL.Query().Get(name)
}

// QueryInt returns a query param as an int.
func (c *Context) QueryInt(name string) (int, error) {
	value := c.Query(name)
	if value == "" {
		return 0, apperr.BadRequest(name+" is required", nil)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, apperr.BadRequest(name+" must be an integer", err)
	}
	return parsed, nil
}

// QueryInt64 returns a query param as an int64.
func (c *Context) QueryInt64(name string) (int64, error) {
	value := c.Query(name)
	if value == "" {
		return 0, apperr.BadRequest(name+" is required", nil)
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, apperr.BadRequest(name+" must be an integer", err)
	}
	return parsed, nil
}

// QueryBool returns a query param as a bool.
func (c *Context) QueryBool(name string) (bool, error) {
	value := c.Query(name)
	if value == "" {
		return false, apperr.BadRequest(name+" is required", nil)
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, apperr.BadRequest(name+" must be a boolean", err)
	}
	return parsed, nil
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

const DefaultMultipartMemory int64 = 32 << 20

// BindForm binds URL-encoded form values into dst.
func (c *Context) BindForm(dst any) error {
	if err := c.Request.ParseForm(); err != nil {
		return apperr.BadRequest("invalid form", err)
	}
	values := c.Request.PostForm
	if len(values) == 0 {
		values = c.Request.Form
	}
	return bindValues(values, dst)
}

// BindMultipart binds multipart form values into dst.
func (c *Context) BindMultipart(dst any, maxMemory int64) error {
	if maxMemory <= 0 {
		maxMemory = DefaultMultipartMemory
	}
	if err := c.Request.ParseMultipartForm(maxMemory); err != nil {
		return apperr.BadRequest("invalid multipart form", err)
	}
	values := url.Values{}
	if c.Request.MultipartForm != nil {
		for key, vals := range c.Request.MultipartForm.Value {
			values[key] = vals
		}
	}
	return bindValues(values, dst)
}

// FormFile returns a file header from a multipart request.
func (c *Context) FormFile(name string, maxMemory int64) (*multipart.FileHeader, error) {
	if maxMemory <= 0 {
		maxMemory = DefaultMultipartMemory
	}
	if err := c.Request.ParseMultipartForm(maxMemory); err != nil {
		return nil, apperr.BadRequest("invalid multipart form", err)
	}
	file, header, err := c.Request.FormFile(name)
	if err != nil {
		return nil, err
	}
	_ = file.Close()
	return header, nil
}

// SaveUploadedFile writes a multipart file to disk.
func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	if file == nil {
		return apperr.BadRequest("file is required", nil)
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

// Render uses a custom render function.
func (c *Context) Render(status int, fn render.RenderFunc) error {
	return render.Custom(c.ResponseWriter, status, fn)
}

// RequestID returns the request id header.
func (c *Context) RequestID() string {
	return RequestIDFromHeader(c.Request)
}

func bindValues(values url.Values, dst any) error {
	if dst == nil {
		return apperr.BadRequest("destination is required", nil)
	}

	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return apperr.BadRequest("destination must be a pointer", nil)
	}

	rv = rv.Elem()

	switch rv.Kind() {
	case reflect.Struct:
		return bindStruct(values, rv)
	case reflect.Map:
		return bindMap(values, rv)
	default:
		return apperr.BadRequest("destination must be a struct or map", nil)
	}
}

func bindMap(values url.Values, rv reflect.Value) error {
	if rv.Type().Key().Kind() != reflect.String {
		return apperr.BadRequest("map key must be a string", nil)
	}
	if rv.IsNil() {
		rv.Set(reflect.MakeMap(rv.Type()))
	}

	valueType := rv.Type().Elem()
	switch valueType.Kind() {
	case reflect.String:
		for key, vals := range values {
			if len(vals) == 0 {
				continue
			}
			rv.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(vals[0]))
		}
	case reflect.Slice:
		if valueType.Elem().Kind() != reflect.String {
			return apperr.BadRequest("map value must be []string", nil)
		}
		for key, vals := range values {
			copyVals := append([]string{}, vals...)
			rv.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(copyVals))
		}
	default:
		return apperr.BadRequest("map value must be string or []string", nil)
	}
	return nil
}

func bindStruct(values url.Values, rv reflect.Value) error {
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name := bindFieldName(field)
		if name == "" || name == "-" {
			continue
		}
		vals, ok := values[name]
		if !ok || len(vals) == 0 {
			continue
		}
		if err := setFieldValue(rv.Field(i), name, vals); err != nil {
			return err
		}
	}
	return nil
}

func bindFieldName(field reflect.StructField) string {
	if tag, ok := tagName(field.Tag.Get("form")); ok {
		return tag
	}
	if tag, ok := tagName(field.Tag.Get("json")); ok {
		return tag
	}
	return field.Name
}

func tagName(tag string) (string, bool) {
	if tag == "" {
		return "", false
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		return "", false
	}
	return name, true
}

func setFieldValue(field reflect.Value, name string, values []string) error {
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFieldValue(field.Elem(), name, values)
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(values[0])
		return nil
	case reflect.Bool:
		parsed, err := strconv.ParseBool(values[0])
		if err != nil {
			return apperr.BadRequest(name+" must be a boolean", err)
		}
		field.SetBool(parsed)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(values[0], 10, field.Type().Bits())
		if err != nil {
			return apperr.BadRequest(name+" must be an integer", err)
		}
		field.SetInt(parsed)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(values[0], 10, field.Type().Bits())
		if err != nil {
			return apperr.BadRequest(name+" must be an integer", err)
		}
		field.SetUint(parsed)
		return nil
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(values[0], field.Type().Bits())
		if err != nil {
			return apperr.BadRequest(name+" must be a number", err)
		}
		field.SetFloat(parsed)
		return nil
	case reflect.Slice:
		if field.Type().Elem().Kind() != reflect.String {
			return apperr.BadRequest(name+" must be a list of strings", nil)
		}
		copyVals := append([]string{}, values...)
		field.Set(reflect.ValueOf(copyVals))
		return nil
	default:
		return apperr.BadRequest(name+" is not assignable", nil)
	}
}
