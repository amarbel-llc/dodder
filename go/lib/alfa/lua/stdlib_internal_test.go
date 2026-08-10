//go:build test

package lua

import (
	"testing"

	glua "github.com/yuin/gopher-lua"
)

// TestApplySandboxRestrictions_ResetsOverwrittenGlobals proves the guards that
// the VM pool re-runs on every repool actually restore the sandbox after a
// script has overwritten a blocked global — the cross-borrow isolation property
// dodder#389 depends on, exercised directly (no sync.Pool nondeterminism).
func TestApplySandboxRestrictions_ResetsOverwrittenGlobals(t *testing.T) {
	ls := glua.NewState(glua.Options{SkipOpenLibs: true})
	defer ls.Close()
	openSafeLibs(ls)

	// Baseline: the dofile stub is blocked.
	if err := ls.DoString("dofile('x')"); err == nil {
		t.Fatal("expected dofile to be blocked after openSafeLibs")
	}

	// A script overwrites the stub with a no-op, defeating the block for the
	// rest of this VM's life.
	if err := ls.DoString("dofile = function() end"); err != nil {
		t.Fatalf("overwrite dofile: %v", err)
	}
	if err := ls.DoString("dofile('x')"); err != nil {
		t.Fatal("expected overwritten dofile to succeed (pollution not established)")
	}

	// Repool re-arms the sandbox.
	applySandboxRestrictions(ls)

	if err := ls.DoString("dofile('x')"); err == nil {
		t.Error("expected dofile to be blocked again after applySandboxRestrictions")
	}
}

// TestSandbox_DodderTodayAvailable proves the lua package installs dodder_today
// itself (the os.date replacement the os proxy advertises), independent of any
// per-VM apply hook, and that it survives a repool.
func TestSandbox_DodderTodayAvailable(t *testing.T) {
	ls := glua.NewState(glua.Options{SkipOpenLibs: true})
	defer ls.Close()
	openSafeLibs(ls)

	const check = `
		local d = dodder_today()
		assert(type(d) == "string", "dodder_today must return a string")
		assert(#d == 10, "dodder_today must return YYYY-MM-DD")
	`

	if err := ls.DoString(check); err != nil {
		t.Errorf("dodder_today() after openSafeLibs: %v", err)
	}

	applySandboxRestrictions(ls)

	if err := ls.DoString(check); err != nil {
		t.Errorf("dodder_today() after applySandboxRestrictions: %v", err)
	}
}

// TestSandbox_BlockedGlobalRejectsWrite proves the __newindex guard: a script
// cannot re-enable a blocked global by assigning through it (io.open = fn),
// which without __newindex would raw-set the key and shadow __index for every
// later read.
func TestSandbox_BlockedGlobalRejectsWrite(t *testing.T) {
	ls := glua.NewState(glua.Options{SkipOpenLibs: true})
	defer ls.Close()
	openSafeLibs(ls)

	if err := ls.DoString("io.open = function() return 'escaped' end"); err == nil {
		t.Error("expected assigning io.open to raise (blocked-global __newindex)")
	}
}
