package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

func (local *Repo) MakeInventoryList(
	query *queries.Query,
) (list *sku.HeapTransacted, err error) {
	list = sku.MakeListTransacted()

	if err = local.GetStore().QueryTransacted(
		query,
		quiter.MakeSyncSerializer(
			func(object *sku.Transacted) (err error) {
				cloned, _ := object.CloneTransacted() //repool:owned
				return list.Add(cloned)
			},
		),
	); err != nil {
		err = errors.Wrap(err)
		return list, err
	}

	return list, err
}
