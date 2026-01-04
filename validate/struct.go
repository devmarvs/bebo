package validate

import (
	"errors"
	"net/mail"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/devmarvs/bebo/apperr"
)

// FieldError describes a validation failure for a field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Errors holds multiple field errors.
type Errors struct {
	Fields []FieldError
}

// Error implements the error interface.
func (e *Errors) Error() string {
	return "validation failed"
}

// As extracts validation errors if present.
func As(err error) (*Errors, bool) {
	if err == nil {
		return nil, false
	}
	var verr *Errors
	if errors.As(err, &verr) {
		return verr, true
	}
	return nil, false
}

// Struct validates struct fields using `validate` tags.
func Struct(value any) error {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	rt := rv.Type()
	var errs []FieldError

	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}
		fieldValue := rv.Field(i)
		if fieldValue.Kind() != reflect.String {
			continue
		}

		name := fieldName(field)
		value := fieldValue.String()
		rules := strings.Split(tag, ",")

		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			switch {
			case rule == "required":
				if value == "" {
					errs = append(errs, FieldError{Field: name, Message: name + " is required"})
				}
			case rule == "email":
				if value != "" {
					if _, err := mail.ParseAddress(value); err != nil {
						errs = append(errs, FieldError{Field: name, Message: name + " must be a valid email"})
					}
				}
			case strings.HasPrefix(rule, "min="):
				min, err := strconv.Atoi(strings.TrimPrefix(rule, "min="))
				if err == nil && utf8.RuneCountInString(value) < min {
					errs = append(errs, FieldError{Field: name, Message: name + " is too short"})
				}
			case strings.HasPrefix(rule, "max="):
				max, err := strconv.Atoi(strings.TrimPrefix(rule, "max="))
				if err == nil && utf8.RuneCountInString(value) > max {
					errs = append(errs, FieldError{Field: name, Message: name + " is too long"})
				}
			}
		}
	}

	if len(errs) > 0 {
		return apperr.Validation("validation failed", &Errors{Fields: errs})
	}

	return nil
}

func fieldName(field reflect.StructField) string {
	if tag := field.Tag.Get("json"); tag != "" {
		name := strings.Split(tag, ",")[0]
		if name != "" && name != "-" {
			return name
		}
	}
	return field.Name
}
