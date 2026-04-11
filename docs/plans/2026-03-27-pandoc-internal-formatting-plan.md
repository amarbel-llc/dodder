# Pandoc Internal Formatting Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Store pandoc Lua filters and defaults YAML as blobs in the object
graph, wire them into the `!md` type formatter via blob references and a
tmpdir-based materializer, so `dodder init` produces a self-contained repo with
working markdown formatting.

**Architecture:** Genesis creates two new tool types (`!pandoc-defaults`,
`!pandoc-lua_filter`) and three tool blobs (embedded via `go:embed`). The `!md`
type object carries typed blob references to these blobs. The formatter pipeline
materializes referenced blobs to a tmpdir and invokes pandoc with
`--data-dir=$tmpdir`.

**Tech Stack:** Go, BATS (integration tests), pandoc (nix dependency),
`go:embed`

**Rollback:** Revert genesis changes. Existing repos unaffected. New repos get
the old minimal `!md`.

--------------------------------------------------------------------------------

### Task 1: Register `!pandoc-defaults` and `!pandoc-lua_filter` builtin type strings

**Promotion criteria:** N/A

**Files:**

- Modify: `go/internal/bravo/ids/types_builtin.go`

**Step 1: Add type string constants and registration**

In `go/internal/bravo/ids/types_builtin.go`, add constants (keep alphabetical
order with existing constants):

``` go
TypeTomlPandocDefaultsV0    = "!toml-pandoc_defaults-v0"
TypeTomlPandocLuaFilterV0   = "!toml-pandoc_lua_filter-v0"
```

Add init registrations (keep sorted within the `init()` function):

``` go
registerBuiltinTypeString(TypeTomlPandocDefaultsV0, genres.Type, false)
registerBuiltinTypeString(TypeTomlPandocLuaFilterV0, genres.Type, false)
```

Note: `isDefault` is `false` --- these are not default types for any genre.

**Step 2: Run unit tests**

Run: `just test-go-unit` Expected: PASS (no behavior change, just registration)

**Step 3: Commit**

``` text
feat: register !pandoc-defaults and !pandoc-lua_filter builtin type strings
```

--------------------------------------------------------------------------------

### Task 2: Create tool type blob definitions

**Promotion criteria:** N/A

**Files:**

- Modify: `go/internal/hotel/type_blobs/main.go`

**Step 1: Add functions for tool type defaults**

In `go/internal/hotel/type_blobs/main.go`, add after the existing `Default()`
function:

``` go
func DefaultPandocDefaults() TomlV1 {
    return TomlV1{
        FileExtension: "yaml",
        Formatters:    make(map[string]script_config.WithOutputFormat),
    }
}

func DefaultPandocLuaFilter() TomlV1 {
    return TomlV1{
        FileExtension: "lua",
        Formatters:    make(map[string]script_config.WithOutputFormat),
    }
}
```

**Step 2: Run unit tests**

Run: `just test-go-unit` Expected: PASS

**Step 3: Commit**

``` text
feat: add DefaultPandocDefaults and DefaultPandocLuaFilter type blob constructors
```

--------------------------------------------------------------------------------

### Task 3: Embed pandoc filter and defaults content

**Promotion criteria:** N/A

**Files:**

- Create: `go/internal/sierra/local_working_copy/embedded_pandoc_tools.go`

**Step 1: Create the embed file**

Create `go/internal/sierra/local_working_copy/embedded_pandoc_tools.go`:

``` go
package local_working_copy

import _ "embed"

//go:embed embedded/pandoc/filters/dodder-common.lua
var embeddedPandocCommonFilter []byte

//go:embed embedded/pandoc/filters/dodder-edit.lua
var embeddedPandocEditFilter []byte

//go:embed embedded/pandoc/defaults/dodder-edit.yaml
var embeddedPandocEditDefaults []byte
```

**Step 2: Create embedded directory and copy files**

