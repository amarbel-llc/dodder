package env_dir

import (
	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/0/repo_id"
	mad_env_dir "github.com/amarbel-llc/madder/go/pkgs/env_dir"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/echo/debug"
	"github.com/amarbel-llc/purse-first/libs/dewey/echo/xdg"
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
func configFor(_ string, do debug.Options) mad_env_dir.Config {
	return dodder_env.OwnConfig(do)
}

// MakeDefault forwards to mad_env_dir.MakeDefault. The dodder
// EnvVarNames bundle is injected via configFor so the previous
// dodder-only env-var contracts (DODDER_XDG_UTILITY_OVERRIDE etc.)
// keep working.
func MakeDefault(
	context errors.Context,
	utilityName string,
	debugOptions debug.Options,
) mad_env_dir.Env {
	return mad_env_dir.MakeDefault(
		context,
		configFor(utilityName, debugOptions),
		utilityName,
	)
}

func MakeDefaultNoInit(
	context errors.Context,
	utilityName string,
	debugOptions debug.Options,
) mad_env_dir.Env {
	return mad_env_dir.MakeDefaultNoInit(
		context,
		configFor(utilityName, debugOptions),
		utilityName,
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
) mad_env_dir.Env {
	return mad_env_dir.MakeFromXDGDotenvPath(
		context,
		dodder_env.OwnConfig(debugOptions),
		xdgDotenvPath,
	)
}

func MakeDefaultAndInitialize(
	context errors.Context,
	utilityName string,
	do debug.Options,
	repoId repo_id.Id,
) mad_env_dir.Env {
	return mad_env_dir.MakeDefaultAndInitialize(
		context,
		configFor(utilityName, do),
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
) mad_env_dir.Env {
	return mad_env_dir.MakeWithDefaultHome(
		context,
		configFor(utilityName, debugOptions),
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
) mad_env_dir.Env {
	return mad_env_dir.MakeWithXDGRootOverrideHomeAndInitialize(
		context,
		configFor(utilityName, debugOptions),
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
) mad_env_dir.Env {
	return mad_env_dir.MakeWithHomeAndInitialize(
		context,
		configFor(utilityName, debugOptions),
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
) mad_env_dir.Env {
	return mad_env_dir.MakeWithXDG(
		context,
		configFor(x.UtilityName, debugOptions),
		x,
	)
}
