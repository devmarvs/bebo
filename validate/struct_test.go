package validate

import "testing"

type userInput struct {
	Name  string `json:"name" validate:"required,min=3"`
	Email string `json:"email" validate:"required,email"`
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
