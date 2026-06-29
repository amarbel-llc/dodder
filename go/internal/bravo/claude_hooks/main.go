// Package claude_hooks implements the Claude Code hook protocol for the
// dodder clown plugin (#244). The plugin's hooks/hooks.json registers a
// PreToolUse hook scoped to dodder's own MCP tools (matcher
// "mcp__plugin_dodder_dodder__.*" — the hook handler spawns a process
// per event, so non-dodder tools are filtered out client-side); the
// handler script execs `dodder hook`, which routes stdin/stdout through
// Run. Keeping the decision table in Go (instead of a hooks.json
// matcher regex) follows spinclass's internal/hooks: the decision is
// unit-testable and has room to grow without touching the shipped
// plugin payload shape.
//
// The decision is mostly the tool's shared capability classification
// (mcp_tool_perms.Permission): read-only tools auto-approve, the
// destructive tool always prompts. Write tools are context-aware — the
// hook reads the tool's data-flow (mcp_tool_perms.WriteFlow) and, for
// scoped writes, the call's repo_id, so a mutation of the repo you are
// in is prompt-free while a write that reaches a different repo, or a
// push that sends data out, still falls through to normal gating.
package claude_hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/mcp_tool_perms"
	"github.com/amarbel-llc/madder/go/pkgs/scoped_id"
)

// hookInput carries the subset of Claude Code's hook-event payload the
// decision table consumes. tool_input is decoded lazily (RawMessage) so
// the scoped-write branch can read its repo_id; other protocol fields
// (session_id, cwd) are deliberately not decoded — the scope signal
// lives in the tool arguments, not the filesystem cwd.
type hookInput struct {
	HookEventName string          `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
}

// Dodder ships as a Claude Code plugin named "dodder" whose clown.json
// registers an MCP server also named "dodder", so its tools appear to
// Claude Code as mcp__plugin_dodder_dodder__<tool>. Stripping this
// prefix yields the bare tool name that mcp_tool_perms classifies.
const toolNamePrefix = "mcp__plugin_dodder_dodder__"

// Run decodes one Claude Code hook event from reader and writes a
// permission decision to writer when one applies. The decision derives
// from the tool's shared classification (mcp_tool_perms — the same
// source the MCP server reads to annotate the tool, #251): read-only
// tools are auto-approved, the destructive tool always prompts, and
// write tools route through decideWrite for a context-aware decision.
// Unknown names get no opinion so Claude Code falls through to its
// normal permission flow.
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

	case mcp_tool_perms.PermissionWrite:
		return decideWrite(writer, name, input.ToolInput)

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

// decideWrite applies the context-aware auto-approval policy for write
// tools, keyed by the tool's data-flow (mcp_tool_perms.WriteFlow):
//
//   - Unconditional (import) auto-approves regardless of target — it
//     reads user-named inventory paths into a repo, no cross-repo prompt.
//   - Scoped (new, edit, organize_commit, checkin, pull) auto-approves
//     only when the write targets the server's default repo or an
//     explicitly cwd-scoped one; a write reaching a different repo falls
//     through.
//   - Gated (push) and any unflagged write fall through to normal
//     gating — push sends data OUT to another repo.
func decideWrite(
	writer io.Writer,
	name string,
	toolInput json.RawMessage,
) error {
	switch mcp_tool_perms.WriteFlowOf(name) {
	case mcp_tool_perms.WriteFlowUnconditional:
		return writeDecision(
			writer,
			"allow",
			"dodder import reads user-named inventory paths into the repo; no cross-repo egress",
		)

	case mcp_tool_perms.WriteFlowScoped:
		if !writesLocalRepo(toolInput) {
			return nil
		}

		return writeDecision(
			writer,
			"allow",
			"dodder write scoped to the server's default repo or a cwd-scoped repo, no cross-repo reach",
		)

	default: // WriteFlowGated, and defensively WriteFlowNone
		return nil
	}
}

// writesLocalRepo reports whether a scoped write targets a repo the hook
// auto-approves: the server's default (no repo_id given — e.g. checkin,
// which has no repo_id param, or an omitted/empty one) or an explicitly
// cwd-scoped repo (.name). A repo_id naming a different XDG-user or
// system repo — or one that does not parse — is treated as non-local so
// the call falls through to normal gating (fail-safe).
func writesLocalRepo(toolInput json.RawMessage) bool {
	if len(toolInput) == 0 {
		return true
	}

	var p struct {
		RepoId string `json:"repo_id"`
	}

	if err := json.Unmarshal(toolInput, &p); err != nil {
		return false
	}

	if p.RepoId == "" {
		return true
	}

	var id scoped_id.Id
	if err := id.Set(p.RepoId); err != nil {
		return false
	}

	return id.GetLocationType() == scoped_id.LocationTypeCwd
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
