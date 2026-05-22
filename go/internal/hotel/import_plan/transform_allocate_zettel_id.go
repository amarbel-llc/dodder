package import_plan

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/zettel_id_index"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
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
