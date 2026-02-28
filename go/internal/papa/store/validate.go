package store

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func (store *Store) validateAndFinalize(
	daughter *sku.Transacted,
	mother *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	if err = store.validateIfNecessary(daughter, mother, options); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.finalizer.WriteLockfile(
		daughter,
		options.LockfileOptions,
		store.streamIndex.ReadOneMarklIdAdded,
		store.streamIndex.ReadOneMarklId,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) validateIfNecessary(
	daughter *sku.Transacted,
	mother *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	if !options.Validate {
		return err
	}

	switch daughter.GetSku().GetGenre() {
	case genres.Type:
		tipe := daughter.GetType()

		var repool interfaces.FuncRepool

		if _, repool, _, err = store.GetTypedBlobStore().Type.ParseTypedBlob(
			tipe,
			daughter.GetSku().GetBlobDigest(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer repool()
	}

	return err
}
