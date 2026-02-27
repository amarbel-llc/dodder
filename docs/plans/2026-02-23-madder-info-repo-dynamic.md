# Madder info-repo Dynamic Key Access Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the hard-coded switch in madder's `info-repo` with interface-driven dynamic key lookup so any blob store config field is queryable without code changes.

**Architecture:** Add a `ConfigKeyValues(Config) map[string]string` function in `golf/blob_store_configs` that type-asserts the config to each interface and collects key-value pairs. Madder's `info-repo` command then looks up requested keys in this map, falling back to special non-config keys (`config-immutable`, `config-path`, `dir-blob_stores`, `xdg`). Key names match TOML struct tags.

**Tech Stack:** Go interfaces, type assertions, `fmt.Sprint`/`Stringer` for formatting

---

### Task 1: Add `ConfigKeyValues` function with base Config support

**Files:**
- Create: `go/internal/golf/blob_store_configs/key_values.go`

**Step 1: Write the function**

```go
package blob_store_configs

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
)

// ConfigKeyValues returns a map of TOML-tag-named keys to string-formatted
// values for the given config. The set of keys depends on which interfaces the
// config implements.
func ConfigKeyValues(config Config) map[string]string {
	keyValues := make(map[string]string)

	keyValues["blob-store-type"] = config.GetBlobStoreType()

	if configHashType, ok := config.(ConfigHashType); ok {
		keyValues["hash_type-id"] = configHashType.GetDefaultHashTypeId()
		keyValues["supports-multi-hash"] = fmt.Sprint(
			configHashType.SupportsMultiHash(),
		)
	}

	if blobIOWrapper, ok := config.(domain_interfaces.BlobIOWrapper); ok {
		keyValues["encryption"] = blobIOWrapper.GetBlobEncryption().
			StringWithFormat()
		keyValues["compression-type"] = fmt.Sprint(
			blobIOWrapper.GetBlobCompression(),
		)
	}

	if configLocal, ok := config.(ConfigLocalHashBucketed); ok {
		keyValues["hash_buckets"] = fmt.Sprint(
			configLocal.GetHashBuckets(),
		)
		keyValues["lock-internal-files"] = fmt.Sprint(
			configLocal.GetLockInternalFiles(),
		)
	}

	if configArchive, ok := config.(ConfigInventoryArchive); ok {
		keyValues["loose-blob-store-id"] = configArchive.
			GetLooseBlobStoreId().String()
		keyValues["max-pack-size"] = fmt.Sprint(
			configArchive.GetMaxPackSize(),
		)
	}

	if configDelta, ok := config.(DeltaConfigImmutable); ok {
		keyValues["delta.enabled"] = fmt.Sprint(
			configDelta.GetDeltaEnabled(),
		)
		keyValues["delta.algorithm"] = configDelta.GetDeltaAlgorithm()
		keyValues["delta.min-blob-size"] = fmt.Sprint(
			configDelta.GetDeltaMinBlobSize(),
		)
		keyValues["delta.max-blob-size"] = fmt.Sprint(
			configDelta.GetDeltaMaxBlobSize(),
		)
		keyValues["delta.size-ratio"] = fmt.Sprint(
			configDelta.GetDeltaSizeRatio(),
		)
	}

	if configSFTP, ok := config.(ConfigSFTPConfigExplicit); ok {
		keyValues["host"] = configSFTP.GetHost()
		keyValues["port"] = fmt.Sprint(configSFTP.GetPort())
		keyValues["user"] = configSFTP.GetUser()
		keyValues["private-key-path"] = configSFTP.GetPrivateKeyPath()
		keyValues["remote-path"] = configSFTP.GetRemotePath()
	} else if configSFTPRemote, ok := config.(ConfigSFTPRemotePath); ok {
		keyValues["remote-path"] = configSFTPRemote.GetRemotePath()
	}

	return keyValues
}

// ConfigKeyNames returns sorted key names available for a config.
func ConfigKeyNames(config Config) []string {
	keyValues := ConfigKeyValues(config)
	keys := make([]string, 0, len(keyValues))

	for key := range keyValues {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
```

Note: Will need `"sort"` import for `ConfigKeyNames`.

**Step 2: Build to verify compilation**

Run: `just build` from `go/` directory
Expected: PASS

**Step 3: Commit**

```
feat: add ConfigKeyValues for interface-driven config introspection
```

---

### Task 2: Write failing BATS test for dynamic info-repo keys

**Files:**
- Create: `zz-tests_bats/blob_store_info_repo.bats`

**Step 1: Write the failing tests**

