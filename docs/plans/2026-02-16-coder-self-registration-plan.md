# Coder Self-Registration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace central coder map literals with per-struct self-registration across all `*_blobs` / `*_configs` packages, plus a static analyzer for completeness checking.

**Architecture:** Each versioned struct file calls a package-level `register[T]()` generic function at var-init time, populating the coder map. The `echo/ids` builtin type registry stays unchanged. A new `golang.org/x/tools/go/analysis` analyzer verifies all non-deprecated builtin types have coder registrations.

**Tech Stack:** Go 1.23+ generics, `golang.org/x/tools/go/analysis`, `singlechecker`

---

### Task 1: Migrate `golf/blob_store_configs` remaining types to `registerToml`

The `registerToml` function already exists. Two types already use it (`TomlPointerV0`, `TomlInventoryArchiveV0`). Migrate the five types still in the map literal.

**Files:**
- Modify: `go/src/golf/blob_store_configs/coding.go:43-89` (delete map literal entries)
- Modify: `go/src/golf/blob_store_configs/toml_v0.go` (add register call)
- Modify: `go/src/golf/blob_store_configs/toml_v1.go` (add register call)
- Modify: `go/src/golf/blob_store_configs/toml_v2.go` (add register call)
- Modify: `go/src/golf/blob_store_configs/toml_sftp_v0.go` (add register call)
- Modify: `go/src/golf/blob_store_configs/toml_sftp_via_ssh_config_v0.go` (add register call)

**Step 1: Add `registerToml` calls to each struct file**

Add to each file's `var` block (matching existing pattern from `toml_pointer_v0.go`):

```go
// toml_v0.go
var _ = registerToml[TomlV0](Coder.Blob, ids.TypeTomlBlobStoreConfigV0)

// toml_v1.go
var _ = registerToml[TomlV1](Coder.Blob, ids.TypeTomlBlobStoreConfigV1)

// toml_v2.go
var _ = registerToml[TomlV2](Coder.Blob, ids.TypeTomlBlobStoreConfigV2)

// toml_sftp_v0.go
var _ = registerToml[TomlSFTPV0](Coder.Blob, ids.TypeTomlBlobStoreConfigSftpExplicitV0)

// toml_sftp_via_ssh_config_v0.go
var _ = registerToml[TomlSFTPViaSSHConfigV0](Coder.Blob, ids.TypeTomlBlobStoreConfigSftpViaSSHConfigV0)
```

**Step 2: Remove those five entries from the map literal in `coding.go:46-87`**

Replace the map literal with an empty map. The `Coder` var becomes:

```go
var Coder = triple_hyphen_io.CoderToTypedBlob[Config]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[Config]{},
	Blob: triple_hyphen_io.CoderTypeMapWithoutType[Config](
		make(map[string]interfaces.CoderBufferedReadWriter[*Config]),
	),
}
```

**Step 3: Verify compilation**

Run: `cd go && go build ./src/golf/blob_store_configs/...`

**Step 4: Run tests**

Run: `cd go && go test -v -tags test,debug ./src/golf/blob_store_configs/...`

Then full test suite: `cd go && just test-go`

**Step 5: Commit**

```
feat(blob_store_configs): migrate all types to registerToml self-registration
```

---

### Task 2: Add `register` function to `golf/repo_blobs`

**Files:**
- Modify: `go/src/golf/repo_blobs/coding.go` (replace map literal with register pattern)
- Modify: `go/src/golf/repo_blobs/toml_uri_v0.go` (add register call)
- Modify: `go/src/golf/repo_blobs/toml_local_override_path_v0.go` (add register call)
- Modify: `go/src/golf/repo_blobs/toml_xdg_v0.go` (add register call)

**Step 1: Rewrite `coding.go`**

Replace the entire file with:

```go
package repo_blobs

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
)

var coderMap = make(map[string]interfaces.CoderBufferedReadWriter[*Blob])

func register[IMPL any, IMPL_PTR interface {
	Blob
	interfaces.Ptr[IMPL]
}](typeString string) struct{} {
	if _, ok := coderMap[typeString]; ok {
		panic(fmt.Sprintf(
			"coder for type %q registered more than once",
			typeString,
		))
	}

	coderMap[typeString] = triple_hyphen_io.CoderToml[Blob, *Blob]{
		Progenitor: func() Blob {
			var impl IMPL
			return IMPL_PTR(&impl)
		},
	}

	return struct{}{}
}

var Coder = triple_hyphen_io.CoderToTypedBlob[Blob]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[Blob]{},
	Blob:     triple_hyphen_io.CoderTypeMapWithoutType[Blob](coderMap),
}
```

