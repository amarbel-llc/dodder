package lua

import (
	lua "github.com/yuin/gopher-lua"
)

// openSafeLibs opens the sandboxed subset of gopher-lua's standard library.
// Allowlist: base (minus dofile, loadfile, load, loadstring), package (preload
// searcher only — filesystem searcher replaced with a hard error), string, table, math.
// Not opened: io, os, coroutine, channel, debug.
// Blocked globals get diagnostic stubs that raise actionable errors naming the
// available alternatives (e.g. dodder_today() for os.date).
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

	// dofile/loadfile/load/loadstring are opened by OpenBase but give arbitrary
	// code-execution paths. Replace them with stubs that raise a clear error.
	for _, name := range []string{"dofile", "loadfile", "load", "loadstring"} {
		name := name
		ls.SetGlobal(name, ls.NewFunction(func(ls *lua.LState) int {
			ls.RaiseError("%s is not available in dodder Lua scripts", name)
			return 0
		}))
	}

	// Install diagnostic proxy tables for io and os so that scripts indexing
	// into them (e.g. os.date, io.open) receive an actionable error message
	// rather than the generic "attempt to index a nil value" Lua produces for
	// nil globals.
	setBlockedGlobalProxy(ls, "os",
		"os is not available in dodder Lua scripts; use dodder_today() for the current date")
	setBlockedGlobalProxy(ls, "io",
		"io is not available in dodder Lua scripts")

	// Replace the filesystem searcher (loaders[2]) with a hard error so that
	// require("anything") that isn't pre-loaded fails loudly instead of trying
	// the filesystem. The preload searcher at loaders[1] is kept intact so
	// PreloadModule and the der/dodder/zit custom module resolver still work.
	packageTable := ls.GetGlobal("package").(*lua.LTable)
	loaderTable := ls.GetField(packageTable, "loaders").(*lua.LTable)
	loaderTable.RawSetInt(2, ls.NewFunction(func(ls *lua.LState) int {
		ls.RaiseError("require: filesystem access is not available in dodder Lua scripts")
		return 0
	}))
}

// setBlockedGlobalProxy installs a table with an __index metamethod that raises
// message, so foo.bar on the blocked global produces the message instead of a
// generic nil-index panic.
func setBlockedGlobalProxy(ls *lua.LState, name, message string) {
	proxy := ls.NewTable()
	mt := ls.NewTable()
	ls.SetField(mt, "__index", ls.NewFunction(func(ls *lua.LState) int {
		ls.RaiseError("%s", message)
		return 0
	}))
	ls.SetMetatable(proxy, mt)
	ls.SetGlobal(name, proxy)
}
