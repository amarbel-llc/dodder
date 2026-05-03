package sku_json_fmt

import (
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type WithDormant struct {
	Transacted

	Dormant bool `json:"dormant"`
}

func (json *WithDormant) FromStringAndMetadata(
	objectId string,
	metadata objects.MetadataMutable,
	blobStore mad_domain_interfaces.BlobStore,
) (err error) {
	if err = json.Transacted.FromObjectIdStringAndMetadata(
		objectId,
		metadata,
		blobStore,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	json.Dormant = metadata.GetIndex().GetDormant().Bool()

	return err
}

func (json *WithDormant) FromTransacted(
	object *sku.Transacted,
	blobStore mad_domain_interfaces.BlobStore,
) (err error) {
	return json.FromStringAndMetadata(
		object.GetObjectId().String(),
		object.GetMetadataMutable(),
		blobStore,
	)
}

func (json *WithDormant) ToTransacted(
	object *sku.Transacted,
	blobStore mad_domain_interfaces.BlobStore,
) (err error) {
	if err = json.Transacted.ToTransacted(object, blobStore); err != nil {
		err = errors.Wrap(err)
		return err
	}

	object.GetMetadataMutable().GetIndexMutable().GetDormantMutable().SetBool(json.Dormant)

	return err
}