**Step 2: Add register calls to struct files**

```go
// toml_uri_v0.go — add to existing var block:
var _ = register[TomlUriV0](ids.TypeTomlRepoUri)

// toml_local_override_path_v0.go — add to existing var block:
var _ = register[TomlLocalOverridePathV0](ids.TypeTomlRepoLocalOverridePath)

// toml_xdg_v0.go — add to existing var block:
var _ = register[TomlXDGV0](ids.TypeTomlRepoDotenvXdgV0)
```

Each file needs to add `"code.linenisgreat.com/dodder/go/src/echo/ids"` to its imports if not already present.

**Step 3: Verify compilation**

Run: `cd go && go build ./src/golf/repo_blobs/...`

**Step 4: Run tests**

Run: `cd go && just test-go`

**Step 5: Commit**

```
feat(repo_blobs): add coder self-registration via register generic
```

---

### Task 3: Add `register` function to `golf/repo_configs`

**Files:**
- Modify: `go/src/golf/repo_configs/coding.go` (replace map literal with register pattern)
- Modify: `go/src/golf/repo_configs/v0.go` (add register call)
- Modify: `go/src/golf/repo_configs/v1.go` (add register call)
- Modify: `go/src/golf/repo_configs/v2.go` (add register call)

**Step 1: Rewrite `coding.go`**

```go
package repo_configs

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
)

var coderMap = make(map[string]interfaces.CoderBufferedReadWriter[*ConfigOverlay])

func register[IMPL any, IMPL_PTR interface {
	ConfigOverlay
	interfaces.Ptr[IMPL]
}](typeString string) struct{} {
	if _, ok := coderMap[typeString]; ok {
		panic(fmt.Sprintf(
			"coder for type %q registered more than once",
			typeString,
		))
	}

	coderMap[typeString] = triple_hyphen_io.CoderToml[ConfigOverlay, *ConfigOverlay]{
		Progenitor: func() ConfigOverlay {
			var impl IMPL
			return IMPL_PTR(&impl)
		},
	}

	return struct{}{}
}

var Coder = triple_hyphen_io.CoderToTypedBlob[ConfigOverlay]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[ConfigOverlay]{},
	Blob:     triple_hyphen_io.CoderTypeMapWithoutType[ConfigOverlay](coderMap),
}
```

**Step 2: Add register calls to struct files**

```go
// v0.go:
var _ = register[V0](ids.TypeTomlConfigV0)

// v1.go:
var _ = register[V1](ids.TypeTomlConfigV1)

// v2.go:
var _ = register[V2](ids.TypeTomlConfigV2)
```

Each file needs `"code.linenisgreat.com/dodder/go/src/echo/ids"` in imports.

**Step 3: Verify compilation**

Run: `cd go && go build ./src/golf/repo_configs/...`

**Step 4: Run tests**

Run: `cd go && just test-go`

**Step 5: Commit**

```
feat(repo_configs): add coder self-registration via register generic
```

---

### Task 4: Add `register` function to `hotel/genesis_configs`

This is the special case with two coder maps (Private and Public). Each version registers both variants.

**Files:**
- Modify: `go/src/hotel/genesis_configs/coder.go` (replace map literals with register pattern)
- Modify: `go/src/hotel/genesis_configs/toml_v1.go` (add register call)
- Modify: `go/src/hotel/genesis_configs/toml_v2.go` (add register call)

**Step 1: Rewrite `coder.go`**

