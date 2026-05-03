package directory_layout

import (
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
	madder_dl "github.com/amarbel-llc/madder/go/pkgs/directory_layout"
)

func GetDefaultBlobStore(
	directoryLayout madder_dl.BlobStore,
) madder_dl.BlobStorePath {
	return GetBlobStorePath(
		directoryLayout,
		"default",
	)
}

func GetBlobStorePath(
	directoryLayout madder_dl.BlobStore,
	idString string,
) madder_dl.BlobStorePath {
	return madder_dl.MakeBlobStorePath(
		blob_store_id.MakeWithLocation(
			idString,
			directoryLayout.GetLocationType(),
		),
		DirBlobStore(directoryLayout, idString),
		DirBlobStore(directoryLayout, idString, madder_dl.FileNameBlobStoreConfig),
	)
}
