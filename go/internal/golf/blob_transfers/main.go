package blob_transfers

import (
	"time"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/madder/go/pkgs/blob_stores"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func MakeBlobImporter(
	envRepo env_repo.BlobStoreEnv,
	src mad_domain_interfaces.BlobStore,
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
	Src                    mad_domain_interfaces.BlobStore
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

// ImportObjectBlobClosure copies an object's full blob closure into the
// importer's destination store(s): the object's own blob (when non-null) plus
// every field-level blob reference. Content-addressed, so blobs already present
// are skipped and re-runs are cheap. When tolerateMissing is set, a blob absent
// from the source is skipped rather than erroring (for staged, intentionally
// incomplete migration passes).
//
// This is the transform pipeline's self-containment copy (dodder#392): every
// referenced blob is duplicated into the target so it survives deleting the
// source. remote_transfer's receive path has a parallel but policy-divergent
// copy of the same closure (references gated on a remote store being present,
// and only not-found errors skipped — dodder#325) that could migrate onto a
// generalized form of this method later (dodder#394).
func (blobImporter *BlobImporter) ImportObjectBlobClosure(
	object *sku.Transacted,
	tolerateMissing bool,
) (err error) {
	metadata := object.GetMetadata()

	copyOne := func(blobId mad_domain_interfaces.MarklId) error {
		if err := blobImporter.ImportBlobIfNecessary(blobId, object); err != nil {
			if tolerateMissing {
				return nil
			}

			return errors.Wrapf(err, "copying referenced blob %s", blobId)
		}

		return nil
	}

	if blobDigest := metadata.GetBlobDigest(); !blobDigest.IsNull() {
		if err = copyOne(blobDigest); err != nil {
			return err
		}
	}

	for refDigest := range metadata.AllBlobReferences() {
		if err = copyOne(refDigest); err != nil {
			return err
		}
	}

	return err
}

func (blobImporter *BlobImporter) ImportBlobIfNecessary(
	blobId mad_domain_interfaces.MarklId,
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
	blobId mad_domain_interfaces.MarklId,
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
	blobId mad_domain_interfaces.MarklId,
	object *sku.Transacted,
) (copyResult sku.BlobCopyResult) {
	copyResult.ObjectOrNil = object

	var progressWriter env_ui.ProgressWriter

	if err := errors.RunChildContextWithPrintTicker(
		blobImporter.EnvBlobStore,
		func(ctx errors.Context) {
			blobImporter.Counts.Total++

			var hashType mad_domain_interfaces.FormatHash

			if blobImporter.UseDestinationHashType {
				hashType = dst.GetDefaultHashType()
			}

			copyResult.CopyResult = blob_stores.CopyBlobIfNecessary(
				blobImporter.EnvBlobStore,
				dst.GetBlobStore(),
				blobImporter.Src,
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
