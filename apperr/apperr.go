package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

const (
	CodeInternal         = "internal"
	CodeNotFound         = "not_found"
	CodeBadRequest       = "bad_request"
	CodeValidation       = "validation"
	CodeUnauthorized     = "unauthorized"
	CodeForbidden        = "forbidden"
	CodePayloadTooLarge  = "payload_too_large"
	CodeRateLimited      = "rate_limited"
	CodeTimeout          = "timeout"
	CodeMethodNotAllowed = "method_not_allowed"
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

// Internal creates an internal error.
func Internal(message string, cause error) *Error {
	return New(CodeInternal, http.StatusInternalServerError, message, cause)
}

// NotFound creates a not found error.
func NotFound(message string, cause error) *Error {
	return New(CodeNotFound, http.StatusNotFound, message, cause)
}

// BadRequest creates a bad request error.
func BadRequest(message string, cause error) *Error {
	return New(CodeBadRequest, http.StatusBadRequest, message, cause)
}

// Validation creates a validation error.
func Validation(message string, cause error) *Error {
	return New(CodeValidation, http.StatusBadRequest, message, cause)
}

// Unauthorized creates an unauthorized error.
func Unauthorized(message string, cause error) *Error {
	return New(CodeUnauthorized, http.StatusUnauthorized, message, cause)
}

// Forbidden creates a forbidden error.
func Forbidden(message string, cause error) *Error {
	return New(CodeForbidden, http.StatusForbidden, message, cause)
}

// PayloadTooLarge creates a payload too large error.
func PayloadTooLarge(message string, cause error) *Error {
	return New(CodePayloadTooLarge, http.StatusRequestEntityTooLarge, message, cause)
}

// RateLimited creates a rate limited error.
func RateLimited(message string, cause error) *Error {
	return New(CodeRateLimited, http.StatusTooManyRequests, message, cause)
}

// Timeout creates a timeout error.
func Timeout(message string, cause error) *Error {
	return New(CodeTimeout, http.StatusGatewayTimeout, message, cause)
}

// MethodNotAllowed creates a method not allowed error.
func MethodNotAllowed(message string, cause error) *Error {
	return New(CodeMethodNotAllowed, http.StatusMethodNotAllowed, message, cause)
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
