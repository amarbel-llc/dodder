//go:build test

package claude_hooks

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func makeInput(t *ui.T, eventName, toolName string) []byte {
	t.Helper()

	input := map[string]any{
		"hook_event_name": eventName,
		"tool_name":       toolName,
		"tool_input":      map[string]any{},
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

func TestMutatingToolsFallThrough(t1 *testing.T) {
	t := ui.MakeT(t1)

	mutatingTools := []string{
		"mcp__plugin_dodder_dodder__new",
		"mcp__plugin_dodder_dodder__edit",
		"mcp__plugin_dodder_dodder__checkin",
	}

	for _, toolName := range mutatingTools {
		var stdout bytes.Buffer

		if err := Run(
			bytes.NewReader(makeInput(&t, "PreToolUse", toolName)),
			&stdout,
		); err != nil {
			t.Fatalf("unexpected error for %s: %s", toolName, err)
		}

		if stdout.Len() != 0 {
			t.Errorf(
				"expected no output for mutating tool %s, got %q",
				toolName,
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
