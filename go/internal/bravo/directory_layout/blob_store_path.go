package directory_layout

import (
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
	mad_directory_layout "github.com/amarbel-llc/madder/go/pkgs/directory_layout"
)

func GetDefaultBlobStore(
	directoryLayout mad_directory_layout.BlobStore,
) mad_directory_layout.BlobStorePath {
	return GetBlobStorePath(
		directoryLayout,
		"default",
	)
}

func GetBlobStorePath(
	directoryLayout mad_directory_layout.BlobStore,
	idString string,
) mad_directory_layout.BlobStorePath {
	return mad_directory_layout.MakeBlobStorePath(
		blob_store_id.MakeWithLocation(
			idString,
			directoryLayout.GetLocationType(),
		),
		DirBlobStore(directoryLayout, idString),
		DirBlobStore(directoryLayout, idString, mad_directory_layout.FileNameBlobStoreConfig),
	)
}
