---
name: design_patterns-typed_error_sentinels
description: >
  Use when creating error sentinels, defining package-specific error types,
  checking errors by type rather than value, or encountering Typed, IsTyped,
  MakeTypedSentinel, WrapWithType, or pkgErrDisamb in code.
triggers:
  - error sentinel
  - IsTyped
  - MakeTypedSentinel
  - WrapWithType
  - pkgErrDisamb
  - typed error
  - NewWithType
  - error type checking
---

# Typed Error Sentinels

## Overview

Dodder uses phantom type parameters to create error sentinels with compile-time
namespace isolation. Each package defines its own disambiguation type (an empty
struct), making it impossible for errors from different packages to accidentally
match even if they share the same message string.

## Core Types

```go
// alfa/errors/sentinels.go
type Typed[DISAMB any] interface {
    error
    GetErrorType() DISAMB
}
```

The `DISAMB` type parameter is a phantom type — it exists only at compile time
for type discrimination. It is never instantiated at runtime.

## Defining Package Errors

### Step 1: Declare the Disambiguation Type

```go
// mypackage/errors.go
type (
    pkgErrDisamb struct{}                    // phantom type — never instantiated
    pkgError     = errors.Typed[pkgErrDisamb] // convenience alias
)
```

### Step 2: Create Sentinels

```go
var (
    ErrSomethingFailed = errors.NewWithType[pkgErrDisamb]("something failed")
    ErrNotReady        = errors.NewWithType[pkgErrDisamb]("not ready")
)
```

Or use the factory that returns both sentinel and checker:

```go
var ErrBadInput, IsBadInput = errors.MakeTypedSentinel[pkgErrDisamb]("bad input")
```

### Step 3: Export a Checker

```go
func IsMyPackageError(err error) bool {
    return errors.IsTyped[pkgErrDisamb](err)
}
```

## Checking Errors

### By package type (any error from this package)

```go
if errors.IsTyped[pkgErrDisamb](err) {
    // error originated from this package
}
```

### By specific sentinel (standard Go pattern)

```go
switch err {
case ErrSomethingFailed:
    // handle specific case
case ErrNotReady:
    // handle specific case
}
```

## Wrapping Errors with Type

Preserve package typing when wrapping an existing error:

```go
func wrapAsPkgError(err error) pkgError {
    return errors.WrapWithType[pkgErrDisamb](err)
}
```

The wrapped error maintains `Unwrap()` support for standard error chain
traversal.

## Built-in Sentinels

Dodder provides common sentinels in `alfa/errors/`:

| Sentinel | Checker | Purpose |
|----------|---------|---------|
| `errStopIteration` | `IsStopIteration(err)` | Early termination of iteration |
| `ErrNotFound` | `errors.Is(err, ErrNotFound{})` | Value not found (carries `.Value` field) |

## How It Differs from Standard Go

| Standard Go | Typed Sentinels |
|-------------|-----------------|
| `var ErrFoo = errors.New("foo")` | `var ErrFoo = errors.NewWithType[myDisamb]("foo")` |
| `errors.Is(err, ErrFoo)` — value comparison | `errors.IsTyped[myDisamb](err)` — type comparison |
| Same message in two packages could collide | Phantom types provide compile-time isolation |
| No category checking | `IsTyped` checks any error from a package |

## Common Mistakes

| Mistake | Correct Approach |
|---------|-----------------|
| Reusing another package's `pkgErrDisamb` | Each package defines its own phantom type |
| Using `errors.Is` for typed category checks | Use `errors.IsTyped[T]` for type-based checking |
| Forgetting `Unwrap()` when wrapping | Use `WrapWithType` which provides `Unwrap()` automatically |
| Exporting `pkgErrDisamb` | Keep it unexported — export checker functions instead |
