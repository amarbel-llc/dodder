//go:build test

package lua_test

import (
	"testing"

	lua "code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestVMPoolSandbox_BlockedGlobals(t1 *testing.T) {
	t := ui.MakeT(t1)

	pool, err := (&lua.VMPoolBuilder{}).WithScript("return {}").Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	vm, repool := pool.GetWithRepool()
	defer repool()

	// dofile/loadfile/load/loadstring must raise errors when called.
	for _, script := range []string{
		"dofile('x')",
		"loadfile('x')",
		"load('return 1')()",
		"loadstring('return 1')()",
	} {
		if err := vm.DoString(script); err == nil {
			t.Errorf("expected %q to fail in sandbox", script)
		}
	}

	// io and os must raise errors when indexed (diagnostic proxy, not nil).
	for _, script := range []string{
		"io.open('/tmp/x', 'r')",
		"os.date('!%Y-%m-%d')",
	} {
		if err := vm.DoString(script); err == nil {
			t.Errorf("expected %q to fail in sandbox", script)
		}
	}

	// Safe standard library globals must be present.
	for _, name := range []string{"string", "table", "math", "print"} {
		if vm.GetGlobal(name) == lua.LNil {
			t.Errorf("expected %s to be available in sandbox", name)
		}
	}
}

func TestVMPoolSandbox_RequireFilesystemBlocked(t1 *testing.T) {
	t := ui.MakeT(t1)

	pool, err := (&lua.VMPoolBuilder{}).WithScript("return {}").Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	vm, repool := pool.GetWithRepool()
	defer repool()

	if err := vm.DoString("require('nonexistent')"); err == nil {
		t.Error("expected require('nonexistent') to fail in sandbox")
	}
}
