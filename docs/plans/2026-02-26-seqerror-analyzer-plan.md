# SeqError Static Analyzer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a `go vet`-compatible static analyzer that forbids dropping the
error value when ranging over `iter.Seq2[T, error]`.

**Architecture:** AST-only `go/analysis` pass. Walk `*ast.RangeStmt` nodes,
type-check the range expression for `iter.Seq2[T, error]` shape, then inspect
the loop body for qualifying error usage (nil-check with scope exit, or yield
pass-through). Mirrors the existing repool analyzer's structure and test harness.

**Tech Stack:** `golang.org/x/tools/go/analysis`, `go/ast`, `go/types`,
`analysistest`

**Design doc:** `docs/plans/2026-02-26-seqerror-analyzer-design.md`

**Reference implementation:** `go/lib/alfa/analyzers/repool/` — same directory
layout, test harness, build/invocation pattern.

---

### Task 1: Scaffold and test harness with first failing test case

**Files:**
- Create: `go/lib/alfa/analyzers/seqerror/analyzer.go`
- Create: `go/lib/alfa/analyzers/seqerror/analyzer_test.go`
- Create: `go/lib/alfa/analyzers/seqerror/testdata/src/a/a.go`
- Create: `go/lib/alfa/analyzers/seqerror/cmd/main.go`

**Step 1: Create testdata with a single blank-error test case**

The `analysistest` framework uses `// want "..."` comments to assert diagnostics.
The testdata package needs a helper that returns an `iter.Seq2[T, error]` so the
analyzer has something to detect.

Create `go/lib/alfa/analyzers/seqerror/testdata/src/a/a.go`:

```go
package a

import "iter"

func makeSeq() iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {}
}

// --- Rule 1: blank error variable ---

func blankError() {
	for x, _ := range makeSeq() { // want "error from iter.Seq2 range is discarded"
		_ = x
	}
}
```

**Step 2: Create the test file**

Create `go/lib/alfa/analyzers/seqerror/analyzer_test.go`:

```go
package seqerror_test

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/alfa/analyzers/seqerror"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, seqerror.Analyzer, "a")
}
```

**Step 3: Create minimal analyzer stub**

Create `go/lib/alfa/analyzers/seqerror/analyzer.go` with a stub `Analyzer` var
and empty `run` function that returns `(nil, nil)`:

```go
package seqerror

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var Analyzer = &analysis.Analyzer{
	Name:     "seqerror",
	Doc:      "check error from iter.Seq2 range is not discarded",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (any, error) {
	return nil, nil
}
```

**Step 4: Create cmd entry point**

Create `go/lib/alfa/analyzers/seqerror/cmd/main.go`:

```go
package main

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/analyzers/seqerror"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(seqerror.Analyzer)
}
```

**Step 5: Run the test, verify it fails**

Run from `go/`:
```
go test -v -tags test ./src/alfa/analyzers/seqerror/
```

Expected: FAIL — the `// want` annotation expects a diagnostic but the stub
produces none.

**Step 6: Commit scaffold**

```
git add go/lib/alfa/analyzers/seqerror/
git commit -m "test: scaffold seqerror analyzer with blank-error test case"
```

---

### Task 2: Implement Rule 1 — blank error variable detection

**Files:**
- Modify: `go/lib/alfa/analyzers/seqerror/analyzer.go`

**Step 1: Implement `isSeq2Error` type checker**

Add a function that takes a `types.Type` and returns `true` if its underlying
type matches `func(func(T, error) bool)`:

```go
func isSeq2Error(t types.Type) bool {
	sig, ok := t.Underlying().(*types.Signature)
	if !ok || sig.Params().Len() != 1 {
		return false
	}

	yieldSig, ok := sig.Params().At(0).Type().Underlying().(*types.Signature)
	if !ok || yieldSig.Params().Len() != 2 || yieldSig.Results().Len() != 1 {
		return false
	}

	// Second param of yield must be the error interface
	errParam := yieldSig.Params().At(1).Type()
	return types.Identical(errParam, types.Universe.Lookup("error").Type())
}
```

**Step 2: Implement the AST walk and blank-error detection**

Fill in the `run` function to use `inspector.Preorder` on `*ast.RangeStmt`
nodes. For each, type-check the range expression with `pass.TypesInfo.TypeOf`.
If it's a Seq2-error type and `.Value` is `*ast.Ident` with `Name == "_"`,
report — unless `//seq:err-checked` comment is on the same line.

The suppression check should mirror repool's `hasRepoolOwnedComment`: iterate
over `pass.Files` comment groups looking for `//seq:err-checked` on the same
line as the range statement.

**Step 3: Run the test, verify it passes**

```
go test -v -tags test ./src/alfa/analyzers/seqerror/
```

Expected: PASS — the blank-error case is now detected.

**Step 4: Commit**

