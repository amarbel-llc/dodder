# Dodder info-repo Dynamic Key Access Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the hard-coded switch in dodder's `info-repo` with interface-driven dynamic key lookup (same pattern as madder's `info-repo`), drop all `blob_stores-0-*` legacy keys, and add CLI completion for all available keys.

**Architecture:** Repo-level keys (`config-immutable`, `store-version`, `id`, `pubkey`, `seckey`, `xdg`) stay hard-coded since they require genesis config. All blob-store config keys are handled dynamically via `blob_store_configs.ConfigKeyValues()` on the default blob store. Unknown keys error with a sorted list of all available keys. A `Complete` method exposes all keys for shell completion.

**Tech Stack:** Go interfaces, type assertions, `blob_store_configs.ConfigKeyValues`, `command.Completer` interface

---

### Task 1: Rewrite dodder info-repo with dynamic key lookup and completion

**Files:**
- Modify: `go/internal/yankee/commands_dodder/info_repo.go`

**Step 1: Replace the file contents**

```go
package commands_dodder

import (
	"sort"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/delta/xdg"
	"code.linenisgreat.com/dodder/go/internal/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/hotel/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/india/env_local"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
)

func init() {
	// TODO rename to repo-info
	utility.AddCmd("info-repo", &InfoRepo{})
}

type InfoRepo struct {
	command_components_dodder.EnvRepo
}

var repoSpecialKeys = []string{
	"config-immutable",
	"id",
	"pubkey",
	"seckey",
	"store-version",
	"xdg",
}

func (cmd InfoRepo) Run(req command.Request) {
	args := req.PopArgs()
	env := cmd.MakeEnvRepo(req, false)

	configPublicTypedBlob := env.GetConfigPublic()
	configPublicBlob := configPublicTypedBlob.Blob

	configPrivateTypedBlob := env.GetConfigPrivate()
	configPrivateBlob := configPrivateTypedBlob.Blob

	defaultBlobStore := env.GetDefaultBlobStore()

	if len(args) == 0 {
		args = []string{"store-version"}
	}

	configKVs := blob_store_configs.ConfigKeyValues(
		defaultBlobStore.Config.Blob,
	)

	for _, arg := range args {
		switch strings.ToLower(arg) {
		case "config-immutable":
			if _, err := genesis_configs.CoderPublic.EncodeTo(
				&configPublicTypedBlob,
				env.GetUIFile(),
			); err != nil {
				env.Cancel(err)
			}

		case "store-version":
			env.GetUI().Print(configPublicBlob.GetStoreVersion())

		case "id":
			env.GetUI().Print(configPublicBlob.GetRepoId())

		case "pubkey":
			env.GetUI().Print(
				configPublicBlob.GetPublicKey().StringWithFormat(),
			)

		case "seckey":
			env.Cancel(errors.Err405MethodNotAllowed)

			env.GetUI().Print(
				configPrivateBlob.GetPrivateKey().StringWithFormat(),
			)

		case "xdg":
			exdg := env.GetXDG()

			dotenv := xdg.Dotenv{
				XDG: &exdg,
			}

			if _, err := dotenv.WriteTo(env.GetUIFile()); err != nil {
				env.Cancel(err)
			}

		default:
			value, ok := configKVs[arg]
			if !ok {
				allKeys := allAvailableKeys(
					defaultBlobStore.Config.Blob,
				)

				errors.ContextCancelWithBadRequestf(
					env,
					"unsupported info key: %q\navailable keys: %s",
					arg,
					strings.Join(allKeys, ", "),
				)

				return
			}

			env.GetUI().Print(value)
		}
	}
}

func allAvailableKeys(config blob_store_configs.Config) []string {
	configKeys := blob_store_configs.ConfigKeyNames(config)
	allKeys := make([]string, 0, len(repoSpecialKeys)+len(configKeys))
	allKeys = append(allKeys, repoSpecialKeys...)
	allKeys = append(allKeys, configKeys...)
	sort.Strings(allKeys)

	return allKeys
}

func (cmd InfoRepo) Complete(
	req command.Request,
	envLocal env_local.Env,
	_ command.CommandLineInput,
) {
	env := cmd.MakeEnvRepo(req, false)
	defaultBlobStore := env.GetDefaultBlobStore()
	keys := allAvailableKeys(defaultBlobStore.Config.Blob)

	for _, key := range keys {
		envLocal.GetUI().Print(key)
	}
}
```

