package apperr

import (
	"errors"
	"fmt"
)

const (
	CodeInternal     = "internal"
	CodeNotFound     = "not_found"
	CodeBadRequest   = "bad_request"
	CodeValidation   = "validation"
	CodeUnauthorized = "unauthorized"
	CodeForbidden    = "forbidden"
)

// Error represents a structured application error.
type Error struct {
	Code    string
	Status  int
	Message string
	Cause   error
}

// New creates a new Error.
func New(code string, status int, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Status:  status,
		Message: message,
		Cause:   cause,
	}
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
}

// Unwrap returns the root cause.
func (e *Error) Unwrap() error {
	return e.Cause
}

// As extracts an *Error if present.
func As(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}
