# FSck TAP-14 Output Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace ad-hoc text output in all three fsck commands with TAP version 14.

**Architecture:** Each fsck command creates a `tap.NewWriter(os.Stdout)` and
emits test points inline during verification. Structured error fields are
extracted via type assertions into YAML diagnostic maps. Progress tickers become
TAP comments.

**Tech Stack:** Go, `github.com/amarbel-llc/tap-dancer/go` (TAP-14 writer),
`bats` (integration tests)

**Design doc:** `docs/plans/2026-02-22-fsck-tap14-output-design.md`

---

### Task 1: Add tap-dancer Go dependency

**Files:**
- Modify: `go/go.mod`
- Modify: `go/go.sum`

**Step 1: Add the dependency**

Run from `go/`:
```bash
go get github.com/amarbel-llc/tap-dancer/go@latest
```

**Step 2: Verify import works**

Run:
```bash
cd go && go build ./...
```
Expected: builds successfully (no new code uses it yet, just verifying resolution)

**Step 3: Commit**

```bash
git add go/go.mod go/go.sum
git commit -m "deps: add tap-dancer/go for TAP-14 output"
```

---

### Task 2: Export `ErrIsNull` in `echo/markl`

The `errIsNull` type is unexported. Export it so fsck can extract the `Purpose`
field for YAML diagnostics. Follow the existing `ErrNotEqual` pattern.

**Files:**
- Modify: `go/src/echo/markl/errors.go:77-92`

**Step 1: Write the failing test**

Create `go/src/echo/markl/errors_test.go`:

```go
package markl_test

import (
	"errors"
	"testing"

	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

func TestErrIsNullPurposeExtractable(t *testing.T) {
	id := markl.MakeNullId()
	err := markl.AssertIdIsNotNullWithPurpose(id, "object-dig")

	var errIsNull markl.ErrIsNull
	if !errors.As(err, &errIsNull) {
		t.Fatalf("expected ErrIsNull, got %T", err)
	}

	if errIsNull.Purpose != "object-dig" {
		t.Errorf("expected purpose %q, got %q", "object-dig", errIsNull.Purpose)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./src/echo/markl/ -run TestErrIsNullPurposeExtractable`
Expected: FAIL — `ErrIsNull` not defined

**Step 3: Export the type**

In `go/src/echo/markl/errors.go`, rename `errIsNull` to `ErrIsNull` and export
the `purpose` field to `Purpose`:

```go
type ErrIsNull struct {
	Purpose string
}

func (err ErrIsNull) Error() string {
	return fmt.Sprintf("markl id is null for purpose %q", err.Purpose)
}

func (err ErrIsNull) Is(target error) bool {
	_, ok := target.(ErrIsNull)
	return ok
}

func (err ErrIsNull) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}
```

Update all references in the same file:
- `AssertIdIsNotNull`: `errIsNull{purpose:` → `ErrIsNull{Purpose:`
- `AssertIdIsNotNullWithPurpose`: same
- `IsErrNull`: `errIsNull{}` → `ErrIsNull{}`

**Step 4: Run test to verify it passes**

Run: `cd go && go test -v -tags test,debug ./src/echo/markl/ -run TestErrIsNullPurposeExtractable`
Expected: PASS

**Step 5: Run full build to check nothing breaks**

Run: `cd go && go build ./...`
Expected: builds clean

**Step 6: Commit**

```bash
git add go/src/echo/markl/errors.go go/src/echo/markl/errors_test.go
git commit -m "refactor: export ErrIsNull for structured TAP-14 diagnostics"
```

---

### Task 3: Create TAP diagnostic builder helper

Create a shared helper that extracts structured fields from fsck error types
into `map[string]string` for `tap.NotOk()`. This avoids duplicating
type-assertion logic across three commands.

**Files:**
- Create: `go/src/bravo/ui/tap_diagnostics.go`
- Create: `go/src/bravo/ui/tap_diagnostics_test.go`

**Step 1: Write the failing test**

Create `go/src/bravo/ui/tap_diagnostics_test.go`:

```go
package ui_test

import (
	"fmt"
	"testing"

	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

func TestTapDiagnosticsFromErrNotEqual(t *testing.T) {
	expected := markl.MakeIdFromString("sha256:aaa")
	actual := markl.MakeIdFromString("sha256:bbb")
	err := markl.ErrNotEqual{Expected: expected, Actual: actual}

	diag := ui.TapDiagnosticsFromError(err)

	if diag["severity"] != "fail" {
		t.Errorf("expected severity fail, got %q", diag["severity"])
	}
	if diag["expected"] == "" {
		t.Error("expected 'expected' field to be set")
	}
	if diag["actual"] == "" {
		t.Error("expected 'actual' field to be set")
	}
}

func TestTapDiagnosticsFromErrIsNull(t *testing.T) {
	err := markl.ErrIsNull{Purpose: "object-dig"}

	diag := ui.TapDiagnosticsFromError(err)

	if diag["severity"] != "fail" {
		t.Errorf("expected severity fail, got %q", diag["severity"])
	}
	if diag["field"] != "object-dig" {
		t.Errorf("expected field %q, got %q", "object-dig", diag["field"])
	}
}

func TestTapDiagnosticsFromGenericError(t *testing.T) {
	err := fmt.Errorf("something went wrong")

	diag := ui.TapDiagnosticsFromError(err)

	if diag["severity"] != "fail" {
		t.Errorf("expected severity fail, got %q", diag["severity"])
	}
	if diag["message"] != "something went wrong" {
		t.Errorf("expected message, got %q", diag["message"])
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./src/bravo/ui/ -run TestTapDiagnostics`
Expected: FAIL — `TapDiagnosticsFromError` not defined

**Step 3: Implement the helper**

Create `go/src/bravo/ui/tap_diagnostics.go`:

```go
package ui

import (
	"errors"
	"fmt"

	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

func TapDiagnosticsFromError(err error) map[string]string {
	diag := map[string]string{
		"severity": "fail",
		"message":  err.Error(),
	}

	var errNotEqual markl.ErrNotEqual
	if errors.As(err, &errNotEqual) {
		diag["expected"] = fmt.Sprintf("%s", errNotEqual.Expected)
		diag["actual"] = fmt.Sprintf("%s", errNotEqual.Actual)
		return diag
	}

	var errIsNull markl.ErrIsNull
	if errors.As(err, &errIsNull) {
		diag["field"] = errIsNull.Purpose
		return diag
	}

	return diag
}
```

Note: `bravo/ui` importing `echo/markl` follows the NATO hierarchy (bravo <
echo). Verify this is acceptable — if not, move the helper to a package at
`echo` level or higher. Check existing imports in `bravo/ui` first.

**Important:** If `bravo/ui` cannot import `echo/markl` due to the NATO
hierarchy, move this helper to `go/src/echo/fsck_diagnostics/` or
`go/src/hotel/tap_diagnostics/` instead and adjust all references below.

**Step 4: Run test to verify it passes**

Run: `cd go && go test -v -tags test,debug ./src/bravo/ui/ -run TestTapDiagnostics`
Expected: PASS

**Step 5: Commit**

```bash
git add go/src/bravo/ui/tap_diagnostics.go go/src/bravo/ui/tap_diagnostics_test.go
git commit -m "feat: add TapDiagnosticsFromError for structured TAP-14 YAML"
```

---

### Task 4: Rewrite `dodder fsck` to emit TAP-14

This is the largest change. Restructure `runVerification` to emit one TAP test
point per object inline, with YAML diagnostics on failure.

**Files:**
- Modify: `go/src/yankee/commands_dodder/fsck.go:86-235`

**Step 1: Write the failing BATS test**

Update `zz-tests_bats/fsck.bats`, replacing the first test:

```bash
function fsck_basic_tap14 { # @test
  run_dodder_init_disable_age

  run_dodder fsck
  assert_success
  assert_output --partial "TAP version 14"
  assert_output --partial "1.."
  refute_output --partial "not ok"
}
```

**Step 2: Run to verify it fails**

Run: `just test-bats-targets fsck.bats`
Expected: FAIL — output still contains "verification complete", not "TAP version 14"

**Step 3: Rewrite `runVerification` in `fsck.go`**

Replace the `Run` and `runVerification` methods. Key changes:

