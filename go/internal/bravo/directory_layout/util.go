package directory_layout

import (
	"fmt"
	"path/filepath"

	madder_directory_layout "github.com/amarbel-llc/madder/go/pkgs/directory_layout"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

const FileNameBlobStoreConfig = madder_directory_layout.FileNameBlobStoreConfig

func GetBlobStoreConfigPaths(
	ctx interfaces.ActiveContext,
	directoryLayout BlobStore,
) []string {
	globPattern := DirBlobStore(
		directoryLayout,
		fmt.Sprintf("*/%s", FileNameBlobStoreConfig),
	)

	var configPaths []string

	{
		var err error

		if configPaths, err = filepath.Glob(globPattern); err != nil {
			ctx.Cancel(err)
			return configPaths
		}
	}

	return configPaths
}

func PathBlobStore(
	layout BlobStore,
	targets ...string,
) interfaces.DirectoryLayoutPath {
	return layout.MakePathBlobStore(targets...)
}

func DirBlobStore(
	layout BlobStore,
	targets ...string,
) string {
	return PathBlobStore(layout, targets...).String()
}
