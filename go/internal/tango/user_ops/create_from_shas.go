package user_ops

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

type CreateFromShas struct {
	*local_working_copy.Repo
	sku.Proto
}

func (op CreateFromShas) Run(
	args ...string,
) (results sku.TransactedMutableSet, err error) {
	var lookupStored map[string][]string

	if lookupStored, err = op.GetStore().MakeBlobDigestObjectIdsMap(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	toCreate := make(map[string]*sku.Transacted)

	for _, arg := range args {
		var digest markl.Id

		if err = markl.SetMaybeSha256(
			&digest,
			arg,
		); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		digestBytes := digest.GetBytes()

		if _, ok := toCreate[string(digestBytes)]; ok {
			ui.Err().Printf(
				"%s appears in arguments more than once. Ignoring",
				&digest,
			)
			continue
		}

		if oids, ok := lookupStored[string(digestBytes)]; ok {
			ui.Err().Printf(
				"%s appears in object already checked in (%q). Ignoring",
				&digest,
				oids,
			)
			continue
		}

		object, _ := sku.GetTransactedPool().GetWithRepool()

		object.GetObjectIdMutable().SetGenre(genres.Zettel)
		object.GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(&digest)

		op.Proto.Apply(object, genres.Zettel)

		toCreate[string(digestBytes)] = object
	}

	results = sku.MakeTransactedMutableSet()

	if err = op.Lock(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	for _, object := range toCreate {
		if err = op.GetStore().CreateOrUpdateDefaultProto(
			object,
			sku.StoreOptions{
				ApplyProto: true,
			},
		); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		results.Add(object)
	}

	if err = op.Unlock(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	return results, err
}
