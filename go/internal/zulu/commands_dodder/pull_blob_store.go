package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/india/env_dir"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/lima/command_components_madder"
	"code.linenisgreat.com/dodder/go/internal/oscar/queries"
	"code.linenisgreat.com/dodder/go/internal/uniform/repo"
	"code.linenisgreat.com/dodder/go/internal/yankee/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("pull-blob-store", &PullBlobStore{})
}

type PullBlobStore struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup
	command_components_madder.BlobStore
}

var _ interfaces.CommandComponentWriter = (*PullBlobStore)(nil)

func (cmd *PullBlobStore) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	cmd.LocalWorkingCopyWithQueryGroup.SetFlagDefinitions(f)
}

func (cmd *PullBlobStore) Run(
	req command.Request,
) {
	blobStoreBasePath := req.PopArg("blob_store-base-path")
	blobStoreConfigPath := req.PopArg("blob_store-config-path")

	localWorkingCopy, queryGroup := cmd.MakeLocalWorkingCopyAndQueryGroup(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.InventoryList),
		),
	)

	importerOptions := repo.ImporterOptions{
		ExcludeObjects: true,
		PrintCopies:    true,
	}

	importerOptions.RemoteBlobStore = cmd.MakeBlobStoreFromIdOrConfigPath(
		localWorkingCopy.GetEnvRepo().GetEnvBlobStore(),
		blobStoreBasePath,
		blobStoreConfigPath,
	)

	importer := localWorkingCopy.MakeImporter(
		importerOptions,
		sku.GetStoreOptionsRemoteTransfer(),
	)

	if err := localWorkingCopy.GetStore().QueryTransacted(
		queryGroup,
		func(object *sku.Transacted) (err error) {
			if err = importer.ImportBlobIfNecessary(object); err != nil {
				if env_dir.IsErrBlobMissing(err) {
					err = nil
					localWorkingCopy.GetUI().Printf("Blob missing from remote: %q", object.GetBlobDigest())
				} else {
					err = errors.Wrap(err)
				}

				return err
			}

			return err
		},
	); err != nil {
		req.Cancel(err)
	}
}
