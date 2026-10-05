//go:build test

package lua

import (
	"testing"
	"time"

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

// TestTodayDate_UsesLocalCalendarDate pins dodder#409: an evening instant west
// of UTC is still "today" locally even though the UTC date has already rolled
// over, and a morning instant east of UTC is already "tomorrow" relative to UTC.
func TestTodayDate_UsesLocalCalendarDate(t *testing.T) {
	// 2026-10-04 20:06 at UTC-04:00, the instant from the issue report.
	instant := time.Date(2026, time.October, 5, 0, 6, 0, 0, time.UTC)

	west := time.FixedZone("west", -4*60*60)
	if got := todayDate(instant, west); got != "2026-10-04" {
		t.Errorf("todayDate west of UTC: got %q, want %q", got, "2026-10-04")
	}

	if got := todayDate(instant, time.UTC); got != "2026-10-05" {
		t.Errorf("todayDate in UTC: got %q, want %q", got, "2026-10-05")
	}

	// 2026-10-04 23:30 UTC is already 2026-10-05 08:30 at UTC+09:00.
	lateUTC := time.Date(2026, time.October, 4, 23, 30, 0, 0, time.UTC)
	east := time.FixedZone("east", 9*60*60)
	if got := todayDate(lateUTC, east); got != "2026-10-05" {
		t.Errorf("todayDate east of UTC: got %q, want %q", got, "2026-10-05")
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
