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
