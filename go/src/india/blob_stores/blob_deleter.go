package blob_stores

import "code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"

// BlobDeleter is implemented by blob stores that support removing individual
// blobs by their content address. Used by Pack to delete loose blobs after
// they have been safely written to an archive.
type BlobDeleter interface {
	DeleteBlob(id domain_interfaces.MarklId) error
}
