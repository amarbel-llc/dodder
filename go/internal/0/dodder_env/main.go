// Package dodder_env holds dodder-flavored constants and configuration
// bundles consumed by env_dir construction sites.
//
// Lives at NATO tier 0 because it's a pure data-only package (no behavior,
// no dependencies on other dodder packages) consumed across the tree.
package dodder_env

import (
	"github.com/amarbel-llc/madder/go/pkgs/env_dir"
)

const (
	// XDGUtilityName is dodder's XDG scope segment — the `<scope>` in
	// `$XDG_*_HOME/<scope>/`. Used to construct dodder's own-state
	// env_dir (cache, state, config, log).
	XDGUtilityName = "dodder"

	// EnvDir is the env-var name dodder reads to override the
	// repository base path. Honored by env_repo.Make when set.
	EnvDir = "DIR_DODDER"
)

// EnvVarNames is the env-var-names bundle dodder injects into its
// own-scope env_dir constructors. Madder-scoped env_dirs (constructed
// for blob-store ops) use madder_env.DefaultEnvVarNames instead.
var EnvVarNames = env_dir.EnvVarNames{
	Binary:             "BIN_DODDER",
	XDGUtilityOverride: "DODDER_XDG_UTILITY_OVERRIDE",
	VerifyOnCollision:  "DODDER_VERIFY_ON_COLLISION",
}
