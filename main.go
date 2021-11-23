package main

import (
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
}
