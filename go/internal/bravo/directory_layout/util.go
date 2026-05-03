package directory_layout

import (
	"fmt"
	"path/filepath"

	madder_dl "github.com/amarbel-llc/madder/go/pkgs/directory_layout"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

func GetBlobStoreConfigPaths(
	ctx interfaces.ActiveContext,
	directoryLayout madder_dl.BlobStore,
) []string {
	globPattern := DirBlobStore(
		directoryLayout,
		fmt.Sprintf("*/%s", madder_dl.FileNameBlobStoreConfig),
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
	layout madder_dl.BlobStore,
	targets ...string,
) interfaces.DirectoryLayoutPath {
	return layout.MakePathBlobStore(targets...)
}

func DirBlobStore(
	layout madder_dl.BlobStore,
	targets ...string,
) string {
	return PathBlobStore(layout, targets...).String()
}
