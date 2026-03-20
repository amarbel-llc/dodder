package import_plan

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/zettel_id_index"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func MakeAllocateZettelIdTransform(
	index zettel_id_index.Index,
) ObjectTransform {
	return func(object *sku.Transacted) (bool, error) {
		if !object.GetObjectId().IsEmpty() {
			return true, nil
		}

		zettelId, err := index.CreateZettelId()
		if err != nil {
			return false, errors.Wrap(err)
		}

		if err = object.GetObjectIdMutable().SetWithSeq(
			zettelId.ToSeq(),
		); err != nil {
			return false, errors.Wrap(err)
		}

		ui.Log().Printf("pre-allocated zettel id: %s", object.GetObjectId())

		return true, nil
	}
}
