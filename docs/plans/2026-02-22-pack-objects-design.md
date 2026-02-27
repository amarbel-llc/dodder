# pack-objects Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `pack-objects` madder command that accepts files, blob IDs, and blob store IDs — writes files to the loose store, then packs all referenced blobs into the target archive.

**Architecture:** Two-pass approach. Pass 1 processes args like `write` (files → loose store, blob store IDs → switch target, blob hashes → collect). Pass 2 packs collected blob IDs into the target archive via `PackableArchive.Pack`. TAP-14 output throughout.

**Tech Stack:** Go, tap-dancer TAP-14 library, bats integration tests

---

### Task 1: Create pack_objects.go command file

**Files:**
- Create: `go/internal/lima/commands_madder/pack_objects.go`

**Step 1: Write the command file**

```go
package commands_madder

import (
	"fmt"
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/internal/hotel/tap_diagnostics"
	"code.linenisgreat.com/dodder/go/internal/india/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/india/env_local"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/command_components_madder"
	tap "github.com/amarbel-llc/tap-dancer/go"
)

func init() {
	utility.AddCmd("pack-objects", &PackObjects{})
}

type PackObjects struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStoreLocal

	DeleteLoose bool
}

var _ interfaces.CommandComponentWriter = (*PackObjects)(nil)

func (cmd PackObjects) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStores := envBlobStore.GetBlobStores()

	for id, blobStore := range blobStores {
		envLocal.GetOut().Printf("%s\t%s", id, blobStore.GetBlobStoreDescription())
	}
}

func (cmd *PackObjects) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.BoolVar(&cmd.DeleteLoose, "delete-loose", false,
		"validate archive then delete packed loose blobs")
}

func (cmd PackObjects) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStore := envBlobStore.GetDefaultBlobStore()

	tw := tap.NewWriter(os.Stdout)

	var blobStoreId blob_store_id.Id
	var collectedIds []domain_interfaces.MarklId

	sawStdin := false

	// Pass 1: process args — write files to loose, collect blob IDs
	for _, arg := range req.PopArgs() {
		switch {
		case arg == "-" && sawStdin:
			tw.Comment("'-' passed in more than once. Ignoring")
			continue

		case arg == "-":
			sawStdin = true
		}

		var blobReader domain_interfaces.BlobReader

		{
			var err error

			if blobReader, err = env_dir.NewFileReaderOrErrNotExist(
				env_dir.DefaultConfig,
				arg,
			); errors.IsNotExist(err) {
				if err = blobStoreId.Set(arg); err != nil {
					tw.BailOut(err.Error())
					req.Cancel(err)
					return
				}

				blobStore = envBlobStore.GetBlobStore(blobStoreId)
				tw.Comment(fmt.Sprintf("switched to blob store: %s", blobStoreId))
				continue
			} else if err != nil {
				tw.NotOk(arg, tap_diagnostics.FromError(err))
				continue
			}
		}

		blobId, err := cmd.doOne(blobStore, blobReader)

		if err != nil {
			tw.NotOk(arg, tap_diagnostics.FromError(err))
			continue
		}

		if blobId.IsNull() {
			tw.Skip(arg, "null digest")
			continue
		}

		tw.Ok(fmt.Sprintf("%s %s", blobId, arg))
		collectedIds = append(collectedIds, blobId)
	}

	// Pass 2: pack collected blobs into the target archive
	if len(collectedIds) == 0 {
		tw.Plan()
		return
	}

	packable, ok := blobStore.BlobStore.(blob_stores.PackableArchive)
	if !ok {
		tw.NotOk(
			fmt.Sprintf("pack %s", blobStoreId),
			map[string]string{
				"severity": "fail",
				"message":  "not packable",
			},
		)
		tw.Plan()
		return
	}

	if err := packable.Pack(blob_stores.PackOptions{
		DeleteLoose:          cmd.DeleteLoose,
		DeletionPrecondition: blob_stores.NopDeletionPrecondition(),
	}); err != nil {
		tw.NotOk(
			fmt.Sprintf("pack %s", blobStoreId),
			tap_diagnostics.FromError(err),
		)
		req.Cancel(err)
		return
	}

	tw.Ok(fmt.Sprintf("pack %s", blobStoreId))
	tw.Plan()
}

func (cmd PackObjects) doOne(
	blobStore blob_stores.BlobStoreInitialized,
	blobReader domain_interfaces.BlobReader,
) (blobId domain_interfaces.MarklId, err error) {
	defer errors.DeferredCloser(&err, blobReader)

	var writeCloser domain_interfaces.BlobWriter

	if writeCloser, err = blobStore.MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return blobId, err
	}

	defer errors.DeferredCloser(&err, writeCloser)

	if _, err = io.Copy(writeCloser, blobReader); err != nil {
		err = errors.Wrap(err)
		return blobId, err
	}

	blobId = writeCloser.GetMarklId()

	return blobId, err
}
```

**Step 2: Build to verify compilation**

Run: `just build` (from repo root using `/home/sasha/eng/result/bin/just`)
Expected: BUILD SUCCESS

**Step 3: Commit**

```
feat: add pack-objects madder command
```

---

### Task 2: Add pack-objects to completion test

**Files:**
- Modify: `zz-tests_bats/complete.bats:84-156` (the `complete_subcmd` test)

**Step 1: Add `blob_store-pack-objects` to the expected output**

Insert after the `blob_store-pack` line (line 104):

```
blob_store-pack-objects
```

**Step 2: Run the completion test**

Run: `just test-bats-targets complete.bats`
Expected: PASS

**Step 3: Commit**

```
test: add blob_store-pack-objects to completion test
```

---

### Task 3: Write bats integration tests

**Files:**
- Create: `zz-tests_bats/blob_store_pack_objects.bats`

**Step 1: Write the test file**

```bash
#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output
}

function pack_objects_no_args { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-pack-objects
	assert_success
	assert_output --partial 'TAP version 14'
	assert_output --partial '1..0'
}

function pack_objects_file_into_archive { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive .archive
	assert_success

	run_dodder blob_store-pack-objects ..archive <(echo pack-objects-test-content)
	assert_success
	assert_output --partial 'TAP version 14'
	assert_output --partial 'ok 1'
	assert_output --partial 'ok 2 - pack'
	refute_output --partial 'not ok'
}

function pack_objects_multiple_files { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive .archive
	assert_success

	run_dodder blob_store-pack-objects ..archive <(echo content-alpha) <(echo content-beta)
	assert_success
	assert_output --partial 'ok 1'
	assert_output --partial 'ok 2'
	assert_output --partial 'ok 3 - pack'
	refute_output --partial 'not ok'
}

function pack_objects_not_packable_store { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-pack-objects <(echo some-content)
	assert_success
	assert_output --partial 'not ok'
	assert_output --partial 'not packable'
}
```

**Step 2: Run the tests**

Run: `just test-bats-targets blob_store_pack_objects.bats`
Expected: ALL PASS

**Step 3: Commit**

```
test: add bats tests for pack-objects command
```

---

### Task 4: Full test suite verification

**Step 1: Run all tests**

Run: `just test` (from repo root)
Expected: ALL tests pass (318+ tests)

**Step 2: Commit any fixups if needed**

---
