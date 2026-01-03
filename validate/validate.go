package validate

import (
	"net/mail"
	"unicode/utf8"

	"github.com/devmarvs/bebo/apperr"
)

// Required ensures a non-empty string.
func Required(field, value string) error {
	if value == "" {
		return apperr.New(apperr.CodeValidation, 400, field+" is required", nil)
	}
	return nil
}

// MinLen ensures a minimum string length.
func MinLen(field, value string, min int) error {
	if utf8.RuneCountInString(value) < min {
		return apperr.New(apperr.CodeValidation, 400, field+" is too short", nil)
	}
	return nil
}

// MaxLen ensures a maximum string length.
func MaxLen(field, value string, max int) error {
	if utf8.RuneCountInString(value) > max {
		return apperr.New(apperr.CodeValidation, 400, field+" is too long", nil)
	}
	return nil
}

// Email validates an email address.
func Email(field, value string) error {
	if value == "" {
		return nil
	}
	if _, err := mail.ParseAddress(value); err != nil {
		return apperr.New(apperr.CodeValidation, 400, field+" must be a valid email", err)
	}
	return nil
}