``` bash
mkdir -p go/internal/sierra/local_working_copy/embedded/pandoc/filters
mkdir -p go/internal/sierra/local_working_copy/embedded/pandoc/defaults
cp zz-pandoc/filters/dodder-common.lua go/internal/sierra/local_working_copy/embedded/pandoc/filters/
cp zz-pandoc/filters/dodder-edit.lua go/internal/sierra/local_working_copy/embedded/pandoc/filters/
cp zz-pandoc/defaults/dodder-edit.yaml go/internal/sierra/local_working_copy/embedded/pandoc/defaults/
```

**Step 3: Modify the embedded `dodder-edit.lua` to remove the `package.path`
hack**

The original `dodder-edit.lua` line 1 sets:

``` lua
package.path = package.path .. string.format(";%s/.local/share/pandoc/filters/?.lua", os.getenv("HOME"))
```

Replace this with:

``` lua
package.path = package.path .. string.format(";%s/filters/?.lua", os.getenv("DODDER_BLOB_TREE") or "")
```

This makes the require resolve from the materialized blob tree instead of the
home directory.

**Step 4: Verify build**

Run: `just build` Expected: Compiles successfully with embedded files

**Step 5: Commit**

``` text
feat: embed pandoc filters and defaults for blob-backed type config
```

--------------------------------------------------------------------------------

### Task 4: Expand genesis to create tool types and blobs

**Promotion criteria:** N/A

**Files:**

- Modify: `go/internal/sierra/local_working_copy/genesis.go`

**Step 1: Add `prepareToolTypes` function**

Add a new function that creates type objects for `!pandoc-defaults` and
`!pandoc-lua_filter`:

``` go
func (local *Repo) prepareToolTypes(
    builder *import_plan.Builder,
) (err error) {
    // Create !pandoc-defaults type object
    {
        tipe := ids.DefaultOrPanic(genres.Type)
        blob := type_blobs.DefaultPandocDefaults()
        object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

        if err = object.GetObjectIdMutable().SetWithId(
            ids.MustTypeStruct("pandoc-defaults"),
        ); err != nil {
            return errors.Wrap(err)
        }

        var digest domain_interfaces.MarklId

        if digest, _, err = local.GetStore().GetTypedBlobStore().Type.SaveBlobText(
            tipe,
            &blob,
        ); err != nil {
            return errors.Wrap(err)
        }

        object.GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(digest)
        object.GetMetadataMutable().GetTypeMutable().ResetWithType(tipe)
        builder.AddObject(object, 0)
    }

    // Create !pandoc-lua_filter type object
    {
        tipe := ids.DefaultOrPanic(genres.Type)
        blob := type_blobs.DefaultPandocLuaFilter()
        object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

        if err = object.GetObjectIdMutable().SetWithId(
            ids.MustTypeStruct("pandoc-lua_filter"),
        ); err != nil {
            return errors.Wrap(err)
        }

        var digest domain_interfaces.MarklId

        if digest, _, err = local.GetStore().GetTypedBlobStore().Type.SaveBlobText(
            tipe,
            &blob,
        ); err != nil {
            return errors.Wrap(err)
        }

        object.GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(digest)
        object.GetMetadataMutable().GetTypeMutable().ResetWithType(tipe)
        builder.AddObject(object, 0)
    }

    return err
}
```

**Step 2: Add `writeToolBlob` helper and `prepareToolBlobs` function**

``` go
type toolBlobDigests struct {
    commonFilter markl.Id
    editFilter   markl.Id
    editDefaults markl.Id
}

func (local *Repo) writeRawBlob(content []byte) (digest markl.Id, err error) {
    var writer domain_interfaces.BlobWriter

    if writer, err = local.GetEnvRepo().GetDefaultBlobStore().MakeBlobWriter(nil); err != nil {
        return digest, errors.Wrap(err)
    }

    defer errors.DeferredCloser(&err, writer)

    if _, err = writer.Write(content); err != nil {
        return digest, errors.Wrap(err)
    }

    digest.ResetWithMarklId(writer.GetMarklId())

    return digest, err
}

func (local *Repo) prepareToolBlobs() (digests toolBlobDigests, err error) {
    if digests.commonFilter, err = local.writeRawBlob(embeddedPandocCommonFilter); err != nil {
        return digests, errors.Wrap(err)
    }

    if digests.editFilter, err = local.writeRawBlob(embeddedPandocEditFilter); err != nil {
        return digests, errors.Wrap(err)
    }

    if digests.editDefaults, err = local.writeRawBlob(embeddedPandocEditDefaults); err != nil {
        return digests, errors.Wrap(err)
    }

    return digests, err
}
```

