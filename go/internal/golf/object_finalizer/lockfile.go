package object_finalizer

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
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

func (finalizer finalizer) writeReferencedObjectLockIfNecessary(
	metadata objects.MetadataMutable,
	ref ids.SeqId,
	funcs ...sku.FuncReadOne,
) (err error) {
	if ref.IsEmpty() {
		err = ErrEmptyLockKey
		return err
	}

	refLock := metadata.GetReferencedObjectLockMutable(ref)

	if !refLock.GetValue().IsNull() {
		return err
	}

	refObject, repool := sku.GetTransactedPool().GetWithRepool()
	defer repool()

	if ok := sku.ReadOneObjectIdBespoke(ref, refObject, funcs...); ok {
		refLock.GetValueMutable().ResetWithMarklId(refObject.GetMetadataMutable().GetObjectSig())
	} else {
		err = ErrFailedToReadCurrentLockObject
		return err
	}

	return err
}

func (finalizer finalizer) writeBlobReferenceTypeLockIfNecessary(
	metadata objects.MetadataMutable,
	blobId markl.Id,
	funcs ...sku.FuncReadOne,
) (err error) {
	typeLock := metadata.GetBlobReferenceTypeLockMutable(blobId)

	if typeLock == nil {
		err = ErrBlobReferenceMissingType
		return err
	}

	tipe := typeLock.GetKey()

	if tipe.IsEmpty() {
		err = ErrBlobReferenceMissingType
		return err
	}

	if ids.IsBuiltin(tipe) {
		return err
	}

	if !typeLock.GetValue().IsNull() {
		return err
	}

	typeObject, repool := sku.GetTransactedPool().GetWithRepool()
	defer repool()

	if ok := sku.ReadOneObjectIdBespoke(tipe, typeObject, funcs...); ok {
		typeLock.GetValueMutable().ResetWithMarklId(typeObject.GetMetadataMutable().GetObjectSig())
	} else {
		err = errors.Wrapf(
			ErrFailedToReadCurrentLockObject,
			"blob reference type: %q",
			tipe,
		)
		return err
	}

	return err
}
