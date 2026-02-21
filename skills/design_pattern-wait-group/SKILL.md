---
name: design_pattern-wait-group
description: >
  Use when running concurrent operations with error collection, structuring
  parallel or serial work with cleanup, or encountering MakeWaitGroupParallel,
  MakeWaitGroupSerial, DoAfter, or GroupBuilder in code.
triggers:
  - WaitGroup
  - MakeWaitGroupParallel
  - MakeWaitGroupSerial
  - DoAfter
  - GroupBuilder
  - error aggregation
  - parallel operations
  - concurrent error handling
---

# Wait Group with Error Aggregation

## Overview

Dodder's wait groups extend Go's `sync.WaitGroup` with automatic error
collection, ordered cleanup via `DoAfter`, and stack frame capture in debug
builds. Two variants exist: parallel (concurrent goroutines) and serial
(sequential execution). Both implement the same `WaitGroup` interface and
aggregate all errors into a single return value.

## Interface

```go
// alfa/errors (interfaces)
type WaitGroup interface {
    Do(FuncErr) bool     // Queue work. Returns false if already done.
    DoAfter(FuncErr)     // Queue cleanup (runs in reverse order after Do completes).
    GetError() error     // Wait for all work + cleanup, return aggregated errors.
}
```

## Creating Wait Groups

```go
wg := errors.MakeWaitGroupParallel()  // concurrent — each Do() runs in its own goroutine
wg := errors.MakeWaitGroupSerial()    // sequential — Do() functions run in order during GetError()
```

## Usage Pattern

```go
wg := errors.MakeWaitGroupParallel()

wg.Do(func() error {
    return flushIndex()
})

wg.Do(func() error {
    return flushAbbreviations()
})

wg.DoAfter(func() error {
    return releaseLock()
})

if err := wg.GetError(); err != nil {
    // err contains all errors from Do() and DoAfter()
}
```

## How It Works

### Parallel

- Each `Do()` call launches a goroutine immediately
- Errors are collected thread-safely via `GroupBuilder` (mutex-protected)
- `GetError()` waits for all goroutines, then runs `DoAfter` functions in
  reverse order (LIFO), then returns aggregated errors

### Serial

- `Do()` stores functions in a slice
- `GetError()` executes them sequentially, collecting errors
- `DoAfter` functions run in reverse order after all `Do` functions complete

### Error Aggregation

`GroupBuilder` collects errors thread-safely:

```go
// alfa/errors/group_builder.go
type GroupBuilder struct {
    lock  sync.Mutex
    group Group
}
```

- `Add()` automatically flattens nested `Group` errors when collecting
- `GetError()` returns `nil` if no errors, or a `Group` containing all errors

## DoAfter Ordering

`DoAfter` functions execute in **reverse registration order** (LIFO), similar
to `defer`:

```go
wg.DoAfter(func() error { return cleanup1() })  // runs second
wg.DoAfter(func() error { return cleanup2() })  // runs first
```

## Debug Stack Capture

In debug builds, `Do()` captures the caller's stack frame via
`stack_frame.MakeFrame(1)`. This aids in diagnosing which call site produced an
error when multiple parallel operations fail.

## Real-World Examples

### Parallel flush

```go
// tango/store/flush.go
wg := errors.MakeWaitGroupParallel()
wg.Do(func() error { return store.streamIndex.Flush(printerHeader) })
wg.Do(store.GetAbbrStore().Flush)
wg.Do(store.zettelIdIndex.Flush)
if err = wg.GetError(); err != nil {
    return errors.Wrap(err)
}
```

### Parallel query with cleanup

```go
// papa/store_fs/query.go
wg := errors.MakeWaitGroupParallel()
wg.Do(func() (err error) {
    for item := range store.probablyCheckedOut.All() {
        if err = process(item); err != nil { return }
    }
    return
})
wg.DoAfter(index.Flush)
err = wg.GetError()
```

## Common Mistakes

| Mistake | Correct Approach |
|---------|-----------------|
| Using `sync.WaitGroup` + manual error channels | Use `MakeWaitGroupParallel()` for built-in error aggregation |
| Ignoring `Do()` return value | Returns `false` if wait group is already done — check when conditional |
| Putting cleanup in `Do()` instead of `DoAfter()` | `DoAfter` guarantees execution after all `Do` work completes |
| Calling `GetError()` multiple times | Call once — it waits and drains all work |
