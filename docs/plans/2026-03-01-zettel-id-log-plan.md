# Zettel ID Log Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace mutable flat Yin/Yang provider files with content-addressed delta blobs tracked by a signed append-only zettel ID log.

**Architecture:** New `zettel_id_log` package at `go/internal/charlie/` defines the log entry types and append-only reader/writer. Provider loading replays the log or falls back to flat files for pre-migration repos. Three new commands (`add-zettel-ids-yin`, `add-zettel-ids-yang`, `migrate-zettel-ids`) and genesis changes write blobs + log entries instead of flat files. Also renames `object_id_provider` to `zettel_id_provider` and adds `ohio.MakeLineSeqFromReader` utility.

**Tech Stack:** Go, `triple_hyphen_io` box-format encoding, content-addressed blob store, BATS integration tests.

**Design doc:** `docs/plans/2026-03-01-zettel-id-log-design.md`

**Rollback:** Delete the `zettel_id_log` file from any repo; provider automatically falls back to flat Yin/Yang files.

**Skills to reference:**
- `features-zettel_ids` — ZettelId system overview
- `design_patterns-hamster_style` — Go code conventions for dodder
- `robin:bats-testing` — BATS integration test patterns

---

### Task 1: Add `MakeLineSeqFromReader` to ohio

**Files:**
- Create: `go/lib/charlie/ohio/buffered_reader_line_seq.go`

**Step 1: Write the implementation**

```go
package ohio

import (
	"bufio"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func MakeLineSeqFromReader(
	reader *bufio.Reader,
) interfaces.SeqError[string] {
	return func(yield func(string, error) bool) {
		for {
			line, err := reader.ReadString('\n')

			if len(line) > 0 {
				if !yield(line, nil) {
					return
				}
			}

			if err != nil {
				if !errors.IsEOF(err) {
					yield("", errors.Wrap(err))
				}

				return
			}
		}
	}
}
```

**Step 2: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 3: Commit**

```
git add go/lib/charlie/ohio/buffered_reader_line_seq.go
git commit -m "feat: add MakeLineSeqFromReader iterator to ohio"
```

---

### Task 2: Register TypeZettelIdLogV1 and Remove TypeZettelIdListV0

**Files:**
- Modify: `go/internal/bravo/ids/types_builtin.go`

**Step 1: Update the const block**

In `go/internal/bravo/ids/types_builtin.go`, replace `TypeZettelIdListV0` (line 53) with the new type constants. The const block should read (around lines 26-53):

```go
	TypeWasmTagV1                                   = "!wasm-tag-v1"
	TypeZettelIdLogV1                               = "!zettel_id_log-v1"
	TypeZettelIdLogVCurrent                         = TypeZettelIdLogV1
	TypeTomlBlobStoreConfigSftpExplicitV0           = "!toml-blob_store_config_sftp-explicit-v0"
```

Remove line 53 (`TypeZettelIdListV0`).

**Step 2: Update init()**

Replace `registerBuiltinTypeString(TypeZettelIdListV0, genres.Unknown, false)` (line 133) with:

```go
	registerBuiltinTypeString(TypeZettelIdLogV1, genres.Unknown, false)
```

**Step 3: Verify no references to TypeZettelIdListV0**

Use LSP references on `TypeZettelIdListV0` to confirm nothing else uses it.

**Step 4: Run tests**

Run: `just test-go`
Expected: PASS

**Step 5: Commit**

```
git add go/internal/bravo/ids/types_builtin.go
git commit -m "feat: register TypeZettelIdLogV1, remove orphan TypeZettelIdListV0"
```

---

### Task 3: Add FileZettelIdLog to Directory Layout

**Files:**
- Modify: `go/internal/bravo/directory_layout/main.go:24-45` — add to `Repo` interface
- Modify: `go/internal/bravo/directory_layout/v3.go:118-129` — add method + update `DirsGenesis`

**Step 1: Add to Repo interface**

In `go/internal/bravo/directory_layout/main.go`, add `FileZettelIdLog() string` to the `Repo` interface after `FileInventoryListLog()` (line 42):

