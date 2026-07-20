//go:build test

package claude_hooks

import (
	"bytes"
	"encoding/json"
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func makeInput(t *ui.T, eventName, toolName string) []byte {
	t.Helper()

	return makeInputWithToolInput(t, eventName, toolName, map[string]any{})
}

func makeInputWithToolInput(
	t *ui.T,
	eventName, toolName string,
	toolInput map[string]any,
) []byte {
	t.Helper()

	input := map[string]any{
		"hook_event_name": eventName,
		"tool_name":       toolName,
		"tool_input":      toolInput,
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshaling hook input: %s", err)
	}

	return data
}

func parseDecision(t *ui.T, output []byte) (decision, reason string) {
	t.Helper()

	var result map[string]any

	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("expected valid JSON, got %q: %s", string(output), err)
	}

	hookSpecificOutput, ok := result["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("expected hookSpecificOutput in output, got %q", string(output))
	}

	if hookSpecificOutput["hookEventName"] != "PreToolUse" {
		t.Errorf(
			"expected hookEventName PreToolUse, got %v",
			hookSpecificOutput["hookEventName"],
		)
	}

	decision, _ = hookSpecificOutput["permissionDecision"].(string)
	reason, _ = hookSpecificOutput["permissionDecisionReason"].(string)

	return decision, reason
}

// TestResetLockAlwaysAsks pins #249's permission posture: reset-lock
// forcibly breaks the repo's env lock, so the hook must force a user
// prompt on EVERY call — even when the user has otherwise allowlisted
// dodder tools — rather than falling through to the default flow.
func TestResetLockAlwaysAsks(t1 *testing.T) {
	t := ui.MakeT(t1)

	var stdout bytes.Buffer

	if err := Run(
		bytes.NewReader(
			makeInput(&t, "PreToolUse", "mcp__plugin_dodder_dodder__reset-lock"),
		),
		&stdout,
	); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if stdout.Len() == 0 {
		t.Fatalf("expected ask output for reset-lock, got none")
	}

	decision, reason := parseDecision(&t, stdout.Bytes())

	if decision != "ask" {
		t.Errorf("expected permissionDecision ask, got %q", decision)
	}

	if reason == "" {
		t.Errorf("expected a permissionDecisionReason")
	}
}

func TestReadOnlyToolsAutoApproved(t1 *testing.T) {
	t := ui.MakeT(t1)

	readOnlyTools := []string{
		"mcp__plugin_dodder_dodder__show",
		"mcp__plugin_dodder_dodder__query",
		"mcp__plugin_dodder_dodder__query-tag",
		"mcp__plugin_dodder_dodder__query-type",
		"mcp__plugin_dodder_dodder__format-blob",
		"mcp__plugin_dodder_dodder__status",
		"mcp__plugin_dodder_dodder__diff",
		"mcp__plugin_dodder_dodder__read-checked_out",
	}

	for _, toolName := range readOnlyTools {
		var stdout bytes.Buffer

		if err := Run(
			bytes.NewReader(makeInput(&t, "PreToolUse", toolName)),
			&stdout,
		); err != nil {
			t.Fatalf("unexpected error for %s: %s", toolName, err)
		}

		if stdout.Len() == 0 {
			t.Fatalf("expected allow output for %s, got none", toolName)
		}

		decision, reason := parseDecision(&t, stdout.Bytes())

		if decision != "allow" {
			t.Errorf(
				"expected permissionDecision allow for %s, got %q",
				toolName,
				decision,
			)
		}

		if reason == "" {
			t.Errorf("expected a permissionDecisionReason for %s", toolName)
		}
	}
}

// scopedWriteTools mutate the repo the call addresses; the hook
// auto-approves them only when that repo is local (the server default,
// or a cwd-scoped .name).
var scopedWriteTools = []string{
	"mcp__plugin_dodder_dodder__new",
	"mcp__plugin_dodder_dodder__edit",
	"mcp__plugin_dodder_dodder__organize_commit",
	"mcp__plugin_dodder_dodder__checkin",
	"mcp__plugin_dodder_dodder__pull",
}

// TestScopedWritesAutoApprovedWhenLocal: a scoped write with no repo_id
// (the server's default repo) or a cwd-scoped repo_id (.name) is
// auto-approved. checkin has no repo_id param at all and so is always
// local.
func TestScopedWritesAutoApprovedWhenLocal(t1 *testing.T) {
	t := ui.MakeT(t1)

	localInputs := []map[string]any{
		{},                      // no repo_id → server default
		{"repo_id": ".default"}, // cwd-scoped default
		{"repo_id": ".work"},    // cwd-scoped named repo
		{"repo_id": "..backup"}, // cwd ancestor (still cwd-scoped)
	}

	for _, toolName := range scopedWriteTools {
		for _, toolInput := range localInputs {
			var stdout bytes.Buffer

			if err := Run(
				bytes.NewReader(makeInputWithToolInput(
					&t, "PreToolUse", toolName, toolInput,
				)),
				&stdout,
			); err != nil {
				t.Fatalf("unexpected error for %s %v: %s", toolName, toolInput, err)
			}

			if stdout.Len() == 0 {
				t.Fatalf(
					"expected allow output for %s with %v, got none",
					toolName,
					toolInput,
				)
			}

			decision, reason := parseDecision(&t, stdout.Bytes())

			if decision != "allow" {
				t.Errorf(
					"expected allow for %s with %v, got %q",
					toolName,
					toolInput,
					decision,
				)
			}

			if reason == "" {
				t.Errorf("expected a reason for %s with %v", toolName, toolInput)
			}
		}
	}
}

