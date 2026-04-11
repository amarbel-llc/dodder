# Repo Disambiguation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add `-repo_id` flag and `DODDER_REPO_ID` env var to select between CWD (`.`) and XDG user repos, then remove `-override-xdg-with-cwd`.

**Architecture:** Extract `xdg_location_type` package from `blob_store_id`, create `repo_id` package consuming it, wire into `repo_config_cli` as global flag, consume in `env_dir` construction, replace all `-override-xdg-with-cwd` usage.

**Tech Stack:** Go, BATS integration tests

**Rollback:** Revert commits. No persistent state affected.

---

### Task 1: Extract `xdg_location_type` package

**Files:**
- Create: `go/internal/0/xdg_location_type/main.go`
- Modify: `go/internal/0/blob_store_id/location.go`
- Modify: `go/internal/0/blob_store_id/main.go`

**Step 1: Create `xdg_location_type` package**

Create `go/internal/0/xdg_location_type/main.go` with the `Type` enum extracted from `blob_store_id/location.go`. Keep the same values and prefix logic:

```go
package xdg_location_type

import "code.linenisgreat.com/dodder/go/lib/bravo/errors"

type (
	Type interface {
		TypeGetter
		xdgLocationType()
		GetPrefix() rune
	}

	TypeGetter interface {
		GetLocationType() Type
	}

	//go:generate stringer -type=typee
	typee int
)

const (
	Unknown = typee(iota)
	Cwd
	XDGUser
	XDGSystem
)

var (
	_ TypeGetter = typee(0)
	_ Type       = typee(0)
)

func (typee) xdgLocationType() {}

func (t typee) GetLocationType() Type { return t }

func (t *typee) SetPrefix(firstChar rune) (err error) {
	switch firstChar {
	case '/':
		*t = XDGSystem
	case '~':
		*t = XDGUser
	case '.':
		*t = Cwd
	case '_':
		*t = Unknown
	default:
		err = errors.Errorf(
			"unsupported rune for location type: %q",
			string(firstChar),
		)
		return err
	}
	return err
}

func (t typee) IsPrefix(r rune) bool {
	switch r {
	case '/', '~', '.', '_':
		return true
	default:
		return false
	}
}

func (t typee) GetPrefix() rune {
	switch t {
	case XDGSystem:
		return '/'
	case XDGUser:
		return 0
	case Cwd:
		return '.'
	case Unknown:
		return '_'
	default:
		panic(errors.Errorf("unsupported location type: %q", t))
	}
}
```

**Step 2: Refactor `blob_store_id/location.go`**

Replace `location` type with a type alias to `xdg_location_type.typee`. Keep the public `LocationType*` constants as aliases:

```go
package blob_store_id

import "code.linenisgreat.com/dodder/go/internal/0/xdg_location_type"

type (
	LocationType       = xdg_location_type.Type
	LocationTypeGetter = xdg_location_type.TypeGetter
)

var (
	LocationTypeUnknown   = xdg_location_type.Unknown
	LocationTypeCwd       = xdg_location_type.Cwd
	LocationTypeXDGUser   = xdg_location_type.XDGUser
	LocationTypeXDGSystem = xdg_location_type.XDGSystem
)
```

**Step 3: Update `blob_store_id/main.go`**

Change the `location` field type from internal `location` to `xdg_location_type.Type`. Update `Set()`, `GetLocationType()`, `String()`, `Less()` to use the new type's methods.

**Step 4: Build and verify**

Run: `cd go && go build ./...`
Expected: compiles with no errors

**Step 5: Run unit tests**

Run: `just test-go`
Expected: all pass

**Step 6: Commit**

Message: `refactor: extract xdg_location_type package from blob_store_id`

---

### Task 2: Create `repo_id` package

**Files:**
- Create: `go/internal/0/repo_id/main.go`

**Step 1: Create `repo_id/main.go`**

```go
package repo_id

import (
	"code.linenisgreat.com/dodder/go/internal/0/xdg_location_type"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type Id struct {
	locationType xdg_location_type.Type
	isSet        bool
}

func (id Id) IsEmpty() bool {
	return !id.isSet
}

func (id Id) GetLocationType() xdg_location_type.Type {
	return id.locationType
}

func (id *Id) Set(value string) (err error) {
	switch value {
	case "":
		id.isSet = false
		return nil

	case ".":
		id.locationType = xdg_location_type.Cwd
		id.isSet = true
		return nil

	case "/":
		id.locationType = xdg_location_type.XDGSystem
		id.isSet = true
		return nil

	default:
		if len(value) > 1 && value[0] == '/' {
			err = errors.Errorf(
				"remote repo selection (/%s) not yet implemented",
				value[1:],
			)
			return err
		}

		err = errors.Errorf("invalid repo_id: %q (expected . or /)", value)
		return err
	}
}

func (id Id) String() string {
	if !id.isSet {
		return ""
	}

	prefix := id.locationType.GetPrefix()
	if prefix == 0 {
		return ""
	}

	return string(prefix)
}

func (id Id) IsCwd() bool {
	return id.isSet && id.locationType == xdg_location_type.Cwd
}

func (id Id) IsSystem() bool {
	return id.isSet && id.locationType == xdg_location_type.XDGSystem
}
```

