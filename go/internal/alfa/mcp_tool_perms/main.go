// Package mcp_tool_perms is the single source of truth for how each
// dodder MCP tool may affect the store. Two consumers read it, in tiers
// that cannot see each other: the MCP server (tango/mcp_dodder) maps a
// Permission to the go-mcp ToolAnnotations it registers, and the clown
// plugin's PreToolUse hook (bravo/claude_hooks) maps it to a permission
// decision. Classifying a tool in one place but not the other used to
// silently drift — a new read-only tool would lose auto-approval, or a
// renamed one would leave a stale entry (#251). Both now derive from
// ByName here.
package mcp_tool_perms

// Permission is what a tool can do to the store.
type Permission int

const (
	// PermissionUnknown is the zero value: a tool not classified here.
	// Consumers treat it as the most-restrictive option (no
	// auto-approval, normal gating) so an unclassified tool fails safe.
	PermissionUnknown Permission = iota

	// PermissionReadOnly: cannot mutate the store or workspace; safe to
	// auto-approve.
	PermissionReadOnly

	// PermissionWrite: mutates the store; keeps normal permission
	// gating.
	PermissionWrite

	// PermissionDestructive: discards state (e.g. reset-lock forcibly
	// breaks the env lock); must prompt the user on every call.
	PermissionDestructive
)

// ByName classifies every MCP tool by its bare name (no Claude Code
// plugin prefix). Keep in lockstep with the tools registered in
// tango/mcp_dodder/server.go; the consistency test there guards it.
var ByName = map[string]Permission{
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

// Of returns the Permission for a bare tool name, or PermissionUnknown
// if the name is not classified.
func Of(name string) Permission {
	return ByName[name]
}

// WriteFlow refines a PermissionWrite tool by how its write moves data
// relative to the repo the call targets. Only the PreToolUse hook
// (bravo/claude_hooks) reads it, to make a context-aware auto-approval
// decision; the capability classification above (Permission, consumed
// by the MCP server's tool annotations) stays deliberately separate,
// because auto-approving a write does not make it read-only — a
// WriteFlowScoped tool is still annotated PermissionWrite. Only
// PermissionWrite tools carry a flow; every other tool is WriteFlowNone.
type WriteFlow int

const (
	// WriteFlowNone is the zero value: the tool is not a write tool (or
	// is unclassified), so WriteFlow says nothing about it.
	WriteFlowNone WriteFlow = iota

	// WriteFlowScoped: the write lands in the repo the call addresses
	// via its repo_id. The hook auto-approves it only when that repo is
	// the server's default (no repo_id given) or an explicitly
	// cwd-scoped repo (.name); a write aimed at a different XDG-user or
	// system repo falls through to normal gating.
	WriteFlowScoped

	// WriteFlowUnconditional: always safe to auto-approve regardless of
	// the target repo. import reads user-named inventory paths into a
	// repo — the user has already chosen the source files, so it carries
	// no cross-repo prompt.
	WriteFlowUnconditional

	// WriteFlowGated: never auto-approved; always falls through to
	// normal gating. push sends objects OUT to another repo — the one
	// write direction that leaves the local repo.
	WriteFlowGated
)

// WriteFlowByName classifies each PermissionWrite tool by its data flow.
// Every tool ByName marks PermissionWrite MUST appear here (the
// consistency test guards it); tools of any other permission are absent
// and resolve to WriteFlowNone.
var WriteFlowByName = map[string]WriteFlow{
	"new":             WriteFlowScoped,
	"edit":            WriteFlowScoped,
	"organize_commit": WriteFlowScoped,
	"checkin":         WriteFlowScoped,
	"pull":            WriteFlowScoped,
	"import":          WriteFlowUnconditional,
	"push":            WriteFlowGated,
}

// WriteFlowOf returns the WriteFlow for a bare tool name, or
// WriteFlowNone if the tool is not a classified write tool.
func WriteFlowOf(name string) WriteFlow {
	return WriteFlowByName[name]
}
