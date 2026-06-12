package store

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func (store *Store) ReadTransactedFromObjectId(
	objectId domain_interfaces.ObjectId,
) (object *sku.Transacted, err error) {
	var objectRepool interfaces.FuncRepool
	object, objectRepool = sku.GetTransactedPool().GetWithRepool()

	if err = store.ReadOneInto(objectId, object); err != nil {
		objectRepool()
		object = nil

		err = errors.Wrap(err)
		return object, err
	}

	_ = objectRepool //repool:owned — ownership transfers to caller via returned object

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
	typeObject, typeObjectRepool = sku.GetTransactedPool().GetWithRepool() //repool:suppress ownership transfer via return

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

// IsInlineType resolves whether objects of the given type render their blob
// inline (i.e. the type's blob has binary = false). Resolution is deterministic:
// the type id is looked up to its latest type object via the signature-backed
// stream index, then the type blob is parsed and its binary flag read. This
// replaces the former approximate scheme (a never-populated InlineTypes set plus
// an ids.IsBuiltin name guess) which silently returned false for user types like
// !md on read-only commands.
func (store *Store) IsInlineType(tipe ids.Type) bool {
	if tipe.IsEmpty() {
		return true
	}

	// Builtin types carry no stored type object, so the index cannot resolve
	// them. Treat builtins as inline. NOTE: this is a deliberate seam — if binary
	// builtin types are ever introduced, their binary-ness must be resolved here
	// from the builtin type-blob definition rather than blanket-inlined.
	if ids.IsBuiltin(tipe) {
		return true
	}

	typeObject, err := store.ReadOneObjectId(tipe)
	if err != nil {
		// A non-builtin type that cannot be resolved is a genuinely broken or
		// missing type object, not expected runtime variance (the sig-backed
		// index is deterministic and immutable). Render conservatively as
		// metadata-only rather than dumping an unresolvable blob inline.
		return false
	}

	if typeObject == nil || typeObject.GetBlobDigest().IsNull() {
		return false
	}

	commonBlob, repool, _, err := store.typedBlobStore.Type.ParseTypedBlob(
		typeObject.GetType(),
		typeObject.GetBlobDigest(),
	)
	if err != nil {
		if repool != nil {
			repool()
		}

		return false
	}

	defer repool()

	if commonBlob == nil {
		return false
	}

	return !commonBlob.GetBinary()
}

// TODO https://github.com/amarbel-llc/dodder/issues/27
// Transition to context-based panic/cancel semantics
func (store *Store) ReadOneObjectId(
	objectId domain_interfaces.ObjectId,
) (object *sku.Transacted, err error) {
	if objectId.IsEmpty() {
		return object, err
	}

	object, _ = sku.GetTransactedPool().GetWithRepool() //repool:owned

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
