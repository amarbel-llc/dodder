# Object Reference Discovery Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Types define an external command that discovers object references from blob content, automatically populating the References collection before commit.

**Architecture:** Add `ObjectReferencesConfig` to `TomlV1` type blobs. During commit, after `SaveBlob` but before pre-commit hooks, the store loads the type blob, pipes blob content to the discovery command's stdin, parses stdout for reference lines, and merges discovered references into metadata. The lock finalizer then pins their signatures as usual.

**Tech Stack:** Go, `os/exec` via `script_config.ScriptConfig`, TOML type blobs

**Rollback:** Purely additive. Remove the `discoverReferences` call in `tryPrecommit` and the feature is disabled. Types with `[object-references]` sections have an unused field.

---

### Task 1: ObjectReferencesConfig struct

**Promotion criteria:** N/A

**Files:**
- Create: `go/internal/hotel/type_blobs/object_references_config.go`

**Step 1: Create the config struct**

Create the file with a struct that embeds `ScriptConfig` and adds `Optional`:

```go
package type_blobs

import "code.linenisgreat.com/dodder/go/lib/delta/script_config"

type ObjectReferencesConfig struct {
	script_config.ScriptConfig
	Optional bool `toml:"optional,omitempty"`
}
```

**Step 2: Run tests to verify no regressions**

Run: `just test-go`
Expected: PASS (new file, no behavior change)

**Step 3: Commit**

```
feat: add ObjectReferencesConfig struct for type-driven reference discovery
```

---

### Task 2: Add ObjectReferences field to TomlV1

**Promotion criteria:** N/A

**Files:**
- Modify: `go/internal/hotel/type_blobs/toml_v1.go`
- Modify: `go/internal/hotel/type_blobs/toml_v0.go`
- Modify: `go/internal/hotel/type_blobs/main.go`

**Step 1: Add field and getter to TomlV1**

In `toml_v1.go`, add the field to the struct (after `Hooks`):

```go
type TomlV1 struct {
	Binary        bool                                      `toml:"binary,omitempty"`
	FileExtension string                                    `toml:"file-extension,omitempty"`
	MimeType      string                                    `toml:"mime-type,omitempty"`
	ExecCommand   *script_config.ScriptConfig               `toml:"exec-command,omitempty"`
	VimSyntaxType string                                    `toml:"vim-syntax-type"`
	UTIGroups     map[string]UTIGroup                       `toml:"uti-groups"`
	Formatters    map[string]script_config.WithOutputFormat `toml:"formatters,omitempty"`

	// TODO migrate to properly-typed hooks
	Hooks any `toml:"hooks"`

	ObjectReferences *ObjectReferencesConfig `toml:"object-references,omitempty"`
}
```

Update `Reset()` to clear it:

```go
func (blob *TomlV1) Reset() {
	// ... existing resets ...
	blob.ObjectReferences = nil
}
```

Add getter method:

```go
func (blob *TomlV1) GetObjectReferences() *ObjectReferencesConfig {
	return blob.ObjectReferences
}
```

**Step 2: Add getter to TomlV0 (returns nil)**

In `toml_v0.go`, add:

```go
func (blob *TomlV0) GetObjectReferences() *ObjectReferencesConfig {
	return nil
}
```

**Step 3: Add to Blob interface**

In `main.go`, add `WithObjectReferences` to the interface composition and define
the sub-interface:

```go
type Blob interface {
	GetFileExtension() string
	GetBinary() bool
	GetMimeType() string
	GetVimSyntaxType() string

	WithFormatters
	WithFormatterUTIGroups
	WithStringLuaHooks
	WithObjectReferences
}

type WithObjectReferences interface {
	GetObjectReferences() *ObjectReferencesConfig
}
```

**Step 4: Run tests**

Run: `just test-go`
Expected: PASS

**Step 5: Commit**

```
feat: add ObjectReferences field to type blob interface
```

---

### Task 3: Reference output parser

**Promotion criteria:** N/A

**Files:**
- Create: `go/internal/papa/store/reference_discovery.go`
- Create: `go/internal/papa/store/reference_discovery_test.go`

**Step 1: Write the failing test**

Create `reference_discovery_test.go`:

