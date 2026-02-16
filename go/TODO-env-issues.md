# Environment Issues (worktree devshell)

Discovered during pool_cleanup branch work. Both are pre-existing and
unrelated to pool changes.

## 1. `stringer` not in PATH

`just build` fails at `go generate ./...` because `stringer` is missing:

```
src/alfa/cmp/result.go:18: running "stringer": exec: "stringer": executable file not found in $PATH
```

12 files across the codebase use `//go:generate stringer`. Workaround:
skip `go generate` and build binaries directly with
`just build-go-binary debug build/debug`.

**Fix:** add `stringer` (from `golang.org/x/tools/cmd/stringer`) to the
devenv flake or the go devshell.

## 2. `bats-support` library not found

`just test-bats` fails during fixture generation:

```
Could not find library 'bats-support' relative to test file or in BATS_LIB_PATH
```

Also `chflags_and_rm` is not on PATH.

The bats devenv at `devenvs/bats/` likely needs to be loaded for this
worktree, or `BATS_LIB_PATH` needs to include the nix store path for
`bats-support`.

**Fix:** ensure the worktree `.envrc` loads the bats devenv, or add
`bats-support`, `bats-assert`, and project utility scripts to the go
devshell.
