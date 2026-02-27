package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
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
