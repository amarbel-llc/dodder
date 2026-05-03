package directory_layout

import (
	"fmt"
	"path/filepath"

	mad_directory_layout "github.com/amarbel-llc/madder/go/pkgs/directory_layout"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

func GetBlobStoreConfigPaths(
	ctx interfaces.ActiveContext,
	directoryLayout mad_directory_layout.BlobStore,
) []string {
	globPattern := DirBlobStore(
		directoryLayout,
		fmt.Sprintf("*/%s", mad_directory_layout.FileNameBlobStoreConfig),
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
	layout mad_directory_layout.BlobStore,
	targets ...string,
) interfaces.DirectoryLayoutPath {
	return layout.MakePathBlobStore(targets...)
}

func DirBlobStore(
	layout mad_directory_layout.BlobStore,
	targets ...string,
) string {
	return PathBlobStore(layout, targets...).String()
}