```go
package genesis_configs

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
)

var (
	privateCoderMap = make(map[string]interfaces.CoderBufferedReadWriter[*ConfigPrivate])
	publicCoderMap  = make(map[string]interfaces.CoderBufferedReadWriter[*ConfigPublic])
)

func register[
	PRIV any, PRIV_PTR interface {
		ConfigPrivate
		interfaces.Ptr[PRIV]
	},
	PUB any, PUB_PTR interface {
		ConfigPublic
		interfaces.Ptr[PUB]
	},
](typeString string) struct{} {
	if _, ok := privateCoderMap[typeString]; ok {
		panic(fmt.Sprintf(
			"genesis coder for type %q registered more than once",
			typeString,
		))
	}

	privateCoderMap[typeString] = triple_hyphen_io.CoderToml[
		ConfigPrivate,
		*ConfigPrivate,
	]{
		Progenitor: func() ConfigPrivate {
			var impl PRIV
			return PRIV_PTR(&impl)
		},
	}

	publicCoderMap[typeString] = triple_hyphen_io.CoderToml[
		ConfigPublic,
		*ConfigPublic,
	]{
		Progenitor: func() ConfigPublic {
			var impl PUB
			return PUB_PTR(&impl)
		},
	}

	return struct{}{}
}

var CoderPrivate = triple_hyphen_io.CoderToTypedBlob[ConfigPrivate]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[ConfigPrivate]{},
	Blob:     triple_hyphen_io.CoderTypeMapWithoutType[ConfigPrivate](privateCoderMap),
}

var CoderPublic = triple_hyphen_io.CoderToTypedBlob[ConfigPublic]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[ConfigPublic]{},
	Blob:     triple_hyphen_io.CoderTypeMapWithoutType[ConfigPublic](publicCoderMap),
}
```

**Step 2: Add register calls to struct files**

```go
// toml_v1.go — add after existing var blocks:
var _ = register[TomlV1Private, TomlV1Public](ids.TypeTomlConfigImmutableV1)

// toml_v2.go — add after existing var blocks:
var _ = register[TomlV2Private, TomlV2Public](ids.TypeTomlConfigImmutableV2)
```

**Step 3: Verify compilation**

Run: `cd go && go build ./src/hotel/genesis_configs/...`

**Step 4: Run tests**

Run: `cd go && just test-go`

**Step 5: Commit**

```
feat(genesis_configs): add dual coder self-registration for private/public
```

---

### Task 5: Add `register` function to `hotel/workspace_config_blobs`

**Files:**
- Modify: `go/src/hotel/workspace_config_blobs/io.go` (replace map literal with register pattern)
- Modify: `go/src/hotel/workspace_config_blobs/v0.go` (add register call)

**Step 1: Rewrite `io.go`**

```go
package workspace_config_blobs

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
)

var coderMap = make(map[string]interfaces.CoderBufferedReadWriter[*Config])

func register[IMPL any, IMPL_PTR interface {
	Config
	interfaces.Ptr[IMPL]
}](typeString string) struct{} {
	if _, ok := coderMap[typeString]; ok {
		panic(fmt.Sprintf(
			"coder for type %q registered more than once",
			typeString,
		))
	}

	coderMap[typeString] = triple_hyphen_io.CoderToml[Config, *Config]{
		Progenitor: func() Config {
			var impl IMPL
			return IMPL_PTR(&impl)
		},
	}

	return struct{}{}
}

var Coder = triple_hyphen_io.CoderToTypedBlob[Config]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[Config]{},
	Blob:     triple_hyphen_io.CoderTypeMapWithoutType[Config](coderMap),
}
```

**Step 2: Add register call to `v0.go`**

```go
var _ = register[V0](ids.TypeTomlWorkspaceConfigV0)
```

Add `"code.linenisgreat.com/dodder/go/src/echo/ids"` to imports.

**Step 3: Verify compilation**

Run: `cd go && go build ./src/hotel/workspace_config_blobs/...`

**Step 4: Run tests**

Run: `cd go && just test-go`

**Step 5: Commit**

```
feat(workspace_config_blobs): add coder self-registration
```

---

### Task 6: Convert `lima/type_blobs` to factory registration

This is the special case that uses `blob_library.MakeBlobStore` with `envRepo` context rather than `CoderTypeMapWithoutType`.

**Files:**
- Create: `go/src/lima/type_blobs/register.go` (factory registration infrastructure)
- Modify: `go/src/lima/type_blobs/coder.go` (build from registered factories)
- Modify: `go/src/lima/type_blobs/toml_v0.go` (add register call)
- Modify: `go/src/lima/type_blobs/toml_v1.go` (add register call)

**Step 1: Create `register.go`**

