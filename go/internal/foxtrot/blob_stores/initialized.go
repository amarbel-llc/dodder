package blob_stores

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/delta/blob_store_configs"
)

type BlobStoreInitialized struct {
	blob_store_configs.ConfigNamed
	domain_interfaces.BlobStore
}

func (blobStoreInitialized BlobStoreInitialized) GetBlobStore() domain_interfaces.BlobStore {
	return blobStoreInitialized.BlobStore
}
