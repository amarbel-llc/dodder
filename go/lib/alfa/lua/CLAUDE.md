# lua

Type aliases for the gopher-lua library to simplify imports.

## Re-exported Types

- `LTable`, `LValue`, `LState`, `LFunction`, `LString`, `LBool`
- `FunctionProto`, `LGFunction`

## Constants

- `LTNil`, `LTFunction`, `LTTable`, `LTBool`, `MultRet`, `LNil`

## VM Pool Sandbox

All VMs created by `VMPool` are sandboxed via `lua.Options{SkipOpenLibs: true}`
plus a selective `openSafeLibs` call (`stdlib.go`). The sandbox is applied in
`SetCompiled`'s factory function, before `PrepareVM` runs.

**Allowed stdlib:**

| Library | Notes |
|---------|-------|
| base | `dofile`, `loadfile`, `load`, `loadstring` replaced with hard-error stubs |
| package | preload searcher (`loaders[1]`) kept; filesystem searcher (`loaders[2]`) replaced with a hard-error stub |
| string | fully open |
| table | fully open |
| math | fully open |

**Blocked (never opened):** `io`, `os`, `coroutine`, `channel`, `debug`.

`io` and `os` are set to diagnostic proxy tables whose `__index` **and**
`__newindex` metamethods both raise an actionable error message instead of the
generic "attempt to index nil" Lua produces for absent globals. `__newindex` is
required so a script cannot re-enable a blocked global by assigning through it
(`io.open = fn`), which would otherwise raw-set the key and shadow `__index`.
The `os` proxy names `dodder_today()` as the replacement for `os.date("!%Y-%m-%d")`.

### Repool re-arming

`SetCompiled`'s pool factory calls `openSafeLibs` once per VM, but the pool
(a `sync.Pool`) hands the same `LState` back on the next borrow. The repool
closure therefore re-runs `applySandboxRestrictions` (the overwritable-global
subset of the guards: the `dofile`/`loadfile`/`load`/`loadstring` stubs, the
`io`/`os` proxies, and `dodder_today`) so a script that clobbered one of them
cannot leak that mutation into the next script sharing the VM slot. The
filesystem-searcher block lives in `openSafeLibs`, **not**
`applySandboxRestrictions`: `PrepareVM` inserts the custom searcher at
`loaders[1]` after construction, so re-running `RawSetInt(2)` on repool would
clobber the preload searcher — and a script can't reconstruct filesystem
loading anyway without the blocked `io`/`os`.

**Go-side date globals:**

- `dodder_today()` — returns current UTC date as `YYYY-MM-DD`; replaces
  `os.date("!%Y-%m-%d")`. Registered by the lua package itself
  (`applySandboxRestrictions`, `stdlib.go`), so it is present in **every**
  sandboxed VM and restored on repool.
- `dodder_advance_date(date, duration)` — advances a `YYYY-MM-DD` date by an
  ISO-8601 duration. Needs the dodder `iso_duration` package, so it is
  registered per-VM by `registerDateHelpers` in
  `go/internal/hotel/tag_blobs/lua_v1.go` (called from the V1 and V2 tag-filter
  apply hooks).
