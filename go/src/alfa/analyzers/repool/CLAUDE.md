# repool

Static analyzer that checks `FuncRepool` functions returned by `GetWithRepool`
are properly called.

## Usage

```sh
just check-go-repool          # build analyzer + run on all packages
go vet -vettool=build/repool-analyzer ./...  # run directly
```

## What It Detects

1. **Discarded repool (blank `_`)**: Reports when the repool function is
   assigned to `_` without a `//repool:owned` suppression comment on the same
   line.

2. **Incomplete call paths**: Uses CFG analysis to find repool variables that are
   not called on all code paths before the function returns (potential leaks).

## Suppression

Add `//repool:owned` on the assignment line to indicate the caller intentionally
takes ownership of the element's lifetime:

```go
hash, _ := config.hashFormat.GetHash() //repool:owned
```

## Implementation

- Built on `golang.org/x/tools/go/analysis` framework
- Requires `inspect.Analyzer` and `ctrlflow.Analyzer` passes
- Looks for the `FuncRepool` named type in packages ending with `interfaces`
- Handles assignments, var declarations, deferred calls, struct field storage,
  and pass-to-function patterns
- Test cases in `testdata/src/a/a.go`