```go
		FileInventoryListLog() string
		FileZettelIdLog() string
```

**Step 2: Add v3 implementation**

In `go/internal/bravo/directory_layout/v3.go`, add after `FileInventoryListLog()` (line 120):

```go
func (layout v3) FileZettelIdLog() string {
	return layout.MakeDirData("zettel_id_log").String()
}
```

**Step 3: Run tests**

Run: `just test-go`
Expected: PASS

**Step 4: Commit**

```
git add go/internal/bravo/directory_layout/main.go go/internal/bravo/directory_layout/v3.go
git commit -m "feat: add FileZettelIdLog to directory layout"
```

---

### Task 4: Create Zettel ID Log Package

**Files:**
- Create: `go/internal/charlie/zettel_id_log/main.go` — Side enum + Entry interface
- Create: `go/internal/charlie/zettel_id_log/v1.go` — V1 struct
- Create: `go/internal/charlie/zettel_id_log/coding.go` — Coder variable

**Step 1: Create main.go**

```go
package zettel_id_log

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
)

type Side uint8

const (
	SideYin  Side = iota
	SideYang
)

type Entry interface {
	GetSide() Side
	GetTai() ids.Tai
	GetMarklId() markl.Id
	GetWordCount() int
}
```

**Step 2: Create v1.go**

```go
package zettel_id_log

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
)

var _ Entry = V1{}

type V1 struct {
	Side      Side     `toml:"side"`
	Tai       ids.Tai  `toml:"tai"`
	MarklId   markl.Id `toml:"markl-id"`
	WordCount int      `toml:"word-count"`
}

func (v V1) GetSide() Side {
	return v.Side
}

func (v V1) GetTai() ids.Tai {
	return v.Tai
}

func (v V1) GetMarklId() markl.Id {
	return v.MarklId
}

func (v V1) GetWordCount() int {
	return v.WordCount
}
```

**Step 3: Create coding.go**

Follow the `delta/blob_store_configs/coding.go` pattern (lines 43-54).

```go
package zettel_id_log

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

var Coder = triple_hyphen_io.CoderToTypedBlob[Entry]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[Entry]{},
	Blob: triple_hyphen_io.CoderTypeMapWithoutType[Entry](
		map[string]interfaces.CoderBufferedReadWriter[*Entry]{
			ids.TypeZettelIdLogV1: triple_hyphen_io.CoderToml[
				Entry,
				*Entry,
			]{
				Progenitor: func() Entry {
					return &V1{}
				},
			},
		},
	),
}
```

**Step 4: Run tests**

Run: `just test-go`
Expected: PASS (compiles, no consumers yet)

**Step 5: Commit**

```
git add go/internal/charlie/zettel_id_log/
git commit -m "feat: add zettel ID log entry interface, V1 struct, and coder"
```

---

### Task 5: Implement Zettel ID Log Reader/Writer

**Files:**
- Create: `go/internal/charlie/zettel_id_log/log.go`

**Step 1: Study the reference pattern**

Read `go/internal/golf/env_repo/genesis.go:104-135` — the `writeInventoryListLog()` method shows how `triple_hyphen_io` encodes typed blobs to files.

**Step 2: Write log.go**

