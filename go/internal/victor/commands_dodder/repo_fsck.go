package commands_dodder

import (
	"os"

	"code.linenisgreat.com/dodder/go/internal/charlie/tap_diagnostics"
	"code.linenisgreat.com/dodder/go/internal/delta/env_ui"
	"code.linenisgreat.com/dodder/go/internal/echo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components_madder"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	tap "github.com/amarbel-llc/purse-first/packages/tap-dancer/go"
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
