# `-omit-tags` Transform Pipeline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add `-omit-tags <regex>` flag to `der import` that strips matching tags from objects before plan classification, backed by an `ObjectTransform` pipeline on the Builder.

**Architecture:** Add an `ObjectTransform` function type and a `transforms` slice to the Builder. Transforms run in `AddObject` before classification. The CLI constructs the omit-tags transform from flag values and passes it via a new `Builder.AddTransform` method. The tag omission transform iterates tags, collects non-matching ones, then uses `ResetTags()`+`AddTagString()` to rebuild the tag set.

**Tech Stack:** Go stdlib `regexp`, existing `ids.Tag`/`metadata.ResetTags()`/`AddTagString()` APIs.

**Rollback:** Purely additive — revert the commits.

---

### Task 1: ObjectTransform type and Builder.AddTransform

**Files:**
- Modify: `go/internal/india/import_plan/builder.go`

**Step 1: Add the ObjectTransform type and transforms field**

Add to `builder.go` after the imports:

```go
// ObjectTransform mutates an object before plan classification.
// Return true to keep the object, false to drop it entirely.
type ObjectTransform func(*sku.Transacted) (keep bool, err error)
```

Add `transforms []ObjectTransform` field to the `Builder` struct.

**Step 2: Add Builder.AddTransform method**

```go
func (b *Builder) AddTransform(t ObjectTransform) {
	b.transforms = append(b.transforms, t)
}
```

**Step 3: Apply transforms in AddObject**

In `AddObject`, after the config genre skip (line 94) and before `b.abbrIndex.AddObject` (line 96), add:

```go
for _, transform := range b.transforms {
	keep, err := transform(object)
	if err != nil {
		return
	}
	if !keep {
		return
	}
}
```

**Step 4: Build and verify compilation**

Run: `just build` from repo root.
Expected: Compiles cleanly. No test changes needed — transforms slice is nil by default, so existing behavior is unchanged.

**Step 5: Commit**

```
feat: add ObjectTransform pipeline to import plan Builder

Transforms run in AddObject before classification. Each transform can
mutate the object (keep=true) or drop it (keep=false). When no
transforms are registered, behavior is unchanged.
```

---

### Task 2: OmitTagsTransform function

**Files:**
- Create: `go/internal/india/import_plan/transform_omit_tags.go`

**Step 1: Write unit test**

Create `go/internal/india/import_plan/transform_omit_tags_test.go`:

```go
//go:build test

package import_plan

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/golf/sku"
)

func TestOmitTagsTransformRemovesMatchingTags(t *testing.T) {
	transform, err := MakeOmitTagsTransform([]string{"^tag-[12]$"})
	if err != nil {
		t.Fatal(err)
	}

	var object sku.Transacted
	object.GetMetadataMutable().AddTagString("tag-1")
	object.GetMetadataMutable().AddTagString("tag-2")
	object.GetMetadataMutable().AddTagString("tag-3")

	keep, err := transform(&object)
	if err != nil {
		t.Fatal(err)
	}
	if !keep {
		t.Fatal("expected keep=true")
	}

	tags := collectTagStrings(&object)
	if len(tags) != 1 || tags[0] != "tag-3" {
		t.Fatalf("expected [tag-3], got %v", tags)
	}
}

func TestOmitTagsTransformKeepsAllWhenNoMatch(t *testing.T) {
	transform, err := MakeOmitTagsTransform([]string{"^archived$"})
	if err != nil {
		t.Fatal(err)
	}

	var object sku.Transacted
	object.GetMetadataMutable().AddTagString("tag-1")
	object.GetMetadataMutable().AddTagString("tag-2")

	keep, err := transform(&object)
	if err != nil {
		t.Fatal(err)
	}
	if !keep {
		t.Fatal("expected keep=true")
	}

	tags := collectTagStrings(&object)
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", tags)
	}
}

func TestOmitTagsTransformMultiplePatterns(t *testing.T) {
	transform, err := MakeOmitTagsTransform([]string{"^tag-1$", "^tag-3$"})
	if err != nil {
		t.Fatal(err)
	}

	var object sku.Transacted
	object.GetMetadataMutable().AddTagString("tag-1")
	object.GetMetadataMutable().AddTagString("tag-2")
	object.GetMetadataMutable().AddTagString("tag-3")

	keep, err := transform(&object)
	if err != nil {
		t.Fatal(err)
	}
	if !keep {
		t.Fatal("expected keep=true")
	}

	tags := collectTagStrings(&object)
	if len(tags) != 1 || tags[0] != "tag-2" {
		t.Fatalf("expected [tag-2], got %v", tags)
	}
}

func TestOmitTagsTransformRemovesAllTags(t *testing.T) {
	transform, err := MakeOmitTagsTransform([]string{".*"})
	if err != nil {
		t.Fatal(err)
	}

	var object sku.Transacted
	object.GetMetadataMutable().AddTagString("tag-1")

	keep, err := transform(&object)
	if err != nil {
		t.Fatal(err)
	}
	if !keep {
		t.Fatal("expected keep=true even when all tags removed")
	}

	tags := collectTagStrings(&object)
	if len(tags) != 0 {
		t.Fatalf("expected no tags, got %v", tags)
	}
}

func TestOmitTagsTransformInvalidRegex(t *testing.T) {
	_, err := MakeOmitTagsTransform([]string{"[invalid"})
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func collectTagStrings(object *sku.Transacted) []string {
	var result []string
	for tag := range object.GetMetadata().AllTags() {
		result = append(result, tag.String())
	}
	return result
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -tags test,debug ./go/internal/india/import_plan/` from repo root (or `just test-go` for all).
Expected: FAIL — `MakeOmitTagsTransform` undefined.

