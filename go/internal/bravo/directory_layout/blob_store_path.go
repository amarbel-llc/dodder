package directory_layout

import (
	madder_dl "github.com/amarbel-llc/madder/go/pkgs/directory_layout"

	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
)

type BlobStorePath = madder_dl.BlobStorePath

func MakeBlobStorePath(id blob_store_id.Id, base, config string) BlobStorePath {
	return madder_dl.MakeBlobStorePath(id, base, config)
}

func GetDefaultBlobStore(
	directoryLayout BlobStore,
) BlobStorePath {
	return GetBlobStorePath(
		directoryLayout,
		"default",
	)
}

func GetBlobStorePath(
	directoryLayout BlobStore,
	idString string,
) BlobStorePath {
	return MakeBlobStorePath(
		blob_store_id.MakeWithLocation(
			idString,
			directoryLayout.GetLocationType(),
		),
		DirBlobStore(directoryLayout, idString),
		DirBlobStore(directoryLayout, idString, FileNameBlobStoreConfig),
	)
}

func GetBlobStorePathForCustomPath(
	idString,
	basePath string,
	configPath string,
) BlobStorePath {
	return madder_dl.GetBlobStorePathForCustomPath(
		idString,
		basePath,
		configPath,
	)
}