```bash
#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_and_rm
}

# bats file_tags=user_story:blob_store,user_story:repo_info

function info_repo_encryption_default_store { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-info-repo encryption
	assert_success
	assert_output ''
}

function info_repo_compression_type_default_store { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-info-repo compression-type
	assert_success
	assert_output 'zstd'
}

function info_repo_hash_type_id_default_store { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-info-repo hash_type-id
	assert_success
	assert_output 'blake2b256'
}

function info_repo_lock_internal_files_default_store { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-info-repo lock-internal-files
	assert_success
	assert_output 'false'
}

function info_repo_archive_encryption { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive -encryption generate .archive
	assert_success

	run_dodder blob_store-info-repo .archive encryption
	assert_success
	assert_output --regexp '.+'
}

function info_repo_archive_delta_enabled { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive -delta .archive
	assert_success

	run_dodder blob_store-info-repo .archive delta.enabled
	assert_success
	assert_output 'true'
}

function info_repo_archive_max_pack_size { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive .archive
	assert_success

	run_dodder blob_store-info-repo .archive max-pack-size
	assert_success
	assert_output '0'
}

function info_repo_unknown_key_fails { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-info-repo nonexistent-key
	assert_failure
}
```

**Step 2: Run tests to verify they fail**

Run: `just build && just test-bats-run` from `go/` directory
Expected: All new tests FAIL (unknown keys or wrong output format)

**Step 3: Commit**

```
test: add BATS tests for dynamic blob_store-info-repo keys
```

---

### Task 3: Rewrite madder info-repo to use ConfigKeyValues

**Files:**
- Modify: `go/internal/lima/commands_madder/info_repo.go`

**Step 1: Rewrite the command handler**

Replace the hard-coded switch with:
1. Check special keys first (`config-immutable`, `config-path`, `dir-blob_stores`, `xdg`)
2. For all other keys, look up in `ConfigKeyValues(blobStoreConfig.Blob)`
3. On unknown key, error listing available keys

```go
package commands_madder

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/delta/xdg"
	"code.linenisgreat.com/dodder/go/internal/echo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/india/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/command_components_madder"
)

func init() {
	// TODO rename to repo-info
	utility.AddCmd("info-repo", &InfoRepo{})
}

type InfoRepo struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStoreConfig
	command_components_madder.BlobStore
}

func (cmd InfoRepo) Run(req command.Request) {
	env := cmd.MakeEnvBlobStore(req)

	var blobStore blob_stores.BlobStoreInitialized
	var keys []string

	switch req.RemainingArgCount() {
	case 0:
		blobStore = env.GetDefaultBlobStore()
		keys = []string{"config-immutable"}

	case 1:
		blobStore = env.GetDefaultBlobStore()
		keys = []string{req.PopArg("blob store config key")}

	case 2:
		blobStoreIndex := req.PopArg("blob store index")
		blobStore = cmd.MakeBlobStoreFromIdString(env, blobStoreIndex)
		keys = []string{req.PopArg("blob store config key")}

	default:
		blobStoreIndex := req.PopArg("blob store index")
		blobStore = cmd.MakeBlobStoreFromIdString(env, blobStoreIndex)
		keys = req.PopArgs()
	}

	blobStoreConfig := blobStore.Config
	configKVs := blob_store_configs.ConfigKeyValues(blobStoreConfig.Blob)

	for _, key := range keys {
		switch strings.ToLower(key) {
		case "config-immutable":
			if _, err := blob_store_configs.Coder.EncodeTo(
				&blobStoreConfig,
				env.GetUIFile(),
			); err != nil {
				env.Cancel(err)
			}

		case "config-path":
			env.GetUI().Print(
				directory_layout.GetDefaultBlobStore(env).GetConfig(),
			)

		case "dir-blob_stores":
			env.GetUI().Print(env.MakePathBlobStore())

		case "xdg":
			ecksDeeGee := env.GetXDG()

			dotenv := xdg.Dotenv{
				XDG: &ecksDeeGee,
			}

			if _, err := dotenv.WriteTo(env.GetUIFile()); err != nil {
				env.Cancel(err)
			}

		default:
			value, ok := configKVs[key]
			if !ok {
				availableKeys := blob_store_configs.ConfigKeyNames(
					blobStoreConfig.Blob,
				)

				errors.ContextCancelWithBadRequestf(
					env,
					"unsupported info key: %q\navailable keys: %s",
					key,
					strings.Join(availableKeys, ", "),
				)

				return
			}

			env.GetUI().Print(value)
		}
	}
}
```

**Step 2: Build and run tests**

Run: `just build && just test-bats-run` from `go/` directory
Expected: All new `blob_store_info_repo.bats` tests PASS. All existing tests PASS.

**Step 3: Commit**

```
refactor: rewrite madder info-repo with interface-driven key lookup
```

---

### Task 4: Update init.bats encryption test to use info-repo

**Files:**
- Modify: `zz-tests_bats/init.bats`

**Step 1: Simplify the archive encryption test**

