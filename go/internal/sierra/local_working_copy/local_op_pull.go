package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/quebec/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/remote_transfer"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
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

	return err
}