```go
//go:build test

package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReferenceOutputEmpty(t *testing.T) {
	refs, err := parseReferenceOutput("")
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestParseReferenceOutputSimpleRefs(t *testing.T) {
	input := "one/dos\ntwo/uno\n"
	refs, err := parseReferenceOutput(input)
	require.NoError(t, err)
	require.Len(t, refs, 2)
	assert.Equal(t, "one/dos", refs[0].ObjectId)
	assert.Equal(t, "", refs[0].Alias)
	assert.Equal(t, "two/uno", refs[1].ObjectId)
	assert.Equal(t, "", refs[1].Alias)
}

func TestParseReferenceOutputWithAliases(t *testing.T) {
	input := "blog-template = one/uno\none/dos\n"
	refs, err := parseReferenceOutput(input)
	require.NoError(t, err)
	require.Len(t, refs, 2)
	assert.Equal(t, "one/uno", refs[0].ObjectId)
	assert.Equal(t, "blog-template", refs[0].Alias)
	assert.Equal(t, "one/dos", refs[1].ObjectId)
	assert.Equal(t, "", refs[1].Alias)
}

func TestParseReferenceOutputSkipsCommentsAndBlanks(t *testing.T) {
	input := "# this is a comment\n\none/dos\n  \n# another comment\n"
	refs, err := parseReferenceOutput(input)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "one/dos", refs[0].ObjectId)
}
```

**Step 2: Run test to verify it fails**

Run: `just test-go`
Expected: FAIL — `parseReferenceOutput` undefined

**Step 3: Write minimal implementation**

Create `reference_discovery.go`:

```go
package store

import (
	"strings"
)

type discoveredReference struct {
	ObjectId string
	Alias    string
}

func parseReferenceOutput(output string) ([]discoveredReference, error) {
	var refs []discoveredReference

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var ref discoveredReference

		if idx := strings.Index(line, " = "); idx != -1 {
			ref.Alias = strings.TrimSpace(line[:idx])
			ref.ObjectId = strings.TrimSpace(line[idx+3:])
		} else {
			ref.ObjectId = line
		}

		refs = append(refs, ref)
	}

	return refs, nil
}
```

**Step 4: Run tests**

Run: `just test-go`
Expected: PASS

**Step 5: Commit**

```
feat: add reference output parser for discovery command stdout
```

---

### Task 4: Discovery execution function

**Promotion criteria:** N/A

**Files:**
- Modify: `go/internal/papa/store/reference_discovery.go`

**Step 1: Add the discoverReferences method**

Add to `reference_discovery.go`. This method:
1. Checks if hooks are enabled via `options.RunHooks`
2. Loads the type blob
3. Checks if `ObjectReferences` is defined on the type blob
4. Opens the blob reader from the default blob store
5. Creates exec command from `ScriptConfig.Cmd()`
6. Pipes blob content to stdin, captures stdout
7. Parses stdout into references
8. Merges each discovered reference into `metadata.References`

