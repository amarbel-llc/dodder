package commands_dodder

import (
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/internal/tango/mcp_dodder"
)

func init() {
	utility.AddCmd("mcp", &Mcp{})
}

func (cmd Mcp) GetDescription() command.Description {
	return command.Description{
		Short: "start the MCP server",
	}
}

type Mcp struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*Mcp)(nil)

// GetArgs returns nil: no positional arguments.
func (cmd *Mcp) GetArgs() []command.ArgGroup { return nil }

func (cmd Mcp) Run(req command.Request) {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())
	repo := cmd.MakeLocalWorkingCopy(req)

	// The XDG-user-home repos/ dir, resolved with the cwd walk-up disabled
	// (MakeStandardXDGUser), so the dodder:///repos listing can show the
	// user scope alongside the startup (cwd) scope like `info-repo repos`
	// (FDR-0019 #276). Resolves paths only; an absent dir reads as empty.
	userDir := env_dir.MakeStandardXDGUser(
		req,
		dodder_env.XDGUtilityName,
		config.Debug,
		"",
	)
	userReposDir := filepath.Join(userDir.GetXDG().Data.ActualValue, "repos")

	// The MCP server starts cleanly whether or not the CWD is a dodder
	// workspace. mcp_dodder.RunServer inspects the workspace env and
	// advertises only the tools that make sense in the current mode (see
	// github.com/amarbel-llc/dodder/issues/116). config.RepoId is the
	// startup repo: explicit -repo_id pins the server to it; auto/default
	// lets each tool call select a repo via the repo_id param (FDR-0019).
	if err := mcp_dodder.RunServer(
		req.Utility,
		repo,
		config.RepoId,
		userReposDir,
	); err != nil {
		req.Cancel(err)
	}
}
