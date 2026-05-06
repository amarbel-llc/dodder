// Package env_dir provides dodder's blob I/O wrappers (NewReader,
// NewWriter, NewMover) and its hash-bucket / temporary-file helpers,
// plus thin forwarders for env_dir construction that delegate to
// madder/pkgs/env_dir.
//
// #151 bucket B Stage B reduces this package to dodder-specific
// concerns (blob I/O and utilities) — the env interface and its
// concrete implementation now live in madder's pkgs/env_dir, with
// dodder's Env type as a type alias preserving the import-path
// surface so existing callers don't have to be rewritten en masse.
package env_dir

import (
	mad_env_dir "github.com/amarbel-llc/madder/go/pkgs/env_dir"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
)

const (
	// EnvDir is dodder's repository-base-path env var; honored by
	// env_repo.Make. Mirrored from dodder_env.EnvDir for backward
	// compatibility with callers that still reference it via this
	// package.
	EnvDir = dodder_env.EnvDir

	// XDGUtilityNameDodder is dodder's XDG scope segment. Mirrored
	// from dodder_env.XDGUtilityName.
	XDGUtilityNameDodder = dodder_env.XDGUtilityName
)

// Env is aliased to madder's env_dir.Env — the env interface lives
// upstream now. Concrete env values are produced by the madder
// constructors invoked from this package's MakeXxx forwarders.
type Env = mad_env_dir.Env

// RelativePath is aliased upstream too. Implementations live in
// madder; dodder callers reference the alias for stable import
// paths.
type RelativePath = mad_env_dir.RelativePath

// Config is dodder's blob I/O config (hash format, compression,
// encryption). Distinct from madder env_dir.Config (env-construction
// inputs); the name overlap is local-only since callers import this
// package as `env_dir` and madder's as `mad_env_dir`.
//
// (defined in blob_config.go)

// TemporaryFS is aliased upstream too.
//
// (defined in temp.go)
