package store_fs

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/_/vim_cli_options_builder"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/checkout_mode"
	"code.linenisgreat.com/dodder/go/lib/delta/editor"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

func (store *Store) Open(
	m checkout_mode.Mode,
	ph interfaces.FuncIter[string],
	zsc sku.SkuTypeSet,
) (err error) {
	wg := errors.MakeWaitGroupParallel()

	if m.IncludesMetadata() {
		wg.Do(func() error {
			return store.openZettels(ph, zsc)
		})
	}

	if m.IncludesBlob() {
		wg.Do(func() error {
			return store.openBlob(ph, zsc)
		})
	}

	if err = wg.GetError(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) openZettels(
	ph interfaces.FuncIter[string],
	zsc sku.SkuTypeSet,
) (err error) {
	var filesZettels []string

	if filesZettels, err = store.MakeObjectPathSlice(zsc); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var e editor.Editor

	if e, err = editor.MakeEditorWithVimOptions(
		ph,
		vim_cli_options_builder.New().
			WithCursorLocation(2, 3).
			WithFileType("dodder-object").
			WithInsertMode().
			Build(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = e.Run(filesZettels); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) openBlob(
	ph interfaces.FuncIter[string],
	zsc sku.SkuTypeSet,
) (err error) {
	var filesBlobs []string

	if filesBlobs, err = store.MakeBlobPathSlice(zsc); err != nil {
		err = errors.Wrap(err)
		return err
	}

	opOpenFiles := OpenFiles{}

	if err = opOpenFiles.Run(ph, filesBlobs...); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
