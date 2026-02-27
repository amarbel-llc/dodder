# pool

Generic sync.Pool wrappers with reset support.

## Key Types

- `pool[SWIMMER, SWIMMER_PTR]` - pointer-based pool wrapper
- `value[SWIMMER]` - value-based pool wrapper

## Key Functions

- `Make()` - creates pool with custom New/Reset functions
- `MakeWithResetable()` - creates pool for types implementing Resetable
- `MakeValue()` - creates value-based pool

## Key Methods

- `Get()` - retrieves element from pool
- `GetWithRepool()` - gets element with automatic return function
- `Put()` - returns element to pool (with reset)

## Features

- Type-safe generic pools
- Automatic reset on Put()
- Bespoke pools (bespoke.go) and fake pools (fake_pool.go) for testing
- FakePool with error injection via WithError wrapper

## Debug Poisoning (`repool_debug.go` / `repool_release.go`)

Build with `-tags debug` to enable runtime repool guards:

- **Double-repool detection**: Wraps every `FuncRepool` with an `atomic.Bool`
  that panics with caller location on second call
- **Outstanding borrow tracking**: `pool.OutstandingBorrows()` returns the count
  of elements borrowed but not yet returned
- **Zero overhead in release**: `repool_release.go` compiles to no-op passthrough

## Discarding Repool (`//repool:owned`)

When a pooled element's lifetime cannot be bounded to a single scope (e.g., a
`hash.Hash` embedded in a blob writer that outlives the constructor), discard the
repool function and annotate with `//repool:owned` to suppress the static
analyzer:

```go
hash, _ := config.hashFormat.GetHash() //repool:owned
```

## Value Pool Pitfall

`MakeValue[T]` returns copies of T, but if T contains interface fields (like
`hash.Hash`), copies share the underlying pointer. A `Reset()` via repool on one
copy corrupts all copies. Never repool a value-pool element while any copy is
still in use.
