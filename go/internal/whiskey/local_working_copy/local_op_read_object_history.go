package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func (local *Repo) ReadObjectHistory(
	objectId *ids.ObjectId,
) (objects []*sku.Transacted, err error) {
	streamIndex := local.GetStore().GetStreamIndex()

	if objects, err = streamIndex.ReadManyObjectId(
		objectId,
	); err != nil {
		err = errors.Wrap(err)
		return objects, err
	}

	return objects, err
}
