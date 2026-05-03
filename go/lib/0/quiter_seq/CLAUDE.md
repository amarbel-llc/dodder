# quiter_seq

Iterator (Seq) utility functions for Go's iter package integration.

## Key Functions

- `Seq()` - creates Seq from variadic elements
- `Any()` - gets first element from Seq
- `SeqWithIndex()` - converts Seq to Seq2 with indices
- `Strings()` - converts Seq[Stringer] to Seq[string]
- `SeqErrorToSeqAndPanic()` - converts SeqError to Seq (panics on error)

## Features

- Integrates with Go 1.23+ iter package
- Type-safe iterator transformations
- Error-to-panic conversion for SeqError iterators
