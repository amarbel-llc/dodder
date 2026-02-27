package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/internal/india/env_dir"
	"code.linenisgreat.com/dodder/go/internal/juliett/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/lima/blob_transfers"
	"code.linenisgreat.com/dodder/go/internal/lima/command_components_madder"
	"code.linenisgreat.com/dodder/go/internal/uniform/repo"
	"code.linenisgreat.com/dodder/go/internal/victor/remote_transfer"
	"code.linenisgreat.com/dodder/go/internal/yankee/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("import", &Import{})
}

type Import struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.InventoryLists
	command_components_madder.BlobStore
	command_components_madder.Complete

	repo.ImporterOptions

	Proto sku.Proto

	BlobStoreId blob_store_id.Id
}

var _ interfaces.CommandComponentWriter = (*Import)(nil)

func (cmd *Import) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	cmd.ImporterOptions.SetFlagDefinitions(flagDefinitions)
	cmd.Proto.SetFlagDefinitions(flagDefinitions)

	flagDefinitions.Var(
		cmd.Complete.GetFlagValueBlobIds(&cmd.BlobStoreId),
		"blob_store-id",
		"The name of the existing madder blob store to use",
	)
}

func (cmd Import) Run(req command.Request) {
	inventoryListPath := req.PopArg("inventory_list-path")

	local := cmd.MakeLocalWorkingCopy(req)

	if inventoryListPath == "" {
		errors.ContextCancelWithBadRequestf(req, "empty inventory list")
	}

	cmd.DedupingFormatId = markl.PurposeV5MetadataDigestWithoutTai
	cmd.CheckedOutPrinter = local.PrinterCheckedOutConflictsForRemoteTransfers()

	if !cmd.BlobStoreId.IsEmpty() {
		cmd.RemoteBlobStore = local.GetEnvRepo().GetEnvBlobStore().GetBlobStore(
			cmd.BlobStoreId,
		)
	}

	req.AssertNoMoreArgs()

	var afterDecoding func(*sku.Transacted) error

	blobImporter := blob_transfers.MakeBlobImporter(
		local.GetEnvRepo().GetEnvBlobStore(),
		cmd.RemoteBlobStore,
		blob_stores.MakeBlobStoreMap(local.GetBlobStore()),
	)

	blobImporter.UseDestinationHashType = true

	blobImporter.CopierDelegate = sku.MakeBlobCopierDelegate(
		local.GetUI(),
		false,
	)

	finalizer := local.GetObjectFinalizer()

	// TODO traverse object graph and rewrite all signature in topological order
	// TODO move this to the importer directly
	if cmd.OverwriteSignatures {
		afterDecoding = func(object *sku.Transacted) (err error) {
			object.GetMetadataMutable().GetObjectDigestMutable().Reset()
			object.GetMetadataMutable().GetObjectSigMutable().Reset()
			object.GetMetadataMutable().GetRepoPubKeyMutable().Reset()

			if err = blobImporter.ImportBlobIfNecessary(
				object.GetMetadata().GetBlobDigest(),
				object,
			); err != nil { // TODO rewrite blob
				var errNotEqual markl.ErrNotEqual

				if errors.As(err, &errNotEqual) {
					if errNotEqual.IsDifferentHashTypes() {
						err = nil
						object.GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(
							errNotEqual.Actual,
						)
					} else {
						err = errors.Wrap(err)
						return err
					}
				} else if env_dir.IsErrBlobAlreadyExists(err) {
					err = nil
				} else {
					err = errors.Wrap(err)
					return err
				}
			}

			// TODO add mother?
			// TODO rewrite time?
			if err = finalizer.FinalizeAndSignOverwrite(
				object,
				local.GetEnvRepo().GetConfigPrivate().Blob,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		}
	}

	seq := cmd.MakeSeqFromPath(
		local,
		local.GetInventoryListCoderCloset(),
		inventoryListPath,
		afterDecoding,
	)

	importer := local.MakeImporter(
		cmd.ImporterOptions,
		sku.GetStoreOptionsImport(),
	)

	if err := local.ImportSeq(
		seq,
		importer,
	); err != nil {
		if !errors.Is(err, remote_transfer.ErrNeedsMerge) {
			err = errors.Wrap(err)
		}

		local.Cancel(err)
	}
}
