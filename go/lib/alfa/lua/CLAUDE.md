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

`io` and `os` are set to diagnostic proxy tables whose `__index` metamethod
raises an actionable error message instead of the generic "attempt to index nil"
Lua produces for absent globals. The `os` proxy names `dodder_today()` as the
replacement for `os.date("!%Y-%m-%d")`.

**Go-side globals available in hook VMs** (registered by `registerDateHelpers`
in `go/internal/hotel/tag_blobs/lua_v1.go`):

- `dodder_today()` — returns current UTC date as `YYYY-MM-DD`; replaces `os.date("!%Y-%m-%d")`
- `dodder_advance_date(date, duration)` — advances a `YYYY-MM-DD` date by an ISO-8601 duration