```
git add go/lib/alfa/analyzers/seqerror/analyzer.go
git commit -m "feat(seqerror): detect blank error variable in Seq2 range"
```

---

### Task 3: Add blank-error suppression test case

**Files:**
- Modify: `go/lib/alfa/analyzers/seqerror/testdata/src/a/a.go`

**Step 1: Add suppression test case to testdata**

Append to `a.go`:

```go
func blankErrorSuppressed() {
	for x, _ := range makeSeq() { //seq:err-checked
		_ = x
	}
}
```

No `// want` annotation — this should produce no diagnostic.

**Step 2: Run test, verify it passes**

```
go test -v -tags test ./src/alfa/analyzers/seqerror/
```

Expected: PASS.

**Step 3: Commit**

```
git add go/lib/alfa/analyzers/seqerror/testdata/
git commit -m "test(seqerror): add blank-error suppression case"
```

---

### Task 4: Add test cases for Rule 2 — unchecked named error (failing)

**Files:**
- Modify: `go/lib/alfa/analyzers/seqerror/testdata/src/a/a.go`

**Step 1: Add failing test cases for named-but-unchecked errors**

Append to `a.go`:

```go
// --- Rule 2: named but unchecked error ---

func namedButUnchecked() {
	for x, err := range makeSeq() { // want `error variable "err" from iter.Seq2 range is never checked or propagated`
		_ = err
		_ = x
	}
}

func namedButEmptyCheck() {
	for x, err := range makeSeq() { // want `error variable "err" is checked but not handled`
		if err != nil {
		}
		_ = x
	}
}
```

**Step 2: Run test, verify it fails**

```
go test -v -tags test ./src/alfa/analyzers/seqerror/
```

Expected: FAIL — analyzer doesn't detect these yet.

**Step 3: Commit failing tests**

```
git add go/lib/alfa/analyzers/seqerror/testdata/
git commit -m "test(seqerror): add unchecked-error and empty-check test cases"
```

---

### Task 5: Implement Rule 2 — unchecked named error detection

**Files:**
- Modify: `go/lib/alfa/analyzers/seqerror/analyzer.go`

**Step 1: Implement body analysis**

After confirming the range statement has a named (non-blank) error variable,
analyze the loop body. Resolve the error variable's `*types.Var` via
`pass.TypesInfo.Defs` or `pass.TypesInfo.Uses`.

Check for qualifying usage by walking the body AST:

1. **Yield pass-through**: Any `*ast.CallExpr` where the error variable appears
   as an argument. If found, the usage is valid — return early.

2. **If-check with scope exit**: An `*ast.IfStmt` whose `Cond` references the
   error variable. If found, inspect the `Body` for at least one: `*ast.ReturnStmt`,
   `*ast.BranchStmt` (break/continue), expression statement calling `panic`, or
   a `*ast.CallExpr` that passes the error variable.

If the `IfStmt` is found but its body has no qualifying statement, report the
"checked but not handled" diagnostic.

If neither yield pass-through nor if-check is found, report the "never checked
or propagated" diagnostic.

Helper functions to add:
- `condReferencesVar(cond ast.Expr, v *types.Var, info *types.Info) bool` —
  walks the condition expression looking for the variable
- `bodyHasQualifyingUsage(body *ast.BlockStmt, v *types.Var, info *types.Info) bool` —
  checks for return/break/continue/panic or call passing the error
- `callPassesVar(call *ast.CallExpr, v *types.Var, info *types.Info) bool` —
  checks if any argument to the call is the variable

**Step 2: Run test, verify it passes**

```
go test -v -tags test ./src/alfa/analyzers/seqerror/
```

Expected: PASS.

**Step 3: Commit**

```
git add go/lib/alfa/analyzers/seqerror/analyzer.go
git commit -m "feat(seqerror): detect unchecked and unhandled error variables"
```

---

### Task 6: Add test cases for valid patterns (OK cases)

**Files:**
- Modify: `go/lib/alfa/analyzers/seqerror/testdata/src/a/a.go`

**Step 1: Add all valid-pattern test cases**

Append to `a.go`. None of these have `// want` annotations — they must produce
no diagnostics:

```go
// --- Valid patterns (no diagnostics expected) ---

func checkedWithReturn() (string, error) {
	for x, err := range makeSeq() {
		if err != nil {
			return "", err
		}
		_ = x
	}
	return "", nil
}

func checkedWithContinue() {
	for x, err := range makeSeq() {
		if err != nil {
			continue
		}
		_ = x
	}
}

func checkedWithBreak() {
	for x, err := range makeSeq() {
		if err != nil {
			break
		}
		_ = x
	}
}

func checkedWithPanic() {
	for x, err := range makeSeq() {
		if err != nil {
			panic(err)
		}
		_ = x
	}
}

func checkedWithFuncCall() {
	for x, err := range makeSeq() {
		if err != nil {
			handleErr(err)
			return
		}
		_ = x
	}
}

func handleErr(err error) {}

func yieldPassThrough() {
	_ = func(yield func(string, error) bool) {
		for x, err := range makeSeq() {
			if !yield(x, err) {
				return
			}
		}
	}
}

func yieldWithNilCheck() {
	_ = func(yield func(string, error) bool) {
		for x, err := range makeSeq() {
			if err != nil {
				yield("", err)
				return
			}
			if !yield(x, nil) {
				return
			}
		}
	}
}
```

