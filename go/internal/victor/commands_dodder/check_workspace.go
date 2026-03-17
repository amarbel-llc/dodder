package commands_dodder

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd(
		"check-workspace",
		&CheckWorkspace{},
	)
}

type CheckWorkspace struct {
	command_components_dodder.LocalWorkingCopy
}

func (cmd CheckWorkspace) Run(req command.Request) {
	subcmd := req.PopArg("subcommand (dirty)")

	if subcmd != "dirty" {
		req.Cancel(errors.BadRequestf("unknown check-workspace subcommand: %s", subcmd))
		return
	}

	cmd.runDirty(req)
}

func (cmd CheckWorkspace) runDirty(req command.Request) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)
	envWorkspace := localWorkingCopy.GetEnvWorkspace()

	if envWorkspace.IsTemporary() {
		fmt.Fprintln(os.Stderr, "not in a workspace")
		os.Exit(2)
	}

	syncTai, syncDigest := envWorkspace.GetSyncBaseline()
	if syncTai == "" {
		// No baseline stored — treat as dirty (workspace predates this feature
		// or is a V0 lightweight workspace with no sync tracking)
		os.Exit(0)
	}

	last, err := localWorkingCopy.GetInventoryListStore().ReadLast()
	if err != nil {
		localWorkingCopy.Cancel(err)
		return
	}

	currentTai := last.GetTai().String()
	currentDigest := last.GetMetadata().GetObjectDigest().String()

	// Compare TAI first (cheapest), then digest as belt-and-suspenders
	if currentTai != syncTai || currentDigest != syncDigest {
		os.Exit(0) // dirty
	}

	os.Exit(1) // clean
}
