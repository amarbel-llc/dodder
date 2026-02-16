# Pool Management

Pool management in dodder uses `sync.Pool`-backed generic pools with a
repool-function pattern that ensures every borrowed element is returned exactly
once. Three enforcement layers -- static analysis, runtime debug poisoning, and
CI lint -- work together to prevent leaks and double-returns.

## GetWithRepool() Lifecycle

Every pool interaction follows the same pattern:

```go
element, repool := pool.GetWithRepool()
// ... use element ...
repool()  // MUST call exactly once
```

`GetWithRepool()` returns a pointer to the pooled element and a `FuncRepool`
closure. Calling `repool()` resets the element and returns it to the underlying
`sync.Pool`. Failing to call `repool()` leaks the element. Calling it more than
once corrupts pool state (caught by debug poisoning).

Typical usage with defer:

```go
element, repool := somePool.GetWithRepool()
defer repool()
// ... use element for the rest of the function scope ...
```

When the element must outlive the current scope, pass the repool function to the
consumer or store it alongside the element for later invocation.

## Three-Layer Safety System

### 1. Static Analyzer (`just check-go-repool`)

A CFG-based `go vet` checker located in `go/src/alfa/analyzers/repool/analyzer.go`.
Built on `golang.org/x/tools/go/analysis` with `inspect.Analyzer` and
`ctrlflow.Analyzer` passes.

**What it detects:**

- **Discarded repool (blank `_`):** Reports when the repool function is assigned
  to `_` without a `//repool:owned` suppression comment on the same line.
- **Incomplete call paths:** Performs depth-first search of the control flow
  graph starting from the assignment. Reports when any path from the definition
  to a function exit does not call the repool variable.

**How it works:**

1. Skips packages that do not import `interfaces` (the package containing
   `FuncRepool`).
2. Iterates over all function declarations and function literals.
3. For each assignment or var declaration whose right-hand side returns a
   `FuncRepool`, identifies the repool variable.
4. Locates the defining block in the CFG.
5. Searches successor blocks depth-first for any path to return that does not
   reference the repool variable.

**Suppression:** Add `//repool:owned` on the assignment line to indicate
intentional lifetime ownership transfer:

```go
hash, _ := config.hashFormat.GetHash() //repool:owned
```

### 2. Runtime Debug Poisoning (build tag `debug`)

Located in `go/src/alfa/pool/repool_debug.go`. Activated by compiling with
`-tags debug` (the default for `just build` debug binaries and `just test-go`).

**How it works:**

- `wrapRepoolDebug()` wraps every repool closure with an `atomic.Bool` guard.
- The first call sets the bool to `true` and proceeds normally.
- A second call finds the bool already `true` and panics with a message
  including the file and line where the element was originally borrowed:
  `"repool: double-repool detected (originally borrowed at file.go:42)"`.
- An `atomic.Int64` counter (`outstandingBorrows`) increments on borrow and
  decrements on repool. Query via `pool.OutstandingBorrows()`.

**Zero overhead in release builds:** `go/src/alfa/pool/repool_release.go`
compiles when the `debug` tag is absent. It passes through the repool function
unchanged and returns `0` from `OutstandingBorrows()`.

### 3. CI Lint Check (`bin/lint.bash`)

A grep-based check that scans for discarded repool functions (blank `_`
assignments from `GetWithRepool` calls) missing the `//repool:owned` annotation.
Runs as part of the CI pipeline.

## sku.Transacted Rules

`sku.Transacted` is the central versioned object type. It is always managed
through pools and must never be copied via pointer dereference.

### WRONG: Direct Dereference

```go
// WRONG: creates a shallow copy that shares internal pointers
value := *object
```

```go
// WRONG: dereference into struct field
someStruct.Field = *object
```

Both forms violate pool management. The shallow copy shares mutable internal
state with the original, leading to subtle corruption when either side modifies
fields.

### CORRECT: ResetWith for Local Values

```go
// Copy field data safely into a local value
var local sku.Transacted
sku.TransactedResetter.ResetWith(&local, src)
```

`ResetWith` performs a deep, field-by-field copy that properly handles internal
state. Use this when a value-typed copy is needed for temporary processing.

### CORRECT: ResetWith into Existing Pointer

```go
// Reset an existing pool-managed object from another source
obj := sku.GetTransactedPool().Get()
defer sku.GetTransactedPool().Put(obj)
sku.TransactedResetter.ResetWith(obj, src)
```

Use this when obtaining a new pool element and populating it from an existing
object.

### CORRECT: CloneTransacted for Persistent Objects

```go
// Clone creates a new pool-managed copy for objects that must persist
cloned := original.CloneTransacted()
defer sku.GetTransactedPool().Put(cloned)
```

`CloneTransacted()` allocates from the pool and performs a deep copy. Always
defer the `Put()` to ensure the clone returns to the pool when done.

### CORRECT: ResetWith into Typed Blob Struct

```go
typedBlob := &triple_hyphen_io.TypedBlob[sku.Transacted]{
    Type: tipe,
}
sku.TransactedResetter.ResetWith(&typedBlob.Blob, sourcePointer)
return encoder.EncodeTo(typedBlob, writer)
```

When populating a generic struct that contains a `Transacted` field, use
`ResetWith` to fill the field without dereferencing.

## Common Pitfall: Hash Lifetime in Blob Writers

When a pooled `hash.Hash` is embedded in a blob reader or writer via
`markl_io.MakeWriter`, the hash's lifetime extends from construction through all
reads and writes to `GetMarklId()` AFTER `Close()`.

The `localFileMover` pattern illustrates why this matters:

1. Construct writer with embedded hash.
2. Write blob content (hash accumulates digest).
3. Call `writer.Close()`.
4. Call `writer.GetMarklId()` to compute the final digest for the destination
   path.

Any repool of the hash before step 4 corrupts the digest because the pooled
hash gets reset and potentially reused.

**Additional hazard with value pools:** `pool.MakeValue[Hash]` returns copies of
the hash struct. Because `hash.Hash` is an interface, copies share the
underlying pointer. A `Reset()` via repool on one copy corrupts all copies
simultaneously.

**Solution:** Discard the repool function with `//repool:owned` when the hash
lifetime cannot be bounded to a single scope:

```go
hash, _ := config.hashFormat.GetHash() //repool:owned
writer.digester = markl_io.MakeWriter(hash, nil)
// hash lives as long as writer; no repool needed
```

The `//repool:owned` annotation suppresses the static analyzer. The hash is
effectively "owned" by the writer and will be garbage-collected when the writer
is no longer referenced.

## Debugging Pool Leaks

Use `pool.OutstandingBorrows()` in debug builds to check for unreturned pool
elements:

```go
// At a known quiescent point (e.g., end of a test or shutdown)
if outstanding := pool.OutstandingBorrows(); outstanding != 0 {
    panic(fmt.Sprintf("pool leak: %d outstanding borrows", outstanding))
}
```

This function returns `0` in release builds (no overhead). In debug builds, it
queries the global `atomic.Int64` counter that tracks all `GetWithRepool` /
repool pairs across all pool instances.

To isolate a leak:

1. Build with `-tags debug` (or use `just build` which produces debug binaries).
2. Add `OutstandingBorrows()` checks at strategic points.
3. Run the failing test or operation.
4. The debug wrapper records the borrowing call site (file:line) in each repool
   closure, so a double-repool panic message identifies exactly where the
   element was originally borrowed.
5. For leaks (missing repool), instrument suspect code paths with logging around
   `GetWithRepool()` calls and verify each has a matching `repool()`.