**Step 2: Build**

Run: `cd go && go build ./...`
Expected: compiles

**Step 3: Commit**

Message: `feat: add repo_id package for repo location selection`

---

### Task 3: Wire `repo_id` into `repo_config_cli` and `env_dir`

**Files:**
- Modify: `go/internal/charlie/repo_config_cli/main.go`
- Modify: `go/internal/echo/env_dir/construction.go`
- Modify: `go/internal/uniform/command_components_dodder/env_repo.go`
- Modify: `go/internal/uniform/command_components_dodder/genesis.go`

**Step 1: Add `RepoId` to `repo_config_cli.Config`**

Add field and flag registration. Read `DODDER_REPO_ID` env var as default:

```go
import (
	"os"
	"code.linenisgreat.com/dodder/go/internal/0/repo_id"
)

// In Config struct:
RepoId repo_id.Id

// In SetFlagDefinitions:
flagSet.Var(&config.RepoId, "repo_id", "repo location: . (cwd) or / (system)")

// After flag definition, set default from env var if flag not set:
// (This happens at flag parse time via the Var interface — the env var
// default is set in Default())
```

In `Default()`, read env var:

```go
func Default() (config *Config) {
	config = &Config{
		Config: *(config_cli.Default()),
	}

	if envRepoId := os.Getenv("DODDER_REPO_ID"); envRepoId != "" {
		if err := config.RepoId.Set(envRepoId); err != nil {
			// env var is invalid — ignore, let flag override or error later
		}
	}

	return config
}
```

Add getter:

```go
func (config Config) GetRepoId() repo_id.Id {
	return config.RepoId
}
```

**Step 2: Update `env_dir.MakeDefaultAndInitialize` signature**

Change `overrideXDGWithCwd bool` parameter to accept `repo_id.Id`:

```go
func MakeDefaultAndInitialize(
	context errors.Context,
	utilityName string,
	do debug.Options,
	repoId repo_id.Id,
) env {
```

Inside, replace the bool switch:

```go
if repoId.IsSystem() {
	panic(errors.WithoutStack(errors.Err501NotImplemented))
}

if repoId.IsCwd() {
	// explicit CWD selection
	var cwd string
	{
		var err error
		if cwd, err = os.Getwd(); err != nil {
			context.Cancel(err)
		}
	}
	return MakeWithXDGRootOverrideHomeAndInitialize(
		context, cwd, utilityName, do,
	)
} else {
	// auto-detect: MakeWithDefaultHome with permitCwdXDGOverride=true
	return MakeWithDefaultHome(
		context, utilityName, do, true, true,
	)
}
```

**Step 3: Update `genesis.go` call site**

Change the `OnTheFirstDay` call from `cmd.OverrideXDGWithCwd` to pass a
`repo_id.Id`. For now, convert the bool: if `OverrideXDGWithCwd` is true, create
a CWD repo_id. This keeps genesis working during the transition.

```go
repoIdForDir := repo_id.Id{}
if cmd.OverrideXDGWithCwd {
	repoIdForDir.Set(".")
}

dir := env_dir.MakeDefaultAndInitialize(
	req, env_dir.XDGUtilityNameDodder, config.Debug, repoIdForDir,
)
```

**Step 4: Update `env_repo.go` MakeEnvRepo**

Pass the repo_id from config through to `env_dir.MakeDefault`. Since
`MakeDefault` currently calls `MakeWithDefaultHome` with auto-detect, and
`MakeEnvRepo` doesn't use `MakeDefaultAndInitialize`, we need to check: if
`RepoId.IsCwd()`, use `MakeDefaultAndInitialize` with the repo_id instead.

```go
func (cmd EnvRepo) MakeEnvRepo(
	req command.Request,
	permitNoDodderDirectory bool,
) env_repo.Env {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

	var dir env_dir.Env
	if config.RepoId.IsCwd() || config.RepoId.IsSystem() {
		dir = env_dir.MakeDefaultAndInitialize(
			req, env_dir.XDGUtilityNameDodder, config.Debug, config.RepoId,
		)
	} else {
		dir = env_dir.MakeDefault(
			req, env_dir.XDGUtilityNameDodder, config.Debug,
		)
	}
	// ... rest unchanged
```

**Step 5: Build and run unit tests**

Run: `cd go && go build ./... && just test-go`
Expected: compiles and passes

**Step 6: Run integration tests**

Run: `just test-bats`
Expected: all pass (no behavior change yet)

**Step 7: Commit**

Message: `feat: wire repo_id flag and DODDER_REPO_ID env var into CLI`

---

### Task 4: Replace `-override-xdg-with-cwd` with `-repo_id .` in tests

