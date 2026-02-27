# string_builder_joined

String builder that automatically inserts a join string between writes.

## Key Types

- `builder`: Wraps strings.Builder with automatic delimiter insertion

## Usage

```go
b := Make(", ")
b.WriteString("a")
b.WriteString("b")
b.String() // "a, b"
```
