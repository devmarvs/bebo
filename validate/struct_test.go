package validate

import (
	"reflect"
	"strings"
	"testing"
)

type userInput struct {
	Name  string `json:"name" validate:"required,min=3"`
	Email string `json:"email" validate:"required,email"`
}

type profileInput struct {
	Age    int      `json:"age" validate:"required,min=18"`
	Score  float64  `json:"score" validate:"max=100"`
	Active bool     `json:"active" validate:"required"`
	Tags   []string `json:"tags" validate:"min=1"`
}

type customInput struct {
	Code string `json:"code" validate:"starts_with=BE"`
}

func TestStructValidation(t *testing.T) {
	input := userInput{Name: "Al", Email: "invalid"}
	err := Struct(input)
	if err == nil {
		t.Fatalf("expected validation error")
	}

	verr, ok := As(err)
	if !ok {
		t.Fatalf("expected validation errors")
	}
	if len(verr.Fields) != 2 {
		t.Fatalf("expected 2 field errors, got %d", len(verr.Fields))
	}
}

func TestStructValidationNonString(t *testing.T) {
	input := profileInput{Age: 16, Score: 101.5, Active: false, Tags: nil}
	err := Struct(input)
	if err == nil {
		t.Fatalf("expected validation error")
	}

	verr, ok := As(err)
	if !ok {
		t.Fatalf("expected validation errors")
	}
	if len(verr.Fields) != 4 {
		t.Fatalf("expected 4 field errors, got %d", len(verr.Fields))
	}
}

func TestCustomValidator(t *testing.T) {
	Register("starts_with", func(field string, value reflect.Value, param string) *FieldError {
		if value.Kind() != reflect.String {
			return nil
		}
		if !strings.HasPrefix(value.String(), param) {
			return &FieldError{Field: field, Message: field + " must start with " + param}
		}
		return nil
	})

	err := Struct(customInput{Code: "XX"})
	if err == nil {
		t.Fatalf("expected validation error")
	}

	verr, ok := As(err)
	if !ok {
		t.Fatalf("expected validation errors")
	}
	if len(verr.Fields) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(verr.Fields))
	}
}
