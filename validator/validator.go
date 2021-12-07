// Package validator implements a lightweight, schema-lite validator for
// JSON data that has already been unmarshaled into map[string]interface{}.
package validator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// Kind identifies the expected Go/JSON type of a field.
type Kind string

// The supported Kind values, corresponding to the Go types produced by
// encoding/json when decoding into interface{}.
const (
	KindString Kind = "string"
	KindNumber Kind = "number"
	KindBool   Kind = "bool"
	KindArray  Kind = "array"
	KindObject Kind = "object"
)

// Rule describes the validation constraints for a single field.
type Rule struct {
	// Kind is the expected JSON type for the field's value.
	Kind Kind

	// Required marks a field as mandatory. A JSON null value is
	// treated the same as an absent field: it triggers the
	// required-field error when Required is true, and is otherwise
	// skipped without further checks.
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
	// Field is the dotted/bracketed path to the offending value, e.g.
	// "address.zip" or "orders[2].qty".
	Field string
	// Message is a human-readable description of why Field is invalid.
	Message string
}

// Error implements the error interface, formatting the failure as
// `field "<Field>": <Message>`.
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
	c := newCollector(false)
	validateFields(c, "", data, schema)
	return c.errs
}

// ValidateFirst checks data against schema and returns only the first
// validation error encountered, or nil when data satisfies schema.
// Validation stops as soon as one error is found, which can be cheaper
// than Validate for large or deeply nested schemas when callers only need
// to know whether data is valid, or want a single actionable error. Fields
// are checked in sorted order, so the result is deterministic.
func ValidateFirst(data map[string]interface{}, schema map[string]Rule) error {
	c := newCollector(true)
	validateFields(c, "", data, schema)
	if len(c.errs) > 0 {
		return c.errs[0]
	}
	return nil
}

// ValidateJSON unmarshals raw as a JSON object and validates it against
// schema in one call. JSON syntax errors (malformed input, or input that
// is not a JSON object) are returned as the second value and are always
// distinguishable from validation errors: when it is non-nil, the first
// return value is nil and no validation was attempted.
func ValidateJSON(raw []byte, schema map[string]Rule) ([]error, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("go-json-validator: invalid JSON: %w", err)
	}
	return Validate(data, schema), nil
}

// FormatErrors joins a slice of validation errors (as returned by Validate
// or ValidateJSON) into a single newline-separated human-readable string.
// It returns an empty string for an empty or nil slice.
func FormatErrors(errs []error) string {
	if len(errs) == 0 {
		return ""
	}

	parts := make([]string, len(errs))
	for i, err := range errs {
		parts[i] = err.Error()
	}
	return strings.Join(parts, "\n")
}

// ValidateJSONString is a convenience wrapper around ValidateJSON for
// callers holding the JSON payload as a string rather than a []byte.
func ValidateJSONString(raw string, schema map[string]Rule) ([]error, error) {
	return ValidateJSON([]byte(raw), schema)
}

// collector accumulates validation errors and, when stopFirst is set,
// signals the recursive walk to stop as soon as one error is recorded.
type collector struct {
	errs      []error
	stopFirst bool
}

func newCollector(stopFirst bool) *collector {
	return &collector{stopFirst: stopFirst}
}

func (c *collector) add(err error) {
	c.errs = append(c.errs, err)
}

// done reports whether the walk should stop early.
func (c *collector) done() bool {
	return c.stopFirst && len(c.errs) > 0
}

// validateFields validates every field in schema against data, prefixing
// resulting error field names with prefix (used for nested objects).
// Fields are visited in sorted order so the returned errors (and the
// error ValidateFirst picks) are deterministic.
func validateFields(c *collector, prefix string, data map[string]interface{}, schema map[string]Rule) {
	for _, field := range sortedKeys(schema) {
		if c.done() {
			return
		}

		rule := schema[field]
		path := joinPath(prefix, field)

		value, present := data[field]
		if !present || value == nil {
			if rule.Required {
				c.add(fieldError(path, rule, msgKeyRequired, "is required but missing"))
			}
			continue
		}

		validateValue(c, path, value, rule)
	}
}

// validateValue checks a single value against rule, recording every
// violation found at or below path into c.
func validateValue(c *collector, path string, value interface{}, rule Rule) {
	if !matchesKind(value, rule.Kind) {
		c.add(fieldError(path, rule, msgKeyType, fmt.Sprintf("expected type %s, got %s", rule.Kind, describeType(value))))
		return
	}

	for _, check := range valueCheckers {
		if c.done() {
			return
		}
		if err := check(path, value, rule); err != nil {
			c.add(err)
		}
	}

	if c.done() {
		return
	}

	if rule.Kind == KindObject && rule.Properties != nil {
		obj := value.(map[string]interface{})
		validateFields(c, path, obj, rule.Properties)
	}

	if rule.Kind == KindArray {
		items := value.([]interface{})
		checkArrayBounds(c, path, items, rule)
		if rule.Items != nil {
			for i, item := range items {
				if c.done() {
					return
				}
				itemPath := fmt.Sprintf("%s[%d]", path, i)
				validateValue(c, itemPath, item, *rule.Items)
			}
		}
	}
}

// checkArrayBounds enforces rule.MinItems and rule.MaxItems against an
// array value, recording any violation into c. It is a no-op when neither
// bound is configured.
func checkArrayBounds(c *collector, path string, items []interface{}, rule Rule) {
	if rule.MinItems != nil && len(items) < *rule.MinItems {
		c.add(fieldError(path, rule, msgKeyMinItems, fmt.Sprintf("must have at least %d items, got %d", *rule.MinItems, len(items))))
	}
	if c.done() {
		return
	}
	if rule.MaxItems != nil && len(items) > *rule.MaxItems {
		c.add(fieldError(path, rule, msgKeyMaxItems, fmt.Sprintf("must have at most %d items, got %d", *rule.MaxItems, len(items))))
	}
}

// sortedKeys returns the keys of schema in ascending order.
func sortedKeys(schema map[string]Rule) []string {
	keys := make([]string, 0, len(schema))
	for field := range schema {
		keys = append(keys, field)
	}
	sort.Strings(keys)
	return keys
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
