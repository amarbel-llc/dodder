# SeqError Static Analyzer Design

## Problem

`interfaces.SeqError[T]` (alias for `iter.Seq2[T, error]`) is used throughout
the codebase for fallible iteration. The error value in each iteration can be
silently dropped by assigning it to `_` or by naming it but never checking it.
This analyzer prevents that class of bug at compile time.

## Target Type

Any `for key, val := range expr` where `expr` has type matching
`func(func(T, error) bool)` — i.e., `iter.Seq2[T, error]` regardless of
aliasing (`interfaces.SeqError`, `sku.Seq`, or direct usage).

Detection: check the underlying type of the range expression. It must be a
function type whose single parameter is itself a function with two parameters
where the second is the `error` interface.

## Detection Rules

### Rule 1 — Blank error variable

The range statement's `Value` is `*ast.Ident` with `Name == "_"`. Report unless
`//seq:err-checked` appears on the same line.

### Rule 2 — Unchecked error variable

The error variable is named but the loop body contains no qualifying usage. A
qualifying usage is either:

- **(a)** An `if` statement whose condition references the error variable (an
  `err != nil` check or equivalent), AND whose body contains at least one
  scope-exiting or propagating statement: `return`, `break`, `continue`,
  `panic(...)`, or a function call passing the error variable.

- **(b)** A function call that passes the error variable as an argument (yield
  pass-through pattern like `yield(elem, err)`).

If neither (a) nor (b) is found, report.

### Rule 3 (opt-in) — Unwrapped error

With `-require-wrap` flag, inside the qualifying `if err != nil` body, the error
must be passed through a call to `errors.Wrap` or `errors.Wrapf` before being
returned/yielded. Pass-through yield is exempt from this rule.

## Diagnostics

| Pattern                        | Message                                                                                         |
| ------------------------------ | ----------------------------------------------------------------------------------------------- |
| `for x, _ := range seq`       | `error from iter.Seq2 range is discarded; must be checked (or add //seq:err-checked)`           |
| Named but unchecked            | `error variable %q from iter.Seq2 range is never checked or propagated`                         |
| Checked but not used           | `error variable %q is checked but not handled; if-body must return, break, continue, or propagate the error` |
| `-require-wrap` violation      | `error variable %q should be wrapped with errors.Wrap before propagation`                       |

## Suppression

`//seq:err-checked` on the `for` line suppresses all diagnostics for that range
statement.

## Location

`go/lib/alfa/analyzers/seqerror/` — same structure as repool:

- `analyzer.go` — the analysis pass
- `cmd/main.go` — `singlechecker.Main(Analyzer)`
- `analyzer_test.go` — `analysistest.Run`
- `testdata/src/a/a.go` — annotated test cases

## Integration

Justfile recipes:

```makefile
build-analyzer-seqerror:
  go build -o build/seqerror-analyzer ./src/alfa/analyzers/seqerror/cmd/

check-go-seqerror: build-analyzer-seqerror
  go vet -vettool=build/seqerror-analyzer ./... || true

check: check-go-vuln check-go-vet check-go-repool check-go-seqerror
```

## Test Cases

```go
// Blank error — flagged
for x, _ := range seqError { ... }

// Blank with suppression — OK
for x, _ := range seqError { ... } //seq:err-checked

// Named and checked with return — OK
for x, err := range seqError {
    if err != nil { return err }
}

// Named and checked with continue — OK
for x, err := range seqError {
    if err != nil { continue }
}

// Named and checked with yield — OK
for x, err := range seqError {
    if err != nil { yield(nil, err); return }
}

// Yield pass-through (no nil check) — OK
for x, err := range seqError {
    if !yield(x, err) { return }
}

// Named but never checked — flagged
for x, err := range seqError {
    _ = err
    process(x)
}

// Checked but empty body — flagged
for x, err := range seqError {
    if err != nil {}
}
```