**Files:**
- Modify: `zz-tests_bats/lib/common.bash` (lines 133, 156, 225)
- Modify: `zz-tests_bats/current_version/init.bats` (lines 136, 181, 207, 240)
- Modify: `zz-tests_bats/current_version/push.bats` (lines 70, 80)
- Modify: `zz-tests_bats/current_version/clone.bats` (lines 20, 61)
- Modify: `zz-tests_bats/current_version/pull.bats` (line 29)
- Modify: `zz-tests_bats/current_version/info.bats` (line 49)
- Modify: `zz-tests_bats/current_version/info_repo.bats` (lines 80, 109)
- Modify: `zz-tests_bats/current_version/remote_add.bats` (line 58)
- Modify: `zz-tests_bats/current_version/init_ecdsa_p256.bats` (line 49)
- Modify: `zz-tests_bats/current_version/import.bats` (line 20)
- Modify: `zz-tests_bats/current_version/inventory_list_json.bats` (lines 23, 79, 134)

**Step 1: Replace all occurrences**

In every file listed above, replace `-override-xdg-with-cwd` with `-repo_id .`
(exact string replacement).

In `common.bash`, the three helper functions (`run_dodder_init`,
`run_dodder_init_sha256`, `run_dodder_init_disable_age`) all use this flag.

**Step 2: Run integration tests**

Run: `just test-bats`
Expected: all pass

**Step 3: Commit**

Message: `test: replace -override-xdg-with-cwd with -repo_id . in all BATS tests`

---

### Task 5: Remove `-override-xdg-with-cwd` from codebase

**Files:**
- Modify: `go/internal/golf/env_repo/big_bang.go` — remove `OverrideXDGWithCwd` field
- Modify: `go/internal/uniform/command_components_dodder/genesis.go` — remove flag registration and bool-to-repo_id conversion
- Modify: skills and docs referencing the old flag

**Step 1: Remove `OverrideXDGWithCwd` from `BigBang`**

In `big_bang.go`, delete the `OverrideXDGWithCwd bool` field.

**Step 2: Update `genesis.go`**

Remove the flag registration for `-override-xdg-with-cwd`. Add `-repo_id` flag
to the genesis command's flag definitions. In `OnTheFirstDay`, read `RepoId`
from config instead of `BigBang`:

```go
dir := env_dir.MakeDefaultAndInitialize(
	req, env_dir.XDGUtilityNameDodder, config.Debug, config.RepoId,
)
```

**Step 3: Build and test**

Run: `cd go && go build ./... && just test`
Expected: compiles and all tests pass

**Step 4: Commit**

Message: `refactor: remove -override-xdg-with-cwd flag, replaced by -repo_id .`

---

### Task 6: Add TODOs and update docs

**Files:**
- Modify: `TODO.md` — add init split TODO
- Modify: skills and docs that reference `-override-xdg-with-cwd`

**Step 1: Add TODO for init command split**

Add to the repo disambiguation section of `TODO.md`:

```
- [ ] split `init` into `init` (XDG) and `init-cwd`; remove `-repo_id` from init entirely
```

**Step 2: Update skill references**

Update `.claude/skills/dodder-usage/SKILL.md` and
`.claude/skills/features-madder_blob_stores/SKILL.md` to reference `-repo_id .`
instead of `-override-xdg-with-cwd`. Update
`.claude/skills/dodder-usage/references/commands.md` flag table.

**Step 3: Update FDR status**

Change `docs/features/0003-repo-disambiguation.md` status from `exploring` to
`experimental`.

**Step 4: Commit**

Message: `docs: update references from -override-xdg-with-cwd to -repo_id .`

---

### Task 7: Add new BATS tests for repo_id

**Files:**
- Create: `zz-tests_bats/current_version/repo_id.bats`

**Step 1: Write tests**

```bash
#!/usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output
}

teardown() {
  teardown_repo
}

# bats test_tags=repo_id
@test "repo_id . selects CWD repo" {
  run_dodder_init -repo_id . test-repo-id
  assert_success

  [[ -d ".dodder" ]]
}

# bats test_tags=repo_id
@test "repo_id . errors when no CWD repo exists for non-init commands" {
  run_dodder show -repo_id .
  assert_failure
}

# bats test_tags=repo_id
@test "repo_id / panics with not implemented" {
  run_dodder show -repo_id /
  assert_failure
  assert_output --partial "501"
}

# bats test_tags=repo_id
@test "DODDER_REPO_ID env var selects CWD repo" {
  DODDER_REPO_ID=. run_dodder_init test-repo-id
  assert_success

  [[ -d ".dodder" ]]
}

# bats test_tags=repo_id
@test "repo_id flag overrides DODDER_REPO_ID env var" {
  run_dodder_init -repo_id . test-repo-id
  assert_success

  [[ -d ".dodder" ]]
}
```

**Step 2: Run new tests**

Run: `just test-bats-targets repo_id.bats`
Expected: all pass

**Step 3: Run full suite**

Run: `just test`
Expected: all pass

**Step 4: Commit**

Message: `test: add BATS tests for -repo_id flag and DODDER_REPO_ID env var`
