package command

import "github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"

type (
	// Arg declares metadata for a single positional argument.
	// Mirrors the flag pattern: flagSet.Var(&cmd.RepoId, "repo", "usage")
	// becomes Arg{Name: "repo-id", Value: &ids.RepoId{}, ...}
	Arg struct {
		// Name is used in synopsis, error messages, and as MCP schema property
		// key. Should match the string passed to PopArg(name).
		Name        string
		Description string
		Required    bool

		// Variadic means this arg consumes all remaining positional arguments.
		// At most one Arg per command may be Variadic, and it must be last.
		Variadic bool

		// EnumValues constrains the arg to listed values. Used for MCP schema
		// enum and shell completion.
		EnumValues []string

		// Value carries type information for schema generation and future
		// auto-parsing. Same interface flags use (StringerSetter). Nil means
		// plain string. When non-nil, the concrete type determines the JSON
		// schema type and Value.Set() provides validation.
		Value interfaces.FlagValue
	}

	// ArgGroup is a named set of args contributed by a command or component.
	// Commands compose ArgGroups from their embedded components.
	ArgGroup struct {
		Name        string // e.g. "query" (empty for inline groups)
		Description string // group-level description for docs
		Args        []Arg
	}

	// CommandWithArgs is the opt-in interface for declarative arg metadata.
	// Commands that implement this get improved manpage SYNOPSIS/ARGUMENTS
	// sections and auto-generated MCP schemas. Commands that don't continue
	// to work as before.
	CommandWithArgs interface {
		GetArgs() []ArgGroup
	}
)

type (
	// MCPAnnotations declares MCP tool hints without importing go-mcp
	// protocol types, keeping the golf layer dependency-free.
	MCPAnnotations struct {
		ReadOnly    bool
		Destructive bool
	}

	// CommandWithMCPAnnotations lets commands declare their MCP hints.
	CommandWithMCPAnnotations interface {
		GetMCPAnnotations() MCPAnnotations
	}
)
