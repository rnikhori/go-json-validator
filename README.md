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

Run the demo:

```sh
go run .
```

## Running tests

```sh
go test ./...
```
