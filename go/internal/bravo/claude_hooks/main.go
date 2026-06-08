// Package claude_hooks implements the Claude Code hook protocol for the
// dodder clown plugin (#244). The plugin's hooks/hooks.json registers a
// PreToolUse hook scoped to dodder's own MCP tools (matcher
// "mcp__plugin_dodder_dodder__.*" — the hook handler spawns a process
// per event, so non-dodder tools are filtered out client-side); the
// handler script execs `dodder hook`, which routes stdin/stdout through
// Run. Keeping the decision table in Go (instead of a hooks.json
// matcher regex) follows spinclass's internal/hooks: the decision is
// unit-testable and has room to grow (deny rules, workspace-aware
// decisions) without touching the shipped plugin payload shape.
package claude_hooks

import (
	"encoding/json"
	"fmt"
	"io"
)

// hookInput carries the subset of Claude Code's hook-event payload the
// decision table consumes; unused protocol fields (session_id,
// tool_input, cwd) are deliberately not decoded.
type hookInput struct {
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
}

// Dodder ships as a Claude Code plugin named "dodder" whose clown.json
// registers an MCP server also named "dodder", so its tools appear to
// Claude Code as mcp__plugin_dodder_dodder__<tool>. The set below is
// the read-only subset of the MCP server's tools (see
// internal/tango/mcp_dodder/server.go): none of them can mutate the
// store or the workspace. The mutating tools (new, edit, checkin) are
// deliberately absent so they keep normal permission gating.
const toolNamePrefix = "mcp__plugin_dodder_dodder__"

var readOnlyTools = map[string]bool{
	toolNamePrefix + "show":             true,
	toolNamePrefix + "query":            true,
	toolNamePrefix + "query-tag":        true,
	toolNamePrefix + "query-type":       true,
	toolNamePrefix + "format-blob":      true,
	toolNamePrefix + "status":           true,
	toolNamePrefix + "diff":             true,
	toolNamePrefix + "read-checked_out": true,
}

// alwaysAskTools force a user prompt on every call, even when the user
// has otherwise allowlisted dodder's MCP tools. reset-lock forcibly
// breaks the repo's env lock — a destructive recovery action that must
// never run on standing approval (#249).
var alwaysAskTools = map[string]string{
	toolNamePrefix + "reset-lock": "reset-lock forcibly breaks the repo's environment lock; every invocation requires explicit user approval",
}

// Run decodes one Claude Code hook event from reader and writes a
// permission decision to writer when one applies. No output means no
// opinion: Claude Code falls through to its normal permission flow.
func Run(reader io.Reader, writer io.Writer) error {
	var input hookInput

	if err := json.NewDecoder(reader).Decode(&input); err != nil {
		return fmt.Errorf("decoding hook input: %w", err)
	}

	if input.HookEventName != "PreToolUse" {
		return nil
	}

	if reason, ok := alwaysAskTools[input.ToolName]; ok {
		return writeDecision(writer, "ask", reason)
	}

	if !readOnlyTools[input.ToolName] {
		return nil
	}

	return writeDecision(
		writer,
		"allow",
		"read-only dodder MCP tool, cannot mutate the store",
	)
}

func writeDecision(writer io.Writer, decision, reason string) error {
	output := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       decision,
			"permissionDecisionReason": reason,
		},
	}

	return json.NewEncoder(writer).Encode(output)
}
