# Sentinel Error Pattern Guide

This guide documents the standard pattern for creating sentinel errors in Dodder using typed generics.

## Why Typed Generics?

The typed generic pattern provides:
- **Compile-time type safety** - Catch errors at compile time, not runtime
- **Zero runtime allocation** - Type checks are compile-time only
- **Consistent API** - Same pattern across all packages
- **Works with `errors.Is()`** - Standard Go error checking

## Standard Pattern

### Basic Package Error

```go
package mypackage

import "code.linenisgreat.com/dodder/go/src/alfa/errors"

// Step 1: Define package-scoped disambiguator
type pkgErrDisamb struct{}

// Step 2: Create type alias for package errors
type pkgError = errors.Typed[pkgErrDisamb]

// Step 3: Constructor for new sentinels
func newPkgError(text string) pkgError {
    return errors.NewWithType[pkgErrDisamb](text)
}

// Step 4: Checker function
func IsPkgError(err error) bool {
    return errors.IsTyped[pkgErrDisamb](err)
}

// Step 5: Define sentinels
var (
    ErrMyError    = newPkgError("my error")
    ErrOtherError = newPkgError("other error")
)
```

### Using the Helper

To reduce boilerplate, use `MakeTypedSentinel()`:

```go
package mypackage

import "code.linenisgreat.com/dodder/go/src/alfa/errors"

type pkgErrDisamb struct{}

var (
    // Creates both the sentinel and checker function
    ErrMyError, IsMyError = errors.MakeTypedSentinel[pkgErrDisamb]("my error")
    ErrOtherError, IsOtherError = errors.MakeTypedSentinel[pkgErrDisamb]("other error")
)
```

## Rich Error Types

For errors that need to carry data, create a custom struct:

```go
type myErrorDisamb struct{}

type MyError struct {
    Value string
    Code  int
}

func (err MyError) Error() string {
    return fmt.Sprintf("my error: %s (code %d)", err.Value, err.Code)
}

func (err MyError) GetErrorType() myErrorDisamb {
    return myErrorDisamb{}
}

// Constructor
func MakeMyError(value string, code int) error {
    return MyError{Value: value, Code: code}
}

// Checker
func IsMyError(err error) bool {
    return errors.IsTyped[myErrorDisamb](err)
}

// Accessor
func GetMyError(err error) (MyError, bool) {
    var myErr MyError
    if errors.As(err, &myErr) {
        return myErr, true
    }
    return MyError{}, false
}
```

## Usage Examples

### Returning Sentinel Errors

```go
func doSomething(id string) error {
    if id == "" {
        return ErrMyError  // Return sentinel directly
    }

    // For rich errors, use constructor
    if invalid(id) {
        return MakeMyError(id, 400)
    }

    return nil
}
```

### Checking Sentinel Errors

```go
func caller() {
    err := doSomething("foo")

    // Check with package function
    if IsMyError(err) {
        // Handle my error
    }

    // Or use errors.Is()
    if errors.Is(err, ErrMyError) {
        // Same result
    }

    // For rich errors, extract data
    if myErr, ok := GetMyError(err); ok {
        fmt.Printf("Error with value: %s, code: %d\n", myErr.Value, myErr.Code)
    }
}
```

### Wrapping Sentinel Errors

```go
// DON'T wrap sentinels - they lose their identity
if notFound {
    return errors.Wrapf(ErrNotFound, "context")  // ❌ BAD - loses sentinel
}

// DO return sentinels directly
if notFound {
    return ErrNotFound  // ✅ GOOD
}

// For rich errors, return directly
if invalid {
    return MakeMyError(value, code)  // ✅ GOOD
}

// If you need to wrap, use WrapWithType
if needsWrapping {
    underlying := doSomething()
    return errors.WrapWithType[pkgErrDisamb](underlying)  // ✅ Preserves type
}
```

## Migration from Old Patterns

### Pattern 1: Custom Is() Method

