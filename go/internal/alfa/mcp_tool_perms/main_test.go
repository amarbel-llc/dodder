//go:build test

package mcp_tool_perms

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// TestClassification locks the permission of every MCP tool. The MCP
// server (tango/mcp_dodder) and the clown plugin hook (bravo/
// claude_hooks) both derive their behavior from this map, so an
// accidental edit here changes tool annotations and auto-approval at
// once (#251). The mcp_tools_list bats tests assert which tools exist;
// this asserts how each is classified.
func TestClassification(t1 *testing.T) {
	t := ui.MakeT(t1)

	want := map[string]Permission{
		"show":             PermissionReadOnly,
		"query":            PermissionReadOnly,
		"query-tag":        PermissionReadOnly,
		"query-type":       PermissionReadOnly,
		"format-blob":      PermissionReadOnly,
		"status":           PermissionReadOnly,
		"diff":             PermissionReadOnly,
		"read-checked_out": PermissionReadOnly,
		"organize_plan":    PermissionReadOnly,
		"new":              PermissionWrite,
		"edit":             PermissionWrite,
		"checkin":          PermissionWrite,
		"organize_commit":  PermissionWrite,
		"import":           PermissionWrite,
		"push":             PermissionWrite,
		"pull":             PermissionWrite,
		"reset-lock":       PermissionDestructive,
	}

	for name, permission := range want {
		if got := Of(name); got != permission {
			t.Errorf("Of(%q) = %v, want %v", name, got, permission)
		}
	}

	if len(ByName) != len(want) {
		t.Errorf(
			"ByName has %d entries, want %d — a tool was added or removed without updating this test",
			len(ByName),
			len(want),
		)
	}

	if Of("no-such-tool") != PermissionUnknown {
		t.Errorf("an unclassified name must be PermissionUnknown (fail-safe)")
	}
}

// TestWriteFlowClassification locks the data-flow refinement the
// PreToolUse hook reads for context-aware auto-approval, and pins it to
// the capability map: every PermissionWrite tool must carry a flow, and
// nothing else may. A new write tool added without a flow (or a
// reclassification that leaves a stale flow) fails here rather than
// silently changing auto-approval.
func TestWriteFlowClassification(t1 *testing.T) {
	t := ui.MakeT(t1)

	want := map[string]WriteFlow{
		"new":             WriteFlowScoped,
		"edit":            WriteFlowScoped,
		"organize_commit": WriteFlowScoped,
		"checkin":         WriteFlowScoped,
		"pull":            WriteFlowScoped,
		"import":          WriteFlowUnconditional,
		"push":            WriteFlowGated,
	}

	for name, flow := range want {
		if got := WriteFlowOf(name); got != flow {
			t.Errorf("WriteFlowOf(%q) = %v, want %v", name, got, flow)
		}
	}

	// The flow map and the capability map must agree on which tools are
	// writes: exactly the PermissionWrite tools carry a flow.
	for name, permission := range ByName {
		_, hasFlow := WriteFlowByName[name]

		if permission == PermissionWrite && !hasFlow {
			t.Errorf("write tool %q has no WriteFlow entry", name)
		}

		if permission != PermissionWrite && hasFlow {
			t.Errorf("non-write tool %q must not have a WriteFlow entry", name)
		}
	}

	if len(WriteFlowByName) != len(want) {
		t.Errorf(
			"WriteFlowByName has %d entries, want %d — a write tool was added or removed without updating this test",
			len(WriteFlowByName),
			len(want),
		)
	}

	if WriteFlowOf("show") != WriteFlowNone {
		t.Errorf("a read-only tool must be WriteFlowNone")
	}

	if WriteFlowOf("reset-lock") != WriteFlowNone {
		t.Errorf("a destructive tool must be WriteFlowNone")
	}

	if WriteFlowOf("no-such-tool") != WriteFlowNone {
		t.Errorf("an unclassified name must be WriteFlowNone")
	}
}
