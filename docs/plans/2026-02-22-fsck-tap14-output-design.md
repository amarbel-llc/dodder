# Design: TAP-14 Output for fsck Commands

## Goal

Replace the ad-hoc text output of all three fsck commands with TAP version 14,
making verification results machine-parseable and consistent with the project's
TAP-14 convention. Uses the `tap-dancer/go` library.

## Scope

All three fsck commands:

- `dodder fsck` — object verification (digests, signatures, probes, blobs)
- `dodder repo-fsck` — inventory list integrity
- `madder fsck` — blob store integrity

## Decisions

| Decision | Choice |
|---|---|
| Default or opt-in | Always TAP-14 (replaces current output) |
| Progress output | TAP comments (`# ...`) |
| Plan placement | Trailing (count unknown upfront) |
| Granularity (`dodder fsck`) | One test point per object |
| Multi-store (`madder fsck`) | Single TAP stream, store boundaries as comments |
| YAML diagnostics | Structured fields extracted via type assertions |

## TAP-14 Output Formats

### `dodder fsck`

One test point per object. All checks (digest, signature, probes, blobs) run for
each object; failures aggregate into YAML diagnostics.

```
TAP version 14
# verification for ":" objects in progress...
# (in progress) 5 verified, 0 errors
ok 1 - [md !md:zz-test/one @3e8a92]
ok 2 - [md !md:zz-test/two @7fb301]
not ok 3 - [md !md:zz-test/three @a1c4d0]
  ---
  message: "blob verification failed: digest mismatch"
  severity: fail
  expected: "sha256:abc123..."
  actual: "sha256:def456..."
  ...
1..3
```

### `dodder repo-fsck`

One test point per inventory list object.

```
TAP version 14
ok 1 - [inventory_list @ab12cd]
not ok 2 - [inventory_list @ef34gh]
  ---
  message: "blob missing"
  severity: fail
  ...
1..2
```

### `madder fsck`

One test point per blob, single stream across all stores.

```
TAP version 14
# (blob_store: local) starting fsck...
# (blob_store: local) 10 blobs / 1.2 MiB verified, 0 errors
ok 1 - sha256:abc123...
not ok 2 - sha256:def456...
  ---
  message: "blob verification failed"
  severity: fail
  expected: "sha256:aaa..."
  actual: "sha256:bbb..."
  ...
# (blob_store: remote-s3) starting fsck...
ok 3 - sha256:789ghi...
1..3
```

## YAML Diagnostic Field Extraction

Error types are inspected via type assertions to extract structured fields
beyond the flat message string.

### `markl.ErrNotEqual` (blob/digest mismatch)

Richest error type. Fields: `Expected`, `Actual` (both `MarklId`).

```yaml
---
message: "expected digest \"sha256:abc\" but got \"sha256:def\""
severity: fail
expected: "sha256:abc123..."
actual: "sha256:def456..."
...
```

### `errIsNull` (null digest/signature)

Field: `purpose` (string). The `errIsNull` type is unexported, so extraction
requires either exporting it or using `errors.As` with an interface. Fallback:
parse the `purpose` from the error message or add a `Purpose() string` method.

```yaml
---
message: "markl id is null for purpose \"object-dig\""
severity: fail
field: "object-dig"
...
```

### Probe mismatch (`errors.Errorf`)

No structured type — expected/actual are embedded in the format string. Include
the full message only.

```yaml
---
message: "probe \"objectid\" points to wrong object: expected foo@123, got bar@456"
severity: fail
...
```

### Fallback

Any error without a recognized structured type:

```yaml
---
message: "the error string"
severity: fail
...
```

## Structural Changes

### `dodder fsck` — per-object error accumulation

Current code collects all errors globally, then prints them at the end. For TAP
output, emit one test point per object as verification completes. Restructure:

1. Inside the object iteration loop, run all checks (digest, sig, probes, blobs)
2. Collect errors for the current object into a local slice
3. If no errors: `tw.Ok(objectDescription)`
4. If errors: `tw.NotOk(objectDescription, yamlDiagnostics)`
5. Move to the next object

This replaces the global `objectErrors` slice and the post-loop error printing.

### `dodder repo-fsck` — inline test points

Replace the two-phase approach (collect missing, then print) with inline TAP
emission during the iteration.

### `madder fsck` — single writer across stores

Create the `tap.Writer` once before the blob store loop. Use `tw.Comment()` to
mark store boundaries. Emit `tw.Plan()` after all stores complete.

### Progress tickers

Replace `ui.Out().Printf(...)` in ticker callbacks with `tw.Comment(...)`. The
TAP writer must be accessible from the ticker closure.

## Files Modified

| File | Change |
|---|---|
| `go/src/yankee/commands_dodder/fsck.go` | Rewrite `runVerification` to emit TAP |
| `go/src/yankee/commands_dodder/repo_fsck.go` | Rewrite `Run` to emit TAP |
| `go/src/lima/commands_madder/fsck.go` | Rewrite `Run` to emit TAP |
| `go/src/bravo/ui/main.go` | Remove `// TODO add a TAP printer` comment |
| `go/src/lima/commands_madder/sync.go` | Remove `// TODO output TAP` comment (separate task) |
| `zz-tests_bats/fsck.bats` | Update assertions for TAP output |
| `go.mod` / `go.sum` | Add `tap-dancer/go` dependency |

## Dependencies

- `github.com/amarbel-llc/tap-dancer/go` — TAP-14 writer library (already a
  flake input, needs `go get`)

## Error Type Visibility

`errIsNull` in `echo/markl/errors.go` is unexported. Options:

1. Export it as `ErrIsNull` (preferred — follows `ErrNotEqual` pattern)
2. Add an `IsNullPurpose() string` interface and implement it
3. Fall back to message-only YAML for null errors

Recommend option 1 for consistency with `ErrNotEqual`.

## BATS Test Updates

Current assertions like:
```bash
assert_output --partial "verification complete"
assert_output --partial "objects with errors: 0"
```

Become:
```bash
assert_output --partial "TAP version 14"
assert_output --partial "1.."
refute_output --partial "not ok"
```
