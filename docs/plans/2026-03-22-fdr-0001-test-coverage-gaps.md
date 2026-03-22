# FDR-0001 Test Coverage Gaps Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Close the moderate-priority test gaps identified in
[#36](https://github.com/amarbel-llc/dodder/issues/36) for FDR-0001 (Object
Locks): discovery script error handling, multiple blob refs with heterogeneous
types, and blob reference alias cross-format round-trips.

**Architecture:** Go unit tests for `parseReferenceOutput` error cases; BATS
integration tests for multi-type blob refs and alias round-trips across
hyphence, box, and pull (which exercises binary stream index) formats.

**Tech Stack:** Go `testing`, BATS (`bats-assert`), existing dodder test helpers
(`run_dodder`, `assert_success`, `assert_output --regexp`).

**Rollback:** N/A --- purely additive test coverage.

--------------------------------------------------------------------------------

## Task 1: Discovery Script --- Malformed Output

Unit tests for `parseReferenceOutput` handling garbage, partial lines, and mixed
valid/invalid content.

**Files:** - Modify: `go/internal/papa/store/reference_discovery_test.go`

**Step 1: Write failing tests**

Add to `reference_discovery_test.go`:

``` go
func TestParseReferenceOutputBinaryGarbage(t1 *testing.T) {
    t := ui.T{T: t1}

    input := "one/dos\n\x00\xff\xfe binary garbage\ntwo/uno\n"
    refs, err := parseReferenceOutput(input)
    t.AssertNoError(err)

    // Parser should yield refs for lines it can parse; garbage lines become
    // object refs with the raw string (they'll fail downstream validation,
    // but parseReferenceOutput itself shouldn't panic or error).
    if len(refs) != 3 {
        t.Fatalf("expected 3 refs, got %d", len(refs))
    }

    t.AssertEqualStrings("one/dos", refs[0].ObjectId)
    t.AssertEqualStrings("two/uno", refs[2].ObjectId)
}

func TestParseReferenceOutputPartialBlobRef(t1 *testing.T) {
    t := ui.T{T: t1}

    // Blob ref without digest — just "@"
    input := "@\none/dos\n"
    refs, err := parseReferenceOutput(input)
    t.AssertNoError(err)

    if len(refs) != 2 {
        t.Fatalf("expected 2 refs, got %d", len(refs))
    }

    // Empty blob ID from bare "@"
    t.AssertEqualStrings("", refs[0].BlobId)
    t.AssertEqualStrings("one/dos", refs[1].ObjectId)
}

func TestParseReferenceOutputBlobRefWithAlias(t1 *testing.T) {
    t := ui.T{T: t1}

    input := "hero = @blake2b256-abc123 !image-png\n"
    refs, err := parseReferenceOutput(input)
    t.AssertNoError(err)

    if len(refs) != 1 {
        t.Fatalf("expected 1 ref, got %d", len(refs))
    }

    t.AssertEqualStrings("blake2b256-abc123", refs[0].BlobId)
    t.AssertEqualStrings("hero", refs[0].Alias)
    t.AssertEqualStrings("!image-png", refs[0].TypeId)
}
```

**Step 2: Run tests to verify they pass (or fail as expected)**

Run:
`go test -v -tags test,debug -run TestParseReferenceOutput ./internal/papa/store/`
from `go/` directory.

Evaluate: the binary garbage and partial blob ref tests document current
behavior. If any panic, that's a bug to fix before proceeding. If they pass
as-is, they serve as regression tests.

