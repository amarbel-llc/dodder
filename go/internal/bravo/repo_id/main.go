// Package repo_id implements dodder's repo *location* selector
// (FDR-0019, "Scoped Repo Resolution").
//
// It extends the legacy location-only selector (`.`, `/`, empty —
// FDR-0003) with an optional name and a cwd dot-depth, mirroring
// madder's blob_store_id grammar so several named repos can coexist
// per scope under a repos/<name>/ layout.
//
// This is the dodder-side prototype that lands ahead of the eventual
// madder env_dir.RepoId grammar change the FDR calls for. Two
// deliberate prototype limitations:
//
//   - Only single-dot cwd depth resolves. `..name` (depth > 0) parses
//     but callers reject it at resolution time.
//   - `/name` resolves to the system scope; the FDR's remote-first
//     lookup (try a defined remote, then fall back to system) is not
//     implemented, so `/name` and `//name` behave identically here.
//
// This selector is distinct from ids.RepoId (the repo *object* genre),
// which the FDR leaves untouched.
package repo_id

import (
	"strings"

	mad_env_dir "github.com/amarbel-llc/madder/go/pkgs/env_dir"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// Scope is the location scope a repo id resolves against.
type Scope int

const (
	// ScopeUser is the XDG user data tree (unprefixed name, or empty).
	ScopeUser Scope = iota
	// ScopeCwd is the nearest ancestor `.dodder/` tree (`.name`).
	ScopeCwd
	// ScopeSystem is the XDG system data tree (`//name`, or `/name`).
	ScopeSystem
)

// Id is dodder's repo location selector. The zero value is the empty
// (auto) id.
type Id struct {
	isSet    bool
	scope    Scope
	name     string
	cwdDepth uint
}

// Set parses the repo-id grammar. It satisfies flag.Value via a
// pointer receiver.
func (id *Id) Set(value string) (err error) {
	*id = Id{}

	switch value {
	case "":
		// Auto: nearest cwd repo on the walk-up, else user `default`.
		// The prototype treats empty as the legacy user scope so all
		// existing single-repo trees keep resolving unchanged.
		return nil

	case ".":
		id.isSet = true
		id.scope = ScopeCwd
		return nil

	case "/":
		id.isSet = true
		id.scope = ScopeSystem
		return nil
	}

	id.isSet = true

	switch {
	case strings.HasPrefix(value, "//"):
		id.scope = ScopeSystem
		id.name = value[2:]

	case value[0] == '/':
		// FDR-0019 reserves `/name` for remote-first selection. The
		// remote transport is out of scope, so the prototype resolves
		// it straight to the system scope.
		id.scope = ScopeSystem
		id.name = value[1:]

	case value[0] == '.':
		dots := 0
		for dots < len(value) && value[dots] == '.' {
			dots++
		}

		if dots == len(value) {
			err = errors.Errorf("repo_id is all dots, no name: %q", value)
			return err
		}

		id.scope = ScopeCwd
		id.cwdDepth = uint(dots - 1)
		id.name = value[dots:]

	case value[0] == '~':
		// `~name` is the parse-only user-scope alias; never emitted.
		id.scope = ScopeUser
		id.name = value[1:]

	default:
		id.scope = ScopeUser
		id.name = value
	}

	if err = validateName(id.name); err != nil {
		err = errors.Wrapf(err, "repo_id: %q", value)
		return err
	}

	return err
}

// String renders the canonical form: cwd ids collapse to a single dot
// (depth dropped, #145 precedent), system named ids emit `//name`, and
// user ids emit the bare name with no prefix.
func (id Id) String() string {
	if !id.isSet {
		return ""
	}

	switch id.scope {
	case ScopeCwd:
		if id.name == "" {
			return "."
		}
		return "." + id.name

	case ScopeSystem:
		if id.name == "" {
			return "/"
		}
		return "//" + id.name

	default:
		return id.name
	}
}

// IsEmpty reports whether nothing was selected (the auto id). Legacy
// nameless `.` / `/` are NOT empty — they pin a scope.
func (id Id) IsEmpty() bool {
	return !id.isSet
}

func (id Id) IsCwd() bool {
	return id.isSet && id.scope == ScopeCwd
}

func (id Id) IsSystem() bool {
	return id.isSet && id.scope == ScopeSystem
}

// GetName returns the name portion ("" for the legacy nameless and
// auto forms). A non-empty name triggers the repos/<name>/ nesting.
func (id Id) GetName() string {
	return id.name
}

// GetCwdDepth returns the 0-indexed ancestor depth for cwd-scoped ids
// (0 = single dot = nearest).
func (id Id) GetCwdDepth() uint {
	return id.cwdDepth
}

// GetMad projects this id onto the legacy madder env_dir.RepoId so the
// existing scope routing (MakeDefaultAndInitialize) keeps working. The
// name is carried separately and applied as repos/<name>/ nesting.
func (id Id) GetMad() mad_env_dir.RepoId {
	var mad mad_env_dir.RepoId

	switch {
	case id.IsCwd():
		_ = mad.Set(".")
	case id.IsSystem():
		_ = mad.Set("/")
	}

	return mad
}

// CheckPrototypeSupported rejects the grammar this dodder-side
// prototype does not yet resolve (multi-dot cwd depth). It is removed
// once the madder env_dir.RepoId change lands the full walk-up.
func (id Id) CheckPrototypeSupported() (err error) {
	if id.cwdDepth > 0 {
		err = errors.Errorf(
			"repo_id %q: cwd dot-depth > 1 is not yet implemented",
			id.String(),
		)
		return err
	}

	return err
}

func validateName(name string) (err error) {
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_',
			r == '-':
		default:
			err = errors.Errorf(
				"name may contain only [a-zA-Z0-9_-]; got %q",
				string(r),
			)
			return err
		}
	}

	return err
}