**Step 2: Run test, verify it passes**

```
go test -v -tags test ./src/alfa/analyzers/seqerror/
```

Expected: PASS — none of the valid patterns trigger a diagnostic.

**Step 3: Commit**

```
git add go/lib/alfa/analyzers/seqerror/testdata/
git commit -m "test(seqerror): add valid-pattern test cases"
```

---

### Task 7: Add test case for ranging over non-error Seq2 (no false positive)

**Files:**
- Modify: `go/lib/alfa/analyzers/seqerror/testdata/src/a/a.go`

**Step 1: Add non-error Seq2 test case**

This confirms the analyzer does NOT fire on `iter.Seq2[string, int]` or
`iter.Seq[string]`:

```go
// --- Non-error sequences (no diagnostics expected) ---

func nonErrorSeq2() {
	seq := func(yield func(string, int) bool) {}
	for x, i := range seq {
		_ = x
		_ = i
	}
}

func plainSeq() {
	seq := func(yield func(string) bool) {}
	for x := range seq {
		_ = x
	}
}
```

**Step 2: Run test, verify it passes**

```
go test -v -tags test ./src/alfa/analyzers/seqerror/
```

Expected: PASS.

**Step 3: Commit**

```
git add go/lib/alfa/analyzers/seqerror/testdata/
git commit -m "test(seqerror): add non-error sequence false-positive guards"
```

---

### Task 8: Add test case for single-value range (no Value node)

**Files:**
- Modify: `go/lib/alfa/analyzers/seqerror/testdata/src/a/a.go`

**Step 1: Add single-value range test**

When someone writes `for x := range seqError` (only one variable), the `Value`
field of `*ast.RangeStmt` is nil. The error is implicitly dropped. This should
be flagged:

```go
func singleValueRange() {
	for x := range makeSeq() { // want "error from iter.Seq2 range is discarded"
		_ = x
	}
}
```

**Step 2: Run test, check if it fails**

```
go test -v -tags test ./src/alfa/analyzers/seqerror/
```

If it fails, update the analyzer to also flag when `Value == nil` on a Seq2Error
range. This is the same diagnostic as Rule 1 (blank error).

**Step 3: If needed, update analyzer to handle nil Value**

In the `run` function, after checking `isSeq2Error`, add a check: if
`rangeStmt.Value == nil`, report the blank-error diagnostic (unless suppressed).

**Step 4: Run test, verify it passes**

```
go test -v -tags test ./src/alfa/analyzers/seqerror/
```

Expected: PASS.

**Step 5: Commit**

```
git add go/lib/alfa/analyzers/seqerror/
git commit -m "feat(seqerror): flag single-value range over Seq2 error"
```

---

### Task 9: Add justfile integration

**Files:**
- Modify: `go/justfile`

**Step 1: Add build and check recipes**

After the `check-go-repool` recipe block, add:

```just
build-analyzer-seqerror:
  go build -o build/seqerror-analyzer ./src/alfa/analyzers/seqerror/cmd/

check-go-seqerror: build-analyzer-seqerror
  go vet -vettool=build/seqerror-analyzer ./... || true
```

**Step 2: Update the `check` recipe**

Change:
```just
check: check-go-vuln check-go-vet check-go-repool
```
To:
```just
check: check-go-vuln check-go-vet check-go-repool check-go-seqerror
```

**Step 3: Build and run the analyzer against the full codebase**

```
just check-go-seqerror
```

Expected: exits 0 (the `|| true` ensures it). Review the output — the existing
codebase should have zero violations (the exploration confirmed all current code
handles errors correctly). If any violations appear, investigate whether they are
true positives or false positives and adjust the analyzer.

**Step 4: Commit**

```
git add go/justfile
git commit -m "feat(seqerror): add justfile build and check recipes"
```

---

### Task 10: Run full test suite and final verification

**Files:** none (verification only)

**Step 1: Run the analyzer's own tests**

```
go test -v -tags test ./src/alfa/analyzers/seqerror/
```

Expected: PASS.

**Step 2: Run the full check suite**

```
just check
```

Expected: all checks pass (vuln, vet, repool, seqerror).

**Step 3: Run unit tests to ensure no regressions**

```
just test-go
```

Expected: PASS.

**Step 4: Commit if any adjustments were needed, otherwise done**