```go
package zettel_id_log

import (
	"bufio"
	"os"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

type Log struct {
	Path string
}

func (l Log) AppendEntry(entry Entry) (err error) {
	var file *os.File

	if file, err = files.OpenFile(
		l.Path,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0o666,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	typedBlob := &triple_hyphen_io.TypedBlob[Entry]{
		Type: ids.GetOrPanic(ids.TypeZettelIdLogVCurrent).TypeStruct,
		Blob: entry,
	}

	if _, err = Coder.EncodeTo(typedBlob, file); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (l Log) ReadAllEntries() (entries []Entry, err error) {
	var file *os.File

	if file, err = files.Open(l.Path); err != nil {
		if errors.IsNotExist(err) {
			err = nil
			return entries, err
		}

		err = errors.Wrap(err)
		return entries, err
	}

	defer errors.DeferredCloser(&err, file)

	bufferedReader, repoolBufferedReader := pool.GetBufferedReader(file)
	defer repoolBufferedReader()

	segments, err := segmentEntries(bufferedReader)
	if err != nil {
		err = errors.Wrap(err)
		return entries, err
	}

	for _, segment := range segments {
		var typedBlob triple_hyphen_io.TypedBlob[Entry]

		stringReader, repoolStringReader := pool.GetStringReader(segment)
		defer repoolStringReader()

		if _, err = Coder.DecodeFrom(
			&typedBlob,
			stringReader,
		); err != nil {
			err = errors.Wrap(err)
			return entries, err
		}

		entries = append(entries, typedBlob.Blob)
	}

	return entries, err
}

func segmentEntries(
	reader *bufio.Reader,
) (segments []string, err error) {
	var current strings.Builder
	boundaryCount := 0

	for line, errIter := range ohio.MakeLineSeqFromReader(reader) {
		if errIter != nil {
			err = errIter
			return segments, err
		}

		trimmed := strings.TrimSuffix(line, "\n")

		if trimmed == triple_hyphen_io.Boundary {
			boundaryCount++

			if boundaryCount > 2 && boundaryCount%2 == 1 {
				segments = append(segments, current.String())
				current.Reset()
			}
		}

		current.WriteString(line)
	}

	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments, err
}
```

**Step 3: Verify imports resolve**

Check that `pool.GetBufferedReader`, `pool.GetStringReader`, `files.Open`, `files.OpenFile`, `errors.DeferredCloser`, and `triple_hyphen_io.Boundary` exist on master. Use LSP hover or grep if uncertain.

**Step 4: Run tests**

Run: `just test-go`
Expected: PASS

**Step 5: Commit**

```
git add go/internal/charlie/zettel_id_log/log.go
git commit -m "feat: implement zettel ID log reader and writer"
```

---

### Task 6: Rename object_id_provider to zettel_id_provider

**Files:**
- Rename directory: `go/internal/charlie/object_id_provider/` → `go/internal/charlie/zettel_id_provider/`
- Update all files in the package: change `package object_id_provider` → `package zettel_id_provider`
- Update all import paths across the codebase

**Step 1: Rename the directory**

```bash
mv go/internal/charlie/object_id_provider go/internal/charlie/zettel_id_provider
```

**Step 2: Update package declarations**

In every `.go` file under `go/internal/charlie/zettel_id_provider/`, change `package object_id_provider` to `package zettel_id_provider`.

**Step 3: Update all imports**

Find all files importing `charlie/object_id_provider` and update to `charlie/zettel_id_provider`. Also update all qualified references from `object_id_provider.` to `zettel_id_provider.`.

**Step 4: Run tests**

Run: `just test-go`
Expected: PASS

**Step 5: Commit**

```
git add -A
git commit -m "refactor: rename object_id_provider to zettel_id_provider"
```

---

### Task 7: Add Log Replay to Provider Factory

**Files:**
- Modify: `go/internal/charlie/zettel_id_provider/factory.go` — add `NewFromLog` + `BlobResolver`

**Step 1: Add BlobResolver type and NewFromLog function**

Add after the existing `New()` function in `go/internal/charlie/zettel_id_provider/factory.go`:

```go
// BlobResolver fetches a blob by its MarklId and returns the newline-delimited
// words it contains.
type BlobResolver func(markl.Id) ([]string, error)

// NewFromLog builds a Provider by replaying the zettel ID log. Each log entry
// references a blob containing delta words; resolveBlob fetches those words.
// When the log does not exist or is empty, falls back to reading flat files
// via New.
func NewFromLog(
	directoryLayout directory_layout.RepoMutable,
	resolveBlob BlobResolver,
) (f *Provider, err error) {
	log := zettel_id_log.Log{Path: directoryLayout.FileZettelIdLog()}

	var entries []zettel_id_log.Entry

	if entries, err = log.ReadAllEntries(); err != nil {
		err = errors.Wrap(err)
		return f, err
	}

	if len(entries) == 0 {
		return New(directoryLayout)
	}

	f = &Provider{
		Locker: &sync.Mutex{},
	}

	for _, entry := range entries {
		var words []string

		if words, err = resolveBlob(entry.GetMarklId()); err != nil {
			err = errors.Wrapf(err, "resolving blob for log entry")
			return f, err
		}

		switch entry.GetSide() {
		case zettel_id_log.SideYin:
			f.yin = append(f.yin, words...)

		case zettel_id_log.SideYang:
			f.yang = append(f.yang, words...)

		default:
			err = errors.ErrorWithStackf("unknown side: %d", entry.GetSide())
			return f, err
		}
	}

	return f, err
}
```

