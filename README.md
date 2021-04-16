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

Run the demo:

```sh
go run .
```

## Running tests

```sh
go test ./...
```
