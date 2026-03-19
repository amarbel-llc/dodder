package user_ops

import (
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

type WriteNewZettels struct {
	*local_working_copy.Repo
}

func (op WriteNewZettels) RunMany(
	proto sku.Proto,
	count int,
) (results sku.TransactedMutableSet, err error) {
	zettelIdIndex := op.GetStore().GetZettelIdIndex()

	// Phase 1: allocate zettel IDs and prepare objects before acquiring lock
	planned := make([]*sku.Transacted, 0, count)

	for range count {
		object, _ := proto.Make() //repool:owned

		zettelId, idErr := zettelIdIndex.CreateZettelId()
		if idErr != nil {
			err = errors.Wrap(idErr)
			return results, err
		}

		if err = object.GetObjectIdMutable().SetWithSeq(zettelId.ToSeq()); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		ui.Log().Printf("pre-allocated zettel id: %s", object.GetObjectId())

		planned = append(planned, object)
	}

	// Phase 2: commit all planned objects under lock
	if err = op.Lock(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	results = sku.MakeTransactedMutableSet()

	for _, object := range planned {
		if err = op.GetStore().CreateOrUpdateDefaultProto(
			object,
			sku.StoreOptions{
				ApplyProto: true,
			},
		); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		if err = results.Add(object); err != nil {
			err = errors.Wrap(err)
			return results, err
		}
	}

	if err = op.Unlock(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	return results, err
}

func (op WriteNewZettels) RunOne(
	z sku.Proto,
) (result *sku.Transacted, err error) {
	results, err := op.RunMany(z, 1)
	if err != nil {
		return result, err
	}

	for t := range results.All() {
		result = t
		break
	}

	return result, err
}