**Step 2: Add imports**

Add `markl` and `zettel_id_log` imports:

```go
import (
	"path"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_log"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)
```

**Step 3: Run tests**

Run: `just test-go`
Expected: PASS (no logs exist, so all repos use fallback path)

**Step 4: Commit**

```
git add go/internal/charlie/zettel_id_provider/factory.go
git commit -m "feat: support log-based provider loading with flat file fallback"
```

---

### Task 8: Implement migrate-zettel-ids Command

**Files:**
- Create: `go/internal/victor/commands_dodder/migrate_zettel_ids.go`
- Create: `zz-tests_bats/migrate_zettel_ids.bats`

**Step 1: Write the BATS test**

Create `zz-tests_bats/migrate_zettel_ids.bats`. Reference `robin:bats-testing` skill for test patterns.

```bash
#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"
  common_setup
}

teardown() {
  common_teardown
}

function migrate_zettel_ids { # @test
  run_dodder_init_disable_age

  run_dodder migrate-zettel-ids
  assert_success
  assert_output --partial "migrated Yin"
  assert_output --partial "migrated Yang"
}

function migrate_zettel_ids_idempotent { # @test
  run_dodder_init_disable_age

  run_dodder migrate-zettel-ids
  assert_success

  run_dodder migrate-zettel-ids
  assert_success
  assert_output --partial "already contains entries"
}
```

**Step 2: Run test to verify it fails**

Run: `just test-bats-targets migrate_zettel_ids.bats`
Expected: FAIL — command does not exist

**Step 3: Write the command**

Create `go/internal/victor/commands_dodder/migrate_zettel_ids.go`:

```go
package commands_dodder

import (
	"io"
	"path"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_log"
	"code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_provider"
	"code.linenisgreat.com/dodder/go/internal/delta/env_ui"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

func init() {
	utility.AddCmd("migrate-zettel-ids", &MigrateZettelIds{})
}

type MigrateZettelIds struct {
	command_components_dodder.LocalWorkingCopy
}

func (cmd MigrateZettelIds) Run(req command.Request) {
	req.AssertNoMoreArgs()

	localWorkingCopy := cmd.MakeLocalWorkingCopyWithOptions(
		req,
		env_ui.Options{},
		local_working_copy.OptionsAllowConfigReadError,
	)

	envRepo := localWorkingCopy.GetEnvRepo()
	log := zettel_id_log.Log{Path: envRepo.FileZettelIdLog()}

	entries, err := log.ReadAllEntries()
	if err != nil {
		errors.ContextCancelWithErrorf(req, "reading zettel id log: %s", err)
		return
	}

	if len(entries) > 0 {
		ui.Out().Print("zettel id log already contains entries, skipping migration")
		return
	}

	lockSmith := envRepo.GetLockSmith()

	req.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Lock))
	defer req.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Unlock))

	blobStore := envRepo.GetDefaultBlobStore()
	dirObjectId := envRepo.DirObjectId()
	tai := ids.NowTai()

	sides := []struct {
		side     zettel_id_log.Side
		fileName string
	}{
		{zettel_id_log.SideYin, zettel_id_provider.FilePathZettelIdYin},
		{zettel_id_log.SideYang, zettel_id_provider.FilePathZettelIdYang},
	}

	for _, s := range sides {
		flatPath := path.Join(dirObjectId, s.fileName)
		marklId, wordCount := writeFlatFileAsBlob(req, blobStore, flatPath)

		entry := &zettel_id_log.V1{
			Side:      s.side,
			Tai:       tai,
			MarklId:   marklId,
			WordCount: wordCount,
		}

		if err := log.AppendEntry(entry); err != nil {
			errors.ContextCancelWithErrorf(req, "appending %s log entry: %s", s.fileName, err)
			return
		}

		ui.Out().Printf("migrated %s: %d words, %s", s.fileName, wordCount, marklId)
	}
}

func writeFlatFileAsBlob(
	req command.Request,
	blobStore domain_interfaces.BlobStore,
	flatFilePath string,
) (markl.Id, int) {
	file, err := files.Open(flatFilePath)
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return markl.Id{}, 0
	}

	defer errors.ContextMustClose(req, file)

	reader, repool := pool.GetBufferedReader(file)
	defer repool()

	var wordCount int

	for line, err := range ohio.MakeLineSeqFromReader(reader) {
		if err != nil {
			errors.ContextCancelWithError(req, err)
			return markl.Id{}, 0
		}

		if strings.TrimRight(line, "\n") != "" {
			wordCount++
		}
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		errors.ContextCancelWithError(req, err)
		return markl.Id{}, 0
	}

	blobWriter, err := blobStore.MakeBlobWriter(nil)
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return markl.Id{}, 0
	}

	defer errors.ContextMustClose(req, blobWriter)

	if _, err := io.Copy(blobWriter, file); err != nil {
		errors.ContextCancelWithError(req, err)
		return markl.Id{}, 0
	}

	var id markl.Id
	id.ResetWithMarklId(blobWriter.GetMarklId())

	return id, wordCount
}
```

