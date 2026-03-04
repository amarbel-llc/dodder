# govulncheck SSA panic on go-json-experiment types

## Status

Upstream bug, unresolved. Tracked at
[golang/go#73871](https://github.com/golang/go/issues/73871).

## Symptom

`just check` (specifically `check-go-vuln` / `govulncheck ./...`) panics:

```
panic: got github.com/go-json-experiment/json/jsontext.Value,
       want variadic parameter of unnamed slice or string type
```

Stack: `go/types.NewSignatureType` <-
`golang.org/x/tools/go/ssa.(*subster).signature` (`subst.go:566`).

## Root Cause

`govulncheck@1.1.4` bundles `golang.org/x/tools@v0.29.0`. The SSA builder's
type substitution logic constructs a function signature where
`jsontext.Value` (a named type aliasing `[]byte`) appears as a variadic
parameter. `go/types.NewSignatureType` enforces that variadic params must be
unnamed slice or string types and panics.

## Trigger

`github.com/go-json-experiment/json` is an indirect dependency via
`tailscale.com@v1.94.1`. No `GOEXPERIMENT=jsonv2` flag is needed — the
presence of the module in the dependency graph is sufficient.

## Affected Versions

- Go 1.25.x
- `govulncheck` 1.1.4 (latest as of 2026-03-04)
- `golang.org/x/tools` v0.29.0 (bundled in govulncheck)

## Related Issues

- [golang/go#73871](https://github.com/golang/go/issues/73871) — root cause
  (open)
- [golang/go#75584](https://github.com/golang/go/issues/75584) — closed as
  dup of #73871
- [golang/go#74846](https://github.com/golang/go/issues/74846) — closed as
  dup of #73871
- [CL 689277](https://go-review.googlesource.com/c/go/+/689277) — proposed
  fix in `go/types`

## Workaround

None available from the dodder side. The crash is in govulncheck's bundled
toolchain. Options:

1. Skip `check-go-vuln` until a fixed govulncheck is released
2. Wait for the upstream fix to land in a Go point release or new
   govulncheck version

Removing `tailscale.com` to eliminate the transitive dependency is not viable.

## Resolution

Will self-resolve when either:
- `govulncheck` upgrades its bundled `golang.org/x/tools` past the fix
- A Go point release includes the `go/types` fix from CL 689277
