package env_workspace

import (
	"sync"

	"code.linenisgreat.com/dodder/go/internal/_/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/lima/store_workspace"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type Store struct {
	store_workspace.Supplies
	store_workspace.StoreLike

	didInit   bool
	onceInit  sync.Once
	initError error
}

func (store *Store) Initialize() (err error) {
	store.onceInit.Do(func() {
		store.initError = store.StoreLike.Initialize(store.Supplies)
		store.didInit = true
	})

	err = store.initError

	return err
}

func (store *Store) Flush() (err error) {
	if !store.didInit {
		return err
	}

	if err = store.StoreLike.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) QueryCheckedOut(
	qg *queries.Query,
	f interfaces.FuncIter[sku.SkuType],
) (err error) {
	es, ok := store.StoreLike.(store_workspace.QueryCheckedOut)

	if !ok {
		err = makeErrUnsupportedOperation(store, &store)
		return err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = es.QueryCheckedOut(
		qg,
		f,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) ReadAllExternalItems() (err error) {
	esado, ok := store.StoreLike.(domain_interfaces.WorkspaceStoreReadAllExternalItems)

	if !ok {
		err = errors.ErrorWithStackf("store does not support %T", &esado)
		return err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = esado.ReadAllExternalItems(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) ReadTransactedFromObjectId(
	o sku.CommitOptions,
	k1 ids.Id,
	t *sku.Transacted,
) (e sku.ExternalLike, err error) {
	es, ok := store.StoreLike.(sku.ExternalStoreReadExternalLikeFromObjectIdLike)

	if !ok {
		err = makeErrUnsupportedOperation(store, &store)
		return e, err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return e, err
	}

	if e, err = es.ReadExternalLikeFromObjectIdLike(
		o,
		k1,
		t,
	); err != nil {
		err = errors.Wrap(err)
		return e, err
	}

	return e, err
}

func (store *Store) ReadExternalLikeFromObjectIdLike(
	commitObjects sku.CommitOptions,
	id interfaces.Stringer,
	objectInput *sku.Transacted,
) (objectOutput sku.ExternalLike, err error) {
	storeLike, ok := store.StoreLike.(sku.ExternalStoreReadExternalLikeFromObjectIdLike)

	if !ok {
		err = makeErrUnsupportedOperation(store, &store)
		return objectOutput, err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return objectOutput, err
	}

	if objectOutput, err = storeLike.ReadExternalLikeFromObjectIdLike(
		commitObjects,
		id,
		objectInput,
	); err != nil {
		err = errors.Wrap(err)
		return objectOutput, err
	}

	return objectOutput, err
}

func (store *Store) CheckoutOne(
	options checkout_options.Options,
	transactedGetter sku.TransactedGetter,
) (checkedOut sku.SkuType, err error) {
	es, ok := store.StoreLike.(store_workspace.CheckoutOne)

	if !ok {
		err = makeErrUnsupportedOperation(store, &store)
		return checkedOut, err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	if checkedOut, err = es.CheckoutOne(
		options,
		transactedGetter,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	return checkedOut, err
}

func (store *Store) DeleteCheckedOut(el *sku.CheckedOut) (err error) {
	es, ok := store.StoreLike.(store_workspace.DeleteCheckedOut)

	if !ok {
		err = makeErrUnsupportedOperation(store, &store)
		return err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = es.DeleteCheckedOut(el); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// Takes a given sku.Transacted (internal) and updates it with the state of its
// checked out counterpart (external).
func (store *Store) UpdateTransacted(z *sku.Transacted) (err error) {
	es, ok := store.StoreLike.(store_workspace.UpdateTransacted)

	if !ok {
		err = makeErrUnsupportedOperation(store, &store)
		return err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = es.UpdateTransacted(z); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) UpdateTransactedFromBlobs(z sku.ExternalLike) (err error) {
	es, ok := store.StoreLike.(store_workspace.UpdateTransactedFromBlobs)

	if !ok {
		err = makeErrUnsupportedOperation(store, &es)
		return err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = es.UpdateTransactedFromBlobs(z); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) GetObjectIdsForString(
	idString string,
) (ids []sku.ExternalObjectId, err error) {
	if store == nil {
		err = errors.MakeErrNotFoundString(idString)
		return ids, err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return ids, err
	}

	if ids, err = store.StoreLike.GetObjectIdsForString(idString); err != nil {
		err = errors.Wrapf(err, "IdString: %q", idString)
		return ids, err
	}

	return ids, err
}

func (store *Store) Open(
	m checkout_mode.Mode,
	ph interfaces.FuncIter[string],
	zsc sku.SkuTypeSet,
) (err error) {
	es, ok := store.StoreLike.(store_workspace.Open)

	if !ok {
		err = makeErrUnsupportedOperation(store, &es)
		return err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = es.Open(m, ph, zsc); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) SaveBlob(el sku.ExternalLike) (err error) {
	es, ok := store.StoreLike.(sku.BlobSaver)

	if !ok {
		err = makeErrUnsupportedOperation(store, &es)
		return err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = es.SaveBlob(el); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) UpdateCheckoutFromCheckedOut(
	options checkout_options.OptionsWithoutMode,
	col sku.SkuType,
) (err error) {
	es, ok := store.StoreLike.(store_workspace.UpdateCheckoutFromCheckedOut)

	if !ok {
		err = makeErrUnsupportedOperation(store, &store)
		return err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = es.UpdateCheckoutFromCheckedOut(options, col); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) ReadCheckedOutFromTransacted(
	object *sku.Transacted,
) (checkedOut *sku.CheckedOut, err error) {
	storeLike, ok := store.StoreLike.(store_workspace.ReadCheckedOutFromTransacted)

	if !ok {
		err = makeErrUnsupportedOperation(store, &store)
		return checkedOut, err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	if checkedOut, err = storeLike.ReadCheckedOutFromTransacted(
		object,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	return checkedOut, err
}

func (store *Store) Merge(
	conflicted sku.Conflicted,
) (err error) {
	storeLike, ok := store.StoreLike.(store_workspace.Merge)

	if !ok {
		err = makeErrUnsupportedOperation(store, &store)
		return err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = storeLike.Merge(
		conflicted,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) MergeCheckedOut(
	co *sku.CheckedOut,
	parentNegotiator sku.ParentNegotiator,
	allowMergeConflicts bool,
) (commitOptions sku.CommitOptions, err error) {
	es, ok := store.StoreLike.(store_workspace.MergeCheckedOut)

	if !ok {
		err = makeErrUnsupportedOperation(store, &store)
		return commitOptions, err
	}

	if err = store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return commitOptions, err
	}

	if commitOptions, err = es.MergeCheckedOut(
		co,
		parentNegotiator,
		allowMergeConflicts,
	); err != nil {
		err = errors.Wrap(err)
		return commitOptions, err
	}

	return commitOptions, err
}
