package commands_dodder

import (
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/echo/workspace_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

// initWorkspaceDefaultIgnorePatterns is the default ignore set written
// by init-workspace-default. Dot-prefixed paths (.git, .dodder, …) are
// already skipped by store_fs's scan, so these target common non-dot
// build / dependency directories.
var initWorkspaceDefaultIgnorePatterns = []string{
	"node_modules",
	"target",
	"dist",
	"build",
	"vendor",
}

func init() {
	utility.AddCmd("init-workspace-default", &InitWorkspaceDefault{})
}

// InitWorkspaceDefault creates a workspace in the current directory with
// a sensible default ignore-pattern set (#232) -- the workspace analogue
// of init-default, for an unattended / per-session bootstrap. It writes a
// V3 workspace config and is a no-op when a workspace already exists.
type InitWorkspaceDefault struct {
	command_components_dodder.LocalWorkingCopy
}

var (
	_ interfaces.CommandComponentWriter = (*InitWorkspaceDefault)(nil)
	_ command.CommandWithArgs           = (*InitWorkspaceDefault)(nil)
)

func (cmd *InitWorkspaceDefault) GetArgs() []command.ArgGroup {
	return nil
}

func (cmd InitWorkspaceDefault) GetDescription() command.Description {
	return command.Description{
		Short: "initialize a workspace with default ignore patterns",
	}
}

func (cmd *InitWorkspaceDefault) Run(req command.Request) {
	req.AssertNoMoreArgs()

	cwd, err := os.Getwd()
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	// Idempotency: a workspace already configured here is a no-op, so the
	// bootstrap is safe to re-run and skips repos that ship their own
	// .dodder-workspace.
	if pathExists(filepath.Join(cwd, env_repo.FileWorkspace)) {
		return
	}

	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	blob := &workspace_config_blobs.V3{
		Ignore: initWorkspaceDefaultIgnorePatterns,
	}

	if err := localWorkingCopy.GetEnvWorkspace().CreateWorkspace(
		blob,
	); err != nil {
		req.Cancel(err)
	}
}