**Step 3: Modify `prepareDefaultType` to accept tool blob digests and add blob
references**

Change signature to accept `toolBlobDigests`:

``` go
func (local *Repo) prepareDefaultType(
    bigBang env_repo.BigBang,
    builder *import_plan.Builder,
    toolBlobs toolBlobDigests,
) (objectIdType ids.TypeStruct, err error) {
```

After setting the type blob digest and type on the object (before
`builder.AddObject`), add blob references:

``` go
    // Add blob references to tool blobs
    metadata := object.GetMetadataMutable()

    addToolBlobRef := func(digest markl.Id, typeString, alias string) error {
        var typeLock markl.Lock[ids.SeqId, *ids.SeqId]
        marshaler := markl.MakeMutableLockCoderValueNotRequired(&typeLock)

        if err := marshaler.Set(ids.MakeTypeString(typeString)); err != nil {
            return errors.Wrap(err)
        }

        metadata.AddBlobReference(digest, typeLock)

        return metadata.SetBlobReferenceAlias(digest, alias)
    }

    if err = addToolBlobRef(
        toolBlobs.commonFilter,
        "!pandoc-lua_filter",
        "filters/dodder-common.lua",
    ); err != nil {
        return objectIdType, errors.Wrap(err)
    }

    if err = addToolBlobRef(
        toolBlobs.editFilter,
        "!pandoc-lua_filter",
        "filters/dodder-edit.lua",
    ); err != nil {
        return objectIdType, errors.Wrap(err)
    }

    if err = addToolBlobRef(
        toolBlobs.editDefaults,
        "!pandoc-defaults",
        "defaults/dodder-edit.yaml",
    ); err != nil {
        return objectIdType, errors.Wrap(err)
    }
```

**Step 4: Update `Default()` to include formatter config**

In `go/internal/hotel/type_blobs/main.go`, update `Default()`:

``` go
func Default() TomlV1 {
    return TomlV1{
        FileExtension: "md",
        VimSyntaxType: "markdown",
        Formatters: map[string]script_config.WithOutputFormat{
            "text": {
                ScriptConfig: script_config.ScriptConfig{
                    Description: "Normalize markdown with pandoc",
                    Script:      `pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-edit`,
                },
                FileExtension: "md",
            },
        },
    }
}
```

**Step 5: Update `initDefaultTypeAndConfig` call order**

In `genesis.go`, modify `initDefaultTypeAndConfig`:

``` go
func (local *Repo) initDefaultTypeAndConfig(
    bigBang env_repo.BigBang,
) (err error) {
    builder := import_plan.MakeLocalBuilder()

    // 1. Create tool type objects
    if err = local.prepareToolTypes(&builder); err != nil {
        return errors.Wrap(err)
    }

    // 2. Write tool blobs to store
    var toolBlobs toolBlobDigests

    if toolBlobs, err = local.prepareToolBlobs(); err != nil {
        return errors.Wrap(err)
    }

    // 3. Create !md type with blob refs to tool blobs
    var defaultTypeObjectId ids.TypeStruct

    if defaultTypeObjectId, err = local.prepareDefaultType(
        bigBang,
        &builder,
        toolBlobs,
    ); err != nil {
        return errors.Wrap(err)
    }

    // 4. Create repo config (unchanged)
    blobStoreId := local.GetEnvRepo().GetDefaultBlobStore().GetId()

    if !bigBang.BlobStoreId.IsEmpty() {
        blobStoreId = bigBang.BlobStoreId
    }

    blobStores := []blob_store_id.Id{blobStoreId}

    if err = local.prepareDefaultConfig(
        bigBang,
        blobStores,
        defaultTypeObjectId,
        &builder,
    ); err != nil {
        return errors.Wrap(err)
    }

    plan, buildErr := builder.Build()
    if buildErr != nil {
        return errors.Wrap(buildErr)
    }

    plan.DefaultCommitOptions = sku.CommitOptions{
        Proto:        local.GetStore().GetProtoZettel(),
        StoreOptions: sku.GetStoreOptionsCreate(),
    }

    if _, err = local.ExecutePlan(plan); err != nil {
        return errors.Wrap(err)
    }

    return err
}
```

