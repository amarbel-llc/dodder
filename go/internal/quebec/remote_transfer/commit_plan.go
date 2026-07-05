package remote_transfer

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/internal/oscar/env_box"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	mad_blob_io "github.com/amarbel-llc/madder/go/pkgs/blob_io"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func CommitPlan(
	ctx interfaces.ActiveContext,
	local repo.LocalRepo,
	envBox env_box.Env,
	importerInterface repo.Importer,
	plan *import_plan.Plan,
) (err error) {
	imp, ok := importerInterface.(*importer)
	if !ok {
		return errors.Errorf("CommitPlan requires remote_transfer importer")
	}

	return imp.commitPlan(ctx, local, envBox, plan)
}

func (imp importer) commitPlan(
	ctx interfaces.ActiveContext,
	local repo.LocalRepo,
	envBox env_box.Env,
	plan *import_plan.Plan,
) (err error) {
	ctx.Must(errors.MakeFuncContextFromFuncErr(local.Lock))

	importErrors := errors.MakeGroupBuilder()
	checkedOutPrinter := imp.GetCheckedOutPrinter()

	var errorLog *importErrorLog

	if imp.continueOnError {
		errorLog = &importErrorLog{}
	}

	configGenesis := imp.envRepo.GetConfigPrivate().Blob

	for i := range plan.Entries {
		entry := &plan.Entries[i]

		if !entry.Classification.IsCommittable() {
			continue
		}

		object := entry.GetObject()

		if genres.Make(object.GetGenre()) == genres.InventoryList {
			if _, importErr := imp.Import(object); importErr != nil {
				if imp.skipBloblessType(importErr) {
					continue
				}

				if errors.Is(importErr, ErrSkipped) ||
					errors.Is(importErr, errors.ErrExists) ||
					genres.IsErrUnsupportedGenre(importErr) ||
					IsErrDeduped(importErr) ||
					mad_blob_io.IsErrBlobMissing(importErr) {
					continue
				}

				imp.handleCommitPlanError(
					importErr, object, importErrors, errorLog,
				)
			}

			continue
		}

		if imp.committer.options.OverwriteSignatures {
			if objectErr := imp.commitPlanEntryOverwrite(
				object,
				configGenesis,
			); objectErr != nil {
				imp.handleCommitPlanError(
					objectErr, object, importErrors, errorLog,
				)
				continue
			}
		} else {
			if objectErr := imp.commitPlanEntry(
				object,
				configGenesis,
			); objectErr != nil {
				imp.handleCommitPlanError(
					objectErr, object, importErrors, errorLog,
				)
				continue
			}
		}

		if commitErr := imp.importNewObject(object); commitErr != nil {
			if errors.Is(commitErr, errors.ErrExists) {
				continue
			}

			imp.handleCommitPlanError(
				commitErr, object, importErrors, errorLog,
			)
			continue
		}

		checkedOut, _ := sku.GetCheckedOutPool().GetWithRepool() //repool:owned
		sku.Resetter.ResetWith(checkedOut.GetSkuExternal(), object)

		if printErr := checkedOutPrinter(checkedOut); printErr != nil {
			importErrors.Add(errors.Wrap(printErr))
		}
	}

	if dedupCount := plan.CountByClassification()[import_plan.ClassificationSkipDedup]; dedupCount > 0 {
		ui.Err().Printf("%d objects deduped during import\n", dedupCount)
	}

	if errorLog != nil {
		if closeErr := errorLog.Close(); closeErr != nil {
			ui.Err().Printf("failed to close error log: %s", closeErr)
		}

		if errorLog.Count() > 0 {
			importErrors.Add(errors.WithHelp(
				fmt.Errorf("%d objects failed to import", errorLog.Count()),
				[]string{"One or more objects encountered errors during import"},
				[]string{
					fmt.Sprintf(
						"Review error log: %s",
						errorLog.Path(),
					),
				},
			))
		}
	}

	if importErrors.Len() > 0 {
		err = importErrors.GetError()
	}

	ctx.Must(errors.MakeFuncContextFromFuncErr(local.Unlock))

	return err
}

func (imp importer) commitPlanEntry(
	object *sku.Transacted,
	configGenesis genesis_configs.ConfigPrivate,
) (err error) {
	if err = imp.ImportBlobIfNecessary(object); err != nil {
		err = errors.Wrap(err)
		return err
	}

	object.GetMetadataMutable().GetObjectDigestMutable().Reset()

	if object.GetMetadata().GetObjectSig().IsNull() {
		if err = imp.finalizer.FinalizeAndSignOverwrite(
			object,
			configGenesis,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		// recomputes the digest of an imported object that already
		// carries a sig: derive the digest purpose from that sig so
		// v2-signed imports into a v3 repo (and vice versa) verify
		if err = imp.finalizer.FinalizeUsingObject(
			object,
			object.ObjectDigestPurposeOrDefault(
				imp.envRepo.GetObjectDigestType(),
			),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (imp importer) commitPlanEntryOverwrite(
	object *sku.Transacted,
	configGenesis genesis_configs.ConfigPrivate,
) (err error) {
	metadata := object.GetMetadataMutable()
	metadata.GetObjectDigestMutable().Reset()
	metadata.GetObjectSigMutable().Reset()
	metadata.GetRepoPubKeyMutable().Reset()

	// Reset lock values so WriteLockfileIfNecessary repopulates them from the
	// store, where types/tags/refs were already re-signed earlier in
	// topographic order.
	metadata.GetTypeLockMutable().GetValueMutable().Reset()

	for tag := range metadata.GetTags().All() {
		metadata.GetTagLockMutable(tag).GetValueMutable().Reset()
	}

	for ref := range metadata.AllReferencedObjects() {
		metadata.GetReferencedObjectLockMutable(ref).GetValueMutable().Reset()
	}

	if err = imp.blobImporter.ImportBlobIfNecessary(
		object.GetMetadata().GetBlobDigest(),
		object,
	); err != nil {
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
		} else if mad_blob_io.IsErrBlobAlreadyExists(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = imp.finalizer.FinalizeAndSignOverwrite(
		object,
		configGenesis,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (imp importer) handleCommitPlanError(
	objectErr error,
	object *sku.Transacted,
	importErrors *errors.GroupBuilder,
	errorLog *importErrorLog,
) {
	wrappedErr := errors.Wrapf(objectErr, "Object: %s", sku.String(object))

	if imp.continueOnError {
		ui.Err().Print(wrappedErr)

		if errorLog != nil {
			if logErr := errorLog.LogError(object, objectErr); logErr != nil {
				ui.Err().Printf("failed to write error log: %s", logErr)
			}
		}
	} else {
		importErrors.Add(wrappedErr)
	}
}
