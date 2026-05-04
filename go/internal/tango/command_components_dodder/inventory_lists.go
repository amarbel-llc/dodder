package command_components_dodder

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/box_format"
	"code.linenisgreat.com/dodder/go/internal/hotel/inventory_list_coders"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/alfa/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/files"
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
