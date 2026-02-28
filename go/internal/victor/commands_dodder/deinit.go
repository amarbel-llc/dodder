package commands_dodder

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/comments"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

func init() {
	utility.AddCmd("deinit", &Deinit{})
}

type Deinit struct {
	command_components_dodder.LocalWorkingCopy

	Force bool
}

var _ interfaces.CommandComponentWriter = (*Deinit)(nil)

func (cmd *Deinit) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	flagDefinitions.BoolVar(
		&cmd.Force,
		"force",
		false,
		"force deinit",
	)
}

func (cmd Deinit) Run(req command.Request) {
	repo := cmd.MakeLocalWorkingCopy(req)

	var home string

	{
		var err error

		if home, err = os.UserHomeDir(); err != nil {
			req.Cancel(err)
		}

		if home, err = filepath.Abs(home); err != nil {
			req.Cancel(err)
		}
	}

	exdg := repo.GetEnvRepo().GetXDG()

	comments.Comment(
		"determine if this is a native XDG repo, or an overridden XDG repo",
	)

	var xdgHome string

	{
		var err error

		if xdgHome, err = filepath.Abs(exdg.Home.String()); err != nil {
			req.Cancel(err)
		}
	}

	var filesAndDirectories []string

	if xdgHome == home {
		filesAndDirectories = repo.GetEnvRepo().GetXDG().GetXDGPaths()
	} else {
		filesAndDirectories = []string{xdgHome}
	}

	sort.Strings(filesAndDirectories)

	if !repo.GetEnvWorkspace().IsTemporary() {
		workspaceConfigFilePath := repo.GetEnvWorkspace().GetWorkspaceConfigFilePath()

		if workspaceConfigFilePath != "" {
			filesAndDirectories = append(
				filesAndDirectories,
				workspaceConfigFilePath,
			)
		}
	}

	// TODO decide whether the workspace directory should be deleted too

	if !cmd.Force &&
		!repo.Confirm(
			"are you sure you want to deinit?",
			fmt.Sprintf(
				`The following directories and files would be deleted:

%s`, strings.Join(filesAndDirectories, "\n")),
		) {
		ui.Err().Print("permission denied and -force not specified, aborting")
		return
	}

	base := path.Join(repo.GetEnvRepo().MakeDirData().String())

	if err := files.SetAllowUserChangesRecursive(base); err != nil {
		repo.Cancel(err)
	}

	if err := repo.Delete(
		filesAndDirectories...,
	); err != nil {
		repo.Cancel(err)
	}
}
