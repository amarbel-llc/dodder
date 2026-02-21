---
name: design_pattern-sequence-iteration
description: >
  Use when working with lazy sequences, SeqError iterators, combining or
  chaining iterators, or encountering Seq, SeqError, quiter, quiter_seq,
  MakeSeqErrorFromSeq, Chain, Multiplex, or MergeSeqErrorLeft in code.
triggers:
  - SeqError
  - Seq
  - quiter
  - quiter_seq
  - iterator
  - sequence
  - lazy evaluation
  - Chain
  - Multiplex
  - MergeSeqErrorLeft
---

# Sequence Iteration

## Overview

Dodder uses Go 1.23's `iter.Seq` and `iter.Seq2` as the foundation for lazy,
streaming iteration with integrated error handling. The type aliases `Seq[T]`
and `SeqError[T]` are used throughout the codebase for everything from blob
store enumeration to query results. Combinator packages (`quiter`, `quiter_seq`)
provide chaining, merging, multiplexing, and collection utilities.

## Core Types

```go
// _/interfaces/iter.go
type (
    Seq[ELEMENT any]      = iter.Seq[ELEMENT]
    SeqError[ELEMENT any] = iter.Seq2[ELEMENT, error]
)
```

`SeqError[T]` is a function `func(yield func(T, error) bool)` — each element
comes with an optional error. The consumer returns `false` to stop iteration.

## Creating Sequences

### From a function (most common)

```go
func AllBlobs() interfaces.SeqError[MarklId] {
    return func(yield func(MarklId, error) bool) {
        for _, entry := range entries {
            if !yield(entry, nil) {
                return
            }
        }
    }
}
```

### From variadic elements

```go
// alfa/quiter_seq/main.go
seq := quiter_seq.Seq(element1, element2, element3)
```

### Wrapping a plain Seq as SeqError

```go
// bravo/quiter/iter.go
errSeq := quiter.MakeSeqErrorFromSeq(plainSeq)
```

### Error-only sequence

```go
errSeq := quiter.MakeSeqErrorWithError[T](err)
```

## Consuming Sequences

### Range loop (standard)

```go
for element, err := range someSeqError {
    if err != nil {
        return err
    }
    process(element)
}
```

### Collect into slice

```go
// bravo/quiter/iter.go
slice, err := quiter.CollectError(seqError)
```

### Get first element

```go
first := quiter_seq.Any(seq)
```

### Pull-based iteration

```go
next, stop := quiter.PullError(seqError)
defer stop()
element, err, ok := next()
```

## Combinators

### Chain (sequential concatenation)

```go
// bravo/quiter/chain.go
combined := quiter.Chain(seq1, seq2, seq3)
```

### Multiplex (parallel producers into single sequence)

```go
// bravo/quiter/chain.go
merged := quiter.Multiplex(producer1, producer2)
```

### Merge (sorted merge of two sequences)

```go
// bravo/quiter/merge.go
merged := quiter.MergeSeqErrorLeft(left, right, cmpFunc)
```

Prefers left on equality.

### Map to strings

```go
// alfa/quiter_seq/main.go
strings := quiter_seq.Strings(seqOfStringers)
```

### Add index

```go
indexed := quiter_seq.SeqWithIndex(seq)
// yields (index, element) pairs
```

## Error Handling

Errors propagate as the second yield value. Conventions:

- `yield(element, nil)` — successful element
- `yield(zeroValue, err)` — error during iteration
- Return `errors.MakeErrStopIteration()` to signal early termination from inside
  a callback
- Check `errors.IsStopIteration(err)` to distinguish intentional stops from real
  errors

## Integration with Wait Groups

```go
// bravo/quiter/error_wait_group_apply.go
func ErrorWaitGroupApply[T any](
    wg errors.WaitGroup,
    s interfaces.Collection[T],
    f interfaces.FuncIter[T],
) bool
```

Applies a function to each element in the collection via the wait group,
enabling parallel processing of sequence elements with error aggregation.

## Integration with Pool-Repool

Sequences that yield pooled objects follow the convention that the yielded
pointer is valid only until the next iteration. Consumers must clone if they
need to retain:

```go
// lima/inventory_list_coders/main.go
object, _ := sku.GetTransactedPool().GetWithRepool()
// ... populate object ...
if !yield(object, nil) {
    break
}
// object may be reused on next iteration
```

## Common Mistakes

| Mistake | Correct Approach |
|---------|-----------------|
| Retaining a yielded pointer after the next iteration | Clone with `CloneTransacted()` if you need to keep it |
| Ignoring the error in `SeqError` range loops | Always check `err` before using the element |
| Using `break` without returning from the sequence func | `break` in a range loop sends `false` to yield, which is correct |
| Building a slice manually instead of using `CollectError` | Use `quiter.CollectError()` for error-aware collection |
