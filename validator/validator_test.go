package validator

import (
	"encoding/json"
	"testing"
)

func userSchema() map[string]Rule {
	return map[string]Rule{
		"name":  {Kind: KindString, Required: true},
		"age":   {Kind: KindNumber, Required: true},
		"email": {Kind: KindString, Required: false},
	}
}

func decode(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("failed to unmarshal test payload: %v", err)
	}
	return data
}

func TestValidate_ValidPayload(t *testing.T) {
	data := decode(t, `{"name": "Ada Lovelace", "age": 36, "email": "ada@example.com"}`)

	errs := Validate(data, userSchema())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_ValidPayload_OptionalFieldMissing(t *testing.T) {
	data := decode(t, `{"name": "Grace Hopper", "age": 85}`)

	errs := Validate(data, userSchema())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_MissingRequiredField(t *testing.T) {
	data := decode(t, `{"age": 30}`)

	errs := Validate(data, userSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}

	ve, ok := errs[0].(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", errs[0])
	}
	if ve.Field != "name" {
		t.Fatalf("expected error for field %q, got %q", "name", ve.Field)
	}
}

func lengthSchema() map[string]Rule {
	minLen := 3
	maxLen := 10
	return map[string]Rule{
		"username": {Kind: KindString, Required: true, MinLength: &minLen, MaxLength: &maxLen},
	}
}

func TestValidate_StringLength_Valid(t *testing.T) {
	data := decode(t, `{"username": "alice"}`)

	errs := Validate(data, lengthSchema())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_StringLength_TooShort(t *testing.T) {
	data := decode(t, `{"username": "ab"}`)

	errs := Validate(data, lengthSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].(*ValidationError).Field != "username" {
		t.Fatalf("expected error for field %q, got %q", "username", errs[0].(*ValidationError).Field)
	}
}

func TestValidate_StringLength_TooLong(t *testing.T) {
	data := decode(t, `{"username": "waytoolongusername"}`)

	errs := Validate(data, lengthSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
}

func rangeSchema() map[string]Rule {
	min := 0.0
	max := 120.0
	return map[string]Rule{
		"age": {Kind: KindNumber, Required: true, Min: &min, Max: &max},
	}
}

func TestValidate_NumericRange_Valid(t *testing.T) {
	data := decode(t, `{"age": 42}`)

	errs := Validate(data, rangeSchema())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_NumericRange_TooLow(t *testing.T) {
	data := decode(t, `{"age": -5}`)

	errs := Validate(data, rangeSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
}

func TestValidate_NumericRange_TooHigh(t *testing.T) {
	data := decode(t, `{"age": 200}`)

	errs := Validate(data, rangeSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
}

func patternSchema() map[string]Rule {
	return map[string]Rule{
		"zip": {Kind: KindString, Required: true, Pattern: `^\d{5}$`},
	}
}

func TestValidate_Pattern_Valid(t *testing.T) {
	data := decode(t, `{"zip": "94107"}`)

	errs := Validate(data, patternSchema())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_Pattern_NoMatch(t *testing.T) {
	data := decode(t, `{"zip": "abc"}`)

	errs := Validate(data, patternSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
}

func TestValidate_Pattern_InvalidRegex(t *testing.T) {
	data := decode(t, `{"zip": "94107"}`)
	schema := map[string]Rule{
		"zip": {Kind: KindString, Required: true, Pattern: `(unterminated`},
	}

	errs := Validate(data, schema)
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error for invalid pattern, got %d: %v", len(errs), errs)
	}
}

func TestValidate_WrongType(t *testing.T) {
	data := decode(t, `{"name": "Linus Torvalds", "age": "thirty"}`)

	errs := Validate(data, userSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}

	ve, ok := errs[0].(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", errs[0])
	}
	if ve.Field != "age" {
		t.Fatalf("expected error for field %q, got %q", "age", ve.Field)
	}
}
