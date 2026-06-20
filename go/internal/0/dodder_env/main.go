// Package dodder_env holds dodder-flavored constants and configuration
// bundles consumed by env_dir construction sites.
//
// Lives at NATO tier 0 because it's a pure data-only package (no behavior,
// no dependencies on other dodder packages) consumed across the tree.
package dodder_env

import (
	"github.com/amarbel-llc/madder/go/pkgs/env_dir"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/debug"
)

const (
	// XDGUtilityName is dodder's XDG scope segment — the `<scope>` in
	// `$XDG_*_HOME/<scope>/`. Used to construct dodder's own-state
	// env_dir (cache, state, config, log).
	XDGUtilityName = "dodder"

	// XDGUtilityNameMadder is madder's XDG scope segment. Used to
	// construct the madder-scope env_dir for blob-store operations.
	XDGUtilityNameMadder = "madder"

	// EnvDir is the env-var name dodder reads to override the
	// repository base path. Honored by env_repo.Make when set.
	EnvDir = "DIR_DODDER"

	// EnvSystemRoot is the env-var name dodder reads to override the
	// filesystem root that system-scoped (`//name`) repos resolve under.
	// When set it overrides BOTH the dodder-metadata and madder-blob
	// system roots, so a relocation (or a test sandbox) can point both
	// under one directory; when unset the per-utility defaults apply
	// (DefaultSystemRoot for dodder, madder_env.DefaultSystemRoot for the
	// blob slot). FDR-0019 #280.
	EnvSystemRoot = "DODDER_SYSTEM_ROOT"

	// DefaultSystemRoot is the filesystem root dodder's system-scoped
	// (`//name`) metadata resolves under when EnvSystemRoot is unset — a
	// fixed FHS path for writable system state, parallel to madder's
	// /var/lib/madder. A system repo's metadata lands at
	// <DefaultSystemRoot>/<xdg>/repos/<name>. FDR-0019 #280.
	DefaultSystemRoot = "/var/lib/dodder"
)

// EnvVarNames is the env-var-names bundle dodder injects into its
// own-scope env_dir constructors. Madder-scoped env_dirs (constructed
// for blob-store ops) use madder_env.DefaultEnvVarNames instead.
var EnvVarNames = env_dir.EnvVarNames{
	Binary:             "BIN_DODDER",
	XDGUtilityOverride: "DODDER_XDG_UTILITY_OVERRIDE",
	VerifyOnCollision:  "DODDER_VERIFY_ON_COLLISION",
}

// OwnConfig builds an env_dir.Config for own (dodder-scoped) env_dir
// constructors: dodder env-var names, caller-supplied debug options.
func OwnConfig(debugOptions debug.Options) env_dir.Config {
	return env_dir.Config{
		EnvVarNames:  EnvVarNames,
		DebugOptions: debugOptions,
	}
}
