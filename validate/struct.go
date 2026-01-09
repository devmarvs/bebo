package validate

import (
	"errors"
	"net/mail"
	"reflect"
	"strconv"
	"strings"
	"sync"
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

// ValidatorFunc validates a field with an optional parameter.
type ValidatorFunc func(field string, value reflect.Value, param string) *FieldError

var (
	validatorsMu sync.RWMutex
	validators   = map[string]ValidatorFunc{}
)

// Register adds a custom validator by name.
func Register(name string, fn ValidatorFunc) {
	name = strings.TrimSpace(name)
	if name == "" || fn == nil {
		return
	}
	validatorsMu.Lock()
	validators[name] = fn
	validatorsMu.Unlock()
}

func lookupValidator(name string) ValidatorFunc {
	validatorsMu.RLock()
	fn := validators[name]
	validatorsMu.RUnlock()
	return fn
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

		name := fieldName(field)
		rules := strings.Split(tag, ",")
		fieldValue := rv.Field(i)

		if fieldValue.Kind() == reflect.Pointer {
			if fieldValue.IsNil() {
				if hasRule(rules, "required") {
					errs = append(errs, FieldError{Field: name, Message: name + " is required"})
				}
				continue
			}
			fieldValue = fieldValue.Elem()
		}

		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if rule == "" {
				continue
			}
			nameRule, param := splitRule(rule)
			switch nameRule {
			case "required":
				if isZeroValue(fieldValue) {
					errs = append(errs, FieldError{Field: name, Message: name + " is required"})
				}
			case "email":
				if fieldValue.Kind() != reflect.String {
					continue
				}
				if value := fieldValue.String(); value != "" {
					if _, err := mail.ParseAddress(value); err != nil {
						errs = append(errs, FieldError{Field: name, Message: name + " must be a valid email"})
					}
				}
			case "min":
				if err := validateMin(name, fieldValue, param); err != nil {
					errs = append(errs, *err)
				}
			case "max":
				if err := validateMax(name, fieldValue, param); err != nil {
					errs = append(errs, *err)
				}
			default:
				if fn := lookupValidator(nameRule); fn != nil {
					if err := fn(name, fieldValue, param); err != nil {
						errs = append(errs, *err)
					}
				}
			}
		}
	}

	if len(errs) > 0 {
		err := apperr.Validation("validation failed", &Errors{Fields: errs})
		notifyHooks(value, err)
		return err
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

func splitRule(rule string) (string, string) {
	parts := strings.SplitN(rule, "=", 2)
	name := strings.TrimSpace(parts[0])
	param := ""
	if len(parts) > 1 {
		param = strings.TrimSpace(parts[1])
	}
	return name, param
}

func hasRule(rules []string, name string) bool {
	for _, rule := range rules {
		if strings.TrimSpace(rule) == name {
			return true
		}
		ruleName, _ := splitRule(rule)
		if ruleName == name {
			return true
		}
	}
	return false
}

func isZeroValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	return value.IsZero()
}

func validateMin(name string, value reflect.Value, param string) *FieldError {
	if param == "" {
		return &FieldError{Field: name, Message: name + " is invalid"}
	}
	switch value.Kind() {
	case reflect.String:
		min, err := strconv.Atoi(param)
		if err != nil {
			return &FieldError{Field: name, Message: name + " is invalid"}
		}
		if utf8.RuneCountInString(value.String()) < min {
			return &FieldError{Field: name, Message: name + " is too short"}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		min, err := strconv.ParseFloat(param, 64)
		if err != nil {
			return &FieldError{Field: name, Message: name + " is invalid"}
		}
		if current, ok := numericValue(value); ok && current < min {
			return &FieldError{Field: name, Message: name + " must be at least " + param}
		}
	case reflect.Slice, reflect.Array, reflect.Map:
		min, err := strconv.Atoi(param)
		if err != nil {
			return &FieldError{Field: name, Message: name + " is invalid"}
		}
		if value.Len() < min {
			return &FieldError{Field: name, Message: name + " is too short"}
		}
	}
	return nil
}

func validateMax(name string, value reflect.Value, param string) *FieldError {
	if param == "" {
		return &FieldError{Field: name, Message: name + " is invalid"}
	}
	switch value.Kind() {
	case reflect.String:
		max, err := strconv.Atoi(param)
		if err != nil {
			return &FieldError{Field: name, Message: name + " is invalid"}
		}
		if utf8.RuneCountInString(value.String()) > max {
			return &FieldError{Field: name, Message: name + " is too long"}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		max, err := strconv.ParseFloat(param, 64)
		if err != nil {
			return &FieldError{Field: name, Message: name + " is invalid"}
		}
		if current, ok := numericValue(value); ok && current > max {
			return &FieldError{Field: name, Message: name + " must be at most " + param}
		}
	case reflect.Slice, reflect.Array, reflect.Map:
		max, err := strconv.Atoi(param)
		if err != nil {
			return &FieldError{Field: name, Message: name + " is invalid"}
		}
		if value.Len() > max {
			return &FieldError{Field: name, Message: name + " is too long"}
		}
	}
	return nil
}

func numericValue(value reflect.Value) (float64, bool) {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(value.Uint()), true
	case reflect.Float32, reflect.Float64:
		return value.Float(), true
	default:
		return 0, false
	}
}
