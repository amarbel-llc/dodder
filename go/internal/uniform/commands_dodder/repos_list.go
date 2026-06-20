package commands_dodder

import (
	"os"
	"path/filepath"
	"sort"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// scopedRepo is a repo discovered in a specific scope, carrying the CLI
// spelling that addresses it: `.name` for a cwd-scope repo, `name` for an
// XDG-user repo. So callers emit a directly-usable -repo_id.
type scopedRepo struct {
	Name  string
	IsCwd bool
}

// Spelling is the -repo_id token that addresses this repo from anywhere.
func (r scopedRepo) Spelling() string {
	if r.IsCwd {
		return "." + r.Name
	}

	return r.Name
}

// ScopeLabel is a short human description of the repo's scope.
func (r scopedRepo) ScopeLabel() string {
	if r.IsCwd {
		return "cwd repo"
	}

	return "user repo"
}

// listScopedRepos enumerates the repos addressable from here across both
// scopes. A -repo_id can name a cwd repo (`.name`) and an XDG-user repo
// (`name`) regardless of cwd, so the listing shows both. The active scope
// is the cwd walk-up (MakeDefault, IsOverridden when an ancestor .dodder/
// is in play); when it resolves to a cwd repo, the XDG-user scope is a
// distinct second set, enumerated via MakeStandardXDGUser (cwd walk-up
// disabled). When the active scope already IS the user home there is no
// separate cwd scope to add. Shared by `info-repo repos` and -repo_id
// completion (FDR-0019 #276).
func listScopedRepos(req command.Request) ([]scopedRepo, error) {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

	// An empty repoName keeps the data dir un-nested (the base, not a
	// repos/<name>/ nest), so its repos/ subdir holds every repo. These
	// resolve XDG paths only; dir creation is env_repo's job.
	active := env_dir.MakeDefault(
		req,
		dodder_env.XDGUtilityName,
		config.Debug,
		"",
	)

	activeIsCwd := active.GetXDG().IsOverridden()

	var repos []scopedRepo

	activeNames, err := readRepoNames(active.GetXDG().Data.ActualValue)
	if err != nil {
		return nil, err
	}

	for _, name := range activeNames {
		repos = append(repos, scopedRepo{Name: name, IsCwd: activeIsCwd})
	}

	if activeIsCwd {
		user := env_dir.MakeStandardXDGUser(
			req,
			dodder_env.XDGUtilityName,
			config.Debug,
			"",
		)

		userNames, err := readRepoNames(user.GetXDG().Data.ActualValue)
		if err != nil {
			return nil, err
		}

		for _, name := range userNames {
			repos = append(repos, scopedRepo{Name: name, IsCwd: false})
		}
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Spelling() < repos[j].Spelling()
	})

	return repos, nil
}

// readRepoNames returns the subdirectory names of <data>/repos, or an
// empty slice when that directory does not exist yet (no repos created).
func readRepoNames(dataDir string) ([]string, error) {
	reposDir := filepath.Join(dataDir, "repos")

	entries, err := os.ReadDir(reposDir)
	if err != nil {
		if errors.IsNotExist(err) {
			return nil, nil
		}

		return nil, errors.Wrap(err)
	}

	var names []string

	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}

	return names, nil
}
