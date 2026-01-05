package openapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// ErrUnsupportedMethod is returned for unsupported HTTP methods.
var ErrUnsupportedMethod = errors.New("unsupported method")

// Document describes an OpenAPI document.
type Document struct {
	OpenAPI    string               `json:"openapi"`
	Info       Info                 `json:"info"`
	Servers    []Server             `json:"servers,omitempty"`
	Paths      map[string]*PathItem `json:"paths,omitempty"`
	Components *Components          `json:"components,omitempty"`
	Tags       []Tag                `json:"tags,omitempty"`
}

// Info describes API metadata.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Server describes a server entry.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// Tag describes a tag for grouping operations.
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Components holds reusable schemas and security schemes.
type Components struct {
	Schemas         map[string]Schema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme describes auth schemes.
type SecurityScheme struct {
	Type         string `json:"type,omitempty"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
	Description  string `json:"description,omitempty"`
}

// Schema describes a data schema.
type Schema struct {
	Ref         string            `json:"$ref,omitempty"`
	Type        string            `json:"type,omitempty"`
	Format      string            `json:"format,omitempty"`
	Description string            `json:"description,omitempty"`
	Properties  map[string]Schema `json:"properties,omitempty"`
	Items       *Schema           `json:"items,omitempty"`
	Required    []string          `json:"required,omitempty"`
	Enum        []string          `json:"enum,omitempty"`
}

// PathItem describes available operations on a path.
type PathItem struct {
	Get     *Operation `json:"get,omitempty"`
	Post    *Operation `json:"post,omitempty"`
	Put     *Operation `json:"put,omitempty"`
	Patch   *Operation `json:"patch,omitempty"`
	Delete  *Operation `json:"delete,omitempty"`
	Head    *Operation `json:"head,omitempty"`
	Options *Operation `json:"options,omitempty"`
	Trace   *Operation `json:"trace,omitempty"`
}

// Operation describes a single API operation.
type Operation struct {
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses,omitempty"`
	Security    []map[string][]string `json:"security,omitempty"`
}

// Parameter describes an operation parameter.
type Parameter struct {
	Name        string  `json:"name,omitempty"`
	In          string  `json:"in,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody describes a request payload.
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// MediaType describes content.
type MediaType struct {
	Schema  *Schema `json:"schema,omitempty"`
	Example any     `json:"example,omitempty"`
}

// Response describes a response.
type Response struct {
	Description string               `json:"description,omitempty"`
	Headers     map[string]Header    `json:"headers,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// Header describes a response header.
type Header struct {
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// Builder helps compose a Document.
type Builder struct {
	doc Document
}

// New creates a Builder with default OpenAPI version.
func New(info Info) *Builder {
	return &Builder{
		doc: Document{
			OpenAPI: "3.0.3",
			Info:    info,
			Paths:   map[string]*PathItem{},
		},
	}
}

// Document returns the built document.
func (b *Builder) Document() *Document {
	return &b.doc
}

// AddServer registers a server entry.
func (b *Builder) AddServer(server Server) {
	b.doc.Servers = append(b.doc.Servers, server)
}

// AddTag registers a tag.
func (b *Builder) AddTag(tag Tag) {
	b.doc.Tags = append(b.doc.Tags, tag)
}

// AddRoute adds an operation for a method/path.
func (b *Builder) AddRoute(method, path string, op Operation) error {
	if b.doc.Paths == nil {
		b.doc.Paths = map[string]*PathItem{}
	}
	item := b.doc.Paths[path]
	if item == nil {
		item = &PathItem{}
		b.doc.Paths[path] = item
	}

	switch strings.ToLower(method) {
	case "get":
		item.Get = &op
	case "post":
		item.Post = &op
	case "put":
		item.Put = &op
	case "patch":
		item.Patch = &op
	case "delete":
		item.Delete = &op
	case "head":
		item.Head = &op
	case "options":
		item.Options = &op
	case "trace":
		item.Trace = &op
	default:
		return ErrUnsupportedMethod
	}

	return nil
}

// AddSchema registers a schema in components.
func (b *Builder) AddSchema(name string, schema Schema) {
	if b.doc.Components == nil {
		b.doc.Components = &Components{}
	}
	if b.doc.Components.Schemas == nil {
		b.doc.Components.Schemas = map[string]Schema{}
	}
	b.doc.Components.Schemas[name] = schema
}

// AddSecurityScheme registers a security scheme in components.
func (b *Builder) AddSecurityScheme(name string, scheme SecurityScheme) {
	if b.doc.Components == nil {
		b.doc.Components = &Components{}
	}
	if b.doc.Components.SecuritySchemes == nil {
		b.doc.Components.SecuritySchemes = map[string]SecurityScheme{}
	}
	b.doc.Components.SecuritySchemes[name] = scheme
}

// Handler returns an HTTP handler that serves the document as JSON.
func Handler(doc *Document) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if doc == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(doc)
	})
}

// Handler returns an HTTP handler that serves the builder document as JSON.
func (b *Builder) Handler() http.Handler {
	return Handler(&b.doc)
}