```go
package store

import (
	"bytes"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/type_blobs"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/script_config"
)

type discoveredReference struct {
	ObjectId string
	Alias    string
}

func parseReferenceOutput(output string) ([]discoveredReference, error) {
	var refs []discoveredReference

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var ref discoveredReference

		if idx := strings.Index(line, " = "); idx != -1 {
			ref.Alias = strings.TrimSpace(line[:idx])
			ref.ObjectId = strings.TrimSpace(line[idx+3:])
		} else {
			ref.ObjectId = line
		}

		refs = append(refs, ref)
	}

	return refs, nil
}

func (store *Store) discoverReferences(
	daughter *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	if !options.RunHooks {
		return err
	}

	if daughter.GetBlobDigest().IsNull() {
		return err
	}

	var typeObject *sku.Transacted

	if typeObject, err = store.ReadObjectTypeAndLockIfNecessary(
		daughter,
	); err != nil {
		if errors.IsErrNotFound(err) {
			err = nil
		}

		return err
	} else if typeObject == nil {
		return err
	}

	var blob type_blobs.Blob

	{
		var repool interfaces.FuncRepool

		if blob, repool, _, err = store.GetTypedBlobStore().Type.ParseTypedBlob(
			typeObject.GetType(),
			typeObject.GetBlobDigest(),
		); err != nil {
			return errors.Wrap(err)
		}

		defer repool()
	}

	objectReferences := blob.GetObjectReferences()
	if objectReferences == nil {
		return err
	}

	blobReader, err := store.envRepo.GetDefaultBlobStore().MakeBlobReader(
		daughter.GetBlobDigest(),
	)
	if err != nil {
		return errors.Wrap(err)
	}

	defer errors.DeferredCloser(&err, blobReader)

	var writerTo *script_config.WriterTo

	if writerTo, err = script_config.MakeWriterToWithStdin(
		&objectReferences.ScriptConfig,
		nil,
		blobReader,
	); err != nil {
		if objectReferences.Optional {
			return nil
		}

		return errors.Wrap(err)
	}

	var buf bytes.Buffer

	if _, err = writerTo.WriteTo(&buf); err != nil {
		if objectReferences.Optional {
			return nil
		}

		return errors.Wrap(err)
	}

	var refs []discoveredReference

	if refs, err = parseReferenceOutput(buf.String()); err != nil {
		return errors.Wrap(err)
	}

	metadata := daughter.GetMetadataMutable()
	metadataStruct, ok := metadata.(*objects.MetadataStruct)
	if !ok {
		return err
	}

	for _, ref := range refs {
		var refId ids.SeqId

		if err = refId.Set(ref.ObjectId); err != nil {
			if objectReferences.Optional {
				continue
			}

			return errors.Wrapf(err, "invalid reference: %q", ref.ObjectId)
		}

		if err = metadataStruct.References.Add(refId); err != nil {
			return errors.Wrap(err)
		}

		if ref.Alias != "" {
			// Set alias on the contained object
			for index := range metadataStruct.References.GetSlice() {
				entry := &metadataStruct.References.GetSlice()[index]

				if entry.GetKey().String() == refId.String() {
					entry.Alias.Set(ref.Alias)
					break
				}
			}
		}
	}

	return err
}
```

**Important notes for the implementer:**

- The `WriterTo` type in `script_config` is unexported (`writerTo`). Check the
  actual type name — it may need to be referenced as `*script_config.writerTo`
  or you may need to use the `io.WriterTo` interface instead. Specifically,
  `MakeWriterToWithStdin` returns `(*writerTo, error)` where `writerTo` is
  unexported. Use `io.WriterTo` for the variable type.
- `ids.SeqId` — check the import path. It's likely
  `code.linenisgreat.com/dodder/go/internal/bravo/ids`.
- The alias-setting pattern follows the same approach used in
  `text_parser2.go:readReference` from the referenced object locks implementation.
  Check how aliases are set there and follow the same pattern.
- `ReadObjectTypeAndLockIfNecessary` is the same method used in `hooks.go:145`.
  It reads the type object and writes the type lock if needed.

**Step 2: Run tests**

Run: `just test-go`
Expected: PASS (function exists but not wired in yet)

**Step 3: Commit**

```
feat: add discoverReferences function for type-driven reference extraction
```

---

### Task 5: Wire discovery into commit pipeline

**Promotion criteria:** N/A

**Files:**
- Modify: `go/internal/papa/store/mutating.go:44-94`

**Step 1: Add discoverReferences call**

In `tryPrecommit()`, add the discovery call between the proto/type application
block (line 69) and the `tryPreCommitHooks` call (line 72):

```go
func (commitFacilitator commitFacilitator) tryPrecommit(
	daughter *sku.Transacted,
	mother *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	if err = commitFacilitator.SaveBlob(daughter); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if mother == nil {
		options.Proto.Apply(daughter, daughter)
	}

	// TODO decide if the type proto should actually be applied every time
	if options.ApplyProtoType {
		commitFacilitator.protoZettel.ApplyType(daughter, daughter)
	}

	if genres.Type == daughter.GetGenre() {
		if daughter.GetType().IsEmpty() {
			daughter.GetMetadataMutable().GetTypeMutable().ResetWithObjectId(
				ids.DefaultOrPanic(genres.Type),
			)
		}
	}

	// Discover references from blob content before hooks run
	if err = commitFacilitator.discoverReferences(daughter, options); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// modify pre commit hooks to support import
	if err = commitFacilitator.tryPreCommitHooks(daughter, mother, options); err != nil {
		// ... existing error handling ...
```

