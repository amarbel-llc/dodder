package store

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

func (store *Store) ReadTransactedFromObjectId(
	objectId domain_interfaces.ObjectId,
) (object *sku.Transacted, err error) {
	var objectRepool interfaces.FuncRepool
	object, objectRepool = sku.GetTransactedPool().GetWithRepool()

	if err = store.ReadOneInto(objectId, object); err != nil {
		if errors.IsErrNotFound(err) {
			objectRepool()
			object = nil
		}

		err = errors.Wrap(err)
		return object, err
	}

	return object, err
}

func (store *Store) ReadObjectTypeAndLockIfNecessary(
	object *sku.Transacted,
) (typeObject *sku.Transacted, err error) {
	typeLock := object.GetMetadataMutable().GetTypeLockMutable()
	typeMarklId := typeLock.GetValue()

	if ids.IsBuiltin(typeLock.GetKey()) {
		err = errors.MakeErrNotFound(typeLock.GetKey())
		return typeObject, err
	}

	if !typeMarklId.IsNull() {
		return store.ReadTypeObject(typeLock)
	}

	if typeObject, err = store.ReadOneObjectId(object.GetType()); err != nil {
		err = errors.Wrap(err)
		return typeObject, err
	}

	if typeObject != nil {
		typeLock.GetValueMutable().ResetWithMarklId(typeObject.GetMetadata().GetObjectSig())
	}

	return typeObject, err
}

func (store *Store) ReadTypeObject(
	typeLock objects.TypeLock,
) (typeObject *sku.Transacted, err error) {
	if ids.IsBuiltin(typeLock.GetKey()) {
		err = errors.MakeErrNotFound(typeLock.GetKey())
		return typeObject, err
	}

	if typeLock.GetValue().IsNull() {
		panic(fmt.Sprintf("empty type lock for type: %q", typeLock.GetKey()))
	}

	var typeObjectRepool interfaces.FuncRepool
	typeObject, typeObjectRepool = sku.GetTransactedPool().GetWithRepool()

	if !store.streamIndex.ReadOneMarklId(
		typeLock.GetValue(),
		typeObject,
	) {
		typeObjectRepool()
		typeObject = nil

		err = errors.MakeErrNotFound(typeLock.GetKey())
		return typeObject, err
	}

	return typeObject, err
}

// TODO transition to a context-based panic / cancel semantic
func (store *Store) ReadOneObjectId(
	objectId domain_interfaces.ObjectId,
) (object *sku.Transacted, err error) {
	if objectId.IsEmpty() {
		return object, err
	}

	object, _ = sku.GetTransactedPool().GetWithRepool()

	if err = store.streamIndex.ReadOneObjectId(objectId, object); err != nil {
		if !errors.IsErrNotFound(err) {
			err = errors.Wrap(err)
		}

		return object, err
	}

	return object, err
}

// TODO add support for cwd and sigil
// TODO simplify
func (store *Store) ReadOneInto(
	objectId domain_interfaces.ObjectId,
	out *sku.Transacted,
) (err error) {
	var object *sku.Transacted

	switch objectId.GetGenre() {
	case genres.Zettel:
		var zettelId *ids.ZettelId

		if zettelId, err = store.GetAbbrStore().GetZettelIds().ExpandString(
			objectId.String(),
		); err == nil {
			objectId = zettelId
		} else {
			err = nil
		}

		if object, err = store.ReadOneObjectId(objectId); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case genres.Type, genres.Tag, genres.Repo, genres.InventoryList:
		if object, err = store.ReadOneObjectId(objectId); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case genres.Config:
		object = store.GetConfigStore().GetConfig().GetSku()

		if object.GetTai().IsEmpty() {
			ui.Err().Print("config tai is empty")
		}

	case genres.Blob:
		// var oid ids.ObjectId

		// if err = oid.SetWithIdLike(objectId); err != nil {
		// 	err = collections.MakeErrNotFound(objectId)
		// 	return err
		// }

		if object, err = store.ReadOneObjectId(objectId); err != nil {
			err = errors.Wrap(err)
			return err
		}

	default:
		err = genres.MakeErrUnsupportedGenre(objectId)
		return err
	}

	if object == nil {
		err = errors.MakeErrNotFound(objectId)
		return err
	}

	sku.TransactedResetter.ResetWith(out, object)

	return err
}

func (store *Store) ReadPrimitiveQuery(
	query sku.PrimitiveQueryGroup,
	funcIter interfaces.FuncIter[*sku.Transacted],
) (err error) {
	return store.streamIndex.ReadPrimitiveQuery(query, funcIter)
}
