# Blob Reference Sorting Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Make blob reference output deterministic by sorting entries
lexicographically by markl.Id string, matching the pattern already used by
ContainedObjects for tags and referenced objects.

**Architecture:** Add a compare function for `blobReferenceEntry` and call
`SortWithComparer` after every `Add()`, mirroring the `ContainedObjects.Add()`
pattern. Test with a Go unit test and two BATS integration tests (hyphence show
output and inventory list output).

**Tech Stack:** Go, BATS

**Rollback:** N/A --- purely additive sort; existing insertion-order behavior
was never guaranteed.

--------------------------------------------------------------------------------

### Task 1: Add sorting to BlobReferences.Add()

**Promotion criteria:** N/A

**Files:** - Modify: `go/internal/delta/objects/blob_reference.go:31-42`

**Step 1: Add compare function and sort call**

Add a compare function at the top of `blob_reference.go` (after the type
declarations), and add a sort call to `Add()`:

``` go
// Add after line 15 (after blobReferenceEntry struct closing brace):
func blobReferenceEntryCompareKey(left, right blobReferenceEntry) cmp.Result {
    return cmp.CompareUTF8String(left.Key.String(), right.Key.String(), false)
}
```

Add `"code.linenisgreat.com/dodder/go/lib/alfa/cmp"` to the imports.

Modify `Add()` to sort after appending --- add one line after the `Append` call:

``` go
func (refs *BlobReferences) Add(
    id markl.Id,
    typeLock markl.Lock[ids.SeqId, *ids.SeqId],
) {
    for _, entry := range refs.entries {
        if markl.Equals(&entry.Key, &id) {
            return
        }
    }

    refs.entries.Append(blobReferenceEntry{Key: id, TypeLock: typeLock})
    refs.entries.SortWithComparer(blobReferenceEntryCompareKey)
}
```

**Step 2: Verify it compiles**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/ready-ebony && go build ./go/...`
Expected: success, no errors

**Step 3: Commit**

    fix(objects): sort blob references for deterministic output

    BlobReferences.Add() now sorts entries after insertion, matching the
    ContainedObjects pattern. Without this, blob reference order in box
    format, hyphence format, and inventory lists depended on insertion
    order, which varies across deserialization paths.

--------------------------------------------------------------------------------

### Task 2: Add Go unit test for BlobReferences sorting

**Promotion criteria:** N/A

**Files:** - Create: `go/internal/delta/objects/blob_reference_test.go`

**Step 1: Write the test**

Create `go/internal/delta/objects/blob_reference_test.go`:

``` go
package objects

import (
    "testing"

    "code.linenisgreat.com/dodder/go/internal/bravo/ids"
    "code.linenisgreat.com/dodder/go/internal/bravo/markl"
    "code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
)

func TestBlobReferencesAddSortsByKey(t *testing.T) {
    var refs BlobReferences

    // Create three markl.Id values whose String() representations sort as:
    // blake2b256-aaa... < blake2b256-mmm... < blake2b256-zzz...
    // We add them in reverse order to verify sorting.
    ids := makeThreeMarklIds(t)

    // Add in reverse order: zzz, mmm, aaa
    refs.Add(ids[2], markl.Lock[SeqId, *SeqId]{})
    refs.Add(ids[1], markl.Lock[SeqId, *SeqId]{})
    refs.Add(ids[0], markl.Lock[SeqId, *SeqId]{})

    // Collect results
    var got []string
    for id := range refs.All() {
        got = append(got, id.String())
    }

    if len(got) != 3 {
        t.Fatalf("expected 3 entries, got %d", len(got))
    }

    for i := 1; i < len(got); i++ {
        if got[i-1] >= got[i] {
            t.Errorf(
                "blob references not sorted: got[%d]=%q >= got[%d]=%q",
                i-1, got[i-1], i, got[i],
            )
        }
    }
}

func makeThreeMarklIds(t *testing.T) [3]markl.Id {
    t.Helper()

    // Use real blake2b256 digests with different content to get distinct,
    // sortable String() values.
    format, err := markl.GetFormatOrError("blake2b256")
    if err != nil {
        t.Fatalf("getting blake2b256 format: %v", err)
    }

    size := format.GetSize()
    var result [3]markl.Id

    for i := range result {
        data := make([]byte, size)
        // Fill with different byte values to get distinct blech32 encodings
        for j := range data {
            data[j] = byte((i + 1) * 50) // 50, 100, 150
        }
        if err := result[i].SetMarklId("blake2b256", data); err != nil {
            t.Fatalf("setting markl id %d: %v", i, err)
        }
    }

    // Sort expected order by string so we know which is "first"
    ordered := collections_slice.Slice[markl.Id](result[:])
    ordered.SortByStringFunc(func(id markl.Id) string { return id.String() })
    copy(result[:], ordered)

    return result
}
```

**Step 2: Run test to verify it passes**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/ready-ebony && go test -v -tags test,debug ./go/internal/delta/objects/ -run TestBlobReferencesAddSortsByKey`
Expected: PASS