Key changes from the original:
- Removed `command_components_madder.BlobStoreConfig` embedding (only needed for dropped `blob_stores-0-config` case)
- Removed all `blob_stores-0-*` cases and the `compression-type` hard-coded case
- Removed `directory_layout` import (was for `blob_stores-0-base-path` and `blob_stores-0-config-path`)
- Added `ConfigKeyValues` dynamic lookup in `default:` case
- Added `allAvailableKeys` helper combining repo-level + config keys
- Added `Complete` method implementing `command.Completer` for shell completion
- Added `env_local` and `sort` imports; removed unused imports

**Step 2: Build to verify compilation**

Run: `just build` from `go/` directory
Expected: PASS

**Step 3: Commit**

```
refactor: rewrite dodder info-repo with interface-driven key lookup

Drop all blob_stores-0-* legacy keys and hard-coded compression-type
case. Repo-level keys (config-immutable, store-version, id, pubkey,
seckey, xdg) stay as special cases. All blob store config keys are now
handled dynamically via ConfigKeyValues on the default blob store.
Add Complete method for CLI key completion.
```

---

### Task 2: Update BATS tests for new key names

**Files:**
- Modify: `zz-tests_bats/info_repo.bats`

**Step 1: Update encryption tests to use bare key name**

Change line 71 from `blob_stores-0-encryption` to `encryption`:

```bash
function info_age_none { # @test
	run_dodder_init_disable_age
	run_dodder info-repo encryption
	assert_output ''
}
```

Change line 82 from `blob_stores-0-encryption` to `encryption`:

```bash
function info_age_some { # @test
	run_dodder gen madder-private_key-v1
	assert_output --regexp 'madder-private_key-v1@age_x25519_sec-'
	key="$output"
	echo "$key" >age-key
	run_dodder_init -override-xdg-with-cwd -encryption age-key test-repo-id
	run_dodder info-repo encryption
	assert_output "$key"
}
```

**Step 2: Add test for unknown key error**

Append after the last test:

```bash
function info_repo_unknown_key_fails { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder info-repo nonexistent-key
	assert_failure
}
```

**Step 3: Add test for dynamic config key**

Append after the unknown key test:

```bash
function info_repo_dynamic_config_key { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder info-repo blob-store-type
	assert_success
	assert_output --regexp '.+'
}
```

**Step 4: Build and run BATS tests**

Run: `just build && just test-bats-run` from `go/` directory (via nix devshell)
Expected: All info_repo.bats tests PASS

**Step 5: Commit**

```
test: update info-repo BATS tests for dynamic key lookup

Replace blob_stores-0-encryption with bare encryption key name. Add
tests for unknown key error and dynamic config key lookup.
```

---

### Task 3: Run full test suite

**Step 1: Run all tests**

Run: `just test` from `go/` directory (via nix devshell)
Expected: All tests PASS (unit + BATS)

**Step 2: If migration fixtures changed, commit them separately**

Run: `git status -- zz-tests_bats/migration/`

If changed:
```
chore: regenerate migration fixtures
```

---

## Summary of Changes

| File | Action | Purpose |
|------|--------|---------|
| `go/internal/yankee/commands_dodder/info_repo.go` | Rewrite | Dynamic key lookup via `ConfigKeyValues`, drop legacy keys, add `Complete` |
| `zz-tests_bats/info_repo.bats` | Modify | Update `blob_stores-0-encryption` → `encryption`, add error + dynamic key tests |