```go
package type_blobs

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/charlie/toml"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/kilo/blob_library"
)

type blobStoreFactory struct {
	makeStore func(envRepo env_repo.Env) domain_interfaces.TypedStore[Blob, *Blob]
}

// Use a slice to capture all registrations. The init-time registration just
// records factories; MakeTypeStore iterates them at runtime.
type registeredType[T any] struct {
	typeString string
	makeStore  func(envRepo env_repo.Env) domain_interfaces.TypedStore[T, *T]
	reset      func(*T)
}

var factories []registeredType[any]

// We cannot use a single generic slice, so we store closures that build
// map entries at construction time.
var storeBuilders = make(map[string]func(envRepo env_repo.Env) typedStoreEntry)

type typedStoreEntry struct {
	saveBlobText func(
		blob Blob,
	) (domain_interfaces.MarklId, int64, error)

	getBlob func(
		blobId domain_interfaces.MarklId,
	) (Blob, func(), int64, error)
}

func register[T Blob, T_PTR interface {
	Blob
	*T
}](
	typeString string,
	reset func(*T),
) struct{} {
	if _, ok := storeBuilders[typeString]; ok {
		panic(fmt.Sprintf(
			"type_blobs coder for type %q registered more than once",
			typeString,
		))
	}

	storeBuilders[typeString] = func(envRepo env_repo.Env) typedStoreEntry {
		store := blob_library.MakeBlobStore(
			envRepo,
			blob_library.MakeBlobFormat(
				toml.MakeTomlDecoderIgnoreTomlErrors[T](
					envRepo.GetDefaultBlobStore(),
				),
				toml.TomlBlobEncoder[T, T_PTR]{},
				envRepo.GetDefaultBlobStore(),
			),
			reset,
		)

		return typedStoreEntry{
			saveBlobText: func(blob Blob) (domain_interfaces.MarklId, int64, error) {
				return store.SaveBlobText(blob.(T_PTR))
			},
			getBlob: func(blobId domain_interfaces.MarklId) (Blob, func(), int64, error) {
				blob, repool, err := store.GetBlob(blobId)
				return blob, func() { repool() }, 0, err
			},
		}
	}

	return struct{}{}
}
```

**Step 2: Add register calls to struct files**

```go
// toml_v0.go — add:
var _ = register[TomlV0](ids.TypeTomlTypeV0, func(a *TomlV0) { a.Reset() })

// toml_v1.go — add:
var _ = register[TomlV1](ids.TypeTomlTypeV1, func(a *TomlV1) { a.Reset() })
```

Each needs `"code.linenisgreat.com/dodder/go/src/echo/ids"` in imports.

**Step 3: Rewrite `coder.go` to use registered factories**

Replace the struct with hardcoded stores with a map-based approach built from `storeBuilders`:

```go
package type_blobs

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
)

type Coder struct {
	stores map[string]typedStoreEntry
}

func MakeTypeStore(
	envRepo env_repo.Env,
) Coder {
	stores := make(map[string]typedStoreEntry, len(storeBuilders))

	for typeString, builder := range storeBuilders {
		stores[typeString] = builder(envRepo)
	}

	return Coder{stores: stores}
}

func (store Coder) SaveBlobText(
	tipe domain_interfaces.ObjectId,
	blob Blob,
) (digest domain_interfaces.MarklId, n int64, err error) {
	if err = genres.Type.AssertGenre(tipe); err != nil {
		err = errors.Wrap(err)
		return digest, n, err
	}

	typeString := tipe.String()

	// Preserve existing behavior: empty string defaults to V0
	if typeString == "" {
		typeString = ids.TypeTomlTypeV0
	}

	entry, ok := store.stores[typeString]
	if !ok {
		err = errors.Errorf("unsupported type: %q", tipe)
		return digest, n, err
	}

	return entry.saveBlobText(blob)
}

func (store Coder) ParseTypedBlob(
	tipe domain_interfaces.ObjectId,
	blobId domain_interfaces.MarklId,
) (common Blob, repool interfaces.FuncRepool, n int64, err error) {
	typeString := tipe.String()

	// Preserve existing behavior: empty string defaults to V0
	if typeString == "" {
		typeString = ids.TypeTomlTypeV0
	}

	entry, ok := store.stores[typeString]
	if !ok {
		err = errors.Errorf("unsupported type: %q", tipe)
		return common, repool, n, err
	}

	blob, repoolFn, readN, getErr := entry.getBlob(blobId)
	if getErr != nil {
		err = errors.Wrap(getErr)
		return common, repool, n, err
	}

	return blob, interfaces.FuncRepool(repoolFn), readN, nil
}
```

**Note:** The `register.go` and `coder.go` code above is approximate. The exact type signatures for `typedStoreEntry` may need adjustment to match the return types of `blob_library.MakeBlobStore`. The implementer should verify that `store.GetBlob` returns `(*T, interfaces.FuncRepool, error)` and adapt the wrapper accordingly.