**Step 3: Commit**

    test: add parseReferenceOutput edge case coverage (#36)

--------------------------------------------------------------------------------

## Task 2: Discovery Script --- Non-Zero Exit and Crash

BATS integration test verifying that a crashing discovery script is handled
gracefully (either error or skip, depending on `optional` flag).

**Files:** - Modify: `zz-tests_bats/current_version/show.bats`

**Step 1: Write failing test for required script crash**

Add after `blob_reference_without_type_fails`:

``` bash
# bats test_tags=user_story:referenced_objects
function discovery_script_crash_required_fails { # @test
    run_dodder init-workspace
    assert_success

    # Type with required discovery script that exits non-zero
    cat >crashy.type <<-'TYPEFILE'
        ---
        ! toml-type-v1
        ---

        file-extension = 'md'

        [references]
        shell = ['bash', '-c']
        script = 'exit 1'
    TYPEFILE

    run_dodder checkin -delete crashy.type
    assert_success

    run_dodder new -edit=false - <<-EOM
        ---
        # zettel with crashy type
        ! crashy
        ---

        content here
    EOM

    # Required script crash should cause commit failure
    assert_failure
}
```

**Step 2: Write test for optional script crash**

``` bash
# bats test_tags=user_story:referenced_objects
function discovery_script_crash_optional_succeeds { # @test
    run_dodder init-workspace
    assert_success

    # Type with optional discovery script that exits non-zero
    cat >crashy-opt.type <<-'TYPEFILE'
        ---
        ! toml-type-v1
        ---

        file-extension = 'md'

        [references]
        optional = true
        shell = ['bash', '-c']
        script = 'exit 1'
    TYPEFILE

    run_dodder checkin -delete crashy-opt.type
    assert_success

    run_dodder new -edit=false - <<-EOM
        ---
        # zettel with optional crashy type
        ! crashy-opt
        ---

        content here
    EOM

    # Optional script crash should succeed silently
    assert_success
}
```

**Step 3: Run tests**

Run: `just test-bats-targets current_version/show.bats`

Evaluate: if either test fails unexpectedly, the error handling in
`discoverReferences` (lines 126-145 of `reference_discovery.go`) needs
investigation. The `MakeWriterToWithStdin` / `WriteTo` path should propagate the
exit code as an error, which `discoverReferences` checks against
`objectReferences.Optional`.

**Step 4: Commit**

    test: add discovery script crash handling tests (#36)

--------------------------------------------------------------------------------

## Task 3: Multiple Blob References with Heterogeneous Types

BATS integration test for an object with 2+ blob references, each with a
different type.

**Files:** - Modify: `zz-tests_bats/current_version/show.bats`

**Step 1: Write the test**

Add after `show_blob_references_sorted_in_inventory_list`:

``` bash
# bats test_tags=user_story:referenced_objects
function show_blob_references_with_heterogeneous_types { # @test
    run_dodder init-workspace
    assert_success

    # Create two types: one for images, one for data
    cat >img.type <<-'TYPEFILE'
        ---
        ! toml-type-v1
        ---
        file-extension = 'png'
    TYPEFILE

    cat >data.type <<-'TYPEFILE'
        ---
        ! toml-type-v1
        ---
        file-extension = 'csv'
    TYPEFILE

    run_dodder checkin -delete img.type
    assert_success

    run_dodder checkin -delete data.type
    assert_success

    # Create a zettel with two blob refs of different types
    run_dodder new -edit=false - <<-EOM
        ---
        # mixed types
        - @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !img
        - @blake2b256-qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsk2yde5 !data
        ! md
        ---

        content
    EOM
    assert_success

    # Verify both blob refs appear with their respective types in text format
    run_dodder show -format text two/uno:
    assert_success
    assert_output --regexp '@blake2b256-9ft3.+ !img'
    assert_output --regexp '@blake2b256-qyqs.+ !data'

    # Verify box format also has both
    run_dodder show two/uno
    assert_success
    assert_output --regexp '<@blake2b256-9ft3.+ !img'
    assert_output --regexp '<@blake2b256-qyqs.+ !data'
}
```

**Step 2: Run test**

Run: `just test-bats-targets current_version/show.bats`

Evaluate: if either assertion fails, the issue is in how multiple blob refs with
different type locks are stored or formatted. Check
`delta/objects/blob_reference.go` (collection storage) and
`foxtrot/object_metadata_fmt_hyphence/formatter_components.go` (hyphence
output).

**Step 3: Commit**

    test: verify heterogeneous type locks on blob references (#36)

--------------------------------------------------------------------------------

## Task 4: Blob Reference Alias Round-Trip Through Box Format

The existing alias tests only verify hyphence. This tests box format (which is
used for inventory lists and `show` default output).

**Files:** - Modify: `zz-tests_bats/current_version/show.bats`

**Step 1: Write the test**

Add after `blob_reference_alias_with_quotes_round_trips`:

``` bash
# bats test_tags=user_story:referenced_objects
function blob_reference_alias_round_trips_through_box_format { # @test
    run_dodder init-workspace
    assert_success

    # Create a zettel with an aliased blob reference
    run_dodder new -edit=false - <<-'EOM'
        ---
        # alias box test
        - hero-image < @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md
        ! md
        ---

        content
    EOM
    assert_success

    # Box format should include the alias
    run_dodder show two/uno
    assert_success
    assert_output --regexp 'hero-image<@blake2b256-9ft3'
}
```

**Step 2: Run test**

Run: `just test-bats-targets current_version/show.bats`

Evaluate: if the alias doesn't appear in box output, check
`echo/object_metadata_box_builder/main.go:AddBlobReferences` --- it prepends
`alias + "<@"` for aliased blob refs. This should work but may not round-trip
through read if the box parser doesn't handle the alias prefix.

**Step 3: Commit**

    test: verify blob reference alias in box format output (#36)

--------------------------------------------------------------------------------

## Task 5: Blob Reference Alias Round-Trip Through Pull (Binary Index)

Pull exercises the binary stream index encoder/decoder. This verifies aliases
survive the full write → binary encode → binary decode → read cycle.

**Files:** - Modify: `zz-tests_bats/current_version/pull.bats`

**Step 1: Write the test**

Add after `pull_direct_hyphenated_type_name_no_phantom`:

``` bash
# bats test_tags=user_story:pull,user_story:referenced_objects
function pull_direct_blob_reference_alias_survives { # @test
    them="$BATS_TEST_TMPDIR/them"
    mkdir -p "$them"

    pushd "$them" || exit 1

    run_dodder_init_disable_age

    # Create a type for image blobs
    cat >img.type <<-'TYPEFILE'
        ---
        ! toml-type-v1
        ---
        file-extension = 'png'
    TYPEFILE

    run_dodder checkin -delete img.type
    assert_success

    # Create a zettel with an aliased blob reference
    run_dodder new -edit=false - <<-'EOM'
        ---
        # aliased blob ref
        - hero-image < @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !img
        ! md
        ---

        content
    EOM
    assert_success

    # Verify alias exists in source
    run_dodder show -format text one/uno:
    assert_success
    assert_output --partial 'hero-image'

    popd || exit 1

    # Set up destination repo
    us="$BATS_TEST_TMPDIR/us"
    mkdir -p "$us"
    pushd "$us" || exit 1

    run_dodder_init_disable_age

    # Pull from source
    run_dodder pull -direct "$(realpath "$them")" +zettel,typ,etikett
    assert_success

    # Verify alias survived the pull (binary stream index round-trip)
    run_dodder show -format text one/uno:
    assert_success
    assert_output --partial 'hero-image'
}
```

**Step 2: Run test**

Run: `just test-bats-targets current_version/pull.bats`

Evaluate: if the alias disappears after pull, the issue is in
`india/stream_index/binary_encoder.go` / `binary_decoder.go` --- specifically
the `BlobReferences` key encoding, where alias is the "remaining bytes" after
blob ID and type lock. Check that the alias field is written and read back.

**Step 3: Commit**

    test: verify blob reference alias survives pull via binary index (#36)

--------------------------------------------------------------------------------

## Task 6: Run Full Test Suite and Update Issue

**Step 1: Run full suite**

Run: `just test`

Expected: all tests pass (existing + new).

**Step 2: Update #36**

Comment on the issue noting which gaps are now covered: - Discovery script error
handling: malformed output (unit), crash with required/optional (integration) -
Multiple blob refs with heterogeneous types (integration) - Blob reference
aliases: box format (integration), binary index via pull (integration)

Remaining uncovered from #36: - GC reachability → moved to #39

**Step 3: Commit all together if not already committed per-task**

    docs: update FDR-0001 test coverage status (#36)
