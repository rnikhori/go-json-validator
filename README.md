# go-json-validator

A lightweight, schema-lite JSON validator built on the Go standard library — checks required fields and basic types on `map[string]interface{}` data.

## Usage

```go
schema := map[string]validator.Rule{
	"name":  {Kind: validator.KindString, Required: true},
	"age":   {Kind: validator.KindNumber, Required: true},
	"email": {Kind: validator.KindString, Required: false},
}

var data map[string]interface{}
json.Unmarshal([]byte(`{"name": "Ada", "age": 36}`), &data)

if errs := validator.Validate(data, schema); len(errs) > 0 {
	for _, err := range errs {
		fmt.Println(err)
	}
}
```

## String length rules

Use `MinLength` and `MaxLength` to constrain the number of runes in a string field:

```go
minLen := 3
maxLen := 20

schema := map[string]validator.Rule{
	"username": {Kind: validator.KindString, Required: true, MinLength: &minLen, MaxLength: &maxLen},
}
```

## Numeric range rules

Use `Min` and `Max` to constrain numeric fields (inclusive bounds):

```go
min := 0.0
max := 120.0

schema := map[string]validator.Rule{
	"age": {Kind: validator.KindNumber, Required: true, Min: &min, Max: &max},
}
```

## Pattern rule

Use `Pattern` to require a string field to match a regular expression (as understood by Go's `regexp` package):

```go
schema := map[string]validator.Rule{
	"zip": {Kind: validator.KindString, Required: true, Pattern: `^\d{5}$`},
}
```

An invalid regular expression produces a validation error on the field rather than panicking.

## Enum rule

Use `Enum` to restrict a value to a fixed set of options:

```go
schema := map[string]validator.Rule{
	"status": {Kind: validator.KindString, Required: true, Enum: []interface{}{"active", "inactive", "pending"}},
}
```

## Nested objects

Give a `KindObject` rule a `Properties` sub-schema to validate nested JSON objects. Errors from nested fields use dotted paths, e.g. `address.city`:

```go
schema := map[string]validator.Rule{
	"name": {Kind: validator.KindString, Required: true},
	"address": {
		Kind:     validator.KindObject,
		Required: true,
		Properties: map[string]validator.Rule{
			"city": {Kind: validator.KindString, Required: true},
			"zip":  {Kind: validator.KindString, Required: false},
		},
	},
}
```

## Arrays

Give a `KindArray` rule an `Items` rule to validate every element, and optional `MinItems`/`MaxItems` to constrain the element count. `Items` can itself be a nested object rule, producing paths like `orders[1].id`:

```go
minItems := 1
maxItems := 10

schema := map[string]validator.Rule{
	"tags": {
		Kind:     validator.KindArray,
		Required: true,
		Items:    &validator.Rule{Kind: validator.KindString, MinLength: intPtr(1)},
		MinItems: &minItems,
		MaxItems: &maxItems,
	},
	"orders": {
		Kind: validator.KindArray,
		Items: &validator.Rule{
			Kind: validator.KindObject,
			Properties: map[string]validator.Rule{
				"id":  {Kind: validator.KindString, Required: true},
				"qty": {Kind: validator.KindNumber, Required: true},
			},
		},
	},
}
```

## Custom error messages

Set `Message` on a `Rule` to replace every default error text generated for that field (required, type, and every constraint) with your own wording:

```go
schema := map[string]validator.Rule{
	"age": {Kind: validator.KindNumber, Required: true, Message: "age must be a whole number"},
}
```

Use `Messages` instead to override the message for one specific check, falling back to `Message` and then the default for any check not listed. Valid keys are `required`, `type`, `minLength`, `maxLength`, `min`, `max`, `pattern`, `enum`, `minItems`, and `maxItems`:

```go
schema := map[string]validator.Rule{
	"age": {
		Kind:     validator.KindNumber,
		Required: true,
		Min:      &min,
		Messages: map[string]string{
			"min":      "age cannot be negative",
			"required": "age must be provided",
		},
	},
}
```

## Stopping at the first error

`Validate` always collects every violation. When you only need to know whether data is valid, or want a single actionable error, use `ValidateFirst` instead — it stops as soon as one violation is found and returns it directly (or `nil` when data is valid):

```go
if err := validator.ValidateFirst(data, schema); err != nil {
	fmt.Println(err)
}
```

Fields are always checked in sorted order, so both functions produce deterministic results.

## Validating JSON directly

`ValidateJSON` and `ValidateJSONString` unmarshal and validate in one call, so you don't need a separate `json.Unmarshal` step. A JSON syntax error (or a payload that isn't a JSON object) is returned as a distinct second value, never mixed in with validation errors:

```go
errs, err := validator.ValidateJSON([]byte(`{"name": "Ada", "age": 36}`), schema)
if err != nil {
	// malformed JSON, not a validation failure
	fmt.Println("invalid JSON:", err)
} else if len(errs) > 0 {
	for _, e := range errs {
		fmt.Println(e)
	}
}

// Or, from a string:
errs, err = validator.ValidateJSONString(`{"name": "Ada", "age": 36}`, schema)
```

Run the demo:

```sh
go run .
```

## Running tests

```sh
go test ./...
```