**Old:**
```go
type ErrNotFound string

func (err ErrNotFound) Error() string {
    return fmt.Sprintf("not found: %q", string(err))
}

func (err ErrNotFound) Is(target error) bool {
    _, ok := target.(ErrNotFound)
    return ok  // Type check only - ignores value!
}

func IsErrNotFound(err error) bool {
    return errors.Is(err, ErrNotFound(""))
}
```

**Problem:** The `Is()` method only checks types, so `ErrNotFound("foo")` matches `ErrNotFound("bar")`

**New:**
```go
type errNotFoundDisamb struct{}

type ErrNotFoundTyped struct {
    Value string
}

func (err ErrNotFoundTyped) Error() string {
    if err.Value == "" {
        return "not found"
    }
    return fmt.Sprintf("not found: %q", err.Value)
}

func (err ErrNotFoundTyped) GetErrorType() errNotFoundDisamb {
    return errNotFoundDisamb{}
}

func MakeErrNotFound(value string) error {
    return ErrNotFoundTyped{Value: value}
}

func IsErrNotFound(err error) bool {
    return errors.IsTyped[errNotFoundDisamb](err)
}

// Deprecated: use MakeErrNotFound
func ErrNotFound(value string) ErrNotFoundTyped {
    return ErrNotFoundTyped{Value: value}
}
```

**Migration:** Old code continues to work, gradually migrate to `MakeErrNotFound()`

### Pattern 2: Plain errors.New()

**Old:**
```go
var errStopIteration = errors.New("stop iteration")

func IsStopIteration(err error) bool {
    return errors.Is(err, errStopIteration)
}
```

**New:**
```go
type stopIterationDisamb struct{}

var errStopIteration = errors.NewWithType[stopIterationDisamb]("stop iteration")

func MakeErrStopIteration() error {
    return errStopIteration
}

func IsStopIteration(err error) bool {
    return errors.IsTyped[stopIterationDisamb](err)
}
```

**Benefits:** Compile-time type safety, same runtime behavior

## Common Patterns

### Pattern: Multiple Related Errors

```go
type authErrorDisamb struct{}

var (
    ErrInvalidCredentials, IsInvalidCredentials = errors.MakeTypedSentinel[authErrorDisamb]("invalid credentials")
    ErrExpiredToken, IsExpiredToken = errors.MakeTypedSentinel[authErrorDisamb]("expired token")
    ErrInsufficientPerms, IsInsufficientPerms = errors.MakeTypedSentinel[authErrorDisamb]("insufficient permissions")
)

// Check if any auth error
func IsAuthError(err error) bool {
    return errors.IsTyped[authErrorDisamb](err)
}
```

### Pattern: Error Hierarchy

```go
// Base error type
type ioErrorDisamb struct{}

var (
    ErrIOError, IsIOError = errors.MakeTypedSentinel[ioErrorDisamb]("I/O error")
)

// Specific I/O errors
type fileNotFoundDisamb struct{}

type FileNotFoundError struct {
    Path string
}

func (err FileNotFoundError) Error() string {
    return fmt.Sprintf("file not found: %s", err.Path)
}

func (err FileNotFoundError) GetErrorType() fileNotFoundDisamb {
    return fileNotFoundDisamb{}
}

func MakeFileNotFoundError(path string) error {
    return FileNotFoundError{Path: path}
}

func IsFileNotFoundError(err error) bool {
    return errors.IsTyped[fileNotFoundDisamb](err)
}
```

### Pattern: Error with Recovery

Combine with `errors.Helpful` interface:

```go
type lockErrorDisamb struct{}

type LockError struct {
    Path string
}

func (err LockError) Error() string {
    return fmt.Sprintf("unable to acquire lock: %s", err.Path)
}

func (err LockError) GetErrorType() lockErrorDisamb {
    return lockErrorDisamb{}
}

func (err LockError) GetErrorCause() []string {
    return []string{
        "A previous operation that acquired the lock failed.",
        "The lock is intentionally left behind in case recovery is necessary.",
    }
}

func (err LockError) GetErrorRecovery() []string {
    return []string{
        fmt.Sprintf("The lockfile needs to be removed (`rm %q`).", err.Path),
    }
}

func MakeLockError(path string) error {
    return LockError{Path: path}
}

func IsLockError(err error) bool {
    return errors.IsTyped[lockErrorDisamb](err)
}
```

