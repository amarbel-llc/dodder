package command_components_dodder

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/alfa/pool"
	"code.linenisgreat.com/dodder/go/internal/charlie/files"
	"code.linenisgreat.com/dodder/go/internal/charlie/options_print"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/box_format"
	"code.linenisgreat.com/dodder/go/internal/lima/inventory_list_coders"
)

type InventoryLists struct{}

func (InventoryLists) MakeInventoryListCoderCloset(
	envRepo env_repo.Env,
) inventory_list_coders.Closet {
	boxFormat := box_format.MakeBoxTransactedArchive(
		envRepo,
		options_print.Options{}.WithPrintTai(true),
	)

	return inventory_list_coders.MakeCloset(
		envRepo,
		boxFormat,
	)
}

func (InventoryLists) MakeSeqFromPath(
	ctx interfaces.ActiveContext,
	inventoryListCoderCloset inventory_list_coders.Closet,
	inventoryListPath string,
	afterDecoding func(*sku.Transacted) error,
) interfaces.SeqError[*sku.Transacted] {
	var readCloser io.ReadCloser

	// setup inventory list reader
	{
		var err error

		if readCloser, err = files.Open(
			inventoryListPath,
		); err != nil {
			ctx.Cancel(err)
			return nil
		}
	}

	bufferedReader, repoolBufferedReader := pool.GetBufferedReader(readCloser)

	seq := inventoryListCoderCloset.AllDecodedObjectsFromStream(
		bufferedReader,
		afterDecoding,
	)

	return func(yield func(*sku.Transacted, error) bool) {
		defer errors.ContextMustClose(ctx, readCloser)
		defer repoolBufferedReader()

		for object, err := range seq {
			if !yield(object, err) {
				return
			}
		}
	}
}
