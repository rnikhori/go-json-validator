package main

import (
	"encoding/json"
	"fmt"

	"github.com/rnikhori/go-json-validator/validator"
)

func userSchema() map[string]validator.Rule {
	return map[string]validator.Rule{
		"name":  {Kind: validator.KindString, Required: true},
		"age":   {Kind: validator.KindNumber, Required: true},
		"email": {Kind: validator.KindString, Required: false},
	}
}

func orderSchema() map[string]validator.Rule {
	minItems := 1

	return map[string]validator.Rule{
		"id": {Kind: validator.KindString, Required: true},
		"customer": {
			Kind:     validator.KindObject,
			Required: true,
			Properties: map[string]validator.Rule{
				"name": {Kind: validator.KindString, Required: true},
				"address": {
					Kind:     validator.KindObject,
					Required: true,
					Properties: map[string]validator.Rule{
						"city": {Kind: validator.KindString, Required: true},
						"zip":  {Kind: validator.KindString, Required: true, Pattern: `^\d{5}$`},
					},
				},
			},
		},
		"items": {
			Kind:     validator.KindArray,
			Required: true,
			MinItems: &minItems,
			Items: &validator.Rule{
				Kind: validator.KindObject,
				Properties: map[string]validator.Rule{
					"sku": {Kind: validator.KindString, Required: true},
					"qty": {Kind: validator.KindNumber, Required: true},
				},
			},
		},
	}
}

func registrationSchema() map[string]validator.Rule {
	minUsername := 3
	maxUsername := 20
	minAge := 13.0

	return map[string]validator.Rule{
		"username": {
			Kind:      validator.KindString,
			Required:  true,
			MinLength: &minUsername,
			MaxLength: &maxUsername,
		},
		"age": {
			Kind:     validator.KindNumber,
			Required: true,
			Min:      &minAge,
			Message:  "you must be at least 13 years old to register",
		},
		"plan": {
			Kind:     validator.KindString,
			Required: true,
			Enum:     []interface{}{"free", "pro", "enterprise"},
		},
	}
}

type demo struct {
	label  string
	schema map[string]validator.Rule
	raw    string
}

func demos() []demo {
	return []demo{
		{
			"valid user",
			userSchema(),
			`{"name": "Ada Lovelace", "age": 36, "email": "ada@example.com"}`,
		},
		{
			"invalid user",
			userSchema(),
			`{"age": "thirty-six", "email": "ada@example.com"}`,
		},
		{
			"valid order with nested customer and items",
			orderSchema(),
			`{
				"id": "o-1001",
				"customer": {"name": "Ada Lovelace", "address": {"city": "London", "zip": "94107"}},
				"items": [{"sku": "sku-1", "qty": 2}, {"sku": "sku-2", "qty": 1}]
			}`,
		},
		{
			"invalid order: bad zip and missing item quantity",
			orderSchema(),
			`{
				"id": "o-1002",
				"customer": {"name": "Grace Hopper", "address": {"city": "Arlington", "zip": "abc"}},
				"items": [{"sku": "sku-9"}]
			}`,
		},
		{
			"valid registration",
			registrationSchema(),
			`{"username": "ada", "age": 36, "plan": "pro"}`,
		},
		{
			"invalid registration: short username, underage, unknown plan",
			registrationSchema(),
			`{"username": "ab", "age": 10, "plan": "ultimate"}`,
		},
	}
}

func main() {
	for _, d := range demos() {
		fmt.Printf("--- %s ---\n", d.label)

		errs, jsonErr := validator.ValidateJSON([]byte(d.raw), d.schema)
		if jsonErr != nil {
			fmt.Printf("failed to parse JSON: %v\n\n", jsonErr)
			continue
		}

		if len(errs) == 0 {
			fmt.Println("valid")
		} else {
			fmt.Println(validator.FormatErrors(errs))
		}
		fmt.Println()
	}

	fmt.Println("--- ValidateFirst: stop at the first error ---")
	invalidRegistration := mustDecode(`{"username": "ab", "age": 10, "plan": "ultimate"}`)
	if err := validator.ValidateFirst(invalidRegistration, registrationSchema()); err != nil {
		fmt.Println(err)
	}
	fmt.Println()

	fmt.Println("--- ValidateJSONString: syntax error surfaced separately ---")
	if _, err := validator.ValidateJSONString(`{"username": "ada",`, registrationSchema()); err != nil {
		fmt.Println("invalid JSON:", err)
	}
}

func mustDecode(raw string) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		panic(err)
	}
	return data
}
