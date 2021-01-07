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

func main() {
	schema := userSchema()

	payloads := map[string]string{
		"valid user":   `{"name": "Ada Lovelace", "age": 36, "email": "ada@example.com"}`,
		"invalid user": `{"age": "thirty-six", "email": "ada@example.com"}`,
	}

	for label, raw := range payloads {
		fmt.Printf("--- %s ---\n", label)

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			fmt.Printf("failed to parse JSON: %v\n\n", err)
			continue
		}

		errs := validator.Validate(data, schema)
		if len(errs) == 0 {
			fmt.Println("valid")
		} else {
			for _, err := range errs {
				fmt.Println(err)
			}
		}
		fmt.Println()
	}
}
