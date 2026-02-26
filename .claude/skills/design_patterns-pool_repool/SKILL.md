---
name: design_patterns-pool_repool
description: >
  Use when allocating pooled objects, calling GetWithRepool, handling FuncRepool
  return values, writing code that borrows from sync.Pool, or encountering
  repool analyzer warnings. Also applies when adding //repool:owned annotations,
  debugging pool leaks, or working with CloneTransacted.
triggers:
  - GetWithRepool
  - FuncRepool
  - repool
  - pool leak
  - pool management
  - CloneTransacted
  - repool:owned
  - OutstandingBorrows
---

# Pool-Repool Memory Management

## Overview

Dodder uses object pooling with a mandatory borrow-return lifecycle. Every
pooled allocation returns both the element and a `FuncRepool` closure that must
be called exactly once when the caller is done with the element. Three
enforcement layers ensure compliance: a static analyzer, runtime debug
poisoning, and lint checks.

## Core API

### Pool Interface

```go
// _/interfaces/pools.go
type FuncRepool func()

type Pool[T any] interface {
    GetWithRepool() (T, FuncRepool)
}

type PoolPtr[T any, TPtr Ptr[T]] interface {
    Pool[TPtr]
}
```

### Pool Factory Functions

| Factory | Use When |
|---------|----------|
| `pool.Make(New, Reset)` | Custom allocation and reset logic |
| `pool.MakeWithResetable[T]()` | Type implements `ResetablePtr` (has `Reset()`) |
| `pool.MakeValue(New, Reset)` | Value types (non-pointer) |

### Basic Usage

```go
element, repool := somePool.GetWithRepool()
defer repool()
// use element...
```

### Cloning Pooled Objects

```go
cloned, repool := original.CloneTransacted()
defer repool()
// cloned is a pool-managed copy via ResetWith, not pointer dereference
```

Never dereference `*pointer` to copy a pooled object. Use `ResetWith` or
`CloneTransacted`.

## The Three Enforcement Layers

### 1. Static Analyzer (`alfa/analyzers/repool/`)

Run via `just check`. The CFG-based `go vet` checker detects:

- **Discarded repool**: Assigning repool to `_` without `//repool:owned`
- **Uncalled repool**: Repool variable not called on all code paths

### 2. Runtime Debug Poisoning (build tag `debug`)

In debug builds, every repool function is wrapped with:

- `atomic.Bool` guard that panics on double-repool
- `atomic.Int64` counter tracking outstanding borrows
- Caller location capture for diagnostics

Zero overhead in release builds.

### 3. Lint Check (`bin/lint.bash`)

Grep-based check for discarded repool functions missing the `//repool:owned`
annotation.

## The `//repool:owned` Annotation

Suppresses analyzer warnings when intentionally discarding repool. Use when the
pooled element's lifetime extends beyond the borrowing scope:

```go
hash, _ := config.hashFormat.GetHash() //repool:owned
writer.digester = markl_io.MakeWriter(hash, nil)
// hash lives as long as writer — caller cannot defer repool here
```

## Debugging Pool Leaks

In debug builds, `pool.OutstandingBorrows()` returns the count of borrowed but
unreturned elements. Returns 0 in release builds.

## Common Mistakes

| Mistake | Correct Approach |
|---------|-----------------|
| Discarding repool with `_` | Use `defer repool()` or annotate `//repool:owned` |
| Calling repool twice | Call exactly once. Debug builds panic on double-repool. |
| Dereferencing `*pooledPtr` to copy | Use `ResetWith()` or `CloneTransacted()` |
| Forgetting repool on error paths | The static analyzer catches this via CFG analysis |
| Using `pool.MakeValue` with interface fields | Value pools share underlying pointers across copies — use pointer pools instead |
