package sku_fmt

import (
	"bytes"
	"io"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

type TypeBlobStore interface {
	ParseTypedBlob(
		tipe domain_interfaces.ObjectId,
		blobSha mad_domain_interfaces.MarklId,
	) (common type_blobs.Blob, repool interfaces.FuncRepool, n int64, err error)
}

type FuncReadTypeObject func(objects.TypeLock) (*sku.Transacted, error)

type formatterTypFormatterUTIGroups struct {
	funcReadTypeObject FuncReadTypeObject
	store              TypeBlobStore
}

func MakeFormatterTypeFormatterUTIGroups(
	typeReader FuncReadTypeObject,
	typeBlobStore TypeBlobStore,
) *formatterTypFormatterUTIGroups {
	return &formatterTypFormatterUTIGroups{
		funcReadTypeObject: typeReader,
		store:              typeBlobStore,
	}
}

// TODO rewrite as coder
func (format formatterTypFormatterUTIGroups) Format(
	writer io.Writer,
	object *sku.Transacted,
) (n int64, err error) {
	var typeObject *sku.Transacted

	if typeObject, err = format.funcReadTypeObject(
		object.GetTypeLock(),
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	var blob type_blobs.Blob
	var repool interfaces.FuncRepool

	if blob, repool, _, err = format.store.ParseTypedBlob(
		typeObject.GetType(),
		typeObject.GetBlobDigest(),
	); err != nil {
		if repool != nil {
			repool()
		}

		err = errors.Wrap(err)
		return n, err
	}

	defer repool()

	for groupName, group := range blob.GetFormatterUTIGroups() {
		sb := bytes.NewBuffer(nil)

		sb.WriteString(groupName)

		for uti, formatter := range group.Map() {
			sb.WriteString(" ")
			sb.WriteString(uti)
			sb.WriteString(" ")
			sb.WriteString(formatter)
		}

		sb.WriteString("\n")

		if n, err = io.Copy(writer, sb); err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}
