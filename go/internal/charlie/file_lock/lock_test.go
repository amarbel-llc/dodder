//go:build test

package file_lock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func makeLock(t *ui.T) *Lock {
	t.Helper()
	return New(nil, filepath.Join(t.TempDir(), "lock"), "repo")
}

// TestLockWhileHeldReturnsTypedInProcessError pins the #249 fix's
// detection seam: the in-memory "already locked" branch must be a
// typed sentinel so the MCP layer can classify lock failures instead
// of surfacing a bare stack-frame error.
func TestLockWhileHeldReturnsTypedInProcessError(t1 *testing.T) {
	t := ui.MakeT(t1)
	lock := makeLock(&t)

	if err := lock.Lock(); err != nil {
		t.Fatalf("first Lock: %s", err)
	}

	err := lock.Lock()

	if err == nil {
		t.Fatalf("expected error from second Lock while held")
	}

	if !errors.Is(err, ErrAlreadyLockedInProcess{}) {
		t.Errorf(
			"expected ErrAlreadyLockedInProcess, got %T: %s",
			err,
			err,
		)
	}
}

func TestBreakReleasesHeldLock(t1 *testing.T) {
	t := ui.MakeT(t1)
	lock := makeLock(&t)

	if err := lock.Lock(); err != nil {
		t.Fatalf("Lock: %s", err)
	}

	result, err := lock.Break()
	if err != nil {
		t.Fatalf("Break: %s", err)
	}

	if !result.ReleasedHandle {
		t.Errorf("expected ReleasedHandle for held lock")
	}

	if !result.RemovedFile {
		t.Errorf("expected RemovedFile for held lock")
	}

	if _, err := os.Stat(lock.Path()); !os.IsNotExist(err) {
		t.Errorf("expected lock file to be removed, stat err: %v", err)
	}

	// The lock must be acquirable again after a break.
	if err := lock.Lock(); err != nil {
		t.Errorf("Lock after Break: %s", err)
	}
}

func TestBreakRemovesStaleFile(t1 *testing.T) {
	t := ui.MakeT(t1)
	lock := makeLock(&t)

	// A stale lock file left by another (now dead) process: present on
	// disk, but no in-process handle.
	if err := os.WriteFile(lock.Path(), nil, 0o755); err != nil {
		t.Fatalf("staging stale lock file: %s", err)
	}

	result, err := lock.Break()
	if err != nil {
		t.Fatalf("Break: %s", err)
	}

	if result.ReleasedHandle {
		t.Errorf("expected no ReleasedHandle for stale file")
	}

	if !result.RemovedFile {
		t.Errorf("expected RemovedFile for stale file")
	}

	if err := lock.Lock(); err != nil {
		t.Errorf("Lock after Break: %s", err)
	}
}

func TestBreakNothingHeldIsNoOp(t1 *testing.T) {
	t := ui.MakeT(t1)
	lock := makeLock(&t)

	result, err := lock.Break()
	if err != nil {
		t.Fatalf("Break: %s", err)
	}

	if result.ReleasedHandle || result.RemovedFile {
		t.Errorf(
			"expected no-op break, got %+v",
			result,
		)
	}
}
