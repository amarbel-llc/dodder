# Pool Cleanup Design

## Problem

The pool infrastructure has accumulated redundancy and inconsistency across the
NATO module hierarchy:

- Two pool interface families (`Pool`/`PoolValue` and `PoolWithErrors`/`PoolWithErrorsPtr`)
- Duplicate value pool packages (`_/pool_value/` and `alfa/pool/value.go`)
- Identical store pool files (`papa/store_fs/pools.go` and
  `sierra/store_browser/pools.go`) with nil resetters
- Manual `sync.Pool` wrapper in `delta/alfred/item_pool.go`
- Mixed usage of `Get()`/`Put()` vs `GetWithRepool()` across ~44 files
- No consistent initialization pattern (module-level vs `sync.Once`)

## Design

### Single Public API: GetWithRepool

All pool consumers use one pattern:

```go
element, repool := pool.GetWithRepool()
defer repool()
```

`Get()` and `Put()` become unexported on concrete types. The only public method
on the pool interface is `GetWithRepool()`.

### Interfaces

`_/interfaces/pools.go` reduces to:

```go
type FuncRepool func()

type Pool[T any] interface {
    GetWithRepool() (T, FuncRepool)
}

type PoolPtr[T any, TPtr Ptr[T]] interface {
    Pool[TPtr]
}
```

- `Pool[T]` serves value types (interfaces, `hash.Hash`, slices).
- `PoolPtr[T, TPtr]` serves mutable pointer types (`*Transacted`, `*CheckedOut`).
  It embeds `Pool[TPtr]`, so anything accepting `Pool[*Transacted]` also accepts
  `PoolPtr[Transacted, *Transacted]`.

Removed interfaces:

- `PoolValue[T]`
- `PoolWithErrors[T]`
- `PoolWithErrorsPtr[T, TPtr]`
- `PoolablePtr[T]`

### Concrete Types in `alfa/pool/`

| Type | Status | Notes |
|------|--------|-------|
| `pool[T, TPtr]` | Modified | `Get()` -> `get()`, `Put()` -> `put()`, public API is `GetWithRepool()` only |
| `value[T]` | Modified | Same unexport treatment |
| `Bespoke[T]` | Modified | Same unexport treatment |
| `fakePool[T, TPtr]` | Modified | Same treatment, remove `PutMany` |
| `Slice[T, S ~[]T]` | New | Moved from `_/pool_value/slice.go` |
| `poolWithError[T, TPtr]` | Deleted | Lua uses panic semantics |

### Package Deletions

`_/pool_value/` is deleted entirely. Its two types are replaced by:

- `_/pool_value/main.go` -> `alfa/pool/value.go` (already exists)
- `_/pool_value/slice.go` -> new `alfa/pool/slice.go`

### Store Pool Consolidation

`papa/store_fs/pools.go` and `sierra/store_browser/pools.go` are identical
copies with nil resetters. Both are consolidated into a shared location with
proper resetters matching `juliett/sku/pools.go`. `tango/store/pools.go` is
simplified transitively.

### Alfred ItemPool Migration

`delta/alfred/item_pool.go` replaces its manual `sync.Pool` wrapper with
`alfa/pool.Make`, using `Item.Reset` as the resetter. Callers switch to
`GetWithRepool()`.

### Lua VM Pool — Panic Semantics

`VMPool` currently embeds `interfaces.PoolWithErrorsPtr[VM, *VM]`. The
`MakeWithError` constructor already panics on allocation error. The `Get()`
method always returns nil error.

Change: `VMPool` embeds `PoolPtr[VM, *VM]` instead. The `New` function panics
on error (no behavior change). Callers use `GetWithRepool()`.

### Heap Internal Access

The heap manages element lifecycles internally (allocate on Pop, return on
Restore/Reset). It uses the concrete pool type directly with unexported
`get()`/`put()` methods, bypassing the interface. This is the intended escape
hatch for data structures that own element lifetimes.

### Caller Migration

~77 `Get()` call sites and ~44 `Put()` call sites across ~44 files. The
transformation is mechanical:

Before:

```go
object := sku.GetTransactedPool().Get()
defer sku.GetTransactedPool().Put(object)
```

After:

```go
object, repool := sku.GetTransactedPool().GetWithRepool()
defer repool()
```

Helper functions in `alfa/pool/common.go` simplify similarly:

Before:

```go
func GetStringReader(value string) (stringReader *strings.Reader, repool func()) {
    stringReader = stringReaders.Get()
    stringReader.Reset(value)
    repool = func() { stringReaders.Put(stringReader) }
    return
}
```

After:

```go
func GetStringReader(value string) (stringReader *strings.Reader, repool FuncRepool) {
    stringReader, repool = stringReaders.GetWithRepool()
    stringReader.Reset(value)
    return
}
```

## Scope Summary

| Area | Files | Change Type |
|------|-------|-------------|
| `_/interfaces/pools.go` | 1 | Interface reduction |
| `alfa/pool/` | 6 | Unexport Get/Put, add Slice, delete poolWithError |
| `_/pool_value/` | 2 | Delete package |
| `papa/store_fs/pools.go` | 1 | Centralize with resetters |
| `sierra/store_browser/pools.go` | 1 | Centralize with resetters |
| `tango/store/pools.go` | 1 | Simplify |
| `delta/alfred/item_pool.go` | 1 | Migrate to alfa/pool |
| `bravo/lua/vm_pool.go` | 1 | Panic semantics, drop error interface |
| `charlie/heap/` | 2 | Use unexported get/put |
| Caller migration | ~44 | Mechanical Get/Put -> GetWithRepool |
