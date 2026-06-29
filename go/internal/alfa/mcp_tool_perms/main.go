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
