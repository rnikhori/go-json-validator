// Package validator implements a lightweight, schema-lite validator for
// JSON data that has already been unmarshaled into map[string]interface{}.
package validator

import "fmt"

// Kind identifies the expected Go/JSON type of a field.
type Kind string

const (
	KindString Kind = "string"
	KindNumber Kind = "number"
	KindBool   Kind = "bool"
	KindArray  Kind = "array"
	KindObject Kind = "object"
)

// Rule describes the validation constraints for a single field.
type Rule struct {
	Kind     Kind
	Required bool

	// MinLength and MaxLength constrain the length (in runes) of string
	// values. A nil pointer means the bound is not enforced.
	MinLength *int
	MaxLength *int
}

// ValidationError describes a single validation failure for a field.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Message)
}

// Validate checks data against schema and returns every validation error
// found. A nil or empty slice is returned when data satisfies schema.
func Validate(data map[string]interface{}, schema map[string]Rule) []error {
	var errs []error

	for field, rule := range schema {
		value, present := data[field]
		if !present {
			if rule.Required {
				errs = append(errs, &ValidationError{
					Field:   field,
					Message: "is required but missing",
				})
			}
			continue
		}

		if !matchesKind(value, rule.Kind) {
			errs = append(errs, &ValidationError{
				Field:   field,
				Message: fmt.Sprintf("expected type %s, got %s", rule.Kind, describeType(value)),
			})
			continue
		}

		if err := checkStringLength(field, value, rule); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

// checkStringLength enforces rule.MinLength and rule.MaxLength against
// string values. It is a no-op for non-string rules.
func checkStringLength(field string, value interface{}, rule Rule) error {
	if rule.Kind != KindString {
		return nil
	}

	s, ok := value.(string)
	if !ok {
		return nil
	}
	length := len([]rune(s))

	if rule.MinLength != nil && length < *rule.MinLength {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at least %d characters long, got %d", *rule.MinLength, length),
		}
	}
	if rule.MaxLength != nil && length > *rule.MaxLength {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at most %d characters long, got %d", *rule.MaxLength, length),
		}
	}
	return nil
}

// matchesKind reports whether value's runtime type is consistent with the
// way encoding/json decodes JSON values into interface{}.
func matchesKind(value interface{}, kind Kind) bool {
	switch kind {
	case KindString:
		_, ok := value.(string)
		return ok
	case KindNumber:
		_, ok := value.(float64)
		return ok
	case KindBool:
		_, ok := value.(bool)
		return ok
	case KindArray:
		_, ok := value.([]interface{})
		return ok
	case KindObject:
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return false
	}
}

func describeType(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "bool"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", value)
	}
}
