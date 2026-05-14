# repool

Static analyzer that checks `FuncRepool` functions returned by `GetWithRepool`
are properly called.

## Usage

``` sh
just check-go-repool   # nix-builds the analyzer + runs on all packages
```

The analyzer binary lives in the flake as `.#dodder-analyzer-repool`;
the just recipe captures its store path with `nix build --no-link
--print-out-paths` and passes it to `go vet -vettool=...`.

## What It Detects

1.  **Discarded repool (blank `_`)**: Reports when the repool function is
    assigned to `_` without a `//repool:owned` suppression comment on the same
    line.

2.  **Incomplete call paths**: Uses CFG analysis to find repool variables that
    are not called on all code paths before the function returns (potential
    leaks).

## Suppression

### `//repool:owned` --- discarded repool (blank `_`)

Add `//repool:owned` on the assignment line to indicate the caller intentionally
takes ownership of the element's lifetime:

``` go
hash, _ := config.hashFormat.GetHash() //repool:owned
```

### `//repool:suppress` --- incomplete call paths (false positives)

Add `//repool:suppress <reason>` on the assignment line to silence the "not
called on all paths" diagnostic when the analyzer cannot prove safety but the
code is correct. A reason is required (e.g., an issue link or description):

``` go
v, repool := pool.GetWithRepool() //repool:suppress #47 ownership transfer via return
```

## Implementation

- Built on `golang.org/x/tools/go/analysis` framework
- Requires `inspect.Analyzer` and `ctrlflow.Analyzer` passes
- Looks for the `FuncRepool` named type in packages ending with `interfaces`
- Handles assignments, var declarations, deferred calls, struct field storage,
  and pass-to-function patterns
- Test cases in `testdata/src/a/a.go`
