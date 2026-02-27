package blob_transfers

import (
	"time"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/golf/env_ui"
	"code.linenisgreat.com/dodder/go/internal/india/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

func MakeBlobImporter(
	envRepo env_repo.BlobStoreEnv,
	src blob_stores.BlobStoreInitialized,
	dsts blob_stores.BlobStoreMap,
) BlobImporter {
	return BlobImporter{
		EnvBlobStore: envRepo,
		Src:          src,
		Dsts:         dsts,
	}
}

type BlobImporter struct {
	EnvBlobStore           env_repo.BlobStoreEnv
	CopierDelegate         interfaces.FuncIter[sku.BlobCopyResult]
	Src                    blob_stores.BlobStoreInitialized
	Dsts                   blob_stores.BlobStoreMap
	UseDestinationHashType bool

	Counts Counts
}

type Counts struct {
	Succeeded int
	Ignored   int
	Failed    int
	Total     int
}

func (blobImporter *BlobImporter) ImportBlobIfNecessary(
	blobId domain_interfaces.MarklId,
	object *sku.Transacted,
) (err error) {
	if len(blobImporter.Dsts) == 0 {
		return blobImporter.emitMissingBlob(blobId, object)
	}

	for _, blobStore := range blobImporter.Dsts {
		copyResult := blobImporter.ImportBlobToStoreIfNecessary(
			blobStore,
			blobId,
			object,
		)

		if err = copyResult.GetError(); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (blobImporter *BlobImporter) emitMissingBlob(
	blobId domain_interfaces.MarklId,
	object *sku.Transacted,
) (err error) {
	blobCopyResult := sku.BlobCopyResult{
		ObjectOrNil: object,
		CopyResult: blob_stores.CopyResult{
			BlobId: blobId,
		},
	}

	// when this is a dumb HTTP remote, we expect local to push the missing
	// objects to us after the import call

	blobCopyResult.SetBlobMissingLocally()

	if blobImporter.Src.HasBlob(blobId) {
		blobCopyResult.SetBlobExistsLocally()
	}

	if err = blobImporter.emitCopyResultIfNecessary(blobCopyResult); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (blobImporter *BlobImporter) emitCopyResultIfNecessary(
	copyResult sku.BlobCopyResult,
) (err error) {
	if blobImporter.CopierDelegate == nil {
		return err
	}

	if err = blobImporter.CopierDelegate(copyResult); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (blobImporter *BlobImporter) ImportBlobToStoreIfNecessary(
	dst blob_stores.BlobStoreInitialized,
	blobId domain_interfaces.MarklId,
	object *sku.Transacted,
) (copyResult sku.BlobCopyResult) {
	copyResult.ObjectOrNil = object

	var progressWriter env_ui.ProgressWriter

	if err := errors.RunChildContextWithPrintTicker(
		blobImporter.EnvBlobStore,
		func(ctx errors.Context) {
			blobImporter.Counts.Total++

			var hashType domain_interfaces.FormatHash

			if blobImporter.UseDestinationHashType {
				hashType = dst.GetDefaultHashType()
			}

			copyResult.CopyResult = blob_stores.CopyBlobIfNecessary(
				blobImporter.EnvBlobStore,
				dst.GetBlobStore(),
				blobImporter.Src.GetBlobStore(),
				blobId,
				&progressWriter,
				hashType,
			)

			if copyResult.IsError() {
				blobImporter.Counts.Failed++
				ctx.Cancel(copyResult.GetError())
			} else if copyResult.IsMissing() {
				blobImporter.Counts.Failed++
			} else if copyResult.Exists() {
				blobImporter.Counts.Ignored++
			} else {
				blobImporter.Counts.Succeeded++
			}

			if err := blobImporter.emitCopyResultIfNecessary(
				copyResult,
			); err != nil {
				copyResult.SetError(errors.Wrap(err))
				return
			}
		},
		func(time time.Time) {
			ui.Err().Printf(
				"Copying %s... (%s written)",
				blobId,
				progressWriter.GetWrittenHumanString(),
			)
		},
		3*time.Second,
	); err != nil {
		copyResult.SetError(errors.Wrap(err))
		return copyResult
	}

	return copyResult
}
