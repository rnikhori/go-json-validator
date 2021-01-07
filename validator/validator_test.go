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
