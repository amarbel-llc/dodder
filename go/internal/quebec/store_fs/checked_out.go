package store_fs

import (
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
)

func (store *Store) MakeObjectPathSlice(
	objects sku.SkuTypeSet,
) (out []string, err error) {
	return quiter.DerivedValues(
		objects,
		func(col sku.SkuType) (e string, err error) {
			var fds *sku.FSItem

			if fds, err = store.ReadFSItemFromExternal(col.GetSkuExternal()); err != nil {
				err = errors.Wrap(err)
				return e, err
			}

			e = fds.Object.GetPath()

			if e == "" {
				err = errors.MakeErrStopIteration()
				return e, err
			}

			return e, err
		},
	)
}

func (store *Store) MakeBlobPathSlice(
	objects sku.SkuTypeSet,
) (out []string, err error) {
	return quiter.DerivedValues(
		objects,
		func(col sku.SkuType) (e string, err error) {
			var fds *sku.FSItem

			if fds, err = store.ReadFSItemFromExternal(col.GetSkuExternal()); err != nil {
				err = errors.Wrap(err)
				return e, err
			}

			e = fds.Blob.GetPath()

			if e == "" {
				err = errors.MakeErrStopIteration()
				return e, err
			}

			return e, err
		},
	)
}
