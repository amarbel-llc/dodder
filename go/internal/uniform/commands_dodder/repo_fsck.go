package commands_dodder

import (
	"os"

	"code.linenisgreat.com/dodder/go/internal/0/tap_diagnostics"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	mad_blob_io "code.linenisgreat.com/madder/go/pkgs/blob_io"
	env_local "code.linenisgreat.com/madder/go/pkgs/env_local"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	tap "github.com/amarbel-llc/tap/go/pkgs/writer"
)

func init() {
	utility.AddCmd(
		"repo-fsck",
		&RepoFsck{},
	)
}

func (cmd RepoFsck) GetDescription() command.Description {
	return command.Description{
		Short: "verify repository inventory list integrity",
	}
}

type RepoFsck struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.EnvRepo
}

var _ command.CommandWithArgs = (*RepoFsck)(nil)

// GetArgs returns nil: no positional arguments.
func (cmd *RepoFsck) GetArgs() []command.ArgGroup { return nil }

func (cmd RepoFsck) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	envRepo := cmd.MakeEnvRepo(req, false)

	for id, blobStore := range envRepo.GetEnvBlobStore().GetBlobStores() {
		envLocal.GetOut().Printf("%s\t%s", id, blobStore.GetBlobStoreDescription())
	}
}

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

		if mad_blob_io.IsErrBlobMissing(err) {
			diag["message"] = "blob missing"
		}

		tw.NotOk(sku.String(objectWithList.List), diag)
	}

	tw.Plan()
}
