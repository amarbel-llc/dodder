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

// TestBuildSingleVM_ExecutesChunkExactlyOnce proves the single-run execution
// semantics BuildSingleVM provides (dodder#390): unlike the pooled Build path,
// which can re-run the chunk when sync.Pool evicts the trial VM before the
// first borrow, BuildSingleVM compiles once and runs the chunk exactly once on
// one owned VM. A chunk that increments a Lua global therefore observes exactly
// one execution — the property the transform command relies on so blobs.write
// side effects cannot fire twice.
func TestBuildSingleVM_ExecutesChunkExactlyOnce(t1 *testing.T) {
	t := ui.MakeT(t1)

	vm, err := (&lua.VMPoolBuilder{}).
		WithScript("_run_count = (_run_count or 0) + 1\nreturn {}").
		BuildSingleVM()
	if err != nil {
		t.Fatalf("BuildSingleVM: %v", err)
	}

	defer vm.LState.Close()

	if err := vm.DoString(
		"assert(_run_count == 1, 'chunk ran ' .. tostring(_run_count) .. ' times, want 1')",
	); err != nil {
		t.Errorf("single-run assertion failed: %v", err)
	}

	if _, err := vm.GetTopTableOrError(); err != nil {
		t.Errorf("expected the chunk's table return in vm.Top: %v", err)
	}
}
