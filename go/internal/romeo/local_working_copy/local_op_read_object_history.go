package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// ReadObjectHistory returns every committed version of the object, oldest and
// newest alike. The objectId probe behind ReadManyObjectId only points at the
// latest version (see stream_index probe_index.go), so it cannot answer "which
// versions do two repos share?" — which is exactly what the parent negotiator
// needs to find a common ancestor for a 3-way merge. Drive ReadPrimitiveQuery
// instead and collect the versions whose ObjectId matches.
//
// The query is intentionally ignorant of hidden / dormant state:
// MakePrimitiveQueryGroup carries both SigilHistory and SigilHidden, and
// ReadPrimitiveQuery reads the stream directly without consulting the dormant
// index. Ancestry must see the complete chain — a dormant version in the middle
// of the history must not break the common-ancestor search — or the merge
// manufactures false conflicts (#298).
//
// This scans the persisted stream; it is invoked during remote transfers when
// the receiving repo already holds the object, not on a hot path.
func (local *Repo) ReadObjectHistory(
	objectId *ids.ObjectId,
) (objects []*sku.Transacted, err error) {
	streamIndex := local.GetStore().GetStreamIndex()

	target := objectId.String()

	if err = streamIndex.ReadPrimitiveQuery(
		sku.MakePrimitiveQueryGroup(),
		func(object *sku.Transacted) (err error) {
			if object.GetObjectId().String() != target {
				return err
			}

			clone, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned
			sku.TransactedResetter.ResetWith(clone, object)
			objects = append(objects, clone)

			return err
		},
	); err != nil {
		err = errors.Wrap(err)
		return objects, err
	}

	return objects, err
}