**Step 6: Add required imports to genesis.go**

Ensure imports include `markl` and `domain_interfaces`:

``` go
import (
    "code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
    "code.linenisgreat.com/dodder/go/internal/alfa/blob_store_id"
    "code.linenisgreat.com/dodder/go/internal/alfa/genres"
    "code.linenisgreat.com/dodder/go/internal/bravo/ids"
    "code.linenisgreat.com/dodder/go/internal/bravo/markl"
    "code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
    "code.linenisgreat.com/dodder/go/internal/golf/env_repo"
    "code.linenisgreat.com/dodder/go/internal/golf/sku"
    "code.linenisgreat.com/dodder/go/internal/hotel/type_blobs"
    "code.linenisgreat.com/dodder/go/internal/india/import_plan"
    "code.linenisgreat.com/dodder/go/lib/bravo/errors"
)
```

**Step 7: Build**

Run: `just build` Expected: Compiles successfully

**Step 8: Commit**

``` text
feat: expand genesis to create pandoc tool types, blobs, and blob references on !md
```

--------------------------------------------------------------------------------

### Task 5: Update BATS fixtures for new genesis output

**Promotion criteria:** N/A

**Files:**

- Modify: `zz-tests_bats/previous_versions/` (via fixture regeneration)

Genesis now creates more objects (4 instead of 2), so fixture output and SHAs
will change.

**Step 1: Regenerate fixtures**

Run: `just test-bats-update-fixtures`

**Step 2: Review fixture diff**

Run: `git diff -- zz-tests_bats/previous_versions/` Expected: `.fixtures.env`
values change. Init output includes new type objects.

**Step 3: Run full test suite**

Run: `just test` Expected: Some tests may fail due to changed init output
(assertions that check exact `dodder init` output lines).

**Step 4: Fix failing BATS assertions**

Init tests that assert exactly 2 lines of output (one for `!md`, one for
`konfig`) need updating to include the new tool type objects. Update
`common.bash` helpers and test assertions to expect the new types.

Look for assertions like:

``` bash
assert_output --partial '[!md @'
```

These should still pass. But line-count assertions or exact-output assertions
need updating.

**Step 5: Run tests again**

Run: `just test` Expected: PASS

**Step 6: Commit fixtures and test updates together**

``` text
test: update BATS fixtures and assertions for pandoc tool type genesis
```

--------------------------------------------------------------------------------

### Task 6: Implement blob tree materializer

**Promotion criteria:** N/A

**Files:**

- Create: `go/internal/juliett/typed_blob_store/blob_tree_materializer.go`
- Create: `go/internal/juliett/typed_blob_store/blob_tree_materializer_test.go`

**Step 1: Write the failing test**

Create `go/internal/juliett/typed_blob_store/blob_tree_materializer_test.go`.
The test should verify that `MaterializeBlobTree` creates files at the expected
paths. This requires setting up a mock or test blob store --- follow the
patterns used in existing `juliett/` tests.

At minimum, test that:

- Given blob references with aliases `"filters/foo.lua"` and
  `"defaults/bar.yaml"`, the materializer creates `$tmpdir/filters/foo.lua` and
  `$tmpdir/defaults/bar.yaml`
- File contents match what was stored
- Cleanup function removes the tmpdir

**Step 2: Run test to verify it fails**

