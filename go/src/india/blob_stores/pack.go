package blob_stores

import (
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
)

// PackOptions controls the behavior of the Pack operation.
type PackOptions struct {
	// DeleteLoose causes loose blobs to be deleted after they have been
	// packed into the archive and the archive has been validated.
	DeleteLoose bool

	// DeletionPrecondition is checked before any loose blobs are deleted.
	// When nil, deletion proceeds without additional checks.
	DeletionPrecondition DeletionPrecondition

	// BlobFilter restricts packing to only the specified blob IDs. When nil,
	// all loose blobs not yet in the archive are packed.
	BlobFilter map[string]domain_interfaces.MarklId
}

// PackableArchive is implemented by blob stores that support packing loose
// blobs into archive files.
type PackableArchive interface {
	Pack(options PackOptions) error
}
