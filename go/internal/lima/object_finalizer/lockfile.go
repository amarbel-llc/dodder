package object_finalizer

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/hotel/objects"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
)

func (finalizer finalizer) writeTypeLockIfNecessary(
	metadata objects.MetadataMutable,
	tipe ids.Type,
	funcs ...sku.FuncReadOne,
) (err error) {
	if tipe.IsEmpty() {
		err = ErrEmptyLockKey
		return err
	} else if ids.IsBuiltin(tipe) {
		// TODO stop excluding builtin types and create a process for signing those
		// too
		err = ErrBuiltinType
		return err
	}

	typeLock := metadata.GetTypeLockMutable()

	// TODO There are cases where we will want to overwrite the typelock id,
	// should we use CommitOptions?
	if !typeLock.GetValue().IsNull() {
		return err
	}

	typeObject, repool := sku.GetTransactedPool().GetWithRepool()
	defer repool()

	if ok := sku.ReadOneObjectIdBespoke(tipe, typeObject, funcs...); ok {
		typeLock.GetValueMutable().ResetWithMarklId(typeObject.GetMetadataMutable().GetObjectSig())
	} else {
		err = ErrFailedToReadCurrentLockObject
		return err
	}

	return err
}

func (finalizer finalizer) writeTagLockIfNecessary(
	metadata objects.MetadataMutable,
	tag ids.TagStruct, funcs ...sku.FuncReadOne,
) (err error) {
	if tag.IsEmpty() {
		err = ErrEmptyLockKey
		return err
	}

	tagLock := metadata.GetTagLockMutable(tag)

	// TODO There are cases where we will want to overwrite the typelock id,
	// should we use CommitOptions?
	if !tagLock.GetValue().IsNull() {
		return err
	}

	typeObject, repool := sku.GetTransactedPool().GetWithRepool()
	defer repool()

	if ok := sku.ReadOneObjectIdBespoke(tag, typeObject, funcs...); ok {
		tagLock.GetValueMutable().ResetWithMarklId(typeObject.GetMetadataMutable().GetObjectSig())
	} else {
		err = ErrFailedToReadCurrentLockObject
		return err
	}

	return err
}