**Step 3: Write implementation**

Create `go/internal/india/import_plan/transform_omit_tags.go`:

```go
package import_plan

import (
	"regexp"

	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func MakeOmitTagsTransform(
	patterns []string,
) (ObjectTransform, error) {
	compiled := make([]*regexp.Regexp, len(patterns))

	for i, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid -omit-tags pattern: %q", pattern)
		}

		compiled[i] = re
	}

	return func(object *sku.Transacted) (bool, error) {
		var kept []string

		for tag := range object.GetMetadata().AllTags() {
			tagString := tag.String()
			matched := false

			for _, re := range compiled {
				if re.MatchString(tagString) {
					matched = true
					break
				}
			}

			if !matched {
				kept = append(kept, tagString)
			}
		}

		metadata := object.GetMetadataMutable()
		metadata.ResetTags()

		for _, tagString := range kept {
			if err := metadata.AddTagString(tagString); err != nil {
				return false, errors.Wrap(err)
			}
		}

		return true, nil
	}, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test -v -tags test,debug ./go/internal/india/import_plan/`
Expected: All 5 tests PASS.

**Step 5: Commit**

```
feat: add MakeOmitTagsTransform for stripping tags by regex

Compiles patterns at construction time, iterates tags per object,
rebuilds tag set excluding matches. Always returns keep=true — tag
omission never drops objects.
```

---

### Task 3: Wire `-omit-tags` flag in CLI

**Files:**
- Modify: `go/internal/victor/commands_dodder/import.go`

**Step 1: Add OmitTags field and flag registration**

Add to the `Import` struct:

```go
OmitTags []string
```

Note: Go's `flag` package doesn't support repeated string flags natively. Use a
custom `flag.Value` implementation, or use the existing pattern in the codebase.
Check if dodder has a string slice flag type — if not, use a single
comma-separated flag or accumulate via a custom type.

Look at how other repeated flags work in dodder. If there's no precedent,
define a minimal `stringSliceFlag` type locally in import.go:

```go
type stringSliceFlag []string

func (f *stringSliceFlag) String() string { return fmt.Sprintf("%v", *f) }
func (f *stringSliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
```

Register in `SetFlagDefinitions`:

```go
flagDefinitions.Var(
	(*stringSliceFlag)(&cmd.OmitTags),
	"omit-tags",
	"regex pattern for tags to strip during import (repeatable)",
)
```

**Step 2: Construct and register transform in Run**

In `Run()`, after creating the builder (line 105) and before the inventory list
loop (line 107), add:

```go
if len(cmd.OmitTags) > 0 {
	transform, err := import_plan.MakeOmitTagsTransform(cmd.OmitTags)
	if err != nil {
		local.Cancel(errors.Wrap(err))
		return
	}
	builder.AddTransform(transform)
}
```

**Step 3: Build and verify compilation**

Run: `just build`
Expected: Compiles cleanly.

**Step 4: Commit**

```
feat: wire -omit-tags flag to import command

Repeatable flag that registers an OmitTagsTransform on the plan builder.
Patterns are compiled once at startup; invalid regex produces an
immediate error.
```

---

### Task 4: Integration test

**Files:**
- Modify: `zz-tests_bats/current_version/import.bats`

**Step 1: Add integration test**

Add at the end of import.bats:

```bash
function import_omit_tags { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
    popd || exit 1
  )

  # Export from outer repo (objects have tag-1, tag-2, tag-3, tag-4)
  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  pushd inner || exit 1

  # Import with -omit-tags stripping tag-1 and tag-2
  run_dodder import \
    -blob_store-id shared \
    -omit-tags "^tag-[12]$" \
    "$list"
  assert_success

  # Verify imported objects have tag-3 and tag-4 but not tag-1 or tag-2
  run_dodder show one/uno
  assert_success
  # The latest version should only have tag-3 and tag-4
  refute_output --partial "tag-1"
  refute_output --partial "tag-2"
  assert_output --partial "tag-3"
  assert_output --partial "tag-4"
}
```

**Step 2: Run the test**

Run: `just test-bats-targets import.bats`
Expected: All tests PASS including the new `import_omit_tags` test.

**Step 3: If test fails, debug and fix**

Use `xxd` or `cat -A` for invisible whitespace issues. Check that the tag
strings match exactly (tags are stored without `#` prefix in dodder output).

**Step 4: Run full test suite**

Run: `just test`
Expected: All tests pass (unit + integration).

**Step 5: Commit**

```
test: add integration test for -omit-tags import flag

Imports objects with tag-1 through tag-4, uses -omit-tags to strip
tag-1 and tag-2, verifies only tag-3 and tag-4 survive.
```

---

### Task 5: Update implementation status in FDR

**Files:**
- Modify: `docs/features/0002-two-phase-import.md`

**Step 1: Move transform pipeline from "not yet implemented" to "implemented"**

Move "Object transform pipeline and `-omit-tags` flag" from "Not yet
implemented" to "Implemented (experimental)".

**Step 2: Commit**

```
docs: mark -omit-tags and transform pipeline as implemented in FDR-0002
```
