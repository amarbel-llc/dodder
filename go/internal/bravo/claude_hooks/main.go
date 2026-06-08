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
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/mcp_tool_perms"
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
// Claude Code as mcp__plugin_dodder_dodder__<tool>. Stripping this
// prefix yields the bare tool name that mcp_tool_perms classifies.
const toolNamePrefix = "mcp__plugin_dodder_dodder__"

// Run decodes one Claude Code hook event from reader and writes a
// permission decision to writer when one applies. The decision is
// derived from the tool's shared permission classification
// (mcp_tool_perms — the same source the MCP server reads to annotate
// the tool, #251): read-only tools are auto-approved, destructive ones
// always prompt, and everything else (write tools, unknown names) gets
// no opinion so Claude Code falls through to its normal permission
// flow.
func Run(reader io.Reader, writer io.Writer) error {
	var input hookInput

	if err := json.NewDecoder(reader).Decode(&input); err != nil {
		return fmt.Errorf("decoding hook input: %w", err)
	}

	if input.HookEventName != "PreToolUse" {
		return nil
	}

	name, ok := strings.CutPrefix(input.ToolName, toolNamePrefix)
	if !ok {
		return nil
	}

	switch mcp_tool_perms.Of(name) {
	case mcp_tool_perms.PermissionReadOnly:
		return writeDecision(
			writer,
			"allow",
			"read-only dodder MCP tool, cannot mutate the store",
		)

	case mcp_tool_perms.PermissionDestructive:
		return writeDecision(
			writer,
			"ask",
			"destructive dodder MCP tool (e.g. reset-lock forcibly breaks the repo's environment lock); every invocation requires explicit user approval",
		)

	default:
		return nil
	}
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