1. Create `tw := tap.NewWriter(os.Stdout)` at the top of `Run`
2. Replace `ui.Out().Printf("verification for...")` with `tw.Comment(...)`
3. In `runVerification`, iterate objects. For each object:
   - Run all checks (digest null, finalizer verify, probes, blobs)
   - Collect errors for THIS object into a local slice
   - If no errors: `tw.Ok(sku.StringMetadataTaiMerkle(object))`
   - If errors: build `map[string]string` via `ui.TapDiagnosticsFromError`,
     call `tw.NotOk(sku.StringMetadataTaiMerkle(object), diag)`
4. Replace progress ticker with `tw.Comment(fmt.Sprintf(...))`
5. After the loop: `tw.Plan()` (trailing plan)
6. Remove the post-loop error printing code entirely

```go
func (cmd Fsck) Run(req command.Request) {
	repo := cmd.MakeLocalWorkingCopy(req)

	tw := tap.NewWriter(os.Stdout)

	var seq interfaces.SeqError[*sku.Transacted]

	if cmd.InventoryListPath == "" {
		query := cmd.MakeQuery(
			req,
			queries.BuilderOptions(
				queries.BuilderOptionDefaultGenres(genres.All()...),
				queries.BuilderOptionDefaultSigil(
					ids.SigilLatest,
					ids.SigilHistory,
					ids.SigilHidden,
				),
			),
			repo,
			req.PopArgs(),
		)

		seq = repo.GetStore().All(query)

		tw.Comment(fmt.Sprintf("verification for %q objects in progress...", query))
	} else {
		seq = cmd.MakeSeqFromPath(
			repo,
			repo.GetInventoryListCoderCloset(),
			cmd.InventoryListPath,
			nil,
		)
	}

	cmd.runVerification(repo, seq, tw)
}

func (cmd Fsck) runVerification(
	repo *local_working_copy.Repo,
	seq interfaces.SeqError[*sku.Transacted],
	tw *tap.Writer,
) {
	var count atomic.Uint32
	var errorCount atomic.Uint32

	finalizer := object_finalizer.Builder().
		WithVerifyOptions(cmd.VerifyOptions).
		Build()

	if err := errors.RunChildContextWithPrintTicker(
		repo,
		func(ctx errors.Context) {
			for object, errIter := range seq {
				if errIter != nil {
					desc := "iteration error"
					if object != nil {
						desc = sku.StringMetadataTaiMerkle(object)
					}
					tw.NotOk(desc, ui.TapDiagnosticsFromError(errIter))
					errorCount.Add(1)
					continue
				}

				var objectErrors []error

				if err := markl.AssertIdIsNotNull(
					object.GetObjectDigest(),
				); err != nil {
					objectErrors = append(objectErrors, err)
				}

				if err := finalizer.Verify(object); err != nil {
					objectErrors = append(objectErrors, err)
				}

				if !cmd.SkipProbes {
					if err := repo.GetStore().GetStreamIndex().VerifyObjectProbes(
						object,
					); err != nil {
						objectErrors = append(objectErrors, err)
					}
				}

				if !cmd.SkipBlobs {
					blobDigest := object.GetBlobDigest()
					if !blobDigest.IsNull() {
						if err := blob_stores.VerifyBlob(
							repo,
							repo.GetEnvRepo().GetDefaultBlobStore(),
							blobDigest,
							io.Discard,
						); err != nil {
							objectErrors = append(objectErrors,
								errors.Wrapf(err, "blob verification failed"))
						}
					}
				}

				desc := sku.StringMetadataTaiMerkle(object)

				if len(objectErrors) == 0 {
					tw.Ok(desc)
				} else {
					// Use the first error for diagnostics
					diag := ui.TapDiagnosticsFromError(objectErrors[0])

					// If multiple errors, concatenate messages
					if len(objectErrors) > 1 {
						var msgs string
						for i, e := range objectErrors {
							if i > 0 {
								msgs += "; "
							}
							msgs += e.Error()
						}
						diag["message"] = msgs
					}

					tw.NotOk(desc, diag)
					errorCount.Add(1)
				}

				count.Add(1)
			}
		},
		func(time time.Time) {
			tw.Comment(fmt.Sprintf(
				"(in progress) %d verified, %d errors",
				count.Load(),
				errorCount.Load(),
			))
		},
		3*time.Second,
	); err != nil {
		tw.BailOut(err.Error())
		repo.Cancel(err)
		return
	}

	tw.Plan()
}
```

