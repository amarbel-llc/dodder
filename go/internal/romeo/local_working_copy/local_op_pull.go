package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/oscar/store"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/quebec/remote_transfer"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"github.com/amarbel-llc/madder/go/pkgs/blob_stores"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func (local *Repo) PullQueryGroupFromRemote(
	remote repo.Repo,
	qg *queries.Query,
	options repo.ImporterOptions,
) (err error) {
	if err = local.pullQueryGroupFromWorkingCopy(
		remote,
		qg,
		options,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (local *Repo) pullQueryGroupFromWorkingCopy(
	remote repo.Repo,
	queryGroup *queries.Query,
	importerOptions repo.ImporterOptions,
) (err error) {
	var list *sku.HeapTransacted

	if list, err = remote.MakeInventoryList(queryGroup); err != nil {
		err = errors.Wrap(err)
		return err
	}

	explorer := store.MakeEdgeExplorer(
		remote.GetObjectStore(),
		remote.GetBlobStore(),
	)

	edges, err := expandEdges(list, remote.GetObjectStore(), explorer)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	if len(edges.Skipped) > 0 {
		return errors.Errorf("edge traversal had %d failures: %s",
			len(edges.Skipped), edges.Skipped[0])
	}

	importerOptions.CheckedOutPrinter = local.PrinterCheckedOutConflictsForRemoteTransfers()

	if !importerOptions.ExcludeBlobs {
		remoteBlobStore := remote.GetBlobStore()
		importerOptions.RemoteBlobStore = remoteBlobStore
	}

	importerOptions.ParentNegotiator = ParentNegotiatorFirstAncestor{
		Local:  local,
		Remote: remote,
	}

	importer := local.MakeImporter(
		importerOptions,
		sku.GetStoreOptionsImport(),
	)

	if err = local.ImportSeq(
		quiter.MakeSeqErrorFromSeq(list.All()),
		importer,
	); err != nil {
		if errors.Is(err, remote_transfer.ErrNeedsMerge) {
			err = errors.WithoutStack(err)
		} else {
			err = errors.Wrap(err)
		}

		return err
	}

	if !importerOptions.ExcludeBlobs && len(edges.Blobs) > 0 {
		remoteBlobStore := remote.GetBlobStore()
		localBlobStore := local.GetEnvRepo().GetDefaultBlobStore()

		for _, blobDigest := range edges.Blobs {
			copyResult := blob_stores.CopyBlobIfNecessary(
				local.GetEnv(),
				localBlobStore,
				remoteBlobStore,
				blobDigest,
				nil,
				localBlobStore.GetDefaultHashType(),
			)

			if copyErr := copyResult.GetError(); copyErr != nil {
				if errors.IsErrNotFound(copyErr) {
					continue
				}

				return errors.Wrapf(copyErr, "copying additional blob %s", blobDigest.String())
			}
		}
	}

	return err
}
