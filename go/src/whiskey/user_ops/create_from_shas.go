package user_ops

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
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

		object.ObjectId.SetGenre(genres.Zettel)
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