Add to imports:
```go
"fmt"
"os"
tap "github.com/amarbel-llc/tap-dancer/go"
```

Remove unused imports: `"code.linenisgreat.com/dodder/go/src/bravo/collections_slice"`

**Step 4: Build and run the test**

Run: `just build && just test-bats-targets fsck.bats`
Expected: `fsck_basic_tap14` passes

**Step 5: Commit**

```bash
git add go/src/yankee/commands_dodder/fsck.go zz-tests_bats/fsck.bats
git commit -m "feat: dodder fsck emits TAP-14 output"
```

---

### Task 5: Update remaining `fsck.bats` tests for TAP-14

Update all remaining test functions in `fsck.bats` to assert TAP-14 format.

**Files:**
- Modify: `zz-tests_bats/fsck.bats`

**Step 1: Update all tests**

Replace all `assert_output --partial "verification complete"` and
`assert_output --partial "objects with errors: 0"` with TAP assertions:

```bash
assert_output --partial "TAP version 14"
assert_output --partial "1.."
refute_output --partial "not ok"
```

For tests with objects (e.g., `fsck_with_objects`, `fsck_multiple_objects`),
also assert:

```bash
assert_output --partial "ok "
```

**Step 2: Run all fsck tests**

Run: `just test-bats-targets fsck.bats`
Expected: all tests pass

**Step 3: Commit**

```bash
git add zz-tests_bats/fsck.bats
git commit -m "test: update fsck.bats assertions for TAP-14 output"
```

---

### Task 6: Rewrite `dodder repo-fsck` to emit TAP-14

**Files:**
- Modify: `go/src/yankee/commands_dodder/repo_fsck.go`

**Step 1: Write a BATS test for repo-fsck TAP output**

Check if a repo-fsck bats test exists. If not, add one to `fsck.bats`:

```bash
function repo_fsck_tap14 { # @test
  run_dodder_init_disable_age

  run_dodder repo-fsck
  assert_success
  assert_output --partial "TAP version 14"
  assert_output --partial "1.."
  refute_output --partial "not ok"
}
```

**Step 2: Run to verify it fails**

Run: `just test-bats-targets fsck.bats`
Expected: FAIL — repo-fsck still outputs old format

**Step 3: Rewrite repo_fsck.go**

```go
func (cmd RepoFsck) Run(req command.Request) {
	req.AssertNoMoreArgs()

	repo := cmd.MakeLocalWorkingCopyWithOptions(
		req,
		env_ui.Options{},
		local_working_copy.OptionsAllowConfigReadError,
	)

	tw := tap.NewWriter(os.Stdout)

	store := repo.GetStore()

	for objectWithList, err := range store.GetInventoryListStore().AllInventoryListObjectsAndContents() {
		errors.ContextContinueOrPanic(repo)

		if err == nil {
			tw.Ok(sku.String(objectWithList.List))
			continue
		}

		diag := ui.TapDiagnosticsFromError(err)

		if env_dir.IsErrBlobMissing(err) {
			diag["message"] = "blob missing"
		}

		tw.NotOk(sku.String(objectWithList.List), diag)
	}

	tw.Plan()
}
```

Add imports: `"os"`, `tap "github.com/amarbel-llc/tap-dancer/go"`.
Remove unused imports: `"code.linenisgreat.com/dodder/go/src/juliett/sku"` if
`sku.String` is still needed (it is). Remove
`"code.linenisgreat.com/dodder/go/src/bravo/ui"` if no longer used.

**Step 4: Build and run**

Run: `just build && just test-bats-targets fsck.bats`
Expected: PASS

**Step 5: Commit**

```bash
git add go/src/yankee/commands_dodder/repo_fsck.go zz-tests_bats/fsck.bats
git commit -m "feat: dodder repo-fsck emits TAP-14 output"
```

---

### Task 7: Rewrite `madder fsck` to emit TAP-14

**Files:**
- Modify: `go/src/lima/commands_madder/fsck.go`

**Step 1: Write a BATS test**

If a madder fsck bats test exists, update it. Otherwise add to an appropriate
file. Check `zz-tests_bats/` for existing madder tests. Add:

