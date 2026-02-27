package store

import (
	"slices"
	"sync"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/quiter"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/november/queries"
)

func (store *Store) All(
	query *queries.Query,
) interfaces.SeqError[*sku.Transacted] {
	return func(yield func(*sku.Transacted, error) bool) {
		store.QueryTransacted(
			query,
			quiter.MakeSyncSerializer(
				func(object *sku.Transacted) (err error) {
					if !yield(object, nil) {
						err = errors.MakeErrStopIteration()
						return err
					}

					return err
				},
			),
		)
	}
}

// TODO make iterator
func (store *Store) QueryPrimitive(
	group sku.PrimitiveQueryGroup,
	funcIter interfaces.FuncIter[*sku.Transacted],
) (err error) {
	executor := queries.MakeExecutorPrimitive(
		group,
		store.GetStreamIndex().ReadPrimitiveQuery,
		store.ReadOneInto,
	)

	if err = executor.ExecuteTransacted(funcIter); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// TODO make iterator
func (store *Store) QueryTransacted(
	group *queries.Query,
	funcIter interfaces.FuncIter[*sku.Transacted],
) (err error) {
	var executor queries.Executor

	if executor, err = store.makeQueryExecutor(group); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var object *sku.Transacted

	switch {
	case true:
		// TODO why does this not work with trying to read internal
		if object, err = executor.ExecuteExactlyOneExternalObject(false); err != nil {
			err = nil
			break
		}

		if err = funcIter(object); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}

	if err = executor.ExecuteTransacted(funcIter); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// TODO make iterator
func (store *Store) QueryTransactedAsSkuType(
	query *queries.Query,
	funcIter interfaces.FuncIter[sku.SkuType],
) (err error) {
	var executor queries.Executor

	if executor, err = store.makeQueryExecutor(query); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = executor.ExecuteTransactedAsSkuType(funcIter); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// TODO make iterator
func (store *Store) QuerySkuType(
	query *queries.Query,
	output interfaces.FuncIter[sku.SkuType],
) (err error) {
	var executor queries.Executor

	if executor, err = store.makeQueryExecutor(query); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = executor.ExecuteSkuType(output); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) QueryExactlyOneExternal(
	query *queries.Query,
) (object *sku.Transacted, err error) {
	var executor queries.Executor

	if executor, err = store.makeQueryExecutor(query); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	if object, err = executor.ExecuteExactlyOneExternalObject(true); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	return object, err
}

func (store *Store) QueryExactlyOne(
	queryGroup *queries.Query,
) (object *sku.Transacted, err error) {
	var executor queries.Executor

	if executor, err = store.makeQueryExecutor(queryGroup); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	if object, err = executor.ExecuteExactlyOne(); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	return object, err
}

func (store *Store) MakeBlobDigestObjectIdsMap() (blobDigestObjectIds map[string][]string, err error) {
	blobDigestObjectIds = make(map[string][]string)
	var lock sync.Mutex

	if err = store.QueryPrimitive(
		sku.MakePrimitiveQueryGroup(),
		func(object *sku.Transacted) (err error) {
			lock.Lock()
			defer lock.Unlock()

			digestBytes := object.GetMetadata().GetBlobDigest().GetBytes()
			objectIds := blobDigestObjectIds[string(digestBytes)]
			oid := object.ObjectId.String()
			loc, found := slices.BinarySearch(objectIds, oid)

			if found {
				return err
			}

			objectIds = slices.Insert(objectIds, loc, oid)

			bites := object.GetMetadata().GetBlobDigest().GetBytes()
			blobDigestObjectIds[string(bites)] = objectIds

			return err
		},
	); err != nil {
		err = errors.Wrap(err)
		return blobDigestObjectIds, err
	}

	return blobDigestObjectIds, err
}
