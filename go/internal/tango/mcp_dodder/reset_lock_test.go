//go:build test

package mcp_dodder

import (
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/charlie/file_lock"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func assertContainsAll(
	t *ui.T,
	context string,
	text string,
	expected []string,
) {
	t.Helper()

	for _, substring := range expected {
		if !strings.Contains(text, substring) {
			t.Errorf("%s = %q; missing %q", context, text, substring)
		}
	}
}

// TestResetLockResponseTextIsUnambiguous pins #249's response
// contract: every outcome states the resulting lock state in plain
// words, so a caller never has to guess whether the repo is still
// locked.
func TestResetLockResponseTextIsUnambiguous(t1 *testing.T) {
	t := ui.MakeT(t1)

	const path = "/state/dodder/lock"

	cases := []struct {
		result   file_lock.BreakResult
		expected []string
	}{
		{
			file_lock.BreakResult{ReleasedHandle: true, RemovedFile: true},
			[]string{"no longer locked", "in-process lock handle", path},
		},
		{
			file_lock.BreakResult{ReleasedHandle: false, RemovedFile: true},
			[]string{"no longer locked", "stale lock file", path},
		},
		{
			file_lock.BreakResult{ReleasedHandle: true, RemovedFile: false},
			[]string{"no longer locked", "in-process lock handle", "no lock file"},
		},
		{
			file_lock.BreakResult{},
			[]string{"was not locked", path},
		},
	}

	for _, testCase := range cases {
		assertContainsAll(
			&t,
			"resetLockResponseText",
			resetLockResponseText(testCase.result, path),
			testCase.expected,
		)
	}
}

// TestFormatToolErrorClassifiesLockErrors pins #249's other response
// contract: a mutating tool failing on the env lock must say so
// unambiguously — which lock, where, who holds it, and that reset-lock
// (user-approved) is the recovery path — instead of surfacing a bare
// stack tree.
func TestFormatToolErrorClassifiesLockErrors(t1 *testing.T) {
	t := ui.MakeT(t1)

	const path = "/state/dodder/lock"

	assertContainsAll(
		&t,
		"formatToolError(in-process)",
		formatToolError(
			errors.Wrap(file_lock.ErrAlreadyLockedInProcess{Path: path}),
		),
		[]string{"REPO LOCKED", "this MCP server process", path, "reset-lock"},
	)

	assertContainsAll(
		&t,
		"formatToolError(on-disk)",
		formatToolError(
			errors.Wrap(file_lock.ErrUnableToAcquireLock{Path: path}),
		),
		[]string{"REPO LOCKED", path, "reset-lock", "another process"},
	)

	// The shape the bridge actually sees after a declined no-TTY
	// recovery (Recover aborts with the non-retryable carrier).
	assertContainsAll(
		&t,
		"formatToolError(recovery-aborted)",
		formatToolError(
			errors.Wrap(file_lock.ErrLockRecoveryAborted{Path: path}),
		),
		[]string{"REPO LOCKED", path, "reset-lock"},
	)

	plain := errors.Errorf("some other failure")
	text := formatToolError(plain)

	if strings.Contains(text, "REPO LOCKED") {
		t.Errorf(
			"formatToolError(non-lock) = %q; must not claim a lock failure",
			text,
		)
	}
}