**Step 4: Run tests**

Run: `just test-bats-targets migrate_zettel_ids.bats`
Expected: PASS

**Step 5: Commit**

```
git add go/internal/victor/commands_dodder/migrate_zettel_ids.go zz-tests_bats/migrate_zettel_ids.bats
git commit -m "feat: add migrate-zettel-ids command"
```

---

### Task 9: Implement add-zettel-ids-yin and add-zettel-ids-yang Commands

**Files:**
- Create: `go/internal/victor/commands_dodder/add_zettel_ids.go`
- Create: `zz-tests_bats/add_zettel_ids.bats`

**Step 1: Write the BATS test**

Create `zz-tests_bats/add_zettel_ids.bats`:

```bash
#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"
  common_setup
}

teardown() {
  common_teardown
}

function add_zettel_ids_yin { # @test
  run_dodder_init_disable_age

  run_dodder migrate-zettel-ids
  assert_success

  run bash -c 'echo -e "alpha\nbravo\ncharlie" | dodder add-zettel-ids-yin'
  assert_success
  assert_output --partial "added 3 words"
  assert_output --partial "pool size"
}

function add_zettel_ids_yang { # @test
  run_dodder_init_disable_age

  run_dodder migrate-zettel-ids
  assert_success

  run bash -c 'echo -e "golf\nhotel\nindia" | dodder add-zettel-ids-yang'
  assert_success
  assert_output --partial "added 3 words"
}

function add_zettel_ids_dedup { # @test
  run_dodder_init_disable_age

  run_dodder migrate-zettel-ids
  assert_success

  run bash -c 'echo -e "alpha\nbravo" | dodder add-zettel-ids-yin'
  assert_success

  run bash -c 'echo -e "alpha\ncharlie" | dodder add-zettel-ids-yin'
  assert_success
  assert_output --partial "added 1 words"
}

function add_zettel_ids_cross_side_rejection { # @test
  run_dodder_init_disable_age

  run_dodder migrate-zettel-ids
  assert_success

  run bash -c 'echo -e "alpha" | dodder add-zettel-ids-yin'
  assert_success

  # alpha is already in yin, should be rejected from yang
  run bash -c 'echo -e "alpha" | dodder add-zettel-ids-yang'
  assert_success
  assert_output --partial "no new words"
}

function add_zettel_ids_no_new_words { # @test
  run_dodder_init_disable_age

  run_dodder migrate-zettel-ids
  assert_success

  # "one" is already in the yin provider from init
  run bash -c 'echo -e "one" | dodder add-zettel-ids-yin'
  assert_success
  assert_output --partial "no new words"
}
```

