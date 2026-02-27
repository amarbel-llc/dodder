package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/oscar/queries"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
)

func (local *Repo) MakeInventoryList(
	query *queries.Query,
) (list *sku.HeapTransacted, err error) {
	list = sku.MakeListTransacted()

	if err = local.GetStore().QueryTransacted(
		query,
		quiter.MakeSyncSerializer(
			func(object *sku.Transacted) (err error) {
				cloned, _ := object.CloneTransacted()
				return list.Add(cloned)
			},
		),
	); err != nil {
		err = errors.Wrap(err)
		return list, err
	}

	return list, err
}
