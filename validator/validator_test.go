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

func enumSchema() map[string]Rule {
	return map[string]Rule{
		"status": {Kind: KindString, Required: true, Enum: []interface{}{"active", "inactive", "pending"}},
		"level":  {Kind: KindNumber, Required: true, Enum: []interface{}{1.0, 2.0, 3.0}},
	}
}

func TestValidate_Enum_Valid(t *testing.T) {
	data := decode(t, `{"status": "active", "level": 2}`)

	errs := Validate(data, enumSchema())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_Enum_InvalidString(t *testing.T) {
	data := decode(t, `{"status": "archived", "level": 2}`)

	errs := Validate(data, enumSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].(*ValidationError).Field != "status" {
		t.Fatalf("expected error for field %q, got %q", "status", errs[0].(*ValidationError).Field)
	}
}

func TestValidate_Enum_InvalidNumber(t *testing.T) {
	data := decode(t, `{"status": "active", "level": 9}`)

	errs := Validate(data, enumSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].(*ValidationError).Field != "level" {
		t.Fatalf("expected error for field %q, got %q", "level", errs[0].(*ValidationError).Field)
	}
}

func addressSchema() map[string]Rule {
	return map[string]Rule{
		"name": {Kind: KindString, Required: true},
		"address": {
			Kind:     KindObject,
			Required: true,
			Properties: map[string]Rule{
				"city": {Kind: KindString, Required: true},
				"zip":  {Kind: KindString, Required: false},
			},
		},
	}
}

func TestValidate_NestedObject_Valid(t *testing.T) {
	data := decode(t, `{"name": "Ada", "address": {"city": "London", "zip": "W1"}}`)

	errs := Validate(data, addressSchema())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_NestedObject_MissingRequiredNestedField(t *testing.T) {
	data := decode(t, `{"name": "Ada", "address": {"zip": "W1"}}`)

	errs := Validate(data, addressSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].(*ValidationError).Field != "address.city" {
		t.Fatalf("expected error for field %q, got %q", "address.city", errs[0].(*ValidationError).Field)
	}
}

func TestValidate_NestedObject_WrongTypeNestedField(t *testing.T) {
	data := decode(t, `{"name": "Ada", "address": {"city": 42}}`)

	errs := Validate(data, addressSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].(*ValidationError).Field != "address.city" {
		t.Fatalf("expected error for field %q, got %q", "address.city", errs[0].(*ValidationError).Field)
	}
}

func TestValidate_NestedObject_WrongTopLevelType(t *testing.T) {
	data := decode(t, `{"name": "Ada", "address": "not an object"}`)

	errs := Validate(data, addressSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].(*ValidationError).Field != "address" {
		t.Fatalf("expected error for field %q, got %q", "address", errs[0].(*ValidationError).Field)
	}
}

func tagsSchema() map[string]Rule {
	minItems := 1
	maxItems := 3
	return map[string]Rule{
		"tags": {
			Kind:     KindArray,
			Required: true,
			Items:    &Rule{Kind: KindString, MinLength: intPtr(1)},
			MinItems: &minItems,
			MaxItems: &maxItems,
		},
	}
}

func intPtr(v int) *int {
	return &v
}

func TestValidate_Array_Valid(t *testing.T) {
	data := decode(t, `{"tags": ["go", "json"]}`)

	errs := Validate(data, tagsSchema())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_Array_WrongElementType(t *testing.T) {
	data := decode(t, `{"tags": ["go", 42]}`)

	errs := Validate(data, tagsSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].(*ValidationError).Field != "tags[1]" {
		t.Fatalf("expected error for field %q, got %q", "tags[1]", errs[0].(*ValidationError).Field)
	}
}

func TestValidate_Array_TooFewItems(t *testing.T) {
	data := decode(t, `{"tags": []}`)

	errs := Validate(data, tagsSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
}

func TestValidate_Array_TooManyItems(t *testing.T) {
	data := decode(t, `{"tags": ["a", "b", "c", "d"]}`)

	errs := Validate(data, tagsSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
}

func TestValidate_Array_ElementConstraintViolation(t *testing.T) {
	data := decode(t, `{"tags": [""]}`)

	errs := Validate(data, tagsSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].(*ValidationError).Field != "tags[0]" {
		t.Fatalf("expected error for field %q, got %q", "tags[0]", errs[0].(*ValidationError).Field)
	}
}

func ordersSchema() map[string]Rule {
	return map[string]Rule{
		"orders": {
			Kind:     KindArray,
			Required: true,
			Items: &Rule{
				Kind: KindObject,
				Properties: map[string]Rule{
					"id":  {Kind: KindString, Required: true},
					"qty": {Kind: KindNumber, Required: true},
				},
			},
		},
	}
}

func TestValidate_ArrayOfObjects_Valid(t *testing.T) {
	data := decode(t, `{"orders": [{"id": "o1", "qty": 2}, {"id": "o2", "qty": 1}]}`)

	errs := Validate(data, ordersSchema())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_ArrayOfObjects_MissingNestedField(t *testing.T) {
	data := decode(t, `{"orders": [{"id": "o1", "qty": 2}, {"qty": 1}]}`)

	errs := Validate(data, ordersSchema())
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].(*ValidationError).Field != "orders[1].id" {
		t.Fatalf("expected error for field %q, got %q", "orders[1].id", errs[0].(*ValidationError).Field)
	}
}

func TestValidate_CustomMessage_Override(t *testing.T) {
	data := decode(t, `{"age": "not a number"}`)
	schema := map[string]Rule{
		"age": {Kind: KindNumber, Required: true, Message: "age must be a whole number"},
	}

	errs := Validate(data, schema)
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	ve := errs[0].(*ValidationError)
	if ve.Message != "age must be a whole number" {
		t.Fatalf("expected overridden message, got %q", ve.Message)
	}
}

func TestValidate_CustomMessage_AppliesToRequiredCheck(t *testing.T) {
	data := decode(t, `{}`)
	schema := map[string]Rule{
		"age": {Kind: KindNumber, Required: true, Message: "age is mandatory"},
	}

	errs := Validate(data, schema)
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].(*ValidationError).Message != "age is mandatory" {
		t.Fatalf("expected overridden message, got %q", errs[0].(*ValidationError).Message)
	}
}

func TestValidate_Messages_PerCheckOverride(t *testing.T) {
	min := 0.0
	schema := map[string]Rule{
		"age": {
			Kind:     KindNumber,
			Required: true,
			Min:      &min,
			Message:  "generic age problem",
			Messages: map[string]string{
				msgKeyMin: "age cannot be negative",
			},
		},
	}

	errs := Validate(decode(t, `{"age": -1}`), schema)
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].(*ValidationError).Message != "age cannot be negative" {
		t.Fatalf("expected per-check message, got %q", errs[0].(*ValidationError).Message)
	}

	errs = Validate(decode(t, `{}`), schema)
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].(*ValidationError).Message != "generic age problem" {
		t.Fatalf("expected fallback to Message for unconfigured check, got %q", errs[0].(*ValidationError).Message)
	}
}

func TestValidate_DeterministicErrorOrder(t *testing.T) {
	data := decode(t, `{}`)
	schema := map[string]Rule{
		"zeta":  {Kind: KindString, Required: true},
		"alpha": {Kind: KindString, Required: true},
		"mu":    {Kind: KindString, Required: true},
	}

	want := []string{"alpha", "mu", "zeta"}
	for i := 0; i < 20; i++ {
		errs := Validate(data, schema)
		if len(errs) != len(want) {
			t.Fatalf("expected %d errors, got %d: %v", len(want), len(errs), errs)
		}
		for i, err := range errs {
			if err.(*ValidationError).Field != want[i] {
				t.Fatalf("run %d: expected order %v, got field %q at index %d", i, want, err.(*ValidationError).Field, i)
			}
		}
	}
}

func TestValidateFirst_ValidPayload(t *testing.T) {
	data := decode(t, `{"name": "Ada Lovelace", "age": 36}`)

	if err := ValidateFirst(data, userSchema()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateFirst_ReturnsAlphabeticallyFirstError(t *testing.T) {
	data := decode(t, `{}`)
	schema := map[string]Rule{
		"zeta":  {Kind: KindString, Required: true},
		"alpha": {Kind: KindString, Required: true},
	}

	err := ValidateFirst(data, schema)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Field != "alpha" {
		t.Fatalf("expected first error for field %q, got %q", "alpha", ve.Field)
	}
}

func TestValidateFirst_StopsBeforeNestedValidation(t *testing.T) {
	data := decode(t, `{"address": "not an object"}`)

	err := ValidateFirst(data, addressSchema())
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	ve := err.(*ValidationError)
	if ve.Field != "address" {
		t.Fatalf("expected error for field %q, got %q", "address", ve.Field)
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