**Step 2: Run test to verify it fails**

Run: `just test-bats-targets add_zettel_ids.bats`
Expected: FAIL — commands do not exist

**Step 3: Write the command**

Create `go/internal/victor/commands_dodder/add_zettel_ids.go`:

```go
package commands_dodder

import (
	"bufio"
	"io"
	"os"
	"path"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_log"
	"code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_provider"
	"code.linenisgreat.com/dodder/go/internal/delta/env_ui"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/unicorn"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

func init() {
	utility.AddCmd("add-zettel-ids-yin", &AddZettelIds{
		side:         zettel_id_log.SideYin,
		flatFileName: zettel_id_provider.FilePathZettelIdYin,
	})

	utility.AddCmd("add-zettel-ids-yang", &AddZettelIds{
		side:         zettel_id_log.SideYang,
		flatFileName: zettel_id_provider.FilePathZettelIdYang,
	})
}

type AddZettelIds struct {
	command_components_dodder.LocalWorkingCopy
	side         zettel_id_log.Side
	flatFileName string
}

func (cmd AddZettelIds) Run(req command.Request) {
	req.AssertNoMoreArgs()

	candidates := readAndExtractCandidates(req)

	localWorkingCopy := cmd.MakeLocalWorkingCopyWithOptions(
		req,
		env_ui.Options{},
		local_working_copy.OptionsAllowConfigReadError,
	)

	envRepo := localWorkingCopy.GetEnvRepo()
	dirObjectId := envRepo.DirObjectId()

	prov, err := zettel_id_provider.New(envRepo)
	if err != nil {
		errors.ContextCancelWithErrorf(req, "loading zettel id provider: %s", err)
		return
	}

	existingWords := collectExistingWords(prov)

	var filtered []string

	for _, word := range candidates {
		cleaned := zettel_id_provider.Clean(word)

		if cleaned == "" {
			continue
		}

		if !existingWords[cleaned] {
			filtered = append(filtered, cleaned)
		}
	}

	if len(filtered) == 0 {
		ui.Out().Print("no new words to add")
		return
	}

	blobId := writeWordsAsBlob(req, envRepo.GetDefaultBlobStore(), filtered)

	lockSmith := envRepo.GetLockSmith()

	req.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Lock))
	defer req.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Unlock))

	log := zettel_id_log.Log{Path: envRepo.FileZettelIdLog()}
	flatFilePath := path.Join(dirObjectId, cmd.flatFileName)

	entry := &zettel_id_log.V1{
		Side:      cmd.side,
		Tai:       ids.NowTai(),
		MarklId:   blobId,
		WordCount: len(filtered),
	}

	if err := log.AppendEntry(entry); err != nil {
		errors.ContextCancelWithErrorf(req, "appending log entry: %s", err)
		return
	}

	appendWordsToFlatFile(req, flatFilePath, filtered)

	yinCount := prov.Left().Len()
	yangCount := prov.Right().Len()

	if cmd.side == zettel_id_log.SideYin {
		yinCount += len(filtered)
	} else {
		yangCount += len(filtered)
	}

	poolSize := yinCount * yangCount

	ui.Out().Printf(
		"added %d words to %s (pool size: %d)",
		len(filtered),
		cmd.flatFileName,
		poolSize,
	)
}

func readAndExtractCandidates(req command.Request) []string {
	reader := bufio.NewReader(os.Stdin)
	var lines []string

	for line, err := range ohio.MakeLineSeqFromReader(reader) {
		if err != nil {
			errors.ContextCancelWithError(req, err)
			return nil
		}

		lines = append(lines, strings.TrimRight(line, "\n"))
	}

	return unicorn.ExtractUniqueComponents(lines)
}

func collectExistingWords(prov *zettel_id_provider.Provider) map[string]bool {
	existing := make(map[string]bool)

	for _, word := range prov.Left() {
		existing[word] = true
	}

	for _, word := range prov.Right() {
		existing[word] = true
	}

	return existing
}

func writeWordsAsBlob(
	req command.Request,
	blobStore domain_interfaces.BlobStore,
	words []string,
) markl.Id {
	blobWriter, err := blobStore.MakeBlobWriter(nil)
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return markl.Id{}
	}

	defer errors.ContextMustClose(req, blobWriter)

	for _, word := range words {
		if _, err := io.WriteString(blobWriter, word); err != nil {
			errors.ContextCancelWithError(req, err)
			return markl.Id{}
		}

		if _, err := io.WriteString(blobWriter, "\n"); err != nil {
			errors.ContextCancelWithError(req, err)
			return markl.Id{}
		}
	}

	var id markl.Id
	id.ResetWithMarklId(blobWriter.GetMarklId())

	return id
}

func appendWordsToFlatFile(req command.Request, flatFilePath string, words []string) {
	file, err := files.OpenFile(
		flatFilePath,
		os.O_WRONLY|os.O_APPEND,
		0o666,
	)
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	defer errors.ContextMustClose(req, file)

	for _, word := range words {
		if _, err := io.WriteString(file, word); err != nil {
			errors.ContextCancelWithError(req, err)
			return
		}

		if _, err := io.WriteString(file, "\n"); err != nil {
			errors.ContextCancelWithError(req, err)
			return
		}
	}
}
```