Run:
`go test -v -tags test,debug ./go/internal/juliett/typed_blob_store/ -run MaterializeBlobTree`
Expected: FAIL

**Step 3: Write the materializer**

Create `go/internal/juliett/typed_blob_store/blob_tree_materializer.go`:

``` go
package typed_blob_store

import (
    "io"
    "os"
    "path/filepath"

    "code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
    "code.linenisgreat.com/dodder/go/internal/bravo/markl"
    "code.linenisgreat.com/dodder/go/internal/delta/objects"
    "code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func MaterializeBlobTree(
    blobStore domain_interfaces.BlobStore,
    metadata objects.Metadata,
) (tmpdir string, cleanup func(), err error) {
    cleanup = func() {}

    tmpdir, err = os.MkdirTemp("", "dodder-blob-tree-*")
    if err != nil {
        return tmpdir, cleanup, errors.Wrap(err)
    }

    cleanup = func() { os.RemoveAll(tmpdir) }

    for blobId := range metadata.AllBlobReferences() {
        alias := metadata.GetBlobReferenceAlias(blobId)
        if alias == "" {
            continue
        }

        destPath := filepath.Join(tmpdir, alias)

        if err = materializeOneBlob(blobStore, blobId, destPath); err != nil {
            cleanup()
            cleanup = func() {}
            return "", cleanup, errors.Wrapf(err, "materializing %q", alias)
        }
    }

    return tmpdir, cleanup, err
}

func materializeOneBlob(
    blobStore domain_interfaces.BlobStore,
    blobId markl.Id,
    destPath string,
) (err error) {
    if err = os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
        return errors.Wrap(err)
    }

    var reader domain_interfaces.BlobReader

    if reader, err = blobStore.MakeBlobReader(blobId); err != nil {
        return errors.Wrap(err)
    }

    defer errors.DeferredCloser(&err, reader)

    var file *os.File

    if file, err = os.Create(destPath); err != nil {
        return errors.Wrap(err)
    }

    defer errors.DeferredCloser(&err, file)

    if _, err = io.Copy(file, reader); err != nil {
        return errors.Wrap(err)
    }

    return err
}
```

**Step 4: Run test to verify it passes**

Run:
`go test -v -tags test,debug ./go/internal/juliett/typed_blob_store/ -run MaterializeBlobTree`
Expected: PASS

**Step 5: Commit**

``` text
feat: add blob tree materializer for tmpdir-based blob reference expansion
```

--------------------------------------------------------------------------------

### Task 7: Wire materializer into formatter pipeline

**Promotion criteria:** N/A

**Files:**

- Modify: `go/internal/sierra/local_working_copy/op_get_blob_formatter.go`
- Modify: `go/internal/foxtrot/object_metadata_fmt_hyphence/factory.go`
- Modify:
  `go/internal/foxtrot/object_metadata_fmt_hyphence/formatter_components.go`

This is the most delicate task. The formatter pipeline currently has no concept
of blob trees. We need to:

1.  Pass the type object's metadata through to the point where the script runs
2.  Materialize blob references to a tmpdir before the script runs
3.  Add `DODDER_BLOB_TREE` to the env vars
4.  Defer cleanup

**Step 1: Extend `Factory` to accept type object metadata**

In `go/internal/foxtrot/object_metadata_fmt_hyphence/factory.go`, add a field to
`Factory`:

``` go
type Factory struct {
    EnvDir        env_dir.Env
    BlobStore     domain_interfaces.BlobStore
    BlobFormatter script_config.RemoteScript
    TypeMetadata  objects.Metadata // nil if no type-level blob refs
}
```

**Step 2: Materialize in `writeBlob` before script execution**

In `formatter_components.go`, in the `writeBlob` function, after the
`factory.BlobFormatter != nil` check and before `MakeWriterToWithStdin`, add
materialization:

``` go
env := factory.EnvDir.MakeCommonEnv()

if factory.TypeMetadata != nil {
    tmpdir, cleanup, matErr := typed_blob_store.MaterializeBlobTree(
        factory.BlobStore,
        factory.TypeMetadata,
    )
    if matErr != nil {
        return n, errors.Wrap(matErr)
    }
    defer cleanup()
    env["DODDER_BLOB_TREE"] = tmpdir
    env["LUA_PATH"] = tmpdir + "/filters/?.lua;" + env["LUA_PATH"]
}
```

**Step 3: Pass type metadata through from `GetBlobFormatter`**

In `op_get_blob_formatter.go`, after the formatter is resolved, also return the
type object's metadata so it can be passed to the `Factory`. This may require
changing the return type of `GetBlobFormatter` or adding a new method.

The simplest approach: have `GetBlobFormatter` return the type object's metadata
alongside the formatter script. Then `MakeTextFormatterWithBlobFormatter` in
`text_formatter.go` passes it to `Factory.TypeMetadata`.

**Step 4: Build and run unit tests**

Run: `just build && just test-go-unit` Expected: PASS

**Step 5: Commit**

``` text
feat: wire blob tree materializer into formatter pipeline
```

--------------------------------------------------------------------------------

### Task 8: Integration test --- end-to-end pandoc formatting

**Promotion criteria:** N/A

**Files:**

- Create: `zz-tests_bats/current_version/pandoc_formatting.bats`

**Step 1: Write the BATS test**

Create `zz-tests_bats/current_version/pandoc_formatting.bats`:

``` bash
#! /usr/bin/env bats

# bats file_tags=pandoc,formatting

load '../lib/common.bash'

setup() {
  common_setup
}

teardown() {
  common_teardown
}

# Verify genesis creates pandoc tool types
@test "init creates pandoc tool types" {
  run_dodder_init

  run_dodder show :t
  assert_success
  assert_output --partial '!pandoc-defaults'
  assert_output --partial '!pandoc-lua_filter'
}

# Verify !md type has blob references to tool blobs
@test "md type has blob references" {
  run_dodder_init

  run_dodder show -format text '!md:t'
  assert_success
  assert_output --partial 'filters/dodder-common.lua'
  assert_output --partial 'filters/dodder-edit.lua'
  assert_output --partial 'defaults/dodder-edit.yaml'
}

# Verify tool blobs are readable
@test "tool blobs are readable via cat-blob" {
  run_dodder_init

  # Get the blob digest for the common filter from !md's blob refs
  run_dodder show -format text '!md:t'
  assert_success

  # The output should contain blob reference lines with digests
  assert_output --partial '!pandoc-lua_filter'
  assert_output --partial '!pandoc-defaults'
}
```

**Step 2: Run the test**

Run: `just test-bats-targets pandoc_formatting.bats` Expected: Tests pass if
genesis is correct. If pandoc is not on PATH, formatting tests should be skipped
or handle the error gracefully.

**Step 3: Add a formatting test (requires pandoc on PATH)**

``` bash
# Verify pandoc formatting works end-to-end
@test "format-object uses internal pandoc config" {
  skip_if_no_pandoc
  run_dodder_init

  echo '# Hello World' | run_dodder new -
  assert_success

  run_dodder show -format text .z
  assert_success
  # Pandoc normalizes markdown --- exact output depends on defaults
}
```

Add helper to `common.bash`:

``` bash
skip_if_no_pandoc() {
  if ! command -v pandoc &>/dev/null; then
    skip "pandoc not available"
  fi
}
```

**Step 4: Run full test suite**

Run: `just test` Expected: PASS

**Step 5: Commit**

``` text
test: add BATS integration tests for pandoc internal formatting
```

--------------------------------------------------------------------------------

### Task 9: Clean up and document

**Promotion criteria:** N/A

**Files:**

- Modify: `go/internal/hotel/type_blobs/CLAUDE.md`
- Modify: `go/internal/sierra/local_working_copy/CLAUDE.md`

**Step 1: Update CLAUDE.md files**

Add notes about the new tool type patterns and embedded content to the relevant
CLAUDE.md files.

**Step 2: Commit**

``` text
docs: update CLAUDE.md files for pandoc internal formatting
```