**Step 3: Commit**

    test(objects): verify BlobReferences.Add() produces sorted output

--------------------------------------------------------------------------------

### Task 3: Add BATS test for sorted blob references in hyphence and inventory list output

**Promotion criteria:** N/A

**Files:** - Modify: `zz-tests_bats/current_version/show.bats`

**Step 1: Write the BATS test for hyphence output**

Add after the `show_box_format_includes_blob_references` test (after line 1076):

``` bash
# bats test_tags=user_story:referenced_objects
function show_blob_references_sorted_in_hyphence { # @test
    run_dodder init-workspace
    assert_success

    # Create a type whose reference discovery outputs multiple typed blob refs
    cat >ref-multi.type <<-'TYPEFILE'
        ---
        ! toml-type-v1
        ---

        file-extension = 'md'
        vim-syntax-type = 'markdown'

        [references]
        shell = ['bash', '-c']
        script = "grep -oP '@blake2b256-[a-z0-9]+' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !md/'"
    TYPEFILE

    run_dodder checkin -delete ref-multi.type
    assert_success

    # Create a zettel referencing two blobs — the second sorts before the first
    # lexicographically. Content order: zzz... then 9ft3... (9 < z).
    run_dodder new -edit=false - <<-EOM
        ---
        # multi blob refs
        ! ref-multi
        ---

        First @blake2b256-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz and second @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd here.
    EOM
    assert_success

    # Show in hyphence (text) format — blob refs must appear sorted
    run_dodder show -format text two/uno:
    assert_success

    # The 9ft3... digest sorts before zzz... lexicographically
    # Verify the sorted order by checking that 9ft3 appears before zzz
    local line_9ft3 line_zzz
    line_9ft3=$(echo "$output" | grep -n '@blake2b256-9ft3' | head -1 | cut -d: -f1)
    line_zzz=$(echo "$output" | grep -n '@blake2b256-zzzz' | head -1 | cut -d: -f1)

    [[ -n "$line_9ft3" ]] || fail "blob ref 9ft3 not found in output"
    [[ -n "$line_zzz" ]] || fail "blob ref zzzz not found in output"
    [[ "$line_9ft3" -lt "$line_zzz" ]] || fail "blob refs not sorted: 9ft3 (line $line_9ft3) should appear before zzzz (line $line_zzz)"
}
```

**Step 2: Write the BATS test for inventory list (box format) output**

Add after the previous test:

``` bash
# bats test_tags=user_story:referenced_objects
function show_blob_references_sorted_in_inventory_list { # @test
    run_dodder init-workspace
    assert_success

    # Create a type whose reference discovery outputs multiple typed blob refs
    cat >ref-multi.type <<-'TYPEFILE'
        ---
        ! toml-type-v1
        ---

        file-extension = 'md'
        vim-syntax-type = 'markdown'

        [references]
        shell = ['bash', '-c']
        script = "grep -oP '@blake2b256-[a-z0-9]+' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !md/'"
    TYPEFILE

    run_dodder checkin -delete ref-multi.type
    assert_success

    # Create a zettel referencing two blobs in non-sorted content order
    run_dodder new -edit=false - <<-EOM
        ---
        # multi blob refs
        ! ref-multi
        ---

        First @blake2b256-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz and second @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd here.
    EOM
    assert_success

    # Show the inventory list in box format (default show for :b type)
    # Use `show` with the zettel id to get box format output which includes blob refs
    run_dodder show two/uno
    assert_success

    # In box format, blob refs appear as "<@digest !type@sig" fields
    # Verify 9ft3 appears before zzzz in the output
    local pos_9ft3 pos_zzz
    pos_9ft3=$(echo "$output" | grep -boP '<@blake2b256-9ft3' | head -1 | cut -d: -f1)
    pos_zzz=$(echo "$output" | grep -boP '<@blake2b256-zzzz' | head -1 | cut -d: -f1)

    [[ -n "$pos_9ft3" ]] || fail "blob ref 9ft3 not found in box output"
    [[ -n "$pos_zzz" ]] || fail "blob ref zzzz not found in box output"
    [[ "$pos_9ft3" -lt "$pos_zzz" ]] || fail "blob refs not sorted in box format: 9ft3 (pos $pos_9ft3) should appear before zzzz (pos $pos_zzz)"
}
```

**Step 2: Build and run the failing tests first (before Task 1)**

If implementing TDD, run these tests before the fix to see them fail:

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/ready-ebony && just test-bats-targets show.bats`

**Step 3: After Task 1 fix is applied, run to verify pass**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/ready-ebony && just test-bats-targets show.bats`
Expected: all blob reference tests PASS

**Step 4: Commit**

    test: verify blob references are sorted in hyphence and box format output

--------------------------------------------------------------------------------

### Task 4: Run full test suite

**Promotion criteria:** N/A

**Step 1: Run full tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/ready-ebony && just test`
Expected: all tests PASS --- existing tests should not break since sorted output
is a strict subset of possible insertion orders.