**Step 4: Run tests**

Run: `just test-bats-targets add_zettel_ids.bats`
Expected: PASS

**Step 5: Commit**

```
git add go/internal/victor/commands_dodder/add_zettel_ids.go zz-tests_bats/add_zettel_ids.bats
git commit -m "feat: add add-zettel-ids-yin and add-zettel-ids-yang commands"
```

---

### Task 10: Update Genesis to Use Zettel ID Log

**Files:**
- Modify: `go/internal/golf/env_repo/genesis.go:80-94` — replace `CopyFileLines` with blob writes + log entries

**Step 1: Add new imports to genesis.go**

Add these imports to `go/internal/golf/env_repo/genesis.go`:

```go
	"code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_log"
	"code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_provider"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/alfa/unicorn"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
```

Remove the now-unused import:

```go
	"code.linenisgreat.com/dodder/go/lib/echo/ohio_files"
```

**Step 2: Replace CopyFileLines with genesis object IDs method**

Replace lines 80-94 (the two `ohio_files.CopyFileLines` blocks and the two `env.writeFile` calls) with:

```go
	env.BlobStoreEnv = MakeBlobStoreEnv(
		env.Env,
	)

	env.genesisObjectIds(bigBang)

	env.writeFile(env.FileConfig(), "")
	env.writeFile(env.FileCacheDormant(), "")
```

Also move the `env.BlobStoreEnv = MakeBlobStoreEnv(...)` block (lines 99-101) up BEFORE the `genesisObjectIds` call since blob writing needs an initialized blob store.

**Step 3: Add the genesisObjectIds method and helpers**

Add to genesis.go:

```go
func (env *Env) genesisObjectIds(bigBang BigBang) {
	yinWords := readAndCleanFileLines(env, bigBang.Yin)
	yangWords := readAndCleanFileLines(env, bigBang.Yang)

	enforceCrossSideUniqueness(yinWords, yangWords)

	yinSlice := orderedKeys(yinWords)
	yangSlice := orderedKeys(yangWords)

	yinBlobId := genesisWriteWordsAsBlob(env, yinSlice)
	yangBlobId := genesisWriteWordsAsBlob(env, yangSlice)

	tai := ids.NowTai()
	log := zettel_id_log.Log{Path: env.FileZettelIdLog()}

	yinEntry := &zettel_id_log.V1{
		Side:      zettel_id_log.SideYin,
		Tai:       tai,
		MarklId:   yinBlobId,
		WordCount: len(yinSlice),
	}

	if err := log.AppendEntry(yinEntry); err != nil {
		env.Cancel(err)
		return
	}

	yangEntry := &zettel_id_log.V1{
		Side:      zettel_id_log.SideYang,
		Tai:       tai,
		MarklId:   yangBlobId,
		WordCount: len(yangSlice),
	}

	if err := log.AppendEntry(yangEntry); err != nil {
		env.Cancel(err)
		return
	}

	genesisWriteFlatFile(env, filepath.Join(env.DirObjectId(), zettel_id_provider.FilePathZettelIdYin), yinSlice)
	genesisWriteFlatFile(env, filepath.Join(env.DirObjectId(), zettel_id_provider.FilePathZettelIdYang), yangSlice)
}

func readAndCleanFileLines(env *Env, filePath string) map[string]struct{} {
	file, err := files.Open(filePath)
	if err != nil {
		env.Cancel(err)
		return nil
	}

	defer errors.ContextMustClose(env, file)

	reader, repool := pool.GetBufferedReader(file)
	defer repool()

	words := make(map[string]struct{})

	for line, errIter := range ohio.MakeLineSeqFromReader(reader) {
		if errIter != nil {
			env.Cancel(errIter)
			return nil
		}

		cleaned := zettel_id_provider.Clean(line)

		if cleaned == "" {
			continue
		}

		words[cleaned] = struct{}{}
	}

	return words
}

func enforceCrossSideUniqueness(yin, yang map[string]struct{}) {
	for word := range yin {
		delete(yang, word)
	}
}

func orderedKeys(m map[string]struct{}) []string {
	result := make([]string, 0, len(m))

	for k := range m {
		result = append(result, k)
	}

	sort.Strings(result)

	return result
}

func genesisWriteWordsAsBlob(env *Env, words []string) markl.Id {
	blobWriter, err := env.GetDefaultBlobStore().MakeBlobWriter(nil)
	if err != nil {
		env.Cancel(err)
		return markl.Id{}
	}

	defer errors.ContextMustClose(env, blobWriter)

	for _, word := range words {
		if _, err := io.WriteString(blobWriter, word); err != nil {
			env.Cancel(err)
			return markl.Id{}
		}

		if _, err := io.WriteString(blobWriter, "\n"); err != nil {
			env.Cancel(err)
			return markl.Id{}
		}
	}

	var id markl.Id
	id.ResetWithMarklId(blobWriter.GetMarklId())

	return id
}

func genesisWriteFlatFile(env *Env, filePath string, words []string) {
	file, err := files.CreateExclusiveWriteOnly(filePath)
	if err != nil {
		env.Cancel(err)
		return
	}

	defer errors.ContextMustClose(env, file)

	for _, word := range words {
		if _, err := io.WriteString(file, word); err != nil {
			env.Cancel(err)
			return
		}

		if _, err := io.WriteString(file, "\n"); err != nil {
			env.Cancel(err)
			return
		}
	}
}
```

Also add `"sort"` to imports.

**Step 4: Run tests**

Run: `just test`
Expected: PASS (or fixture regeneration needed)

**Step 5: Update fixtures if needed**

Run: `just test-bats-update-fixtures`
Review the diff — genesis output format changed.

**Step 6: Commit**

```
git add go/internal/golf/env_repo/genesis.go
git commit -m "feat: update genesis to use zettel ID log with raw text processing"
```

---

### Task 11: Update Completion Tests and Final Fixture Update

**Files:**
- Modify: `zz-tests_bats/complete.bats:87-161` — add new subcommands

**Step 1: Add new subcommands to completion test**

In `zz-tests_bats/complete.bats`, add to the `complete_subcmd` expected output (after line 88 `add`):

```
		add-zettel-ids-yang
		add-zettel-ids-yin
```

And add after `merge-tool` (around line 145):

```
		migrate-zettel-ids
```

**Step 2: Run completion test**

Run: `just test-bats-targets complete.bats`
Expected: PASS

**Step 3: Run full test suite**

Run: `just test`
Expected: PASS

**Step 4: Update fixtures if needed**

Run: `just test-bats-update-fixtures`
Review and commit any changed fixtures.

**Step 5: Commit**

```
git add zz-tests_bats/complete.bats
git commit -m "test: add zettel ID commands to completion test"
```

If fixtures changed:

```
git add zz-tests_bats/migration/
git commit -m "chore: regenerate fixtures for zettel ID log changes"
```
