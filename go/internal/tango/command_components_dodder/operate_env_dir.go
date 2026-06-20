package command_components_dodder

import (
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"github.com/amarbel-llc/madder/go/pkgs/directory_layout"
	mad_env_dir "github.com/amarbel-llc/madder/go/pkgs/env_dir"
	"github.com/amarbel-llc/madder/go/pkgs/scoped_id"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/xdg"
)

// MakeOperateEnvDir builds one env_dir slot for the nearest-operate repo
// paths (show/query/edit, serve, info xdg/env), honoring every FDR-0019 cwd
// scope that resolves against an *existing* repo:
//
//   - system (`//name`): MakeDefaultAndInitialize roots at the system root (#280).
//   - multi-dot cwd (`..name`, cwdDepth > 0): the Nth same-named ancestor,
//     resolved store-aware via resolveCwdRepoAncestor, then rooted there (#281).
//   - everything else (auto, user, single-dot `.name`): MakeDefault's
//     nearest-ancestor walk, unchanged.
//
// utilityName selects the slot: the dodder metadata slot nests under
// repos/<name>/; the madder blob slot stays flat (repoName ""). Call it once
// per slot. The literal-init paths (genesis, MakeEnvRepo) deliberately stay on
// MakeDefaultAndInitialize(EffectiveId) instead — see repo_id.CheckSupported.
//
// Exported because info's env/xdg display (uniform) shares it with the tango
// repo builders, so `info ..name xdg` reports the same path `show ..name`
// operates on.
func MakeOperateEnvDir(
	req command.Request,
	config repo_config_cli.Config,
	utilityName string,
) mad_env_dir.Env {
	repoId := config.RepoId

	switch {
	case repoId.GetLocationType() == scoped_id.LocationTypeXDGSystem:
		return env_dir.MakeDefaultAndInitialize(
			req,
			utilityName,
			config.Debug,
			repo_id.EffectiveId(repoId),
		)

	case repoId.GetLocationType() == scoped_id.LocationTypeCwd &&
		repoId.GetCwdDepth() > 0:
		ancestor := resolveCwdRepoAncestor(req, config)

		// dodder metadata nests under repos/<name>/; the madder blob pool is
		// flat (see configFor / XDGUtilityNameMadder), so blank its name.
		repoName := repo_id.EffectiveName(repoId)
		if utilityName == XDGUtilityNameMadder {
			repoName = ""
		}

		return env_dir.MakeWithXDGRootOverrideHomeAndInitialize(
			req,
			ancestor,
			utilityName,
			config.Debug,
			repoName,
		)

	default:
		return env_dir.MakeDefault(
			req,
			utilityName,
			config.Debug,
			repo_id.EffectiveName(repoId),
		)
	}
}

// resolveCwdRepoAncestor returns the cwd ancestor a multi-dot (`..name`)
// operate-path id addresses: the GetCwdDepth()-th ancestor (deepest = 0) that
// carries a `.dodder/` override AND actually hosts a dodder repo named
// EffectiveName(id). It is the store-aware OPERATE counterpart to the literal
// Nth-parent INIT walk (madder echo/env_dir.resolveCwdAncestorOrError): the
// operate target must already exist, so non-matching `.dodder/` ancestors are
// skipped (not counted) and an overflow errors rather than clamps. The walk
// always keys on the dodder utility + repo marker even when building the madder
// blob slot — init co-locates `.dodder/` and `.madder/` at the same ancestor,
// so both slots root there.
func resolveCwdRepoAncestor(
	req command.Request,
	config repo_config_cli.Config,
) string {
	cwd, err := os.Getwd()
	if err != nil {
		req.Cancel(err)
	}

	name := repo_id.EffectiveName(config.RepoId)

	ceilings := xdg.ParseCeilingDirectories(
		os.Getenv(xdg.CeilingEnvVarName(dodder_env.XDGUtilityName)),
	)

	// config-seed under the override's XDG data dir is the genesis marker for
	// "a dodder repo named <name> lives here" — layout locked by
	// info_non_xdg.bats: <ancestor>/.dodder/local/share/repos/<name>/...
	// os.Stat keeps the predicate side-effect-free (no mkdir), unlike the
	// *AndInitialize constructors.
	matches := func(ancestor string) bool {
		seed := filepath.Join(
			ancestor,
			"."+dodder_env.XDGUtilityName,
			"local", "share", "repos", name, "config-seed",
		)
		_, statErr := os.Stat(seed)
		return statErr == nil
	}

	ancestor, err := directory_layout.ResolveNthAncestorMatch(
		cwd,
		dodder_env.XDGUtilityName,
		config.RepoId.GetCwdDepth(),
		ceilings,
		matches,
	)
	if err != nil {
		req.Cancel(err)
	}

	return ancestor
}