## Decision Tree

When creating a new error:

```
START
├─ Is this a one-time error message?
│  └─ YES: Use errors.ErrorWithStackf("message")
│  └─ NO: ↓
├─ Is this a sentinel that needs checking?
│  └─ YES: ↓
│  ├─ Does it need to carry data?
│  │  └─ YES: Create struct with GetErrorType() → Rich Error Type
│  │  └─ NO: Use MakeTypedSentinel[T]() → Simple Sentinel
│  └─ NO: Use errors.ErrorWithStackf("message")
```

## Anti-Patterns to Avoid

### ❌ Multiple Disambiguators per Package

```go
// BAD - too many disambiguators
type err1Disamb struct{}
type err2Disamb struct{}
type err3Disamb struct{}

var Err1 = errors.NewWithType[err1Disamb]("error 1")
var Err2 = errors.NewWithType[err2Disamb]("error 2")
var Err3 = errors.NewWithType[err3Disamb]("error 3")
```

```go
// GOOD - one disambiguator for related errors
type pkgErrDisamb struct{}

var (
    Err1, IsErr1 = errors.MakeTypedSentinel[pkgErrDisamb]("error 1")
    Err2, IsErr2 = errors.MakeTypedSentinel[pkgErrDisamb]("error 2")
    Err3, IsErr3 = errors.MakeTypedSentinel[pkgErrDisamb]("error 3")
)
```

### ❌ Exporting Disambiguator Types

```go
// BAD - disambiguator is exported
type PkgErrDisamb struct{}  // Exported!
```

```go
// GOOD - disambiguator is private
type pkgErrDisamb struct{}  // Private
```

### ❌ Wrapping Sentinels

```go
// BAD - wrapping loses sentinel identity
if notFound {
    return errors.Wrapf(ErrNotFound, "looking up %s", id)
}

// Check will fail!
if errors.Is(err, ErrNotFound) {  // false - wrapped!
    // Won't execute
}
```

```go
// GOOD - return sentinel directly
if notFound {
    return MakeErrNotFound(id)  // Rich error with data
}

// Check works
if IsErrNotFound(err) {  // true
    // Executes correctly
}
```

## Testing Sentinel Errors

```go
func TestMyError(t *testing.T) {
    // Test sentinel identity
    err := ErrMyError
    if !IsMyError(err) {
        t.Error("IsMyError should match sentinel")
    }

    if !errors.Is(err, ErrMyError) {
        t.Error("errors.Is should match sentinel")
    }

    // Test rich errors
    richErr := MakeMyError("value", 123)
    if !IsMyError(richErr) {
        t.Error("IsMyError should match rich error")
    }

    myErr, ok := GetMyError(richErr)
    if !ok || myErr.Value != "value" || myErr.Code != 123 {
        t.Error("GetMyError should extract data")
    }

    // Test wrapped errors
    wrapped := errors.Wrap(richErr)
    if !IsMyError(wrapped) {
        t.Error("IsMyError should work on wrapped errors")
    }

    // Test type safety
    otherErr := ErrOtherError
    if IsMyError(otherErr) {
        t.Error("IsMyError should not match different sentinel")
    }
}
```

## Summary

**Standard Pattern:**
1. Define `type pkgErrDisamb struct{}`
2. Use `errors.MakeTypedSentinel[pkgErrDisamb]("message")` for simple sentinels
3. Create rich structs with `GetErrorType()` for data-carrying errors
4. Return sentinels directly, don't wrap them
5. Use `errors.IsTyped[T]()` or package-specific checker functions

**Benefits:**
- Compile-time type safety
- No runtime allocations
- Works with standard `errors.Is()`
- Consistent across codebase
- Easy to test

**Migration:**
- New code uses typed pattern
- Old sentinels continue working
- 6-month deprecation window for old APIs
