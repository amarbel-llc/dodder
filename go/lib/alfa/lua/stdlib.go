package lua

import (
	"time"

	lua "github.com/yuin/gopher-lua"
)

// openSafeLibs opens the sandboxed subset of gopher-lua's standard library and
// installs the sandbox guards. Allowlist: base (minus dofile, loadfile, load,
// loadstring), package (preload searcher only — filesystem searcher replaced
// with a hard error), string, table, math. Not opened: io, os, coroutine,
// channel, debug. Blocked globals get diagnostic stubs that raise actionable
// errors naming the available alternatives (e.g. dodder_today() for os.date).
//
// Called once per VM at construction, before PrepareVM inserts the custom
// der/dodder/zit module searcher.
func openSafeLibs(ls *lua.LState) {
	for _, pair := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{"", lua.OpenBase},
		{"package", lua.OpenPackage},
		{"string", lua.OpenString},
		{"table", lua.OpenTable},
		{"math", lua.OpenMath},
	} {
		ls.Push(ls.NewFunction(pair.fn))
		ls.Push(lua.LString(pair.name))
		ls.Call(1, 0)
	}

	applySandboxRestrictions(ls)

	// Replace the filesystem searcher (package.loaders[2], installed by
	// OpenPackage) with a hard error so require() for anything not preloaded
	// fails loudly instead of touching the filesystem. The preload searcher at
	// loaders[1] is kept intact so PreloadModule and the der/dodder/zit module
	// aliases still resolve.
	//
	// This is deliberately NOT part of applySandboxRestrictions (which the VM
	// pool re-runs on every repool): PrepareVM later does loaders.Insert(1,
	// customSearcher), shifting every entry up one index, so on a reused VM the
	// filesystem searcher is no longer at index 2 and re-running RawSetInt(2)
	// would clobber the preload searcher instead. Re-application is also
	// unnecessary — a script cannot reconstruct filesystem loading without
	// io/os, which stay blocked.
	packageTable := ls.GetGlobal("package").(*lua.LTable)
	loaderTable := ls.GetField(packageTable, "loaders").(*lua.LTable)
	loaderTable.RawSetInt(2, ls.NewFunction(func(ls *lua.LState) int {
		ls.RaiseError("require: filesystem access is not available in dodder Lua scripts")
		return 0
	}))
}

// applySandboxRestrictions (re-)installs the sandbox guards that live in the
// global environment and that a running script could therefore overwrite: the
// hard-error stubs for the filesystem-escaping base functions, the diagnostic
// proxies for the never-opened io/os/coroutine/channel/debug globals, and the
// dodder_today builtin. It is idempotent and order-independent, so the VM pool
// re-runs it on every repool: a script that reassigned dofile, clobbered a
// blocked-global proxy, or shadowed dodder_today cannot leak that mutation into
// the next borrow of the same pooled VM. It deliberately does not touch
// package.loaders — see openSafeLibs.
func applySandboxRestrictions(ls *lua.LState) {
	// dofile/loadfile/load/loadstring are opened by OpenBase but give arbitrary
	// code-execution paths. Replace them with stubs that raise a clear error.
	for _, name := range []string{"dofile", "loadfile", "load", "loadstring"} {
		ls.SetGlobal(name, ls.NewFunction(func(ls *lua.LState) int {
			ls.RaiseError("%s is not available in dodder Lua scripts", name)
			return 0
		}))
	}

	// Install diagnostic proxy tables for the never-opened globals so that
	// scripts indexing into or assigning through them (os.date, io.open = ...)
	// receive an actionable error message rather than the generic "attempt to
	// index a nil value" Lua produces for nil globals. os carries an extra hint
	// toward its supported replacement; io/coroutine/channel/debug share a
	// uniform message (dodder#391 extended these past just io/os).
	setBlockedGlobalProxy(ls, "os",
		"os is not available in dodder Lua scripts; use dodder_today() for the current date")
	for _, name := range []string{"io", "coroutine", "channel", "debug"} {
		setBlockedGlobalProxy(ls, name,
			name+" is not available in dodder Lua scripts")
	}

	// dodder_today() is the sandbox's supported replacement for
	// os.date("!%Y-%m-%d") — the os proxy message above names it. It needs only
	// the time stdlib, so it lives here in the lua package: that guarantees it
	// in every sandboxed VM (not just the ones whose apply hook happens to
	// register it) and restores it on repool. dodder_advance_date (ISO-8601
	// duration math) needs a dodder package and is registered per-VM by
	// tag_blobs.registerDateHelpers.
	ls.SetGlobal("dodder_today", ls.NewFunction(luaTodayDate))
}

func luaTodayDate(ls *lua.LState) int {
	ls.Push(lua.LString(time.Now().UTC().Format("2006-01-02")))
	return 1
}

// setBlockedGlobalProxy installs a table whose __index and __newindex
// metamethods both raise message, so both reading (foo.bar) and writing
// (foo.bar = x) through the blocked global produce the actionable message.
// Without __newindex a script could `io.open = fn`, which raw-sets the key on
// the proxy table and shadows __index for every later read of io.open.
func setBlockedGlobalProxy(ls *lua.LState, name, message string) {
	proxy := ls.NewTable()
	mt := ls.NewTable()
	raise := ls.NewFunction(func(ls *lua.LState) int {
		ls.RaiseError("%s", message)
		return 0
	})
	ls.SetField(mt, "__index", raise)
	ls.SetField(mt, "__newindex", raise)
	ls.SetMetatable(proxy, mt)
	ls.SetGlobal(name, proxy)
}
