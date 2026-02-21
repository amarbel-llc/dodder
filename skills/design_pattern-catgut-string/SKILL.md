---
name: design_pattern-catgut-string
description: >
  Use when working with catgut.String, encountering copy-detection panics,
  pooling strings, or needing mutable buffer-backed strings. Also applies when
  seeing noescape, copyCheck, catgut.GetPool, or MakeFromString in code.
triggers:
  - catgut
  - catgut.String
  - copyCheck
  - noescape
  - string interning
  - string pooling
  - MakeFromString
---

# Catgut Poolable String

## Overview

`catgut.String` is a mutable, pool-managed string type backed by `bytes.Buffer`.
It replaces Go's immutable `string` in hot paths where allocation pressure from
temporary strings would be significant. Copy detection via self-pointer tracking
panics in debug builds when a `catgut.String` is accidentally copied by value.

## Why It Exists

Go strings are immutable and heap-allocated. In dodder's inner loops — parsing
tags, paths, identifiers — creating many transient strings generates GC
pressure. `catgut.String` solves this by:

- **Pooling**: Buffers are reused across operations via `GetPool().GetWithRepool()`
- **Mutability**: `Reset()` clears contents without reallocating
- **Copy protection**: Debug panics prevent accidental value copies that would
  break pool integrity

## Type Definition

```go
// charlie/catgut/string.go
type String struct {
    addr *String      // self-pointer for copy detection
    data bytes.Buffer // actual string data
}
```

## Copy Detection

On every mutating operation, `copyCheck()` verifies the struct hasn't been
copied by value:

```go
func (b *String) copyCheck() {
    if b.addr == nil {
        b.addr = (*String)(noescape(unsafe.Pointer(b)))
        return
    }
    if b.addr != b {
        panic("catgut: illegal use of non-zero String copied by value")
    }
}
```

The `noescape()` function hides the pointer from Go's escape analysis to avoid
forcing heap allocation.

## Pool Integration

```go
// charlie/catgut/pool.go
func GetPool() interfaces.PoolPtr[String, *String] {
    // Returns pool that calls Reset() on Put
}
```

Usage:

```go
str, repool := catgut.GetPool().GetWithRepool()
defer repool()
str.Set("hello")
```

For long-lived singletons:

```go
// delta/key_strings/main.go
var (
    Blob, _        = catgut.MakeFromString("Blob")
    Description, _ = catgut.MakeFromString("Description")
    // repool discarded — these live for the process lifetime
)
```

## Key Methods

| Method | Purpose |
|--------|---------|
| `Set(string)` | Set from Go string |
| `SetBytes([]byte)` | Set from byte slice |
| `Reset()` | Clear contents for reuse |
| `String() string` | Convert to Go string |
| `Bytes() []byte` | Access underlying bytes |
| `Len() int` | Length |
| `IsEmpty() bool` | Empty check |
| `Equals(*String) bool` | Compare two catgut strings |
| `EqualsString(string) bool` | Compare with Go string |
| `Compare(*String) cmp.Result` | Three-way comparison |
| `Write([]byte)` | Append bytes (implements io.Writer) |
| `WriteTo(io.Writer)` | Stream contents out |
| `ReadFrom(io.Reader)` | Read contents in |

## Rules

1. **Always use as pointer** (`*catgut.String`). Never pass by value.
2. **Never assign one catgut.String to another**. The copy detection will panic.
3. **Pool strings used in hot paths**. Use `GetPool().GetWithRepool()` and
   `defer repool()`.
4. **Use `Reset()` before reuse**, not reallocation.

## Common Mistakes

| Mistake | Correct Approach |
|---------|-----------------|
| Passing `catgut.String` by value | Always use `*catgut.String` |
| Assigning `a = b` where both are catgut strings | Use `a.SetBytes(b.Bytes())` |
| Allocating new strings in a loop | Pool with `GetPool().GetWithRepool()` and `Reset()` between iterations |
| Ignoring repool from `MakeFromString` | Either `defer repool()` or discard with `_` for process-lifetime singletons |
