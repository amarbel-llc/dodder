package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	mad_blob_io "github.com/amarbel-llc/madder/go/pkgs/blob_io"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd("pull-blob-store", &PullBlobStore{})
}

func (cmd PullBlobStore) GetDescription() command.Description {
	return command.Description{
		Short: "pull blobs from a remote blob store",
	}
}

type PullBlobStore struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup
	command_components_dodder.BlobStore
}

var (
	_ interfaces.CommandComponentWriter = (*PullBlobStore)(nil)
	_ command.CommandWithArgs           = (*PullBlobStore)(nil)
)

func (cmd *PullBlobStore) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{
		{Args: []command.Arg{
			{
				Name:        "blob_store-base-path",
				Description: "base path of the remote blob store",
				Required:    true,
			},
			{
				Name:        "blob_store-config-path",
				Description: "path to the remote blob store config",
				Required:    true,
			},
		}},
		cmd.Query.GetArgGroup(),
	}
}

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
				if mad_blob_io.IsErrBlobMissing(err) {
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
