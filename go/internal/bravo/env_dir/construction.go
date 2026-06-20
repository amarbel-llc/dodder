package env_dir

import (
	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	mad_env_dir "github.com/amarbel-llc/madder/go/pkgs/env_dir"
	"github.com/amarbel-llc/madder/go/pkgs/scoped_id"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/debug"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/xdg"
)

// configFor returns the env_dir.Config dodder injects into all
// env_dir constructors regardless of utility name.
//
// The original dodder env_dir hardcoded DODDER_XDG_UTILITY_OVERRIDE
// for ALL utility names (see the old before_xdg.go); preserving
// that behavior means even utility="madder" walks read
// DODDER_XDG_UTILITY_OVERRIDE, not MADDER_XDG_UTILITY_OVERRIDE.
// Switching to madder's env-var bundle for utility="madder"
// breaks dodder's existing test harness — utility="madder" walks
// then walk freely until they hit a fixture's .madder/, since the
// harness only sets DODDER_XDG_UTILITY_OVERRIDE.
//
// Per-utility env-var bundles can be revisited as a separate
// change; for #151 bucket B Stage B the priority is preserving
// the bridge behavior bit-for-bit.
//
// FDR-0019: madder blob stores are separate, content-addressed pools
// addressed by store id — they are NEVER nested under a repo's
// repos/<name>/. Only the dodder metadata tree (config-seed, object
// index, inventory-list log, lock) nests. So the madder-scoped env
// (utility dodder_env.XDGUtilityNameMadder) never carries a repo name,
// regardless of what the caller passes; nesting it would hide the blobs
// from madder's flat store addressing.

func configFor(
	utilityName string,
	repoName string,
	do debug.Options,
) mad_env_dir.Config {
	cfg := dodder_env.OwnConfig(do)

	if utilityName != dodder_env.XDGUtilityNameMadder {
		cfg.RepoName = repoName
	}

	return cfg
}

// MakeDefault forwards to mad_env_dir.MakeDefault. The dodder
// EnvVarNames bundle is injected via configFor so the previous
// dodder-only env-var contracts (DODDER_XDG_UTILITY_OVERRIDE etc.)
// keep working.
func MakeDefault(
	context errors.Context,
	utilityName string,
	debugOptions debug.Options,
	repoName string,
) mad_env_dir.Env {
	return mad_env_dir.MakeDefault(
		context,
		configFor(utilityName, repoName, debugOptions),
		utilityName,
	)
}

func MakeDefaultNoInit(
	context errors.Context,
	utilityName string,
	debugOptions debug.Options,
	repoName string,
) mad_env_dir.Env {
	return mad_env_dir.MakeDefaultNoInit(
		context,
		configFor(utilityName, repoName, debugOptions),
		utilityName,
	)
}

// MakeStandardXDGUser builds a fully-initialized env_dir pinned to the
// XDG user home with the cwd walk-up DISABLED (permitCwdXDGOverride
// false), so it addresses the user-scope paths regardless of any ancestor
// .dodder/ override. Unlike MakeDefaultNoInit it runs the standard XDG
// initialization, so Data.ActualValue is populated and safe to read. Use
// it to enumerate the user scope alongside the cwd scope (FDR-0019 #276).
func MakeStandardXDGUser(
	context errors.Context,
	utilityName string,
	debugOptions debug.Options,
	repoName string,
) mad_env_dir.Env {
	return mad_env_dir.MakeWithDefaultHome(
		context,
		configFor(utilityName, repoName, debugOptions),
		utilityName,
		false, // permitCwdXDGOverride: pin to user home, no cwd walk-up
		true,  // initialize: populate XDG paths for reading
	)
}

// MakeFromXDGDotenvPath: scope is determined by the dotenv's XDG;
// configFor is therefore called with the dodder default since the
// scope isn't known until the dotenv is read. dodder's env-var
// bundle is the right choice here regardless because dodder
// processes always honor DODDER_XDG_UTILITY_OVERRIDE.
func MakeFromXDGDotenvPath(
	context errors.Context,
	debugOptions debug.Options,
	xdgDotenvPath string,
	repoName string,
) mad_env_dir.Env {
	cfg := dodder_env.OwnConfig(debugOptions)
	cfg.RepoName = repoName
	return mad_env_dir.MakeFromXDGDotenvPath(
		context,
		cfg,
		xdgDotenvPath,
	)
}

func MakeDefaultAndInitialize(
	context errors.Context,
	utilityName string,
	do debug.Options,
	repoId scoped_id.Id,
) mad_env_dir.Env {
	// madder derives Config.RepoName from the id's name, so strip the name
	// for the madder-scoped (blob-store) env — its blobs are separate and
	// never nest (see configFor). Location is kept so cwd-vs-home routing
	// still applies.
	if utilityName == dodder_env.XDGUtilityNameMadder {
		repoId = scoped_id.MakeWithLocation("", repoId.GetLocationType())
	}

	return mad_env_dir.MakeDefaultAndInitialize(
		context,
		configFor(utilityName, "", do),
		utilityName,
		repoId,
	)
}

func MakeWithDefaultHome(
	context errors.Context,
	utilityName string,
	debugOptions debug.Options,
	permitCwdXDGOverride bool,
	initialize bool,
	repoName string,
) mad_env_dir.Env {
	return mad_env_dir.MakeWithDefaultHome(
		context,
		configFor(utilityName, repoName, debugOptions),
		utilityName,
		permitCwdXDGOverride,
		initialize,
	)
}

// MakeWithXDGRootOverrideHomeAndInitialize keeps dodder's argument
// order (xdgRootOverride before utilityName) for caller stability.
// Forwards to madder's MakeWithXDGRootOverrideHomeAndInitialize
// which uses the inverse order.
func MakeWithXDGRootOverrideHomeAndInitialize(
	context errors.Context,
	xdgRootOverride string,
	utilityName string,
	debugOptions debug.Options,
	repoName string,
) mad_env_dir.Env {
	return mad_env_dir.MakeWithXDGRootOverrideHomeAndInitialize(
		context,
		configFor(utilityName, repoName, debugOptions),
		utilityName,
		xdgRootOverride,
	)
}

// MakeWithHomeAndInitialize keeps dodder's argument order
// (utilityName, home, debug) which differs from madder's
// (Config, scope, home).
func MakeWithHomeAndInitialize(
	context errors.Context,
	utilityName string,
	home string,
	debugOptions debug.Options,
	repoName string,
) mad_env_dir.Env {
	return mad_env_dir.MakeWithHomeAndInitialize(
		context,
		configFor(utilityName, repoName, debugOptions),
		utilityName,
		home,
	)
}

// MakeWithXDG: scope is on the supplied xdg.UtilityName; we route
// through configFor so the right env-var bundle attaches.
func MakeWithXDG(
	context errors.Context,
	debugOptions debug.Options,
	x xdg.XDG,
	repoName string,
) mad_env_dir.Env {
	return mad_env_dir.MakeWithXDG(
		context,
		configFor(x.UtilityName, repoName, debugOptions),
		x,
	)
}
