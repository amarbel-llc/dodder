package commands_dodder

import (
	"os"

	"code.linenisgreat.com/dodder/go/internal/golf/tap_diagnostics"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_ui"
	"code.linenisgreat.com/dodder/go/internal/india/env_dir"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/lima/command_components_madder"
	"code.linenisgreat.com/dodder/go/internal/whiskey/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/yankee/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	tap "github.com/amarbel-llc/tap-dancer/go"
)

func init() {
	utility.AddCmd(
		"repo-fsck",
		&RepoFsck{})
}

type RepoFsck struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.EnvRepo
	command_components_madder.BlobStore
}

// TODO add completion for blob store id's

func (cmd RepoFsck) Run(req command.Request) {
	req.AssertNoMoreArgs()

	repo := cmd.MakeLocalWorkingCopyWithOptions(
		req,
		env_ui.Options{},
		local_working_copy.OptionsAllowConfigReadError,
	)

	tw := tap.NewWriter(os.Stdout)

	store := repo.GetStore()

	for objectWithList, err := range store.GetInventoryListStore().AllInventoryListObjectsAndContents() {
		errors.ContextContinueOrPanic(repo)

		if err == nil {
			tw.Ok(sku.String(objectWithList.List))
			continue
		}

		diag := tap_diagnostics.FromError(err)

		if env_dir.IsErrBlobMissing(err) {
			diag["message"] = "blob missing"
		}

		tw.NotOk(sku.String(objectWithList.List), diag)
	}

	tw.Plan()
}