```bash
function madder_fsck_tap14 { # @test
  run_dodder_init_disable_age

  run_dodder blob_store-fsck
  assert_success
  assert_output --partial "TAP version 14"
  assert_output --partial "1.."
  refute_output --partial "not ok"
}
```

(madder commands are exposed in dodder under the `blob_store-` prefix)

**Step 2: Run to verify it fails**

Run: `just test-bats-targets fsck.bats`
Expected: FAIL

**Step 3: Rewrite madder fsck.go**

Single `tap.Writer` across all stores. Store boundaries as comments.

```go
func (cmd Fsck) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)

	blobStores := cmd.MakeBlobStoresFromIdsOrAll(req, envBlobStore)

	tw := tap.NewWriter(os.Stdout)

	for _, blobStore := range blobStores {
		storeId := fmt.Sprintf("%s", blobStore.Path.GetId())

		tw.Comment(fmt.Sprintf("(blob_store: %s) starting fsck...", storeId))

		var count int
		var progressWriter env_ui.ProgressWriter
		var blobErrors collections_slice.Slice[command_components_madder.BlobError]

		if err := errors.RunChildContextWithPrintTicker(
			envBlobStore,
			func(ctx errors.Context) {
				for digest, err := range blobStore.AllBlobs() {
					errors.ContextContinueOrPanic(ctx)

					if err != nil {
						tw.NotOk("(unknown blob)", ui.TapDiagnosticsFromError(err))
						continue
					}

					count++

					if !blobStore.HasBlob(digest) {
						tw.NotOk(
							fmt.Sprintf("%s", digest),
							map[string]string{
								"severity": "fail",
								"message":  "blob missing",
							},
						)
						continue
					}

					if err = blob_stores.VerifyBlob(
						ctx,
						blobStore,
						digest,
						&progressWriter,
					); err != nil {
						tw.NotOk(
							fmt.Sprintf("%s", digest),
							ui.TapDiagnosticsFromError(err),
						)
						continue
					}

					tw.Ok(fmt.Sprintf("%s", digest))
				}
			},
			func(time time.Time) {
				tw.Comment(fmt.Sprintf(
					"(blob_store: %s) %d blobs / %s verified",
					storeId,
					count,
					progressWriter.GetWrittenHumanString(),
				))
			},
			3*time.Second,
		); err != nil {
			tw.BailOut(err.Error())
			envBlobStore.Cancel(err)
			return
		}

		tw.Comment(fmt.Sprintf(
			"(blob_store: %s) blobs verified: %d, bytes: %s",
			storeId,
			count,
			progressWriter.GetWrittenHumanString(),
		))
	}

	tw.Plan()
}
```

Add imports: `"os"`, `tap "github.com/amarbel-llc/tap-dancer/go"`.
Remove unused imports as needed (`ui` prefix printer, `PrintBlobErrors` if
no longer called).

**Step 4: Build and run**

Run: `just build && just test-bats-targets fsck.bats`
Expected: PASS

**Step 5: Commit**

```bash
git add go/src/lima/commands_madder/fsck.go zz-tests_bats/fsck.bats
git commit -m "feat: madder fsck emits TAP-14 output"
```

---

### Task 8: Clean up TODO comments

**Files:**
- Modify: `go/src/bravo/ui/main.go:85`
- Modify: `go/src/lima/commands_madder/sync.go:60` (leave as-is — sync is separate scope)

**Step 1: Remove the resolved TODO**

In `go/src/bravo/ui/main.go`, remove line 85: `// TODO add a TAP printer`

**Step 2: Build**

Run: `cd go && go build ./...`
Expected: clean

**Step 3: Commit**

```bash
git add go/src/bravo/ui/main.go
git commit -m "chore: remove resolved TAP printer TODO"
```

---

### Task 9: Run full test suite

**Step 1: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 2: Run full integration tests**

Run: `just test-bats`
Expected: all tests pass

**Step 3: Build release**

Run: `just build`
Expected: clean build

---

### Task 10: Update fixtures if needed

If any fixture-dependent tests changed behavior:

**Step 1: Regenerate fixtures**

Run: `just test-bats-update-fixtures`

**Step 2: Review diff**

Run: `git diff -- zz-tests_bats/migration/`

**Step 3: Commit if changed**

```bash
git add zz-tests_bats/migration/
git commit -m "test: regenerate fixtures for TAP-14 fsck output"
```
