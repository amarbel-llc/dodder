package command_components_dodder

import (
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/madder/go/pkgs/directory_layout"
	mad_env_dir "code.linenisgreat.com/madder/go/pkgs/env_dir"
	"code.linenisgreat.com/madder/go/pkgs/scoped_id"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/xdg"
)

// MakeOperateEnvDir builds one env_dir slot for the nearest-operate repo
// paths (show/query/edit, serve, info xdg/env), honoring every FDR-0019 cwd
// scope that resolves against an *existing* repo:
//
//   - system (`//name`) and explicit user (`name`): MakeDefaultAndInitialize
//     preserves the scoped_id's LocationType — system roots at the system root
//     (#280); an explicit XDG-user bare name pins to the XDG-user home with NO
//     cwd walk-up (FDR-0019: bare name is XDG-user scope unconditionally, so it
//     must not be hijacked to a workspace-local repo by an ancestor .dodder/).
//   - multi-dot cwd (`..name`, cwdDepth > 0): the Nth same-named ancestor,
//     resolved store-aware via resolveCwdRepoAncestor, then rooted there (#281).
//   - everything else (auto, single-dot `.name`): MakeDefault's
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
	return makeOperateEnvDir(req, config, utilityName, true)
}

// MakeOperateEnvDirNoInit computes a repo's on-disk paths for a root that may
// not exist yet, without the mkdir side effect, for existence checks (e.g.
// ParentBackedWorkspace.ValidateParentRepo probing a -parent path). Only the
// BasePath branch has genuinely side-effect-free behavior: its base dir is the
// explicit override, computed before any mkdir (madder#260). Every other
// branch still initializes — the default/XDG-user/system/cwd-multi-dot scopes
// address roots that already exist by the time an existence check reaches them
// (the home repo, an XDG-user repo), so the mkdir is a harmless no-op, and the
// only truly no-init constructor for the default branch (MakeDefaultNoInit)
// leaves the XDG base unpopulated — see the default-branch note below.
func MakeOperateEnvDirNoInit(
	req command.Request,
	config repo_config_cli.Config,
	utilityName string,
) mad_env_dir.Env {
	return makeOperateEnvDir(req, config, utilityName, false)
}

func makeOperateEnvDir(
	req command.Request,
	config repo_config_cli.Config,
	utilityName string,
	initialize bool,
) mad_env_dir.Env {
	repoId := config.RepoId

	switch {
	case config.BasePath != "":
		// -dir-dodder (config.BasePath) roots the env_dir tree at an explicit
		// path, taking precedence over the scope's default location — the scope
		// still selects repos/<name> WITHIN that tree. Without this branch
		// -dir-dodder is inert on operate commands (it only reached
		// env_repo.Make / genesis), so `show -dir-dodder <path>` fell back to the
		// default XDG repo. Rooting here lets a filesystem path be addressed as
		// `-dir-dodder <path>` + scope (#343). The madder blob slot never nests a
		// repo name (configFor), so blank it there.
		repoName := repo_id.EffectiveName(repoId)
		if utilityName == XDGUtilityNameMadder {
			repoName = ""
		}

		if !initialize {
			return env_dir.MakeWithXDGRootOverrideHomeNoInit(
				req,
				config.BasePath,
				utilityName,
				config.Debug,
				repoName,
			)
		}

		return env_dir.MakeWithXDGRootOverrideHomeAndInitialize(
			req,
			config.BasePath,
			utilityName,
			config.Debug,
			repoName,
		)

	case repoId.GetLocationType() == scoped_id.LocationTypeXDGSystem ||
		repoId.GetLocationType() == scoped_id.LocationTypeXDGUser:
		// Both are explicit scopes that must NOT take MakeDefault's cwd
		// walk-up: an explicit XDG-user bare `name` would otherwise be
		// hijacked to a workspace-local repo when an ancestor .dodder/ is in
		// play (FDR-0019: bare name is XDG-user scope unconditionally).
		// MakeDefaultAndInitialize preserves the scoped_id's LocationType, so
		// madder roots XDGUser at the user home and XDGSystem at the system
		// root, neither with the override.
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
		// NOTE: the no-init form deliberately does NOT use MakeDefaultNoInit
		// here. MakeDefaultNoInit skips the XDG base initialization (not just the
		// mkdir), leaving GetXDG().Data.ActualValue as a bare "repos/<name>" with
		// no base dir — unusable for path computation. Only the BasePath branch
		// above has a sound no-init form (its base comes from the explicit
		// override, computed pre-mkdir). The default branch addresses the home
		// repo, which already exists, so the initializing form's mkdir is a
		// harmless no-op even on the validation path. See parent_backed_workspace
		// ValidateParentRepo (only -parent paths need the true no-init behavior).
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