Replace the grep-based config file reading with the new info-repo command:

```bash
function init_inventory_archive_with_encryption { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive -encryption generate .archive
	assert_success

	run_dodder blob_store-info-repo .archive encryption
	assert_success
	assert_output --regexp '.+'
}
```

**Step 2: Run tests**

Run: `just build && just test-bats-run` from `go/` directory
Expected: All tests PASS

**Step 3: Commit**

```
refactor: use blob_store-info-repo in archive encryption init test
```

---

### Task 5: Backward-compat for dodder info-repo legacy key names

**Files:**
- Modify: `go/internal/lima/commands_madder/info_repo.go`

The existing dodder `info-repo` tests use `blob_stores-0-encryption`, `blob_stores-0-config-path`, and `compression-type`. The new system uses bare names (`encryption`, `config-path`). The legacy `blob_stores-0-*` prefix keys are used through dodder's own info-repo command (different code path in `yankee/commands_dodder/info_repo.go`), so no backward compat needed in madder's handler. Just verify existing tests still pass.

**Step 1: Run full test suite**

Run: `just test` from `go/` directory (unit + bats)
Expected: All tests PASS

**Step 2: Commit (if any adjustments needed)**

```
chore: verify backward compatibility of info-repo changes
```

---

### Task 6: Add blob_store-info-repo to complete.bats if needed

**Files:**
- Check: `zz-tests_bats/complete.bats` — `blob_store-info-repo` is already listed at line 93. No changes needed unless the command name changes.

**Step 1: Verify completion test passes**

Run: `just build && just test-bats-run` from `go/` directory
Expected: `complete_subcmd` test PASS

---

## Summary of Changes

| File | Action | Purpose |
|------|--------|---------|
| `go/internal/golf/blob_store_configs/key_values.go` | Create | `ConfigKeyValues` + `ConfigKeyNames` functions |
| `go/internal/lima/commands_madder/info_repo.go` | Rewrite | Dynamic key lookup via `ConfigKeyValues` |
| `zz-tests_bats/blob_store_info_repo.bats` | Create | BATS tests for all dynamic keys |
| `zz-tests_bats/init.bats` | Modify | Use `blob_store-info-repo` instead of grep |

---

## Learnings and Strategy

### Why interfaces, not TOML serialization

The initial instinct was to marshal the config to TOML then unmarshal to `map[string]string`. This was rejected: "the serialization format should not be leaned on, we should lean on the interfaces instead." The config type hierarchy already defines clean accessor interfaces (`ConfigHashType`, `ConfigLocalHashBucketed`, `ConfigInventoryArchive`, `DeltaConfigImmutable`, `BlobIOWrapper`, `ConfigSFTPConfigExplicit`, `ConfigSFTPRemotePath`). Each config struct implements a different subset. A single function type-asserts against each interface and collects key-value pairs — so the available keys automatically match what the config actually supports.

### Architecture

```
blob_store_configs (golf layer)
├── ConfigKeyValues(Config) map[string]string   ← dynamic, interface-driven
├── ConfigKeyNames(Config) []string             ← sorted, for error messages
└── key_values.go

commands_madder (lima layer)
└── info_repo.go
    ├── special keys: config-immutable, config-path, dir-blob_stores, xdg
    └── default: lookup in ConfigKeyValues map, error lists available keys
```

`ConfigKeyValues` lives in `golf/blob_store_configs` (same package as the interfaces), not in the command handler. This makes it reusable — dodder's info-repo could use it too.

### Key naming convention

Keys match TOML struct tags exactly (`compression-type`, `hash_type-id`, `lock-internal-files`). Delta keys use dot notation (`delta.enabled`, `delta.algorithm`). Users and tests can look at a config file and know the exact key name.

### Special vs dynamic keys

Four keys need custom handling that can't be reduced to a string value:
- `config-immutable` — encodes full typed config blob to output
- `config-path` — filesystem path from directory layout, not a config value
- `dir-blob_stores` — filesystem path
- `xdg` — writes multi-line dotenv format

Everything else goes through the dynamic map.

### Test strategy

Tests use `blob_store-info-repo` as a first-class introspection tool instead of grepping config files or using dodder's more complex `info-repo`. This exercises the actual config interfaces rather than the serialization format. After implementation, 3 existing init.bats tests were migrated from `info-repo blob_stores-0-encryption` to `blob_store-info-repo encryption`.

### What this enables next

Dodder's `info-repo` (in `yankee/commands_dodder`) has the same hard-coded switch problem plus repo-level keys (`store-version`, `id`, `pubkey`, `seckey`). It could reuse `ConfigKeyValues` for blob store config keys while keeping its own special keys for repo-level config. The `blob_stores-0-*` prefix keys in dodder's info-repo could eventually be deprecated in favor of `blob_store-info-repo`.
