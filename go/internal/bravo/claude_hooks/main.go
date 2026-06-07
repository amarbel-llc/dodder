// Package claude_hooks implements the Claude Code hook protocol for the
// dodder clown plugin (#244). The plugin's hooks/hooks.json registers a
// PreToolUse hook on every tool ("matcher": ".*"); the handler script
// execs `dodder hook`, which routes stdin/stdout through Run. Keeping
// the decision table in Go (instead of a hooks.json matcher regex)
// follows spinclass's internal/hooks: the decision is unit-testable and
// has room to grow (deny rules, workspace-aware decisions) without
// touching the shipped plugin payload shape.
package claude_hooks

import (
	"encoding/json"
	"fmt"
	"io"
)

type hookInput struct {
	HookEventName string         `json:"hook_event_name"`
	SessionID     string         `json:"session_id"`
	ToolName      string         `json:"tool_name"`
	ToolInput     map[string]any `json:"tool_input"`
	CWD           string         `json:"cwd"`
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

	if !readOnlyTools[input.ToolName] {
		return nil
	}

	return writeAllow(
		writer,
		"read-only dodder MCP tool, cannot mutate the store",
	)
}

func writeAllow(writer io.Writer, reason string) error {
	output := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "allow",
			"permissionDecisionReason": reason,
		},
	}

	return json.NewEncoder(writer).Encode(output)
}
