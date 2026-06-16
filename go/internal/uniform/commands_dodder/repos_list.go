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

// listRepoNames returns the names of the repos present in the active dodder
// scope: the subdirectories of <data>/repos/ in the un-nested dodder data
// dir. isCwd reports whether that scope resolved to a cwd/ancestor .dodder
// (true) versus the XDG user home (false), so callers can choose the
// `.name` vs `name` repo-id spelling. Shared by `info-repo repos` and
// -repo_id completion (FDR-0019).
func listRepoNames(
	req command.Request,
) (names []string, isCwd bool, err error) {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

	// An empty repoName keeps the data dir un-nested (the base, not a
	// repos/<name>/ nest), so its repos/ subdir holds every repo. Use
	// MakeDefault (not NoInit) so the cwd walk-up resolves the same
	// ancestor .dodder/ that repo-opening commands use — NoInit returns
	// before the override resolution and would resolve against $HOME
	// instead. This resolves XDG paths only; dir creation is env_repo's job.
	dir := env_dir.MakeDefault(
		req,
		dodder_env.XDGUtilityName,
		config.Debug,
		"",
	)

	xdg := dir.GetXDG()
	isCwd = xdg.IsOverridden()

	reposDir := filepath.Join(xdg.Data.ActualValue, "repos")

	var entries []os.DirEntry

	if entries, err = os.ReadDir(reposDir); err != nil {
		if errors.IsNotExist(err) {
			// No repos created yet — not an error, just an empty list.
			err = nil
			return names, isCwd, err
		}

		err = errors.Wrap(err)
		return names, isCwd, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}

	sort.Strings(names)

	return names, isCwd, err
}
