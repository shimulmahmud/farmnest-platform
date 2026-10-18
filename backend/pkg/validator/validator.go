package validator

import (
	"reflect"
	"strings"

	"github.com/ecommerce/backend/pkg/errors"
)

// Validator provides validation utilities for request payloads.
type Validator struct{}

// New creates a new Validator.
func New() *Validator {
	return &Validator{}
}

// Validate checks that required string fields are non-empty and applies custom rules.
func (v *Validator) Validate(payload interface{}, rules map[string]string) error {
	val := reflect.ValueOf(payload)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	for field, rule := range rules {
		fieldVal := val.FieldByName(field)
		if !fieldVal.IsValid() {
			continue
		}

		switch {
		case rule == "required":
			if isEmptyValue(fieldVal) {
				return errors.New("VALIDATION_ERROR", strings.ToLower(field)+" is required", 422)
			}
		case rule == "email":
			if s, ok := fieldVal.Interface().(string); ok && s != "" {
				if !isValidEmail(s) {
					return errors.New("VALIDATION_ERROR", strings.ToLower(field)+" must be a valid email", 422)
				}
			}
		case rule == "uuid":
			if s, ok := fieldVal.Interface().(string); ok && s != "" {
				if !isValidUUID(s) {
					return errors.New("VALIDATION_ERROR", strings.ToLower(field)+" must be a valid UUID", 422)
				}
			}
		}
	}
	return nil
}

// ValidateRequired checks that given string fields are non-blank.
func (v *Validator) ValidateRequired(fields map[string]string) error {
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			return errors.New("VALIDATION_ERROR", name+" is required", 422)
		}
	}
	return nil
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return strings.TrimSpace(v.String()) == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	default:
		return false
	}
}

func isValidEmail(email string) bool {
	at := strings.Index(email, "@")
	if at < 1 {
		return false
	}
	dot := strings.LastIndex(email[at:], ".")
	return dot > 1 && dot < len(email[at:])-1
}

func isValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !isHexDigit(c) {
				return false
			}
		}
	}
	return true
}

func isHexDigit(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