**Step 4: Verify compilation**

Run: `cd go && go build ./src/lima/type_blobs/...`

**Step 5: Run full test suite**

Run: `cd go && just test`

This task exercises the full encode/decode path, so integration tests are essential.

**Step 6: Commit**

```
feat(type_blobs): convert to factory-based coder self-registration
```

---

### Task 7: Build the `builtin_coders` static analyzer

**Files:**
- Create: `go/src/alfa/analyzers/builtin_coders/analyzer.go`
- Create: `go/src/alfa/analyzers/builtin_coders/analyzer_test.go`
- Create: `go/src/alfa/analyzers/builtin_coders/cmd/main.go`
- Create: `go/src/alfa/analyzers/builtin_coders/testdata/` (test fixtures)
- Modify: `go/justfile:161-167` (add build and check targets)

**Step 1: Create the analyzer**

`go/src/alfa/analyzers/builtin_coders/analyzer.go`:

The analyzer inspects packages for:
1. Calls to `register[...]()` or `registerToml[...]()` — extracts the type string argument
2. References to `ids.Type*` constants — identifies which type constants are used as coder keys

It reports any `ids.Type*` constant that:
- Appears in `ids/types_builtin.go` as a string literal constant (not an alias like `TypeTomlBlobStoreConfigVCurrent`)
- Is not commented `// Deprecated`
- Has no corresponding register call anywhere in the analyzed packages

The implementation should follow the same `analysis.Analyzer` pattern as `repool/analyzer.go`:

```go
package builtin_coders

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var Analyzer = &analysis.Analyzer{
	Name:     "builtin_coders",
	Doc:      "check that builtin type constants have coder registrations",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}
```

The full implementation details will be determined during implementation based on what's most practical with the `analysis` API. The key constraint is that the analyzer sees one package at a time, so it should use `analysis.Fact` to export information across packages:
- The `ids` package exports facts about which type constants exist
- Each `*_blobs` package exports facts about which types it registered
- A top-level package (or the analyzer itself) checks completeness

**Step 2: Create `cmd/main.go`**

```go
package main

import (
	"code.linenisgreat.com/dodder/go/src/alfa/analyzers/builtin_coders"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(builtin_coders.Analyzer)
}
```

**Step 3: Create testdata with a passing and failing case**

Follow the pattern from `repool/testdata/`.

**Step 4: Write analyzer test**

```go
package builtin_coders_test

import (
	"testing"

	"code.linenisgreat.com/dodder/go/src/alfa/analyzers/builtin_coders"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, builtin_coders.Analyzer, "a")
}
```

**Step 5: Run analyzer tests**

Run: `cd go && go test -v ./src/alfa/analyzers/builtin_coders/...`

**Step 6: Add justfile targets**

Add to `go/justfile` after line 165:

```
build-analyzer-builtin-coders:
  go build -o build/builtin-coders-analyzer ./src/alfa/analyzers/builtin_coders/cmd/

check-go-builtin-coders: build-analyzer-builtin-coders
  go vet -vettool=build/builtin-coders-analyzer ./... || true
```

Update the `check` target (line 167) to include the new analyzer:

```
check: check-go-vuln check-go-vet check-go-repool check-go-builtin-coders
```

**Step 7: Run the analyzer on the full codebase**

Run: `cd go && just check-go-builtin-coders`

Expected: No diagnostics (all types should have registrations after Tasks 1-6).

**Step 8: Commit**

```
feat: add builtin_coders static analyzer for coder registration completeness
```

---

### Task 8: Final validation

**Step 1: Run full build**

Run: `cd go && just build`

**Step 2: Run full test suite (unit + integration)**

Run: `cd go && just test`

**Step 3: Run all checks**

Run: `cd go && just check`

**Step 4: Verify no regressions in fixture-dependent tests**

If integration tests fail with fixture issues: `cd go && just test-bats-update-fixtures`

Review diff and commit if needed.

**Step 5: Final commit if any fixups needed**

---

## Dependency Order

Tasks 1-5 are independent of each other (different packages). Task 6 is also independent. Task 7 depends on Tasks 1-6 being complete (the analyzer should pass on the final state). Task 8 validates everything together.

Recommended execution: Tasks 1-6 in parallel, then Task 7, then Task 8.
