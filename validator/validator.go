// Package validator implements a lightweight, schema-lite validator for
// JSON data that has already been unmarshaled into map[string]interface{}.
package validator

import (
	"fmt"
	"reflect"
	"regexp"
)

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

	// Min and Max constrain numeric values (inclusive). A nil pointer
	// means the bound is not enforced.
	Min *float64
	Max *float64

	// Pattern, when non-empty, is a regular expression that string
	// values must match in order to be valid.
	Pattern string

	// Enum, when non-empty, restricts a value to one of the given
	// options. Options should use the same Go types that
	// encoding/json produces (string, float64, bool, etc).
	Enum []interface{}

	// Properties, when Kind is KindObject, describes a sub-schema that
	// the field's value (a nested JSON object) must satisfy.
	Properties map[string]Rule

	// Items, when Kind is KindArray, describes the rule that every
	// element of the array must satisfy.
	Items *Rule

	// MinItems and MaxItems constrain the number of elements in an
	// array value. A nil pointer means the bound is not enforced.
	MinItems *int
	MaxItems *int

	// Message, when non-empty, overrides every default error message
	// produced for this field, regardless of which check failed.
	Message string

	// Messages overrides the error message for a specific failing
	// check on this field, taking precedence over Message. Valid keys
	// are: required, type, minLength, maxLength, min, max, pattern,
	// enum, minItems, maxItems.
	Messages map[string]string
}

const (
	msgKeyRequired = "required"
	msgKeyType     = "type"
	msgKeyMinLen   = "minLength"
	msgKeyMaxLen   = "maxLength"
	msgKeyMin      = "min"
	msgKeyMax      = "max"
	msgKeyPattern  = "pattern"
	msgKeyEnum     = "enum"
	msgKeyMinItems = "minItems"
	msgKeyMaxItems = "maxItems"
)

// ValidationError describes a single validation failure for a field.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Message)
}

// fieldError builds a ValidationError for path. The message is chosen in
// order of precedence: rule.Messages[key], then rule.Message, then def.
func fieldError(path string, rule Rule, key, def string) *ValidationError {
	if msg, ok := rule.Messages[key]; ok && msg != "" {
		return &ValidationError{Field: path, Message: msg}
	}
	if rule.Message != "" {
		return &ValidationError{Field: path, Message: rule.Message}
	}
	return &ValidationError{Field: path, Message: def}
}

// Validate checks data against schema and returns every validation error
// found. A nil or empty slice is returned when data satisfies schema.
func Validate(data map[string]interface{}, schema map[string]Rule) []error {
	return validateFields("", data, schema)
}

// validateFields validates every field in schema against data, prefixing
// resulting error field names with prefix (used for nested objects).
func validateFields(prefix string, data map[string]interface{}, schema map[string]Rule) []error {
	var errs []error

	for field, rule := range schema {
		path := joinPath(prefix, field)

		value, present := data[field]
		if !present {
			if rule.Required {
				errs = append(errs, fieldError(path, rule, msgKeyRequired, "is required but missing"))
			}
			continue
		}

		errs = append(errs, validateValue(path, value, rule)...)
	}

	return errs
}

// validateValue checks a single value against rule, returning every
// violation found at or below path.
func validateValue(path string, value interface{}, rule Rule) []error {
	var errs []error

	if !matchesKind(value, rule.Kind) {
		return append(errs, fieldError(path, rule, msgKeyType, fmt.Sprintf("expected type %s, got %s", rule.Kind, describeType(value))))
	}

	for _, check := range valueCheckers {
		if err := check(path, value, rule); err != nil {
			errs = append(errs, err)
		}
	}

	if rule.Kind == KindObject && rule.Properties != nil {
		obj := value.(map[string]interface{})
		errs = append(errs, validateFields(path, obj, rule.Properties)...)
	}

	if rule.Kind == KindArray {
		items := value.([]interface{})
		errs = append(errs, checkArrayBounds(path, items, rule)...)
		if rule.Items != nil {
			for i, item := range items {
				itemPath := fmt.Sprintf("%s[%d]", path, i)
				errs = append(errs, validateValue(itemPath, item, *rule.Items)...)
			}
		}
	}

	return errs
}

// checkArrayBounds enforces rule.MinItems and rule.MaxItems against an
// array value. It is a no-op when neither bound is configured.
func checkArrayBounds(path string, items []interface{}, rule Rule) []error {
	var errs []error

	if rule.MinItems != nil && len(items) < *rule.MinItems {
		errs = append(errs, fieldError(path, rule, msgKeyMinItems, fmt.Sprintf("must have at least %d items, got %d", *rule.MinItems, len(items))))
	}
	if rule.MaxItems != nil && len(items) > *rule.MaxItems {
		errs = append(errs, fieldError(path, rule, msgKeyMaxItems, fmt.Sprintf("must have at most %d items, got %d", *rule.MaxItems, len(items))))
	}

	return errs
}

// joinPath appends field to prefix using dot notation, omitting the
// separator when prefix is empty (top-level fields).
func joinPath(prefix, field string) string {
	if prefix == "" {
		return field
	}
	return prefix + "." + field
}

// valueCheckers lists the scalar constraint checks applied to a value
// after its Kind has been confirmed. Each checker is a no-op for rules it
// does not apply to.
var valueCheckers = []func(string, interface{}, Rule) error{
	checkStringLength,
	checkNumericRange,
	checkPattern,
	checkEnum,
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
		return fieldError(field, rule, msgKeyMinLen, fmt.Sprintf("must be at least %d characters long, got %d", *rule.MinLength, length))
	}
	if rule.MaxLength != nil && length > *rule.MaxLength {
		return fieldError(field, rule, msgKeyMaxLen, fmt.Sprintf("must be at most %d characters long, got %d", *rule.MaxLength, length))
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

// checkNumericRange enforces rule.Min and rule.Max against numeric values.
// It is a no-op for non-numeric rules.
func checkNumericRange(field string, value interface{}, rule Rule) error {
	if rule.Kind != KindNumber {
		return nil
	}

	n, ok := value.(float64)
	if !ok {
		return nil
	}

	if rule.Min != nil && n < *rule.Min {
		return fieldError(field, rule, msgKeyMin, fmt.Sprintf("must be at least %g, got %g", *rule.Min, n))
	}
	if rule.Max != nil && n > *rule.Max {
		return fieldError(field, rule, msgKeyMax, fmt.Sprintf("must be at most %g, got %g", *rule.Max, n))
	}
	return nil
}

// checkPattern enforces rule.Pattern against string values. It is a no-op
// for non-string rules or rules without a pattern configured.
func checkPattern(field string, value interface{}, rule Rule) error {
	if rule.Kind != KindString || rule.Pattern == "" {
		return nil
	}

	s, ok := value.(string)
	if !ok {
		return nil
	}

	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fieldError(field, rule, msgKeyPattern, fmt.Sprintf("has an invalid pattern configured: %v", err))
	}

	if !re.MatchString(s) {
		return fieldError(field, rule, msgKeyPattern, fmt.Sprintf("must match pattern %q", rule.Pattern))
	}
	return nil
}

// checkEnum enforces rule.Enum, requiring value to equal one of the
// configured options. It is a no-op when no options are configured.
func checkEnum(field string, value interface{}, rule Rule) error {
	if len(rule.Enum) == 0 {
		return nil
	}

	for _, option := range rule.Enum {
		if reflect.DeepEqual(value, option) {
			return nil
		}
	}

	return fieldError(field, rule, msgKeyEnum, fmt.Sprintf("must be one of %v", rule.Enum))
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