The new line goes at line 71 (before the existing line 72 `tryPreCommitHooks`
call). Add exactly this block:

```go
	if err = commitFacilitator.discoverReferences(daughter, options); err != nil {
		err = errors.Wrap(err)
		return err
	}
```

**Step 2: Run tests**

Run: `just test-go`
Expected: PASS (no types have `[object-references]` yet, so discovery is a no-op)

**Step 3: Run full test suite**

Run: `just test`
Expected: PASS — all existing tests still pass since no type blobs define
`[object-references]`

**Step 4: Commit**

```
feat: wire reference discovery into commit pipeline before hooks
```

---

### Task 6: Integration test — reference discovery end-to-end

**Promotion criteria:** N/A

**Files:**
- Modify: `zz-tests_bats/current_version/show.bats`

**Step 1: Write the bats test**

Add a new test function at the end of `show.bats`. This test:

1. Creates a new type `ref-md` with an `[object-references]` section that uses
   `grep` to find `[[...]]` wiki-link patterns in blob content
2. Creates a zettel of that type whose blob contains `[[one/dos]]`
3. Shows the zettel and verifies the `< one/dos@...` reference lock appears

```bash
# bats test_tags=user_story:referenced_objects
function show_zettel_with_discovered_references { # @test
	run_dodder init-workspace
	assert_success

	# Create a type with reference discovery script
	cat >ref-md.type <<-'TYPEFILE'
		---
		! toml-type-v1
		---

		file-extension = 'md'
		vim-syntax-type = 'markdown'

		[object-references]
		shell = ['bash', '-c']
		script = "grep -oP '\\[\\[(.+?)\\]\\]' | sed 's/\\[\\[//;s/\\]\\]//'"
	TYPEFILE

	run_dodder checkin -delete ref-md.type
	assert_success

	# Create a zettel of type ref-md with a wiki-link to one/dos
	run_dodder new -edit=false - <<-EOM
		---
		# zettel with wiki link
		! ref-md
		---

		Check out [[one/dos]] for more info.
	EOM
	assert_success

	# Show the new zettel and verify the reference lock was auto-discovered
	run_dodder show -format text two/uno:
	assert_success
	assert_output --regexp - <<-'EOM'
		---
		# zettel with wiki link
		@ blake2b256-.+
		! ref-md@.+
		< one/dos@ed25519_sig-.+
		---
	EOM
}
```

**Important notes for the implementer:**

- The `grep -oP` pattern extracts `[[...]]` content. The `-P` flag uses Perl
  regex. On some systems (like macOS), `-P` may not be available — if tests fail
  on CI, switch to `grep -o` with basic regex or use `sed`.
- The type is created via `checkin -delete ref-md.type` following the pattern
  from `show_tag_toml` test (line 442).
- The new zettel ID will be `two/uno` (third zettel after one/uno and one/dos).
- The reference lock value is an `ed25519_sig-...` (non-deterministic per key),
  so use regex.
- Use `just test-bats-targets current_version/show.bats` to run just this file.

**Step 2: Run the test**

Run: `just test-bats-targets current_version/show.bats`
Expected: PASS — the reference to `one/dos` should be automatically discovered
from the blob content and locked

**Step 3: Run full test suite**

Run: `just test`
Expected: PASS — all existing tests still pass

**Step 4: Commit**

```
test: add integration test for type-driven reference discovery
```

---

### Task 7: Verify existing tests pass and clean up

**Promotion criteria:** N/A

**Files:** None (verification only)

**Step 1: Run full test suite**

Run: `just test`
Expected: All Go unit tests and all bats integration tests pass.

**Step 2: Review all changes**

Run: `git diff master --stat` and `git log master..HEAD --oneline`

Verify:
- `ObjectReferencesConfig` struct exists in `type_blobs/`
- `TomlV1` and `TomlV0` both implement `GetObjectReferences()`
- `Blob` interface includes `WithObjectReferences`
- `parseReferenceOutput` has unit tests
- `discoverReferences` wired into `tryPrecommit` before hooks
- Integration test verifies end-to-end discovery

**Step 3: Commit (if any cleanup needed)**

```
chore: clean up reference discovery implementation
```