// TestScopedWritesFallThroughWhenCrossRepo: a scoped write that names a
// different XDG-user or system repo, or a repo_id that does not parse,
// gets no opinion so normal gating applies.
func TestScopedWritesFallThroughWhenCrossRepo(t1 *testing.T) {
	t := ui.MakeT(t1)

	crossRepoInputs := []map[string]any{
		{"repo_id": "work"},           // XDG-user repo (bare name)
		{"repo_id": "//config"},       // forced system repo
		{"repo_id": "not a valid id"}, // unparseable → fail-safe
	}

	for _, toolName := range scopedWriteTools {
		for _, toolInput := range crossRepoInputs {
			var stdout bytes.Buffer

			if err := Run(
				bytes.NewReader(makeInputWithToolInput(
					&t, "PreToolUse", toolName, toolInput,
				)),
				&stdout,
			); err != nil {
				t.Fatalf("unexpected error for %s %v: %s", toolName, toolInput, err)
			}

			if stdout.Len() != 0 {
				t.Errorf(
					"expected no output (fall through) for %s with %v, got %q",
					toolName,
					toolInput,
					stdout.String(),
				)
			}
		}
	}
}

// TestImportUnconditionallyAllowed: import auto-approves regardless of
// target repo — it brings user-named inventory paths in.
func TestImportUnconditionallyAllowed(t1 *testing.T) {
	t := ui.MakeT(t1)

	for _, toolInput := range []map[string]any{
		{"paths": []string{"/some/inventory"}},
		{"paths": []string{"/some/inventory"}, "repo_id": "work"},
	} {
		var stdout bytes.Buffer

		if err := Run(
			bytes.NewReader(makeInputWithToolInput(
				&t, "PreToolUse", "mcp__plugin_dodder_dodder__import", toolInput,
			)),
			&stdout,
		); err != nil {
			t.Fatalf("unexpected error for import %v: %s", toolInput, err)
		}

		decision, _ := parseDecision(&t, stdout.Bytes())

		if decision != "allow" {
			t.Errorf("expected allow for import %v, got %q", toolInput, decision)
		}
	}
}

// TestPushFallsThrough: push always falls through to normal gating — it
// sends objects OUT to another repo — even when scoped locally.
func TestPushFallsThrough(t1 *testing.T) {
	t := ui.MakeT(t1)

	for _, toolInput := range []map[string]any{
		{},
		{"repo_id": ".default"},
		{"direct": "/some/local/repo"},
	} {
		var stdout bytes.Buffer

		if err := Run(
			bytes.NewReader(makeInputWithToolInput(
				&t, "PreToolUse", "mcp__plugin_dodder_dodder__push", toolInput,
			)),
			&stdout,
		); err != nil {
			t.Fatalf("unexpected error for push %v: %s", toolInput, err)
		}

		if stdout.Len() != 0 {
			t.Errorf(
				"expected no output (fall through) for push %v, got %q",
				toolInput,
				stdout.String(),
			)
		}
	}
}

func TestUnrelatedToolsFallThrough(t1 *testing.T) {
	t := ui.MakeT(t1)

	for _, toolName := range []string{
		"Bash",
		"Read",
		// substring traps: must not match the read-only set
		"mcp__plugin_dodder_dodder__show-and-mutate",
		"mcp__dodder__show",
	} {
		var stdout bytes.Buffer

		if err := Run(
			bytes.NewReader(makeInput(&t, "PreToolUse", toolName)),
			&stdout,
		); err != nil {
			t.Fatalf("unexpected error for %s: %s", toolName, err)
		}

		if stdout.Len() != 0 {
			t.Errorf(
				"expected no output for unrelated tool %s, got %q",
				toolName,
				stdout.String(),
			)
		}
	}
}

func TestNonPreToolUseEventFallsThrough(t1 *testing.T) {
	t := ui.MakeT(t1)

	var stdout bytes.Buffer

	if err := Run(
		bytes.NewReader(
			makeInput(&t, "PostToolUse", "mcp__plugin_dodder_dodder__show"),
		),
		&stdout,
	); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if stdout.Len() != 0 {
		t.Errorf(
			"expected no output for PostToolUse event, got %q",
			stdout.String(),
		)
	}
}

func TestMalformedInputErrors(t1 *testing.T) {
	t := ui.MakeT(t1)

	var stdout bytes.Buffer

	if err := Run(
		bytes.NewReader([]byte("not json")),
		&stdout,
	); err == nil {
		t.Errorf("expected error for malformed input, got none")
	}
}
