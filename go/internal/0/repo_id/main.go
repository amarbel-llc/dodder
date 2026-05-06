// Package repo_id holds dodder's repo identifier — historically a
// fork of madder's env_dir.RepoId. As part of #151 bucket B Stage B
// the type is aliased to madder's so dodder env_dir constructors
// can pass repo_id.Id straight into madder's MakeDefaultAndInitialize
// without a conversion layer.
package repo_id

import (
	mad_env_dir "github.com/amarbel-llc/madder/go/pkgs/env_dir"
)

// Id is an alias to madder's RepoId. The underlying struct fields
// and methods (IsEmpty, GetLocationType, Set, String, IsCwd,
// IsSystem) are identical between the two; the alias preserves
// dodder's import-path surface (`repo_id.Id`) while letting madder
// own the type.
type Id = mad_env_dir.RepoId
