package web

import (
	"fmt"
	"html"
	"html/template"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/flash"
	"github.com/devmarvs/bebo/middleware"
	"github.com/devmarvs/bebo/render"
)

// TemplateData carries default template fields.
type TemplateData struct {
	Data      any
	CSRFToken string
	Flash     []flash.Message
}

// TemplateDataFrom builds TemplateData from the request context.
func TemplateDataFrom(ctx *bebo.Context, store *flash.Store, data any) (TemplateData, error) {
	view := TemplateData{
		Data:      data,
		CSRFToken: middleware.CSRFToken(ctx),
	}

	if store == nil {
		return view, nil
	}

	messages, err := store.Pop(ctx.ResponseWriter, ctx.Request)
	if err != nil {
		return view, err
	}
	view.Flash = messages
	return view, nil
}

// Funcs returns template helpers for CSRF fields.
func Funcs() render.FuncMap {
	return render.FuncMap{
		"csrfField":      CSRFField,
		"csrfFieldNamed": CSRFFieldNamed,
	}
}

// CSRFField renders a hidden CSRF field using the default name.
func CSRFField(token string) template.HTML {
	return CSRFFieldNamed("csrf_token", token)
}

// CSRFFieldNamed renders a hidden CSRF field with a custom name.
func CSRFFieldNamed(name, token string) template.HTML {
	if token == "" {
		return ""
	}
	return template.HTML(fmt.Sprintf(
		"<input type=\"hidden\" name=\"%s\" value=\"%s\">",
		html.EscapeString(name),
		html.EscapeString(token),
	))
}
